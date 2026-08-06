//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of the
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// Real end-to-end tests against a live AWS Glue/S3 Iceberg catalog. These are
// skipped entirely unless AWS credentials resolve from the environment (a named
// profile via ICEBERG_TEST_AWS_PROFILE, or the default credential chain --
// ~/.aws/credentials, env vars, IMDS) AND an S3 warehouse location is configured
// via ICEBERG_TEST_S3_BUCKET. There is no safe way to guess a writable S3
// location for whatever AWS account ambient credentials happen to belong to, so
// bucket configuration is required even when credentials are found -- this keeps
// `go test ./...` safe by default for anyone with unrelated AWS credentials
// sitting in their environment.
//
// Each test creates its own table (dropping any leftover table of the same name
// first), writes real Parquet data files directly via arrow-go -- the same
// pattern already proven in iceberg_variant_projection_test.go and
// iceberg_variant_predicate_test.go -- registers them via
// table.Transaction.AddFiles (which registers a file as-is, unlike
// Transaction.Append, which round-trips the schema through
// ToRequestedSchema/pqarrow.NewFileWriter and empirically mangles a shredded
// VARIANT's schema enough to fail pqarrow's "record schema does not match
// writer's" check), then drives the real production Scanner
// (NewScanner/LoadTable/ScanAndConvertStream) against it. Tables and their S3
// data are dropped again in cleanup.
//
// Configure with:
//
//	export ICEBERG_TEST_AWS_PROFILE=ICEBERG_TEST_AWS_PROFILE  # or any ~/.aws profile name
//	export ICEBERG_TEST_S3_BUCKET=s3://iceb100
//	export ICEBERG_TEST_GLUE_DATABASE=iceberg                 # optional, defaults to "iceberg"
//	export ICEBERG_TEST_AWS_REGION=us-west-2                  # optional, defaults to "us-west-2"
package external

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/extensions"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	iceberg "github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	"github.com/apache/iceberg-go/catalog/glue"
	icetable "github.com/apache/iceberg-go/table"

	"github.com/couchbase/cbauth"
	"github.com/couchbase/cbauth/cbauthimpl"
)

const (
	envAWSProfile = "ICEBERG_TEST_AWS_PROFILE"
	envS3Bucket   = "ICEBERG_TEST_S3_BUCKET"
	envGlueDB     = "ICEBERG_TEST_GLUE_DATABASE"
	envAWSRegion  = "ICEBERG_TEST_AWS_REGION"
)

// realAWSTestEnv bundles everything a real-AWS Iceberg scenario needs.
type realAWSTestEnv struct {
	cfg      aws.Config
	cat      catalog.Catalog
	s3       *s3.Client
	bucket   string // e.g. "s3://iceb100", no trailing slash
	database string
}

// setupRealAWSTestEnv resolves AWS credentials and required configuration from
// the environment. ok is false (callers must t.Skip and return) when either no
// credentials resolve or ICEBERG_TEST_S3_BUCKET isn't set.
func setupRealAWSTestEnv(t *testing.T) (env realAWSTestEnv, ok bool) {
	t.Helper()

	bucket := os.Getenv(envS3Bucket)
	if bucket == "" {
		t.Skipf("%s not set; skipping real-AWS Iceberg end-to-end test", envS3Bucket)
		return realAWSTestEnv{}, false
	}

	region := os.Getenv(envAWSRegion)
	if region == "" {
		region = "us-west-2"
	}
	database := os.Getenv(envGlueDB)
	if database == "" {
		database = "iceberg"
	}

	ctx := context.Background()
	var cfgOpts []func(*awsconfig.LoadOptions) error
	cfgOpts = append(cfgOpts, awsconfig.WithRegion(region))
	if profile := os.Getenv(envAWSProfile); profile != "" {
		cfgOpts = append(cfgOpts, awsconfig.WithSharedConfigProfile(profile))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, cfgOpts...)
	if err != nil {
		t.Skipf("could not load AWS config from environment: %v", err)
		return realAWSTestEnv{}, false
	}
	if _, err := cfg.Credentials.Retrieve(ctx); err != nil {
		t.Skipf("no AWS credentials found in environment (~/.aws, env vars, IMDS): %v", err)
		return realAWSTestEnv{}, false
	}

	return realAWSTestEnv{
		cfg:      cfg,
		cat:      glue.NewCatalog(glue.WithAwsConfig(cfg)),
		s3:       s3.NewFromConfig(cfg),
		bucket:   bucket,
		database: database,
	}, true
}

