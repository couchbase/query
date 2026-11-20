//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package couchbase

import (
	"testing"
)

// newTestTransactionMutations returns transaction mutations with logSize
// values per log, holding one savepoint marker per name.
func newTestTransactionMutations(t *testing.T, logSize int, savepoints ...string) *TransactionMutations {
	t.Helper()
	tm, err := NewTransactionMutations(false, 0)
	if err != nil {
		t.Fatalf("NewTransactionMutations: %v", err)
	}
	tm.logSize = logSize
	for _, s := range savepoints {
		if err = tm.SetSavepoint(s); err != nil {
			t.Fatalf("SetSavepoint(%s): %v", s, err)
		}
	}
	return tm
}

func undoToSavepoint(t *testing.T, tm *TransactionMutations, sname string) {
	t.Helper()
	slog, sindex, undo, err := tm.GetSavepointRange(sname)
	if err != nil || !undo {
		t.Fatalf("GetSavepointRange(%s): undo %v, err %v", sname, undo, err)
	}
	if err = tm.UndoLog(slog, sindex); err != nil {
		t.Fatalf("UndoLog: %v", err)
	}
}

func assertLogValuesCleared(t *testing.T, name string, s TransactionLogValues, from int) {
	t.Helper()
	for i, v := range s[from:] {
		if v != nil {
			t.Fatalf("%s: slot %d still points at a pooled log value", name, from+i)
		}
	}
}

func assertLogsCleared(t *testing.T, logs []*TransactionLog, from int) {
	t.Helper()
	for i, tl := range logs[from:] {
		if tl != nil {
			t.Fatalf("logs: slot %d still points at a dropped log", from+i)
		}
	}
}

// Rolling back to a savepoint pools the undone log values; the log slices must
// not keep pointers to them, whether the slice is kept (the savepoint's log)
// or pooled (later logs), and the list of logs must not keep the later logs.
func TestUndoLogClearsRolledBackLogValues(t *testing.T) {
	// two values per log: s1, s2 in logs[0], s3, s4 in logs[1]
	tm := newTestTransactionMutations(t, 2, "s1", "s2", "s3", "s4")
	if len(tm.logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(tm.logs))
	}

	logs := tm.logs[:cap(tm.logs)]
	startLog := tm.logs[0].logValues[:cap(tm.logs[0].logValues)]
	laterLog := tm.logs[1].logValues[:cap(tm.logs[1].logValues)]

	undoToSavepoint(t, tm, "s2")

	// s2 and later were undone, s1 is kept
	if _, ok := tm.savepoints["s1"]; !ok || len(tm.savepoints) != 1 {
		t.Fatalf("expected only savepoint s1 to remain, got %v", tm.savepoints)
	}
	if len(tm.logs) != 1 || tm.logs[0].Len() != 1 {
		t.Fatalf("expected 1 log with 1 value, got %d logs", len(tm.logs))
	}

	assertLogValuesCleared(t, "savepoint log", startLog, 1)
	// already cleared by TransactionLogValuesPool.Put (MB-45898): kept as a regression guard
	assertLogValuesCleared(t, "later log", laterLog, 0)
	assertLogsCleared(t, logs, 1)
}

// Rolling back while a statement is writing to a keyspace drops that
// statement's logs without replaying them; the list of logs must not keep
// pointers to them.
func TestUndoLogClearsDroppedKeyspaceLogs(t *testing.T) {
	// s1, s2 in logs[0], then the statement's keyspace marker and k1 in
	// logs[1], k2, k3 in logs[2] and k4 in logs[3]
	tm := newTestTransactionMutations(t, 2, "s1", "s2")
	for _, k := range []string{"k1", "k2", "k3", "k4"} {
		if _, err := tm.Add(MOP_INSERT, "ks", "b", "s", "c", "", 0, k, nil, 0,
			MV_FLAGS_WRITE, 0, nil, nil, nil, 0); err != nil {
			t.Fatalf("Add(%s): %v", k, err)
		}
	}
	if len(tm.logs) != 4 || tm.curKeyspace != "ks" {
		t.Fatalf("expected 4 logs and current keyspace ks, got %d logs, keyspace %q",
			len(tm.logs), tm.curKeyspace)
	}

	logs := tm.logs[:cap(tm.logs)]

	undoToSavepoint(t, tm, "s2")

	if len(tm.logs) != 1 || tm.logs[0].Len() != 1 {
		t.Fatalf("expected 1 log with 1 value, got %d logs", len(tm.logs))
	}
	assertLogsCleared(t, logs, 1)
}

// DeleteAll pools every log's values; the list of logs must not keep the logs.
func TestDeleteAllClearsLogs(t *testing.T) {
	tm := newTestTransactionMutations(t, 2, "s1", "s2", "s3")
	logs := tm.logs[:cap(tm.logs)]

	var memSize int64
	tm.DeleteAll(false, &memSize)

	if len(tm.logs) != 0 {
		t.Fatalf("expected no logs, got %d", len(tm.logs))
	}
	assertLogsCleared(t, logs, 0)
}
