//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/couchbase/query/logging"
)

// runs random queries until signalled to stop
func worker(id uint, stop chan bool, requestParams map[string]interface{}) {
	var failed int
	for rn := 0; ; rn++ {
		select {
		case <-stop:
			logging.Infof("Worker %d stopping after %d requests (%d failures).", id, rn, failed)
			wg.Done()
			return
		default:
		}

		n := rand.Intn(len(Queries))
		if err := Queries[n].Execute(requestParams); err != nil {
			logging.Tracef("Worker %d: %v - %v", id, Queries[n].SQL(""), err)
			time.Sleep(_WAIT_INTERVAL)
			failed++
		}
	}
}

var wg sync.WaitGroup

func runWorkers(num uint, duration time.Duration, requestParams map[string]interface{}) error {
	var err error
	var queryPID int
	// get the current Query service PID for monitoring
	queryPID, err = getPidOf(_QUERY_PROCESS)
	if err != nil {
		logging.Fatalf("Unable to determine Query service PID.")
		return err
	}

	sig_chan := make(chan os.Signal, 2)
	signal.Notify(sig_chan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sig_chan)

	logging.Infof("Starting %d workers running for up to %v.", num, duration)
	stop := make(chan bool)
	for i := uint(0); i < num; i++ {
		wg.Add(1)
		go worker(i, stop, requestParams)
	}
	ticker := time.NewTicker(_WAIT_INTERVAL)
	defer ticker.Stop()
	start := time.Now()
loop:
	for {
		if time.Since(start) > duration {
			logging.Infof("Signalling workers to stop.")
			close(stop)
			break loop
		}
		if err = syscall.Kill(queryPID, 0); err != nil {
			logging.Fatalf("Detected termination of Query service (PID:%d). Aborting run.", queryPID)
			close(stop)
			break loop
		}
		select {
		case s := <-sig_chan:
			logging.Infof("Signal \"%v\" received. Signalling workers to stop.", s)
			close(stop)
			err = fmt.Errorf("Interrupted.")
			break loop
		case <-ticker.C:
		}
	}
	wg.Wait()
	logging.Infof("Worker run complete.")
	return err
}

func RunTest(config map[string]interface{}) error {
	rt, ok := config["runtime"].(map[string]interface{})
	if !ok {
		err := fmt.Errorf("Invalid configuration: \"runtime\" field not found or is not an object.")
		logging.Errorf("%v", err)
		return err
	}

	numClients := uint(1)
	var err error
	var duration time.Duration
	var requestParams map[string]interface{}

	for k, v := range rt {
		switch k {
		case "clients":
			m, ok := v.(map[string]interface{})
			if !ok {
				err := fmt.Errorf("\"runtime\".\"clients\" is not an object.")
				logging.Errorf("%v", err)
				return err
			}
			numClients = NewRandomRange(m, 1).get()
		case "duration":
			s, ok := v.(string)
			if !ok {
				err := fmt.Errorf("\"runtime\".\"duration\" is not a string.")
				logging.Errorf("%v", err)
				return err
			}
			duration, err = time.ParseDuration(s)
			if err != nil {
				logging.Errorf("Invalid duration: %s", s)
				return err
			}
		case "request":
			m, ok := v.(map[string]interface{})
			if !ok {
				err := fmt.Errorf("\"runtime\".\"request\" is not an object.")
				logging.Errorf("%v", err)
				return err
			}
			for k, v := range m {
				if k[0] != '#' { // allow for "commenting out" elements
					if requestParams == nil {
						requestParams = make(map[string]interface{})
					}
					requestParams[k] = v
				}
			}
		case "iterations":
			// do nothing as used/handled elsewhere
		default:
			if k[0] != '#' {
				logging.Debugf("Runtime field \"%v\" ignored.", k)
			}
		}
	}
	if numClients == 0 {
		logging.Infof("No clients specified.")
		return nil
	}
	if duration == 0 {
		logging.Infof("No duration specified.")
		return nil
	}

	logVitals()
	if err = runWorkers(numClients, duration, requestParams); err != nil {
		logging.Debugf("Run failed with: %v", err)
	} else {
		logVitals()
	}
	return err
}