// createFreshTable drops any leftover table of the same name, creates a new one
// with the given schema, and registers a t.Cleanup to drop the table and its S3
// data again at the end of the test.
func (env realAWSTestEnv) createFreshTable(t *testing.T, ctx context.Context, name string, sc *iceberg.Schema) *icetable.Table {
	t.Helper()

	ident := icetable.Identifier{env.database, name}
	location := fmt.Sprintf("%s/%s.db/%s", env.bucket, env.database, name)

	_ = env.cat.DropTable(ctx, ident) // best-effort; table may not exist yet
	env.removeS3Prefix(t, ctx, location)

	tbl, err := env.cat.CreateTable(ctx, ident, sc,
		catalog.WithLocation(location),
		catalog.WithProperties(iceberg.Properties{"format-version": "3"}))
	if err != nil {
		t.Fatalf("CreateTable(%s): %v", name, err)
	}

	t.Cleanup(func() {
		cleanupCtx := context.Background()
		if err := env.cat.DropTable(cleanupCtx, ident); err != nil {
			t.Logf("cleanup: DropTable(%s): %v", name, err)
		}
		env.removeS3Prefix(t, cleanupCtx, location)
	})

	return tbl
}

// removeS3Prefix best-effort deletes every object under an s3://bucket/prefix
// location. Errors are logged, not fatal -- this is cleanup, not the test itself.
func (env realAWSTestEnv) removeS3Prefix(t *testing.T, ctx context.Context, location string) {
	t.Helper()
	bucket, prefix, err := ParseS3URI(location)
	if err != nil {
		t.Logf("removeS3Prefix: %v", err)
		return
	}

	paginator := s3.NewListObjectsV2Paginator(env.s3, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			t.Logf("removeS3Prefix: ListObjectsV2: %v", err)
			return
		}
		for _, obj := range page.Contents {
			if _, err := env.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(bucket),
				Key:    obj.Key,
			}); err != nil {
				t.Logf("removeS3Prefix: DeleteObject(%s): %v", *obj.Key, err)
			}
		}
	}
}

// uploadParquet PUTs raw Parquet bytes to an s3:// path.
func (env realAWSTestEnv) uploadParquet(t *testing.T, ctx context.Context, s3Path string, data []byte) {
	t.Helper()
	bucket, key, err := ParseS3URI(s3Path)
	if err != nil {
		t.Fatalf("uploadParquet: %v", err)
	}
	if _, err := env.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	}); err != nil {
		t.Fatalf("PutObject(%s): %v", s3Path, err)
	}
}

// addFilesAndCommit registers a pre-built Parquet file as a new data file and
// commits the resulting snapshot.
func addFilesAndCommit(t *testing.T, ctx context.Context, tbl *icetable.Table, s3Path string) {
	t.Helper()
	tx := tbl.NewTransaction()
	if err := tx.AddFiles(ctx, []string{s3Path}, nil, false); err != nil {
		t.Fatalf("AddFiles(%s): %v", s3Path, err)
	}
	if _, err := tx.Commit(ctx); err != nil {
		t.Fatalf("Commit: %v", err)
	}
}

// storageCredential builds a cbauth.Credential carrying the resolved AWS
// credentials, for Scanner.CollectionCred -- this is what routes the scan onto
// the parallel-files path (ScanAndConvertParallelFiles), which is where
// resolveColumnIndices/collectVariantLeaves/buildVariantRowGroupMatcher live.
// The plain streaming fallback path goes through iceberg-go's own
// ToArrowRecords instead, which has a documented context-cancellation quirk
// unrelated to anything tested here (see project notes on the v0.4.0->v0.6.0
// iceberg-go bump).
func (env realAWSTestEnv) storageCredential(t *testing.T, ctx context.Context) *cbauth.Credential {
	t.Helper()
	creds, err := env.cfg.Credentials.Retrieve(ctx)
	if err != nil {
		t.Fatalf("Retrieve AWS credentials: %v", err)
	}
	return &cbauth.Credential{
		Type: cbauth.CredentialTypeAWS,
		AWS: &cbauthimpl.AWSPayload{
			AccessKeyID:     creds.AccessKeyID,
			SecretAccessKey: creds.SecretAccessKey,
			SessionToken:    creds.SessionToken,
			Region:          env.cfg.Region,
		},
	}
}

