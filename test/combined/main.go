//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/couchbase/query/logging"
	log_resolver "github.com/couchbase/query/logging/resolver"
)

var configFile = flag.String("config", CONFIG, "Configuration file path")

var DB *Database

var InitialQueries []*Query
var Queries []*Query
var IgnoredErrors map[int]bool

var Iterations = -1

var Notifications map[string]interface{}
var LastNotification = time.Now()

var DataFiles []string

func init() {
	setLogger()
}

func setLogger() {
	logger, _ := log_resolver.NewLogger("golog")
	if logger == nil {
		fmt.Printf("Unable to create logger")
		os.Exit(1)
	}
	logging.SetLogger(logger)
	logging.SetLevel(logging.INFO)
}

// to keep the maintenance overhead as low as possible, we just use the JSON directly rather than building and populating
// native types from the configuration, even though this is less efficient at run time
func loadConfig(config string) (map[string]interface{}, error) {
	f, err := os.Open(config)
	if err != nil {
		logging.Errorf("Error opening config.json: %v", err)
		return nil, err
	}
	var c map[string]interface{}
	d := json.NewDecoder(f)
	for d.More() {
		if err = d.Decode(&c); err != nil {
			f.Close()
			logging.Errorf("Error reading config: %v", err)
			return nil, err
		}
	}
	if c == nil {
		logging.Warnf("Empty config")
	}
	f.Close()

	if lf, ok := c["logfile"].(string); ok {
		f, err = os.OpenFile(lf, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			logging.Errorf("Failed to create/open log file \"%s\": %v", lf, err)
		} else {
			logging.Infof("Redirecting output to \"%s\".", lf)
			var old *os.File
			old, os.Stderr = os.Stderr, f
			setLogger()
			os.Stderr = old
		}
	}

	if ll, ok := c["loglevel"]; ok {
		if ls, ok := ll.(string); ok {
			if l, ok, filter := logging.ParseLevel(ls); ok {
				logging.SetLevel(l)
				if l == logging.DEBUG && filter != "" {
					logging.SetDebugFilter(filter)
				}
			}
		}
	}

	if v, ok := c["database"]; !ok {
		err := fmt.Errorf("Invalid configuration: \"database\" element missing.")
		logging.Errorf("%v", err)
		return nil, err
	} else {
		if database, ok := v.(map[string]interface{}); !ok {
			err := fmt.Errorf("Invalid configuration: \"database\" element is not an object.")
			logging.Errorf("%v", err)
			return nil, err
		} else {
			DB, err = NewDatabase(database)
			if err != nil {
				logging.Errorf("Failed to load database definition: %v", err)
				return nil, err
			}
		}
	}

	if v, ok := c["runtime"]; !ok {
		err := fmt.Errorf("Invalid configuration: \"runtime\" element missing.")
		logging.Errorf("%v", err)
		return nil, err
	} else {
		if rt, ok := v.(map[string]interface{}); !ok {
			err := fmt.Errorf("Invalid configuration: \"runtime\" element is not an object.")
			logging.Errorf("%v", err)
			return nil, err
		} else {
			if n, ok := rt["iterations"].(float64); ok {
				Iterations = int(n)
			}
		}
	}

	if v, ok := c["notifications"]; ok {
		if m, ok := v.(map[string]interface{}); !ok {
			err := fmt.Errorf("Invalid configuration: \"notifications\" element is not an object.")
			logging.Errorf("%v", err)
			return nil, err
		} else {
			Notifications = make(map[string]interface{})
			for k, v := range m {
				if k[0] != '#' {
					Notifications[k] = v
				}
			}
		}
	}
	logging.Infof("Configyration loaded from: %s", *configFile)
	return c, nil
}

// loads queries from "location" (if present) and invoked database random query generation if need be
func getQueries(config map[string]interface{}) error {
	if Queries == nil {
		Queries = make([]*Query, 0, 100)
	} else {
		Queries = Queries[:0]
	}
	stmts, ok := config["statements"].(map[string]interface{})
	if !ok {
		err := fmt.Errorf("Invalid configuration: \"statements\" field not found or is not an object.")
		logging.Errorf("%v", err)
		return err
	}

	loc, ok := stmts["location"].(string)
	if ok {
		if qs, err := LoadQueries(loc); err != nil {
			return err
		} else {
			Queries = append(Queries, qs...)
		}
	} else {
		logging.Debugf("\"location\" not found or is not a string.")
	}

	loc, ok = stmts["initial_statements"].(string)
	if ok {
		if qs, err := LoadQueries(loc); err != nil {
			return err
		} else {
			InitialQueries = append(InitialQueries, qs...)
		}
	} else {
		logging.Debugf("\"initial_statements\" not found or is not a string.")
	}

	rs, ok := stmts["random_statements"].(map[string]interface{})
	if ok {
		// Directly adds to the Queries list so generated entries can be used as sub-queries by subsequent generations
		DB.generateQueries(NewRandomRange(rs, 0).get())
	} else {
		logging.Debugf("\"random_statements\" not found or is not an object.")
	}

	loc, ok = stmts["templates"].(string)
	if ok {
		if templates, err := LoadTemplates(loc); err != nil {
			return err
		} else {
			DB.generateQueriesFromTemplates(templates)
		}
	} else {
		logging.Debugf("\"templates\" not found or is not a string.")
	}

	IgnoredErrors = make(map[int]bool)
	ie, ok := stmts["ignore_errors"].([]interface{})
	if ok {
		for i := range ie {
			if f, ok := ie[i].(float64); ok {
				IgnoredErrors[int(f)] = true
			}
		}
	} else {
		logging.Debugf("\"ignore_errors\" not found or is not an array.")
	}

	if len(Queries) == 0 {
		logging.Fatalf("No queries.")
		return fmt.Errorf("No queries.")
	}
	return nil
}

