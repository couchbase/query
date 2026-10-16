//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of the
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package external

import (
	go_context "context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	icesql "github.com/apache/iceberg-go/catalog/sql"
)

// TestReaderMultiBatchSingleFile covers the iceberg-go streaming path
// (scanAndConvertStreamFallback → Reader) on a table whose only data file spans
// several Arrow record batches. The Reader used to re-range iceberg-go's
// single-use record iterator once per batch; the first early break cancelled the
// scan, so reads either stopped after the first batch with no error or failed
// with "context canceled".
func TestReaderMultiBatchSingleFile(t *testing.T) {
	// iceberg-go reads up to 128K rows per batch; use enough rows for three batches.
	const numRows = 300000

	ctx := go_context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "catalog.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	cat, err := icesql.NewCatalog("sql", db, icesql.SQLite, iceberg.Properties{"warehouse": "file://" + dir})
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	if err := cat.CreateNamespace(ctx, catalog.ToIdentifier("ns"), nil); err != nil {
		t.Fatalf("CreateNamespace: %v", err)
	}
	sc := iceberg.NewSchema(0,
		iceberg.NestedField{ID: 1, Name: "id", Type: iceberg.PrimitiveTypes.Int64, Required: false})
	tbl, err := cat.CreateTable(ctx, catalog.ToIdentifier("ns", "t"), sc)
	if err != nil {
		t.Fatalf("CreateTable: %v", err)
	}

	bldr := array.NewInt64Builder(memory.DefaultAllocator)
	for i := int64(0); i < numRows; i++ {
		bldr.Append(i)
	}
	col := bldr.NewArray()
	arrowSchema := arrow.NewSchema([]arrow.Field{{Name: "id", Type: arrow.PrimitiveTypes.Int64, Nullable: true}}, nil)
	rec := array.NewRecordBatch(arrowSchema, []arrow.Array{col}, numRows)
	arrTbl := array.NewTableFromRecords(arrowSchema, []arrow.RecordBatch{rec})
	defer arrTbl.Release()
	rec.Release()
	col.Release()

	tx := tbl.NewTransaction()
	if err := tx.AppendTable(ctx, arrTbl, numRows, nil); err != nil {
		t.Fatalf("AppendTable: %v", err)
	}
	if _, err := tx.Commit(ctx); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	db.Close()

	// The bug was a race, so a single clean pass proves little; scan repeatedly.
	for pass := 0; pass < 5; pass++ {
		scanCtx, cancel := go_context.WithTimeout(ctx, 2*time.Minute)
		s, err := NewScanner(scanCtx, ScanOptions{
			Database: "ns", Table: "t", SourceType: "SQL", URI: dbPath, CaseSensitive: true,
		}, nil)
		if err != nil {
			cancel()
			t.Fatalf("NewScanner: %v", err)
		}
		if err := s.LoadTable(scanCtx); err != nil {
			cancel()
			t.Fatalf("LoadTable: %v", err)
		}
		if files, err := planFilesWithRecovery(s.table.Scan(), scanCtx); err != nil || len(files) != 1 {
			cancel()
			t.Fatalf("expected exactly 1 data file (single-file path), got %d (err %v)", len(files), err)
		}

		rows, errs := s.ScanAndConvertStream(scanCtx)
		count := 0
		var scanErr error
		for rows != nil || errs != nil {
			select {
			case _, ok := <-rows:
				if !ok {
					rows = nil
					continue
				}
				count++
			case e, ok := <-errs:
				if !ok {
					errs = nil
					continue
				}
				if e != nil && scanErr == nil {
					scanErr = e
				}
			}
		}
		s.Close()
		cancel()

		if scanErr != nil {
			t.Fatalf("pass %d: scan error after %d rows: %v", pass, count, scanErr)
		}
		if count != numRows {
			t.Fatalf("pass %d: got %d rows, want %d", pass, count, numRows)
		}
	}
}