// scanResult captures what a real scan returned plus the pruning stats
// resolveColumnIndices/selectRowGroups recorded along the way.
type scanResult struct {
	rows                          []map[string]interface{}
	rowGroupsKept, rowGroupsTotal int64
	colsSelected, colsTotal       int64
}

// scan drives the real, production Scanner end to end against a real table.
func (env realAWSTestEnv) scan(t *testing.T, ctx context.Context, tableName string,
	selectedFields []string, filterExpr iceberg.BooleanExpression, variantPreds []variantPredicate) scanResult {
	t.Helper()

	opts := ScanOptions{
		Database:          env.database,
		Table:             tableName,
		SourceType:        "AWS_GLUE",
		CaseSensitive:     true,
		AwsConfig:         &env.cfg,
		CollectionCred:    env.storageCredential(t, ctx),
		SelectedFields:    selectedFields,
		FilterExpr:        filterExpr,
		VariantPredicates: variantPreds,
	}

	scanner, err := NewScanner(ctx, opts, nil)
	if err != nil {
		t.Fatalf("NewScanner: %v", err)
	}
	defer scanner.Close()
	if err := scanner.LoadTable(ctx); err != nil {
		t.Fatalf("LoadTable: %v", err)
	}

	rowsCh, errCh := scanner.ScanAndConvertStream(ctx)
	var rows []map[string]interface{}
	for row := range rowsCh {
		rows = append(rows, row)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("scan: %v", err)
	}

	kept, total := scanner.RowGroupStats()
	selCols, totalCols := scanner.ColumnStats()
	return scanResult{
		rows:          rows,
		rowGroupsKept: kept, rowGroupsTotal: total,
		colsSelected: selCols, colsTotal: totalCols,
	}
}

// ---------------------------------------------------------------------------
// Scenario A: plain (non-VARIANT) nested struct.
// ---------------------------------------------------------------------------