func main() {
	defer func() {
		e := recover()
		if e != nil {
			logging.Fatalf("Panic: %v", e)
		}
	}()

	if runtime.GOOS != "linux" {
		logging.Fatalf("This programme must be run on (Debian based) Linux.")
		os.Exit(-1)
	}

	force := false
	var waitTime time.Duration // no wait on first pass
	for iter := 0; iter != Iterations; iter++ {
		time.Sleep(waitTime)
		// once a day sent a notification as a heartbeat
		if time.Since(LastNotification) >= time.Hour*24 {
			notify(fmt.Sprintf("Test iteration %d starting.", iter))
		}
		waitTime = _ITERATION_INTERVAL
		DataFiles = nil
		DB = nil
		Queries = nil
		// load the config every time so that changes are dynamically picked up
		c, err := loadConfig(*configFile)
		if c == nil {
			reportRunFailure(iter, "Failed to load config.", err)
			return
		}

		if err := DB.addRandomKeyspaces(); err != nil {
			reportRunFailure(iter, "Failed to add random keyspaces.", err)
			continue
		}

		if err := DB.addJoins(); err != nil {
			reportRunFailure(iter, "Failed to add joins.", err)
			continue
		}

		logging.Debuga(func() string {
			b, _ := json.MarshalIndent(DB, "  ", "  ")
			return "Database:\n" + string(b)
		})

		if err := getQueries(c); err != nil {
			reportRunFailure(iter, "Failed to get queries.", err)
			continue
		}

		logging.Infof("Loaded/generated %d queries.", len(Queries))
		if logging.LogLevel() == logging.DEBUG {
			for i := range Queries {
				logging.Debugf("%s", Queries[i].SQL(""))
			}
		}

		if installServer(c, force) != nil {
			force = true
			continue
		}
		force = false
		if err := configureInstance(c); err != nil {
			reportRunFailure(iter, "Failed to configure the instance.", err)
			continue
		}
		if err := DB.create(); err != nil {
			reportRunFailure(iter, "Failed to create the database.", err)
			continue
		}
		if err := DB.populate(); err != nil {
			reportRunFailure(iter, "Failed to populate the database.", err)
			continue
		}
		logging.Debugf("KV+GSI breathing space...")
		time.Sleep(_INIT_WAIT)
		if err := RunPrepSQL(); err != nil {
			reportRunFailure(iter, "Failed to run the preparatory SQL.", err)
			continue
		}
		if err := RunTest(c); err != nil {
			reportRunFailure(iter, "Failed to run the test.", err)
			if strings.Contains(err.Error(), "Interrupted") {
				break
			}
			continue
		}
		logging.Infof("Iteration %v complete.", iter+1)
		var report []interface{}
		for i := range Queries {
			b, _ := json.MarshalIndent(Queries[i], "  ", "  ")
			if Queries[i].reportAsFailed() {
				logging.Errorf("%v", string(b))
				report = append(report, b)
			} else {
				logging.Infof("%v", string(b))
			}
		}
		if len(report) > 0 {
			reportRunFailure(iter, report...)
		}

		cleanupTempDataFiles()
	}
	logging.Infof("Test complete.")
	LastNotification = time.Time{} // force final notification always
	notify("Testing complete.")
}

func reportRunFailure(iter int, args ...interface{}) {
	logging.Fatalf("Iteration %d failed ======================================================", iter)

	for i := range DataFiles {
		logging.Infof("Retaining %s (renamed .keep)", DataFiles[i])
		os.Rename(DataFiles[i], DataFiles[i]+".keep")
	}

	if DB != nil {
		logging.Infoa(func() string {
			b, _ := json.MarshalIndent(DB, "  ", "  ")
			return "Database at failure:\n" + string(b)
		})
	}

	content := make([]interface{}, len(args)+1)
	content[0] = fmt.Sprintf("Iteration %d failed.", iter)
	copy(content[1:], args)
	notify(content...)
}

// runs the initial preparatory SQL statements (if any)
func RunPrepSQL() error {
	if len(InitialQueries) == 0 {
		logging.Infof("No preparatory SQL.")
		return nil
	}
	logging.Infof("Running preparatory SQL.")
	for i := range InitialQueries {
		if err := Queries[i].Execute(nil); err != nil {
			return err
		}
	}
	return nil
}