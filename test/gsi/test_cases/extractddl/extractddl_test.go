//  Copyright 2025-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package extractddl

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestExtractDDL(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Println("\n\nSetting up test infrastructure in customer bucket \n\n ")

	// Create a test index on the default collection (which should exist)
	runStmt(qc, "CREATE INDEX test_index ON customer._default._default(name);")

	// Create a sequence statement
	runStmt(qc, "CREATE SEQUENCE `customer`.`_default`.`test_sequence` START WITH 100 INCREMENT BY 5 CACHE 10;")

	// Create a prepared statement
	runStmt(qc, "PREPARE test_prepared_stmt AS SELECT * FROM customer WHERE name = $1;")

	// Create a test JavaScript function
	runStmt(qc, "CREATE FUNCTION test_func() LANGUAGE JAVASCRIPT AS 'function test_func() { return \"hello\"; }';")

	// Create a test inline function
	runStmt(qc, "CREATE FUNCTION test_inline_func(x) {x * 2};")

	runStmt(qc, "CREATE OR REPLACE FUNCTION add_numbers(a, b) LANGUAGE INLINE AS a + b;")

	// Create a scoped function within the customer bucket
	runStmt(qc, "CREATE OR REPLACE FUNCTION `customer`.`_default`.`scoped_multiply`(x, y) LANGUAGE INLINE AS x * y;")

	// Create a variadic function
	runStmt(qc, "CREATE OR REPLACE FUNCTION variadic_func(...) LANGUAGE INLINE AS args[0] + args[1];")

	// Create a function with no parameters
	runStmt(qc, "CREATE OR REPLACE FUNCTION no_param_func() LANGUAGE INLINE AS 42;")

	// Create an external JavaScript function
	runStmt(qc, "CREATE OR REPLACE FUNCTION ejs1() LANGUAGE JAVASCRIPT AS \"ej1\" AT \"lib1\";")

	// Test using JSON test case file
	runMatch("case_extractddl_basic.json", false, true, qc, t)

	// Clean up
	runStmt(qc, "DROP SEQUENCE customer._default.test_sequence IF EXISTS")
	runStmt(qc, "DEALLOCATE test_prepared_stmt")
	runStmt(qc, "DROP FUNCTION test_func IF EXISTS")
	runStmt(qc, "DROP FUNCTION test_inline_func IF EXISTS")
	runStmt(qc, "DROP FUNCTION add_numbers IF EXISTS")
	runStmt(qc, "DROP FUNCTION customer._default.scoped_multiply IF EXISTS")
	runStmt(qc, "DROP FUNCTION variadic_func IF EXISTS")
	runStmt(qc, "DROP FUNCTION no_param_func IF EXISTS")
	runStmt(qc, "DROP FUNCTION ejs1 IF EXISTS")
	runStmt(qc, "DROP INDEX test_index ON customer._default._default IF EXISTS")

	fmt.Println("\n\nExtractDDL test completed \n\n ")
}