func TestRealAWSIcebergNonVariantStruct(t *testing.T) {
	env, ok := setupRealAWSTestEnv(t)
	if !ok {
		return
	}
	ctx := context.Background()
	const tableName = "cq_test_nonvariant_struct"

	sc := iceberg.NewSchema(0,
		iceberg.NestedField{ID: 1, Name: "id", Type: iceberg.PrimitiveTypes.Int64, Required: false},
		iceberg.NestedField{ID: 2, Name: "address", Required: false, Type: &iceberg.StructType{
			FieldList: []iceberg.NestedField{
				{ID: 3, Name: "city", Type: iceberg.PrimitiveTypes.String, Required: false},
				{ID: 4, Name: "state", Type: iceberg.PrimitiveTypes.String, Required: false},
			},
		}},
	)
	tbl := env.createFreshTable(t, ctx, tableName, sc)

	addressType := arrow.StructOf(
		arrow.Field{Name: "city", Type: arrow.BinaryTypes.String},
		arrow.Field{Name: "state", Type: arrow.BinaryTypes.String},
	)
	arrowSchema := arrow.NewSchema([]arrow.Field{
		{Name: "id", Type: arrow.PrimitiveTypes.Int64, Nullable: true},
		{Name: "address", Type: addressType, Nullable: true},
	}, nil)

	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	rows := []struct {
		id          int64
		city, state string
	}{
		{1, "SF", "CA"},
		{2, "NYC", "NY"},
	}

	var buf bytes.Buffer
	wr, err := pqarrow.NewFileWriter(arrowSchema, &buf,
		parquet.NewWriterProperties(parquet.WithDictionaryDefault(false)), pqarrow.DefaultWriterProps())
	if err != nil {
		t.Fatalf("NewFileWriter: %v", err)
	}
	for _, row := range rows {
		idBldr := array.NewInt64Builder(mem)
		idBldr.Append(row.id)
		idArr := idBldr.NewArray()
		idBldr.Release()

		addrBldr := array.NewStructBuilder(mem, addressType)
		addrBldr.Append(true)
		addrBldr.FieldBuilder(0).(*array.StringBuilder).Append(row.city)
		addrBldr.FieldBuilder(1).(*array.StringBuilder).Append(row.state)
		addrArr := addrBldr.NewArray()
		addrBldr.Release()

		rec := array.NewRecordBatch(arrowSchema, []arrow.Array{idArr, addrArr}, -1)
		idArr.Release()
		addrArr.Release()
		if err := wr.Write(rec); err != nil {
			rec.Release()
			t.Fatalf("Write: %v", err)
		}
		rec.Release()
	}
	if err := wr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	s3Path := tbl.Location() + "/data/manual-0001.parquet"
	env.uploadParquet(t, ctx, s3Path, buf.Bytes())
	addFilesAndCommit(t, ctx, tbl, s3Path)

	t.Run("full scan", func(t *testing.T) {
		res := env.scan(t, ctx, tableName, nil, nil, nil)
		if len(res.rows) != 2 {
			t.Fatalf("expected 2 rows, got %d: %v", len(res.rows), res.rows)
		}
		for _, r := range res.rows {
			addr, ok := r["address"].(map[string]interface{})
			if !ok || addr["city"] == nil || addr["state"] == nil {
				t.Errorf("expected both city and state present, got %v", r)
			}
		}
	})

	t.Run("projection pushdown: address.city only", func(t *testing.T) {
		res := env.scan(t, ctx, tableName, []string{"address.city"}, nil, nil)
		if len(res.rows) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(res.rows))
		}
		if res.colsSelected >= res.colsTotal {
			t.Errorf("expected fewer columns selected (%d) than total (%d)", res.colsSelected, res.colsTotal)
		}
		for _, r := range res.rows {
			addr := r["address"].(map[string]interface{})
			if _, present := addr["state"]; present {
				t.Errorf("expected state absent from pruned decode, got %v", addr)
			}
			if _, present := addr["city"]; !present {
				t.Errorf("expected city present, got %v", addr)
			}
		}
	})

	t.Run("filter pushdown: address.city = NYC prunes a row group", func(t *testing.T) {
		filter := iceberg.EqualTo(iceberg.Reference("address.city"), "NYC")
		res := env.scan(t, ctx, tableName, nil, filter, nil)
		if len(res.rows) != 1 {
			t.Fatalf("expected 1 row, got %d: %v", len(res.rows), res.rows)
		}
		if res.rowGroupsKept >= res.rowGroupsTotal {
			t.Errorf("expected row-group pruning: kept=%d total=%d", res.rowGroupsKept, res.rowGroupsTotal)
		}
		addr := res.rows[0]["address"].(map[string]interface{})
		if addr["city"] != "NYC" {
			t.Errorf("expected city=NYC, got %v", addr)
		}
	})
}

// ---------------------------------------------------------------------------
// Scenario B: unshredded VARIANT (metadata+value only, no typed_value).
// ---------------------------------------------------------------------------

