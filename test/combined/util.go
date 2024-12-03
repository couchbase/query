//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/couchbase/query/logging"
)

// "globally" unique number (primarily for unique entity naming)
var _serialNum int

func nextSerial() int {
	_serialNum++
	return _serialNum
}

// this is a copy of the algebra package function, included literally here so as to not add the dependency on sigar.h that importing
// the algebra package entails
func parsePath(path string) []string {
	elements := []string{}
	hasNamespace := false
	start := 0
	end := -1
	inBackTicks := false
	for i, c := range path {
		switch c {
		case '`':
			inBackTicks = !inBackTicks
			if inBackTicks {
				start = i + 1
			} else {
				end = i
			}
		case ':':
			if inBackTicks {
				continue
			}
			if end != i-1 || end == -1 {
				end = i
			}
			elements = append(elements, path[start:end])
			start = i + 1
			end = start
			hasNamespace = true
		case '.':
			if inBackTicks {
				continue
			}
			if !hasNamespace {
				elements = append(elements, "")
				hasNamespace = true
			}
			if end != i-1 || end == -1 {
				end = i
			}
			elements = append(elements, path[start:end])
			start = i + 1
			end = start
		}
	}
	if !hasNamespace {
		elements = append(elements, "")
	}
	if start < len(path) {
		if start < end {
			elements = append(elements, path[start:end])
		} else {
			elements = append(elements, path[start:])
		}
	}
	return elements
}

// handles a GET request to a NODE endpoint that requires authorization
func doNodeGet(uri string) (*http.Response, error) {
	u, _ := url.JoinPath(_NODE_URL, uri)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Combined test")
	req.SetBasicAuth(USER, PASSWORD)
	return http.DefaultClient.Do(req)
}

