//  Copyright 2013-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var tiServer = flag.String("engine", "http://localhost:8093/", "URL to the query service(cbq-engine). By default, cbq connects to: http://localhost:8093\n\n Examples:\n\t cbq \n\t\t Connects to local query node. Same as: cbq -engine=http://localhost:8093\n\t cbq -engine=http://172.23.107.18:8093 \n\t\t Connects to query node at 172.23.107.18 Port 8093 \n\t cbq -engine=https://my.secure.node.com:8093 \n\t\t Connects to query node at my.secure.node.com:8093 using secure https protocol.\n")

var quietFlag = flag.Bool("quiet", false, "Enable/Disable startup connection message for the shell \n\t\t Default : false \n\t\t Possible Values : true/false \n")

func main() {
	flag.Parse()
	if strings.HasSuffix(*tiServer, "/") == false {
		*tiServer = *tiServer + "/"
	}
	if !*quietFlag {
		fmt.Printf("Couchbase query shell connected to %v . Type Ctrl-D to exit.\n", *tiServer)
	}
	HandleInteractiveMode(*tiServer, filepath.Base(os.Args[0]))
}

var transport = &http.Transport{MaxIdleConnsPerHost: 1}

// FIXME we really need a timeout here
var client = &http.Client{Transport: transport}

func execute_internal(tiServer, line string, w *os.File) error {

	url := tiServer + "query"
	if strings.HasPrefix(url, "https") {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	resp, err := client.Post(url, "text/plain", strings.NewReader(line))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(w, resp.Body)
	w.WriteString("\n")
	w.Sync()

	return nil
}