func TestRealAWSIcebergVariantUnshredded(t *testing.T) {
	env, ok := setupRealAWSTestEnv(t)
	if !ok {
		return
	}
	ctx := context.Background()
	const tableName = "cq_test_variant_unshredded"

	sc := iceberg.NewSchema(0,
		iceberg.NestedField{ID: 1, Name: "id", Type: iceberg.PrimitiveTypes.Int64, Required: false},
		iceberg.NestedField{ID: 2, Name: "attrs", Type: iceberg.VariantType{}, Required: false},
	)
	tbl := env.createFreshTable(t, ctx, tableName, sc)

	vt := extensions.NewDefaultVariantType() // unshredded: struct<metadata,value> only
	arrowSchema := arrow.NewSchema([]arrow.Field{
		{Name: "id", Type: arrow.PrimitiveTypes.Int64, Nullable: true},
		{Name: "attrs", Type: vt, Nullable: true},
	}, nil)

	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	rowsData := []struct {
		id    int64
		color string
		size  int64
	}{
		{1, "red", 10},
		{2, "blue", 20},
	}

	var buf bytes.Buffer
	wr, err := pqarrow.NewFileWriter(arrowSchema, &buf,
		parquet.NewWriterProperties(parquet.WithDictionaryDefault(false)), pqarrow.DefaultWriterProps())
	if err != nil {
		t.Fatalf("NewFileWriter: %v", err)
	}
	for _, row := range rowsData {
		idBldr := array.NewInt64Builder(mem)
		idBldr.Append(row.id)
		idArr := idBldr.NewArray()
		idBldr.Release()

		vBldr := vt.NewBuilder(mem)
		jsonRow := fmt.Sprintf(`[{"color": %q, "size": %d}]`, row.color, row.size)
		if err := vBldr.UnmarshalJSON([]byte(jsonRow)); err != nil {
			t.Fatalf("UnmarshalJSON: %v", err)
		}
		attrsArr := vBldr.NewArray()
		vBldr.Release()

		rec := array.NewRecordBatch(arrowSchema, []arrow.Array{idArr, attrsArr}, -1)
		idArr.Release()
		attrsArr.Release()
		if err := wr.Write(rec); err != nil {
			rec.Release()
			t.Fatalf("Write: %v", err)
		}
		rec.Release()
	}
	if err := wr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	s3Path := tbl.Location() + "/data/manual-0001.parquet"
	env.uploadParquet(t, ctx, s3Path, buf.Bytes())
	addFilesAndCommit(t, ctx, tbl, s3Path)

	t.Run("full scan decodes unshredded variant correctly", func(t *testing.T) {
		res := env.scan(t, ctx, tableName, nil, nil, nil)
		if len(res.rows) != 2 {
			t.Fatalf("expected 2 rows, got %d: %v", len(res.rows), res.rows)
		}
		for _, r := range res.rows {
			attrs, ok := r["attrs"].(map[string]interface{})
			if !ok || attrs["color"] == nil || attrs["size"] == nil {
				t.Errorf("expected color and size present, got %v", r)
			}
		}
	})

	t.Run("projection request safely reads whole field (nothing to shred)", func(t *testing.T) {
		// collectVariantLeaves/collectStructSubLeaves both correctly decline (no
		// typed_value present at all), falling back to a full read -- decode must
		// still be fully correct.
		res := env.scan(t, ctx, tableName, []string{"attrs.color"}, nil, nil)
		if len(res.rows) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(res.rows))
		}
		for _, r := range res.rows {
			attrs := r["attrs"].(map[string]interface{})
			if attrs["color"] == nil {
				t.Errorf("expected color present, got %v", attrs)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// Scenario C: shredded VARIANT (metadata+value+typed_value.color/size).
// ---------------------------------------------------------------------------

func TestRealAWSIcebergVariantShredded(t *testing.T) {
	env, ok := setupRealAWSTestEnv(t)
	if !ok {
		return
	}
	ctx := context.Background()
	const tableName = "cq_test_variant_shredded"

	sc := iceberg.NewSchema(0,
		iceberg.NestedField{ID: 1, Name: "id", Type: iceberg.PrimitiveTypes.Int64, Required: false},
		iceberg.NestedField{ID: 2, Name: "attrs", Type: iceberg.VariantType{}, Required: false},
	)
	tbl := env.createFreshTable(t, ctx, tableName, sc)

	vt := extensions.NewShreddedVariantType(arrow.StructOf(
		arrow.Field{Name: "color", Type: arrow.BinaryTypes.String},
		arrow.Field{Name: "size", Type: arrow.PrimitiveTypes.Int64},
		arrow.Field{Name: "geo", Type: arrow.StructOf(
			arrow.Field{Name: "floor", Type: arrow.PrimitiveTypes.Int64},
			arrow.Field{Name: "zone", Type: arrow.PrimitiveTypes.Int64},
		)},
	))
	arrowSchema := arrow.NewSchema([]arrow.Field{
		{Name: "id", Type: arrow.PrimitiveTypes.Int64, Nullable: true},
		{Name: "attrs", Type: vt, Nullable: true},
	}, nil)

	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	// Row groups 1 and 2 hold a single disjoint-colored row each (row-group
	// stats pruning has something to prune). Row group 3 holds two rows of
	// different colors in the SAME Write call, so it survives row-group-stats
	// pruning for either color -- that's what exercises row-level filtering
	// (buildVariantRowMatcher/emitRow): the row group as a whole can't be
	// skipped, but the non-matching row within it still must be.
	rowGroups := [][]struct {
		id    int64
		color string
		size  int64
		floor int64
		zone  int64
	}{
		{{1, "red", 10, 1, 100}},
		{{2, "blue", 20, 2, 200}},
		{
			{3, "green", 30, 3, 300},
			{4, "purple", 40, 4, 400},
		},
	}

	var buf bytes.Buffer
	wr, err := pqarrow.NewFileWriter(arrowSchema, &buf,
		parquet.NewWriterProperties(parquet.WithDictionaryDefault(false)), pqarrow.DefaultWriterProps())
	if err != nil {
		t.Fatalf("NewFileWriter: %v", err)
	}
	for _, rg := range rowGroups {
		idBldr := array.NewInt64Builder(mem)
		var jsonRows []string
		for _, row := range rg {
			idBldr.Append(row.id)
			jsonRows = append(jsonRows, fmt.Sprintf(
				`{"color": %q, "size": %d, "geo": {"floor": %d, "zone": %d}}`,
				row.color, row.size, row.floor, row.zone))
		}
		idArr := idBldr.NewArray()
		idBldr.Release()

		vBldr := vt.NewBuilder(mem)
		jsonBatch := "[" + strings.Join(jsonRows, ",") + "]"
		if err := vBldr.UnmarshalJSON([]byte(jsonBatch)); err != nil {
			t.Fatalf("UnmarshalJSON: %v", err)
		}
		attrsArr := vBldr.NewArray()
		vBldr.Release()

		rec := array.NewRecordBatch(arrowSchema, []arrow.Array{idArr, attrsArr}, -1)
		idArr.Release()
		attrsArr.Release()
		if err := wr.Write(rec); err != nil {
			rec.Release()
			t.Fatalf("Write row group: %v", err)
		}
		rec.Release()
	}
	if err := wr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	s3Path := tbl.Location() + "/data/manual-0001.parquet"
	env.uploadParquet(t, ctx, s3Path, buf.Bytes())
	addFilesAndCommit(t, ctx, tbl, s3Path)

	t.Run("full scan", func(t *testing.T) {
		res := env.scan(t, ctx, tableName, nil, nil, nil)
		if len(res.rows) != 4 {
			t.Fatalf("expected 4 rows, got %d: %v", len(res.rows), res.rows)
		}
		for _, r := range res.rows {
			attrs, ok := r["attrs"].(map[string]interface{})
			if !ok || attrs["color"] == nil || attrs["size"] == nil {
				t.Errorf("expected color and size present, got %v", r)
			}
		}
	})

	t.Run("projection pushdown: attrs.color only", func(t *testing.T) {
		res := env.scan(t, ctx, tableName, []string{"attrs.color"}, nil, nil)
		if len(res.rows) != 4 {
			t.Fatalf("expected 4 rows, got %d", len(res.rows))
		}
		if res.colsSelected >= res.colsTotal {
			t.Errorf("expected pruning: selected=%d total=%d", res.colsSelected, res.colsTotal)
		}
		for _, r := range res.rows {
			attrs := r["attrs"].(map[string]interface{})
			if _, present := attrs["size"]; present {
				t.Errorf("expected size absent from pruned decode, got %v", attrs)
			}
			if _, present := attrs["color"]; !present {
				t.Errorf("expected color present, got %v", attrs)
			}
		}
	})

	t.Run("projection pushdown: attrs.geo.floor only (arbitrary depth)", func(t *testing.T) {
		res := env.scan(t, ctx, tableName, []string{"attrs.geo.floor"}, nil, nil)
		if len(res.rows) != 4 {
			t.Fatalf("expected 4 rows, got %d", len(res.rows))
		}
		if res.colsSelected >= res.colsTotal {
			t.Errorf("expected pruning: selected=%d total=%d", res.colsSelected, res.colsTotal)
		}
		for _, r := range res.rows {
			attrs := r["attrs"].(map[string]interface{})
			geo, ok := attrs["geo"].(map[string]interface{})
			if !ok {
				t.Fatalf("expected 'geo' present as an object, got %v", attrs)
			}
			if _, present := geo["zone"]; present {
				t.Errorf("expected 'zone' absent from pruned decode, got %v", geo)
			}
			if _, present := geo["floor"]; !present {
				t.Errorf("expected 'floor' present, got %v", geo)
			}
			if _, present := attrs["color"]; present {
				t.Errorf("expected 'color' absent (selectedFields requested only attrs.geo.floor), got %v", attrs)
			}
		}
	})

	t.Run("predicate pushdown: attrs.color = blue prunes a row group", func(t *testing.T) {
		res := env.scan(t, ctx, tableName, []string{"attrs.color"},
			nil, []variantPredicate{{path: "attrs.color", op: iceberg.OpEQ, lit: "blue"}})
		if len(res.rows) != 1 {
			t.Fatalf("expected 1 row, got %d: %v", len(res.rows), res.rows)
		}
		if res.rowGroupsKept >= res.rowGroupsTotal {
			t.Errorf("expected row-group pruning: kept=%d total=%d", res.rowGroupsKept, res.rowGroupsTotal)
		}
		attrs := res.rows[0]["attrs"].(map[string]interface{})
		if attrs["color"] != "blue" {
			t.Errorf("expected color=blue, got %v", attrs)
		}
	})

	t.Run("predicate pushdown: attrs.geo.floor > 2 prunes row groups (arbitrary depth)", func(t *testing.T) {
		res := env.scan(t, ctx, tableName, nil,
			nil, []variantPredicate{{path: "attrs.geo.floor", op: iceberg.OpGT, lit: float64(2)}})
		// Row group 1 (floor=1) and row group 2 (floor=2) are both prunable
		// (neither has any row with floor > 2); only row group 3 (floor 3,4)
		// survives.
		if len(res.rows) != 2 {
			t.Fatalf("expected 2 rows, got %d: %v", len(res.rows), res.rows)
		}
		if res.rowGroupsKept != 1 {
			t.Errorf("expected exactly 1 row group kept, got %d/%d", res.rowGroupsKept, res.rowGroupsTotal)
		}
		for _, r := range res.rows {
			attrs := r["attrs"].(map[string]interface{})
			geo := attrs["geo"].(map[string]interface{})
			floor := geo["floor"]
			// decodeVariantScalar sizes decoded integers to the smallest Go
			// type that fits the value (int8/int16/int32/int64), so compare
			// via fmt rather than asserting a specific width.
			if fv, err := strconv.ParseInt(fmt.Sprint(floor), 10, 64); err != nil || fv <= 2 {
				t.Errorf("expected floor > 2, got %v (%T)", floor, floor)
			}
		}
	})

	t.Run("row-level filtering drops the non-matching row from a surviving mixed row group", func(t *testing.T) {
		// "green" only appears in row group 3, alongside "purple" -- that row
		// group's color range (green..purple) can't be excluded by row-group
		// stats alone (a plausible-looking range still spans the literal), so
		// this specifically exercises buildVariantRowMatcher/emitRow dropping
		// "purple" row-by-row rather than relying on row-group pruning alone.
		res := env.scan(t, ctx, tableName, []string{"attrs.color"},
			nil, []variantPredicate{{path: "attrs.color", op: iceberg.OpEQ, lit: "green"}})
		if len(res.rows) != 1 {
			t.Fatalf("expected exactly 1 row (row-level filtering must drop 'purple'), got %d: %v",
				len(res.rows), res.rows)
		}
		if res.rowGroupsKept != 1 {
			t.Errorf("expected exactly 1 row group kept (row groups 1 and 2 are red/blue, prunable), got %d/%d",
				res.rowGroupsKept, res.rowGroupsTotal)
		}
		attrs := res.rows[0]["attrs"].(map[string]interface{})
		if attrs["color"] != "green" {
			t.Errorf("expected color=green, got %v", attrs)
		}
	})
}