// handles posting to a management REST endpoint (application/x-www-form-urlencoded)
func doNodePost(uri string, params map[string]interface{}, body bool) (int, []byte, error) {
	postData := url.Values{}
	for k, v := range params {
		if v == nil {
			postData.Del(k)
		} else {
			postData.Set(k, fmt.Sprintf("%v", v))
		}
	}
	u, _ := url.JoinPath(_NODE_URL, uri)
	req, err := http.NewRequest("POST", u, bytes.NewBufferString(postData.Encode()))
	if err != nil {
		return -1, nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Combined test")
	req.SetBasicAuth(USER, PASSWORD)
	var resp *http.Response
	for retry := 0; retry < _RETRY_COUNT; retry++ {
		resp, err = http.DefaultClient.Do(req)
		if err == nil || !isHttpConnError(err) {
			break
		}
		logging.Debugf("Retrying: %d: %v", retry, err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return -1, nil, err
	}
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusBadRequest && strings.Contains(string(b), "already exists") {
			err = os.ErrExist
		} else {
			err = fmt.Errorf("%s", string(b))
		}
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, b, nil
}

// handles a GET request to a Query endpoint that requires authorization
func doQueryGet(uri string, params map[string]interface{}) (*http.Response, error) {
	args := url.Values{}
	for k, v := range params {
		args.Set(k, fmt.Sprintf("%v", v))
	}
	u, _ := url.Parse(_QUERY_URL)
	u.Path = uri
	u.RawQuery = args.Encode()
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Combined test")
	req.SetBasicAuth(USER, PASSWORD)
	return http.DefaultClient.Do(req)
}

// handles posting to a query REST endpoint (application/json)
func doQueryPost(uri string, data map[string]interface{}, body bool) (int, []byte, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return -1, nil, err
	}
	u, _ := url.JoinPath(_QUERY_URL, uri)
	req, err := http.NewRequest("POST", u, bytes.NewReader(b))
	if err != nil {
		return -1, nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Combined test")
	req.SetBasicAuth(USER, PASSWORD)
	var resp *http.Response
	for retry := 0; retry < _RETRY_COUNT; retry++ {
		resp, err = http.DefaultClient.Do(req)
		if err == nil || !isHttpConnError(err) {
			break
		}
		logging.Debugf("Retrying: %d: %v", retry, err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return -1, nil, err
	}
	b, _ = io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusBadRequest && strings.Contains(string(b), "already exists") {
			err = os.ErrExist
		} else {
			err = fmt.Errorf("%s", string(b))
		}
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, b, nil
}

// checks for common connection error responses
func isHttpConnError(err error) bool {
	estr := strings.ToLower(err.Error())
	return strings.Contains(estr, "broken pipe") ||
		strings.Contains(estr, "broken connection") ||
		strings.Contains(estr, "connection refused") ||
		strings.Contains(estr, "connection reset")
}

// waits if there is an issue accessing the URL given
func checkWait(uri string, msg string) bool {
	for retry := 0; retry <= _WAIT_COUNT; retry++ {
		_, err := http.Get(uri)
		if err != nil {
			if msg != "" {
				logging.Infof(msg)
				msg = ""
				time.Sleep(_INIT_WAIT)
			} else {
				time.Sleep(_RETRY_WAIT)
			}
		} else {
			return true
		}
	}
	logging.Fatalf("%v unavailable after %d attempts", uri, _WAIT_COUNT)
	return false
}

// uses the bucket management REST API to create a bucket
func createBucket(name string, config map[string]interface{}) error {
	var err error
	var resp *http.Response
	config["name"] = name
	status, _, err := doNodePost("/pools/default/buckets", config, false)
	if status == http.StatusAccepted {
		// check availability before returning
		time.Sleep(_INIT_WAIT)
		target := fmt.Sprintf("/pools/default/buckets/%s", name)
		//logging.DBG(target)
		for retry := 1; retry <= _RETRY_COUNT; retry++ {
			logging.Debugf("Checking bucket availability: %d/%d", retry, _RETRY_COUNT)
			resp, err = doNodeGet(target)
			if err != nil || resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				break
			}
			logging.DBG("%v", resp.Status)
			if resp != nil && resp.Body != nil {
				x, _ := io.ReadAll(resp.Body)
				logging.DBG("%v", string(x))
				resp.Body.Close()
			}
			time.Sleep(_RETRY_WAIT)
		}
	}
	return err
}

// uses the bucket management REST API to alter a bucket
func alterBucket(name string, config map[string]interface{}) error {
	var err error
	var resp *http.Response
	status, _, err := doNodePost(fmt.Sprintf("/pools/default/buckets/%s", url.PathEscape(name)), config, false)
	if status == http.StatusAccepted {
		// check availability before returning
		time.Sleep(_INIT_WAIT)
		target := fmt.Sprintf("/pools/default/buckets/%s", name)
		//logging.DBG(target)
		for retry := 1; retry <= _RETRY_COUNT; retry++ {
			logging.Debugf("Checking bucket availability: %d/%d", retry, _RETRY_COUNT)
			resp, err = doNodeGet(target)
			if err != nil || resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				break
			}
			logging.DBG("%v", resp.Status)
			if resp != nil && resp.Body != nil {
				x, _ := io.ReadAll(resp.Body)
				logging.DBG("%v", string(x))
				resp.Body.Close()
			}
			time.Sleep(_RETRY_WAIT)
		}
	}
	return err
}

// uses the bucket management REST API to drop a bucket
func dropBucket(name string) error {
	var err error
	var resp *http.Response
	uri, _ := url.JoinPath(_NODE_URL, "/pools/default/buckets", name)
	req, err := http.NewRequest("DELETE", uri, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Combined test")
	req.SetBasicAuth(USER, PASSWORD)
	for retry := 0; retry < _RETRY_COUNT; retry++ {
		resp, err = http.DefaultClient.Do(req)
		if err == nil || !isHttpConnError(err) {
			break
		}
		logging.Debugf("Retrying: %s %d: %v", uri, retry, err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return os.ErrNotExist
		} else {
			b, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("%s", string(b))
		}
	}
	return nil
}

// uses the scopes & collections management REST API to create a scope
func createScope(bucket string, name string) error {
	if name == "_default" {
		return os.ErrExist
	}
	var err error
	var resp *http.Response
	target := fmt.Sprintf("/pools/default/buckets/%s/scopes", url.PathEscape(bucket))
	status, _, err := doNodePost(target, map[string]interface{}{"name": name}, false)
	if status == http.StatusAccepted || status == http.StatusOK {
		// check availability before returning
		time.Sleep(_RETRY_WAIT) // less initial waiting that for a bucket
		for retry := 1; retry <= _RETRY_COUNT; retry++ {
			logging.Debugf("Checking scope availability: %d/%d", retry, _RETRY_COUNT)
			resp, err = doNodeGet(target)
			if err != nil {
				if resp != nil && resp.Body != nil {
					resp.Body.Close()
				}
				break
			}
			var b []byte
			if resp != nil && resp.Body != nil {
				if resp.StatusCode == http.StatusOK {
					b, _ = io.ReadAll(resp.Body)
				}
				resp.Body.Close()
				if b != nil {
					var res map[string]interface{}
					if json.Unmarshal(b, &res) == nil {
						if s, ok := res["scopes"].([]interface{}); ok {
							for i := range s {
								if sc, ok := s[i].(map[string]interface{}); ok {
									if sc["name"] == name {
										return nil
									}
								}
							}
						}
					}
				}
			}
			time.Sleep(_RETRY_WAIT)
		}
	}
	return err
}

// uses the scopes & collections management REST API to create a collection
func createCollection(bucket string, scope string, name string) error {
	var err error
	var resp *http.Response
	target := fmt.Sprintf("/pools/default/buckets/%s/scopes/%s/collections", url.PathEscape(bucket),
		url.PathEscape(scope))
	status, _, err := doNodePost(target, map[string]interface{}{"name": name}, false)
	if status == http.StatusAccepted || status == http.StatusOK {
		// check availability before returning
		time.Sleep(_RETRY_WAIT) // less initial waiting that for a bucket
		target = fmt.Sprintf("/pools/default/buckets/%s/scopes", url.PathEscape(bucket))
		for retry := 1; retry <= _RETRY_COUNT; retry++ {
			logging.Debugf("Checking collection availability: %d/%d", retry, _RETRY_COUNT)
			resp, err = doNodeGet(target)
			if err != nil {
				if resp != nil && resp.Body != nil {
					resp.Body.Close()
				}
				break
			}
			var b []byte
			if resp != nil && resp.Body != nil {
				if resp.StatusCode == http.StatusOK {
					b, _ = io.ReadAll(resp.Body)
				}
				resp.Body.Close()
				if b != nil {
					var res map[string]interface{}
					if json.Unmarshal(b, &res) == nil {
						if s, ok := res["scopes"].([]interface{}); ok {
							for i := range s {
								if sc, ok := s[i].(map[string]interface{}); ok {
									if sc["name"] == scope {
										if c, ok := sc["collections"].([]interface{}); ok {
											for j := range c {
												if col, ok := c[j].(map[string]interface{}); ok {
													if col["name"] == name {
														return nil
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
			time.Sleep(_RETRY_WAIT)
		}
	}
	return err
}

// drops all existing buckets
// attempts to be slightly space efficient by allowing for streaming of the results
func purgeKeyspaces() error {
	resp, err := doNodeGet("/pools/default/buckets")
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%v", resp.Status)
	}
	if resp.Header.Get("Content-Type") != "application/json" {
		return fmt.Errorf("Invalid content: %v", resp.Header.Get("Content-Type"))
	}
	dec := json.NewDecoder(resp.Body)
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if r, ok := tok.(json.Delim); !ok || r != '[' {
		return fmt.Errorf("Unexpected JSON content.")
	}
	for dec.More() {
		var m map[string]interface{}
		err = dec.Decode(&m)
		if err != nil {
			return err
		}
		if n, ok := m["name"].(string); !ok {
			logging.Warnf("Received bucket information without a valid name element.")
		} else {
			err = dropBucket(n)
			if err != nil {
				logging.Warnf("Failed to drop bucket `%s`: %v", n, err)
			} else {
				logging.Infof("Dropped bucket `%s`.", n)
			}
		}
	}
	tok, err = dec.Token()
	if err != nil {
		return err
	}
	if r, ok := tok.(json.Delim); !ok || r != ']' {
		return fmt.Errorf("Unexpected JSON content.")
	}
	return nil
}

var partialImport = regexp.MustCompile("import failed: ([0-9]+) documents were imported, ([0-9]+) documents")

// imports data into the specified keyspace from the provided file using cbimport
func importData(keyspace string, file string) error {
	logging.Infof("Importing \"%s\"...", file)
	parts := parsePath(keyspace)
	args := []string{"json", "--format", "lines", "-c", _NODE_URL, "-u", USER, "-p", PASSWORD, "-g", "%type%::#MONO_INCR#", "-d",
		fmt.Sprintf("file://%s", file), "-b", parts[1]}
	if len(parts) == 4 {
		args = append(args, "--scope-collection-exp")
		args = append(args, fmt.Sprintf("%s.%s", parts[2], parts[3]))
	}
	ic := exec.Command("/opt/couchbase/bin/cbimport", args...)
	sb := &strings.Builder{}
	ic.Stdout = sb
	err := ic.Run()
	output := sb.String()
	if err != nil {
		if strings.Contains(output, "import failed") {
			if m := partialImport.FindStringSubmatch(output); len(m) == 3 {
				if success, cerr := strconv.Atoi(m[1]); cerr == nil && success > 0 {
					err = nil
					output = fmt.Sprintf("Documents imported: %s Documents failed: %s", m[1], m[2])
				}
			}
		}
	}
	if err != nil {
		logging.Errorf("cbimport %v", strings.Join(args, " "))
		for _, s := range strings.Split(output, "\n") {
			logging.Errorf(">   %v", s)
		}
		logging.Errorf(">   %v", err)
		return err
	}
	if strings.Contains(output, "import failed") {
		if m := partialImport.FindStringSubmatch(output); len(m) == 3 {
			if success, err := strconv.Atoi(m[1]); err != nil || success == 0 {
				return fmt.Errorf("%s", output)
			}
			// consider it successful if some documents were imported
			logging.Infof("%s documents were imported, %s documents failed to be imported", m[1], m[2])
		} else {
			return fmt.Errorf("%s", output)
		}
	} else {
		logging.Infoa(func() string {
			n := strings.Index(sb.String(), "Documents imported:")
			if n == -1 {
				n = 0
			}
			return keyspace + ": " + sb.String()[n:]
		})
	}
	return nil
}

// deletes all data from a keyspace using SQL++
func cleanupKeyspace(name string) error {
	logging.Debugf("Deleting data from %s.", name)
	params := map[string]interface{}{
		"statement": fmt.Sprintf("DELETE FROM %s", name),
	}
	if _, _, err := doNodePost("/_p/query/query/service", params, false); err != nil {
		return err
	}

	params["statement"] = "SELECT RAW name FROM system:indexes WHERE " +
		"NVL2(bucket_id, bucket_id||'.'||scope_id||'.'||keyspace_id, keyspace_id) = $ks"
	params["$ks"] = "\"" + name + "\""
	params["signature"] = false
	params["metrics"] = false
	params["loglevel"] = "info"
	_, b, err := doNodePost("/_p/query/query/service", params, true)
	if err != nil {
		return err
	}

	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	if err != nil {
		return err
	}
	if res, ok := m["results"]; !ok {
		return fmt.Errorf("Invalid response - no \"results\" field.")
	} else if ra, ok := res.([]interface{}); !ok {
		return fmt.Errorf("Invalid response - \"results\" field is not an array.")
	} else {
		for i := range ra {
			iname, ok := ra[i].(string)
			if !ok {
				return fmt.Errorf("Invalid response - \"results\"[%d] is not a string.", i)
			}
			if err = executeSQLWithoutResults(fmt.Sprintf("DROP INDEX %s ON %s", iname, name), nil, false); err != nil {
				return err
			}
		}
	}
	return nil
}

// issues an SQL statement and doesn't care about the returned results (only the succes/failure)
func executeSQLWithoutResults(stmt string, params map[string]interface{}, logResults bool) error {
	logging.Infof("%s", stmt)
	if params == nil {
		params = make(map[string]interface{})
	}
	params["statement"] = stmt
	// using the proxy here just to test another aspect
	_, body, err := doNodePost("/_p/query/query/service", params, logResults)
	//resp, _, err := doQueryPost("/query/service", params, false)
	if err == nil && logResults {
		for _, l := range strings.Split(string(body), "\n") {
			logging.Infof("> %v", l)
		}
	}
	return err
}

// issues an SQL statement and streams the results; returns the resultCount, elapsedTime and an array of error codes (may be empty)
func executeSQLProcessingResults(stmt string, params map[string]interface{}) (int, time.Duration, []int, error) {
	//logging.Debugf("%s", stmt)
	postData := url.Values{}
	postData.Set("signature", "false")
	postData.Set("statement", stmt)
	for k, v := range params {
		if v == nil {
			postData.Del(k)
		} else {
			postData.Set(k, fmt.Sprintf("%v", v))
		}
	}
	u, _ := url.JoinPath(_QUERY_URL, "/query/service")
	req, err := http.NewRequest("POST", u, bytes.NewBufferString(postData.Encode()))
	if err != nil {
		return -1, 0, nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Combined test")
	req.SetBasicAuth(USER, PASSWORD)
	var resp *http.Response
	for retry := 0; retry < _RETRY_COUNT; retry++ {
		resp, err = http.DefaultClient.Do(req)
		if err == nil || !isHttpConnError(err) {
			break
		}
		logging.Debugf("Retrying: %d: %v", retry, err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return -1, 0, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusBadRequest && strings.Contains(string(b), "already exists") {
			err = os.ErrExist
		} else {
			err = fmt.Errorf("Status: %v. %s", resp.Status, string(b))
		}
		return -1, 0, nil, err
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		return -1, 0, nil, fmt.Errorf("Invalid content: %v", resp.Header.Get("Content-Type"))
	}

	// stream the results
	var elapsed time.Duration
	var errs []int
	var results int

	dec := json.NewDecoder(resp.Body)
	tok, err := dec.Token()
	if err != nil {
		return -1, 0, nil, err
	}
	if r, ok := tok.(json.Delim); !ok || r != json.Delim('{') {
		return -1, 0, nil, fmt.Errorf("Unexpected JSON content: missing opening '{'.")
	}
	for dec.More() { // top-level field processing
		f, err := dec.Token()
		if err != nil {
			return -1, 0, nil, err
		}
		fn, ok := f.(string)
		if !ok {
			return -1, 0, nil, fmt.Errorf("Invalid type for field name: %T", f)
		}
		switch fn {
		case "errors":
			// read the errors as a single object
			var i interface{}
			if err = dec.Decode(&i); err != nil {
				return -1, 0, nil, err
			}
			if ai, ok := i.([]interface{}); !ok {
				return -1, 0, nil, fmt.Errorf("Invalid type for errors field: %T", i)
			} else {
				for n := range ai {
					if m, ok := ai[n].(map[string]interface{}); ok {
						if c, ok := m["code"].(float64); ok {
							errs = append(errs, int(c))
						}
					}
				}
			}
		case "metrics":
			// read the metrics as a single object
			var i interface{}
			if err = dec.Decode(&i); err != nil {
				return -1, 0, nil, err
			}
			if m, ok := i.(map[string]interface{}); !ok {
				return -1, 0, nil, fmt.Errorf("Invalid type for metrics field: %T", i)
			} else {
				if e, ok := m["elapsedTime"].(string); !ok {
					return -1, 0, nil, fmt.Errorf("Metrics elapsedTime field missing or invalid")
				} else {
					elapsed, err = time.ParseDuration(e)
					if err != nil {
						return -1, 0, nil, fmt.Errorf("Metrics elapsedTime is invalid: %v", err)
					}
				}
				if r, ok := m["resultCount"].(float64); !ok {
					return -1, 0, nil, fmt.Errorf("Metrics resultCount field missing or invalid")
				} else {
					results = int(r)
				}
			}
		default:
			// all other fields are streamed and discarded
			nesting := 0
			for {
				tok, err = dec.Token()
				if err != nil {
					return -1, 0, nil, err
				}
				if jd, ok := tok.(json.Delim); ok {
					if jd == json.Delim('{') || jd == json.Delim('[') {
						nesting++
					} else if jd == json.Delim('}') || jd == json.Delim(']') {
						// don't have to care about mis-matching closing tokens; Token() will raise an error if invalid/missing
						nesting--
					}
				}
				if nesting == 0 {
					break
				}
			}
		}
	}
	tok, err = dec.Token()
	if err != nil {
		return -1, 0, nil, err
	}
	if r, ok := tok.(json.Delim); !ok || r != json.Delim('}') {
		return -1, 0, nil, fmt.Errorf("Unexpected JSON content: missing closing '}'.")
	}

	return results, elapsed, errs, nil
}

// gets the PID of the given process name
func getPidOf(procName string) (int, error) {
	var pid int
	args := []string{"--no-headers", "-o", "pid", "-C", procName}
	ic := exec.Command("/usr/bin/ps", args...)
	sb := &strings.Builder{}
	ic.Stdout = sb
	err := ic.Run()
	output := sb.String()
	if err == nil {
		pid, err = strconv.Atoi(strings.TrimSpace(output))
	}
	if err != nil {
		logging.Errorf("ps %v", strings.Join(args, " "))
		for _, s := range strings.Split(output, "\n") {
			logging.Errorf(">   %v", s)
		}
		logging.Errorf(">   %v", err)
		return -1, err
	}
	return pid, nil
}

func logVitals() {
	/* SQL alternative
	rParams := map[string]interface{}{"signature":false,"metrics":false,"pretty":true}
	if err = executeSQLWithoutResults("SELECT * FROM system:vitals", rParams, true); err != nil {
		logging.Debugf("Failed to gather vitals information before run.")
	}
	*/
	params := map[string]interface{}{"pretty": true}
	resp, err := doQueryGet("/admin/vitals", params)
	if err != nil || resp == nil || resp.Body == nil {
		logging.Debugf("Failed to read vitals: %v", err)
		return
	}
	logging.Infof("Vitals:")
	r := bufio.NewReader(resp.Body)
	for {
		l, err := r.ReadString('\n')
		if err == nil || len(l) > 0 {
			logging.Infof("> %s", l)
		}
		if err != nil {
			return
		}
	}
}

func notify(bodyContent ...interface{}) {
	if len(Notifications) == 0 {
		return
	}
	min, ok := Notifications["min_interval"].(string)
	if ok {
		d, err := time.ParseDuration(min)
		if err == nil {
			if time.Since(LastNotification) <= d {
				return
			}
		} else {
			logging.Warnf("Invalid \"min_interval\": %v", err)
		}
	}
	server, ok := Notifications["smtp_server"].(string)
	if !ok {
		return
	}
	port, ok := Notifications["smtp_port"]
	if !ok {
		return
	}
	if _, ok = port.(float64); !ok {
		if _, ok = port.(string); !ok {
			return
		}
	}
	var auth smtp.Auth
	user, ok := Notifications["smtp_user"].(string)
	if ok {
		pwd, ok := Notifications["smtp_password"].(string)
		if ok {
			auth = smtp.PlainAuth("", user, pwd, server)
		}
	}
	r, ok := Notifications["receipients"]
	if !ok {
		return
	}
	ra, ok := r.([]interface{})
	if !ok {
		if _, ok = r.(string); !ok {
			return
		}
		ra = []interface{}{r}
	}
	to := make([]string, 0, len(ra))
	for i := range ra {
		if s, ok := ra[i].(string); ok && s[0] != '#' {
			to = append(to, s)
		}
	}
	if len(to) == 0 {
		return
	}
	subject, ok := Notifications["subject"].(string)
	if !ok {
		subject = "Combined test notification"
	}
	var body bytes.Buffer
	body.WriteString("To: ")
	for i := range to {
		if i > 0 {
			body.WriteRune(',')
		}
		body.WriteString(to[i])
	}
	body.WriteString("\r\nSubject: ")
	body.WriteString(subject)
	body.WriteString("\r\n\r\n")
	for i := range bodyContent {
		switch t := bodyContent[i].(type) {
		case string:
			body.WriteString(t)
		case []byte:
			body.Write(t)
		default:
			body.WriteString(fmt.Sprintf("%v", t))
		}
		body.WriteRune('\n')
	}
	if err := smtp.SendMail(fmt.Sprintf("%s:%v", server, port), auth, _EMAIL_FROM, to, body.Bytes()); err != nil {
		logging.Warnf("Failed to send e-mail notification: %v", err)
	}
	LastNotification = time.Now()
}

func cleanupTempDataFiles() {
	dir := os.TempDir()
	d, err := os.Open(dir)
	if err != nil {
		logging.Debugf("Failed to open temp directory %s: %v", dir, err)
		return
	}
	for {
		ents, err := d.ReadDir(10)
		if err == nil {
			for i := range ents {
				if strings.HasPrefix(ents[i].Name(), "import_data_") && !strings.HasSuffix(ents[i].Name(), ".keep") {
					os.Remove(path.Join(dir, ents[i].Name()))
				}
			}
		}
		if err != nil || len(ents) < 10 {
			break
		}
	}
	d.Close()
}
