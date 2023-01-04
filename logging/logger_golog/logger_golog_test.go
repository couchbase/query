//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package logger_golog

import (
	"fmt"
	"os"
	"testing"

	"github.com/couchbase/query/logging"
)

func logMessages(logger *goLogger) {
	logger.Debugf("This is a message from %s", "Debugf")
	logger.Tracef("This is a message from %s", "Tracef")
	logger.Infof("This is a message from %s", "Infof")
	logger.Warnf("This is a message from %s", "Warnf")
	logger.Errorf("This is a message from %s", "Errorf")
	logger.Severef("This is a message from %s", "Severef")
	logger.Fatalf("This is a message from %s", "Fatalf")

	logging.Debugf("This is a message from %s", "Debugf")
	logging.Tracef("This is a message from %s", "Tracef")
	logging.Infof("This is a message from %s", "Infof")
	logging.Warnf("This is a message from %s", "Warnf")
	logging.Errorf("This is a message from %s", "Errorf")
	logging.Severef("This is a message from %s", "Severef")
	logging.Fatalf("This is a message from %s", "Fatalf")

	logger.Debuga(func() string { return "This is a message from Debuga" })
	logger.Tracea(func() string { return "This is a message from Tracea" })
	logger.Infoa(func() string { return "This is a message from Infoa" })
	logger.Warna(func() string { return "This is a message from Warna" })
	logger.Errora(func() string { return "This is a message from Errora" })
	logger.Severea(func() string { return "This is a message from Severea" })
	logger.Fatala(func() string { return "This is a message from Fatala" })

	logging.Debuga(func() string { return "This is a message from Debuga" })
	logging.Tracea(func() string { return "This is a message from Tracea" })
	logging.Infoa(func() string { return "This is a message from Infoa" })
	logging.Warna(func() string { return "This is a message from Warna" })
	logging.Errora(func() string { return "This is a message from Errora" })
	logging.Severea(func() string { return "This is a message from Severea" })
	logging.Fatala(func() string { return "This is a message from Fatala" })
}

func TestStub(t *testing.T) {
	logger := NewLogger(os.Stdout, logging.DEBUG)
	logging.SetLogger(logger)

	logMessages(logger)

	logger.SetLevel(logging.WARN)
	fmt.Printf("Log level is %s\n", logger.Level())

	logMessages(logger)

	fmt.Printf("Changing to standard formatter\n")
	logger.entryFormatter = &standardFormatter{}
	logger.SetLevel(logging.DEBUG)

	logMessages(logger)
}
