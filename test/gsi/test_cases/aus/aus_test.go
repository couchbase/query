//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package aus

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/couchbase/query/migration"
)

func TestAus(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Print("\n\nInserting values into buckets.\n\n")
	runMatch("insert.json", false, false, qc, t)

	fmt.Print("\nTest is waiting for CBO migration to complete.")
	migration.Await("CBO_STATS")

	fmt.Println("Creating indexes.")
	runStmt(qc, "CREATE INDEX product_idx_1 ON product(brand)")
	runStmt(qc, "CREATE INDEX customer_idx_1 ON customer(custId, emailAddress, firstName, age)")
	runStmt(qc, "CREATE INDEX customer_idx_2 ON customer(membership, custId, emailAddress)")

	// Enable and set an AUS schedule that will not run during the test but instead scheduled for the next day.
	// This is because, AUS needs to be enabled so that change information will be collected in CBO statistics
	// that are created during the UPDATE STATISTICS statements executed later in the test
	start := time.Now().Add(24 * time.Hour)
	end := start.Add(30 * time.Minute)
	sched := runStmt(qc, fmt.Sprintf(
		"UPDATE system:aus SET enable=true, "+
			"schedule = {\"start_time\": \"%s\", \"end_time\": \"%s\", \"days\": [\"%s\"], \"timezone\": \"%s\"}",
		start.Format("15:04"), end.Format("15:04"), start.Weekday().String(), time.Local.String()))
	if sched.Err != nil {
		fmt.Printf("\nError creating first AUS schedule: %v", sched.Err)
	}

	// Update CBO statistics
	fmt.Println("Updating CBO statistics.")
	runStmt(qc, "UPDATE STATISTICS FOR shellTest(c1)")
	runStmt(qc, "UPDATE STATISTICS FOR review(id)")
	runStmt(qc, "UPDATE STATISTICS FOR product(brand)")
	runStmt(qc, "UPDATE STATISTICS FOR customer(custId, emailAddress)")
	runStmt(qc, "UPDATE STATISTICS FOR customer(firstName, age) WITH {\"resolution\":0.5}")

	// Adding custom settings for some keyspaces
	runStmt(qc, "INSERT INTO system:aus_settings ( key, value ) VALUES (\"default:review\", {\"enable\": false})")
	runStmt(qc, "INSERT INTO system:aus_settings ( key, value ) VALUES (\"default:customer\", {\"change_percentage\": 10})")

	// Perform some mutations on certain keyspaces
	runStmt(qc, "DELETE FROM product WHERE brand = \"YummyFoods\"")
	runStmt(qc, "UPDATE customer SET firstName = LOWER(firstName) LIMIT 1")

	// Start and end time of the AUS schedule in local timezone
	start = time.Now().Add(time.Minute)
	end = start.Add(30 * time.Minute)

	// Enable and set AUS schedule
	sched = runStmt(qc, fmt.Sprintf(
		"UPDATE system:aus SET enable=true, all_buckets=true, change_percentage=30, "+
			"schedule = {\"start_time\": \"%s\", \"end_time\": \"%s\", \"days\": [\"%s\"], \"timezone\": \"%s\"}",
		start.Format("15:04"), end.Format("15:04"), start.Weekday().String(), time.Local.String()))
	if sched.Err != nil {
		fmt.Printf("\nError creating second AUS schedule: %v", sched.Err)
	}

	// Wait for the task to schedule
	time.Sleep(2 * time.Minute)

	// Check every 1 minute, 10 times to check if the task has completed
	// UPDATE STATISTICS has a default batch timeout of 60 seconds, and the test should result in only 4 batches of stats updates
	// 10 minutes of retrying for the test to complete should ideally be sufficient
	for i := 0; i < 10; i++ {
		stmt := runStmt(qc, "SELECT RAW 1 FROM system:tasks_cache WHERE class = \"auto_update_statistics\" AND state=\"completed\"")
		if len(stmt.Results) > 0 {
			break
		} else if i == 9 {
			fmt.Println("AUS task might not have completed in the test's configured retry period of 10 minutes.")
			break
		} else {
			time.Sleep(time.Minute)
		}
	}

	runMatch("case_aus_tests.json", false, true, qc, t)

	fmt.Println("Performing test cleanup.")
	// Drop all indexes
	runStmt(qc, "DROP INDEX product_idx_1 ON product")
	runStmt(qc, "DROP INDEX customer_idx_1 ON customer")
	runStmt(qc, "DROP INDEX customer_idx_2 ON customer")

	// Delete all CBO statistics in keyspaces used
	runStmt(qc, "UPDATE STATISTICS FOR customer DELETE ALL")
	runStmt(qc, "UPDATE STATISTICS FOR shellTest DELETE ALL")
	runStmt(qc, "UPDATE STATISTICS FOR product DELETE ALL")
	runStmt(qc, "UPDATE STATISTICS FOR review DELETE ALL")

	// Delete documents in keyspaces used
	runStmt(qc, "DELETE FROM customer")
	runStmt(qc, "DELETE FROM shellTest")
	runStmt(qc, "DELETE FROM product")
	runStmt(qc, "DELETE FROM review")

	// Delete AUS coordination documents
	runAdminStmt(qc, "DELETE FROM default:customer._system._query WHERE meta().id LIKE \"aus_coord::%\"")
	runAdminStmt(qc, "DELETE FROM default:shellTest._system._query WHERE meta().id LIKE \"aus_coord::%\"")
	runAdminStmt(qc, "DELETE FROM default:product._system._query WHERE meta().id LIKE \"aus_coord::%\"")
	runAdminStmt(qc, "DELETE FROM default:review._system._query WHERE meta().id LIKE \"aus_coord::%\"")

	// Reset AUS to its default values
	runStmt(qc, "UPDATE system:aus SET enable=false UNSET schedule, change_percentage, all_buckets")

	// Clean up system:aus_settings
	runStmt(qc, "DELETE FROM system:aus_settings")
}
