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
	"fmt"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	iceberg "github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	"github.com/apache/iceberg-go/table"
	"github.com/aws/aws-sdk-go-v2/aws"
)

// stubCatalog satisfies catalog.Catalog for tests that only need a Scanner
// struct populated (e.g. sourceType / parallelScans field checks) without
// an actual catalog connection.
type stubCatalog struct{}

func (s *stubCatalog) CatalogType() catalog.Type { return catalog.REST }
func (s *stubCatalog) CreateTable(_ go_context.Context, _ table.Identifier, _ *iceberg.Schema, _ ...catalog.CreateTableOpt) (*table.Table, error) {
	return nil, fmt.Errorf("stub")
}
func (s *stubCatalog) CommitTable(_ go_context.Context, _ table.Identifier, _ []table.Requirement, _ []table.Update) (table.Metadata, string, error) {
	return nil, "", fmt.Errorf("stub")
}
func (s *stubCatalog) ListTables(_ go_context.Context, _ table.Identifier) iter.Seq2[table.Identifier, error] {
	return func(yield func(table.Identifier, error) bool) {}
}
func (s *stubCatalog) LoadTable(_ go_context.Context, _ table.Identifier) (*table.Table, error) {
	return nil, fmt.Errorf("stub")
}
func (s *stubCatalog) DropTable(_ go_context.Context, _ table.Identifier) error {
	return fmt.Errorf("stub")
}
func (s *stubCatalog) RenameTable(_ go_context.Context, _, _ table.Identifier) (*table.Table, error) {
	return nil, fmt.Errorf("stub")
}
func (s *stubCatalog) CheckTableExists(_ go_context.Context, _ table.Identifier) (bool, error) {
	return false, fmt.Errorf("stub")
}
func (s *stubCatalog) ListNamespaces(_ go_context.Context, _ table.Identifier) ([]table.Identifier, error) {
	return nil, fmt.Errorf("stub")
}
func (s *stubCatalog) CreateNamespace(_ go_context.Context, _ table.Identifier, _ iceberg.Properties) error {
	return fmt.Errorf("stub")
}
func (s *stubCatalog) DropNamespace(_ go_context.Context, _ table.Identifier) error {
	return fmt.Errorf("stub")
}
func (s *stubCatalog) CheckNamespaceExists(_ go_context.Context, _ table.Identifier) (bool, error) {
	return false, fmt.Errorf("stub")
}
func (s *stubCatalog) LoadNamespaceProperties(_ go_context.Context, _ table.Identifier) (iceberg.Properties, error) {
	return nil, fmt.Errorf("stub")
}
func (s *stubCatalog) UpdateNamespaceProperties(_ go_context.Context, _ table.Identifier, _ []string, _ iceberg.Properties) (catalog.PropertiesUpdateSummary, error) {
	return catalog.PropertiesUpdateSummary{}, fmt.Errorf("stub")
}

// This file contains example tests demonstrating the Iceberg scanner usage
// These are not meant to run without proper AWS credentials and Glue setup

func ExampleNewScanner() {
	ctx := go_context.Background()

	// This requires actual AWS credentials and Glue setup
	opts := ScanOptions{
		Database:      "my_database",
		Table:         "my_table",
		CaseSensitive: true,
		AwsConfig:     &aws.Config{},
	}

	scanner, err := NewScanner(ctx, opts, nil)
	if err != nil {
		_ = err
		return
	}
	defer scanner.Close()

	_ = scanner
}

func ExampleNewAWSConfig() {
	accessKeyID := "your-access-key-id"
	secretAccessKey := "your-secret-access-key"
	sessionToken := "" // optional
	region := "us-east-1"

	config, err := NewAWSConfig(accessKeyID, secretAccessKey, sessionToken, region)
	if err != nil {
		_ = err
		return
	}

	_ = config
}

func ExampleCreateEqualFilter() {
	filter := CreateEqualFilter("status", "active")
	_ = filter
}

func ExampleCreateAndFilter() {
	filter := CreateAndFilter(
		CreateEqualFilter("status", "active"),
		CreateEqualFilter("age", 30),
	)
	_ = filter
}

func ExampleCreateOrFilter() {
	filter := CreateOrFilter(
		CreateEqualFilter("status", "active"),
		CreateEqualFilter("status", "pending"),
	)
	_ = filter
}

func ExampleCreateRangeFilter() {
	minAge := 25
	maxAge := 35

	filters, err := CreateRangeFilter("age", minAge, maxAge)
	if err != nil {
		_ = err
		return
	}

	_ = filters
}

func ExampleCreateInFilter() {
	inFilter := CreateInFilter("category", "electronics", "books", "home")
	_ = inFilter
}

func TestParseS3URI(t *testing.T) {
	tests := []struct {
		name           string
		uri            string
		expectedBucket string
		expectedKey    string
		expectError    bool
	}{
		{
			name:           "valid S3 URI with key",
			uri:            "s3://my-bucket/path/to/file.parquet",
			expectedBucket: "my-bucket",
			expectedKey:    "path/to/file.parquet",
			expectError:    false,
		},
		{
			name:           "valid S3 URI without key",
			uri:            "s3://my-bucket",
			expectedBucket: "my-bucket",
			expectedKey:    "",
			expectError:    false,
		},
		{
			name:           "invalid S3 URI",
			uri:            "https://s3.amazonaws.com/my-bucket/file.parquet",
			expectedBucket: "",
			expectedKey:    "",
			expectError:    true,
		},
		{
			name:           "empty URI",
			uri:            "",
			expectedBucket: "",
			expectedKey:    "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bucket, key, err := ParseS3URI(tt.uri)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if bucket != tt.expectedBucket {
				t.Errorf("bucket = %v, want %v", bucket, tt.expectedBucket)
			}

			if key != tt.expectedKey {
				t.Errorf("key = %v, want %v", key, tt.expectedKey)
			}
		})
	}
}

func TestFilterCreation(t *testing.T) {
	t.Run("EqualFilter", func(t *testing.T) {
		filter := CreateEqualFilter("field1", "value1")
		if filter.Op != "=" {
			t.Errorf("filter.Op = %v, want =", filter.Op)
		}
		if filter.Field != "field1" {
			t.Errorf("filter.Field = %v, want field1", filter.Field)
		}
		if filter.Value != "value1" {
			t.Errorf("filter.Value = %v, want value1", filter.Value)
		}
	})

	t.Run("AndFilter", func(t *testing.T) {
		filter := CreateAndFilter(
			CreateEqualFilter("f1", "v1"),
			CreateEqualFilter("f2", "v2"),
		)
		if filter.Op != "and" {
			t.Errorf("filter.Op = %v, want and", filter.Op)
		}
		if len(filter.Children) != 2 {
			t.Errorf("len(filter.Children) = %v, want 2", len(filter.Children))
		}
	})

	t.Run("OrFilter", func(t *testing.T) {
		filter := CreateOrFilter(
			CreateEqualFilter("f1", "v1"),
			CreateEqualFilter("f2", "v2"),
		)
		if filter.Op != "or" {
			t.Errorf("filter.Op = %v, want or", filter.Op)
		}
		if len(filter.Children) != 2 {
			t.Errorf("len(filter.Children) = %v, want 2", len(filter.Children))
		}
	})

	t.Run("InFilter", func(t *testing.T) {
		filter := CreateInFilter("category", "a", "b", "c")
		if filter.Op != "in" {
			t.Errorf("filter.Op = %v, want in", filter.Op)
		}
		values, ok := filter.Value.([]interface{})
		if !ok {
			t.Errorf("filter.Value is not a slice")
		}
		if len(values) != 3 {
			t.Errorf("len(filter.Value) = %v, want 3", len(values))
		}
	})
}

func TestCreateRangeFilter(t *testing.T) {
	t.Run("ValidRange", func(t *testing.T) {
		filters, err := CreateRangeFilter("age", 25, 35)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(filters) != 2 {
			t.Errorf("len(filters) = %v, want 2", len(filters))
		}
		if filters[0].Op != ">=" {
			t.Errorf("filters[0].Op = %v, want >=", filters[0].Op)
		}
		if filters[1].Op != "<=" {
			t.Errorf("filters[1].Op = %v, want <=", filters[1].Op)
		}
	})

	t.Run("MinOnly", func(t *testing.T) {
		filters, err := CreateRangeFilter("age", 25, nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(filters) != 1 {
			t.Errorf("len(filters) = %v, want 1", len(filters))
		}
		if filters[0].Op != ">=" {
			t.Errorf("filters[0].Op = %v, want >=", filters[0].Op)
		}
	})

	t.Run("MaxOnly", func(t *testing.T) {
		filters, err := CreateRangeFilter("age", nil, 35)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(filters) != 1 {
			t.Errorf("len(filters) = %v, want 1", len(filters))
		}
		if filters[0].Op != "<=" {
			t.Errorf("filters[0].Op = %v, want <=", filters[0].Op)
		}
	})

	t.Run("InvalidRange", func(t *testing.T) {
		_, err := CreateRangeFilter("age", nil, nil)
		if err == nil {
			t.Errorf("expected error but got none")
		}
	})
}

// TestScanOptionsSourceTypes validates that ScanOptions accepts all supported source types
func TestScanOptionsSourceTypes(t *testing.T) {
	ctx := go_context.Background()

	// Fake iceberg REST server: returns a minimal valid config response for any request.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"defaults":{}, "overrides":{}}`)
	}))
	defer ts.Close()

	// Mock credentials so SigV4-enabled REST catalogs can sign requests without panicking.
	mockCreds := aws.CredentialsProviderFunc(func(ctx go_context.Context) (aws.Credentials, error) {
		return aws.Credentials{
			AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
			SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		}, nil
	})
	awsConfig := &aws.Config{Region: "us-east-1", Credentials: mockCreds}

	tests := []struct {
		name        string
		opts        ScanOptions
		expectError bool
		errContains string
	}{
		{
			name: "AWS_GLUE default (empty SourceType)",
			opts: ScanOptions{
				Database:  "db",
				Table:     "tbl",
				AwsConfig: awsConfig,
			},
			expectError: false,
		},
		{
			name: "AWS_GLUE explicit",
			opts: ScanOptions{
				Database:   "db",
				Table:      "tbl",
				SourceType: "AWS_GLUE",
				AwsConfig:  awsConfig,
			},
			expectError: false,
		},
		{
			name: "AWS_GLUE_REST requires URI",
			opts: ScanOptions{
				Database:   "db",
				Table:      "tbl",
				SourceType: "AWS_GLUE_REST",
				AwsConfig:  awsConfig,
			},
			expectError: true,
			errContains: "URI is required",
		},
		{
			name: "AWS_GLUE_REST with URI",
			opts: ScanOptions{
				Database:           "db",
				Table:              "tbl",
				SourceType:         "AWS_GLUE_REST",
				URI:                ts.URL,
				SigV4SigningRegion: "us-east-1",
				AwsConfig:          awsConfig,
			},
			expectError: false,
		},
		{
			name: "S3_TABLES requires URI",
			opts: ScanOptions{
				Database:   "db",
				Table:      "tbl",
				SourceType: "S3_TABLES",
				AwsConfig:  awsConfig,
			},
			expectError: true,
			errContains: "URI is required",
		},
		{
			name: "BIGLAKE_METASTORE requires URI",
			opts: ScanOptions{
				Database:   "db",
				Table:      "tbl",
				SourceType: "BIGLAKE_METASTORE",
			},
			expectError: true,
			errContains: "URI is required",
		},
		{
			name: "NESSIE requires URI",
			opts: ScanOptions{
				Database:   "db",
				Table:      "tbl",
				SourceType: "NESSIE",
			},
			expectError: true,
			errContains: "URI is required",
		},
		{
			name: "NESSIE requires http/https URI",
			opts: ScanOptions{
				Database:   "db",
				Table:      "tbl",
				SourceType: "NESSIE",
				URI:        "localhost:19120/iceberg",
			},
			expectError: true,
			errContains: "must start with http",
		},
		{
			name: "NESSIE_REST requires URI",
			opts: ScanOptions{
				Database:   "db",
				Table:      "tbl",
				SourceType: "NESSIE_REST",
			},
			expectError: true,
			errContains: "URI is required",
		},
		{
			name: "unsupported source type",
			opts: ScanOptions{
				Database:   "db",
				Table:      "tbl",
				SourceType: "UNKNOWN_TYPE",
				AwsConfig:  awsConfig,
			},
			expectError: true,
			errContains: "unsupported source type",
		},
		{
			name: "AWS source without AwsConfig",
			opts: ScanOptions{
				Database:   "db",
				Table:      "tbl",
				SourceType: "AWS_GLUE",
			},
			expectError: true,
			errContains: "AWS config is required",
		},
		{
			name: "NESSIE without AwsConfig (allowed)",
			opts: ScanOptions{
				Database:   "db",
				Table:      "tbl",
				SourceType: "NESSIE",
				URI:        ts.URL,
			},
			expectError: false,
		},
		{
			name: "NESSIE_REST without AwsConfig (allowed)",
			opts: ScanOptions{
				Database:   "db",
				Table:      "tbl",
				SourceType: "NESSIE_REST",
				URI:        ts.URL,
			},
			expectError: false,
		},
		{
			// BIGLAKE_METASTORE does not require AwsConfig, but it does require a GCP credential.
			name: "BIGLAKE_METASTORE without AwsConfig (allowed)",
			opts: ScanOptions{
				Database:   "db",
				Table:      "tbl",
				SourceType: "BIGLAKE_METASTORE",
				URI:        "https://biglake.googleapis.com/v1/projects/proj/locations/us",
			},
			expectError: true,
			errContains: "GCP credential not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner, err := NewScanner(ctx, tt.opts, nil)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					if scanner != nil {
						scanner.Close()
					}
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if scanner == nil {
					t.Error("scanner is nil")
					return
				}
				scanner.Close()
			}
		})
	}
}

// TestScanOptionsRequiredFields tests validation of required fields
func TestScanOptionsRequiredFields(t *testing.T) {
	ctx := go_context.Background()
	awsConfig := &aws.Config{Region: "us-east-1"}

	t.Run("MissingDatabase", func(t *testing.T) {
		_, err := NewScanner(ctx, ScanOptions{
			Table:     "tbl",
			AwsConfig: awsConfig,
		}, nil)
		if err == nil {
			t.Error("expected error for missing database")
		}
	})

	t.Run("MissingTable", func(t *testing.T) {
		_, err := NewScanner(ctx, ScanOptions{
			Database:  "db",
			AwsConfig: awsConfig,
		}, nil)
		if err == nil {
			t.Error("expected error for missing table")
		}
	})
}

// TestScanOptionsParallelScans validates ParallelScans parameter behaviour
func TestScanOptionsParallelScans(t *testing.T) {
	ctx := go_context.Background()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"defaults":{}, "overrides":{}}`)
	}))
	defer ts.Close()

	tests := []struct {
		name          string
		parallelScans int
		sourceType    string
		uri           string
		awsConfig     *aws.Config
		cat           catalog.Catalog
	}{
		{
			name:          "default parallelism (0) for AWS_GLUE",
			parallelScans: 0,
			sourceType:    "AWS_GLUE",
			awsConfig:     &aws.Config{Region: "us-east-1"},
		},
		{
			name:          "custom parallelism for AWS_GLUE",
			parallelScans: 8,
			sourceType:    "AWS_GLUE",
			awsConfig:     &aws.Config{Region: "us-east-1"},
		},
		{
			name:          "custom parallelism for NESSIE",
			parallelScans: 4,
			sourceType:    "NESSIE",
			uri:           ts.URL,
		},
		{
			name:          "custom parallelism for NESSIE_REST",
			parallelScans: 2,
			sourceType:    "NESSIE_REST",
			uri:           ts.URL,
		},
		{
			name:          "high parallelism for BIGLAKE_METASTORE",
			parallelScans: 16,
			sourceType:    "BIGLAKE_METASTORE",
			cat:           &stubCatalog{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner, err := NewScanner(ctx, ScanOptions{
				Database:      "db",
				Table:         "tbl",
				SourceType:    tt.sourceType,
				URI:           tt.uri,
				AwsConfig:     tt.awsConfig,
				ParallelScans: tt.parallelScans,
			}, tt.cat)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer scanner.Close()

			// Verify parallelScans is stored correctly
			if scanner.parallelScans != tt.parallelScans {
				t.Errorf("parallelScans = %d, want %d", scanner.parallelScans, tt.parallelScans)
			}
		})
	}
}

// TestScanOptionsSourceTypeNormalization validates that source type is normalized to uppercase
func TestScanOptionsSourceTypeNormalization(t *testing.T) {
	ctx := go_context.Background()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"defaults":{}, "overrides":{}}`)
	}))
	defer ts.Close()

	tests := []struct {
		input    string
		expected string
		uri      string
		aws      *aws.Config
		cat      catalog.Catalog
	}{
		{"aws_glue", "AWS_GLUE", "", &aws.Config{Region: "us-east-1"}, nil},
		{"AWS_GLUE", "AWS_GLUE", "", &aws.Config{Region: "us-east-1"}, nil},
		{"nessie", "NESSIE", ts.URL, nil, nil},
		{"NESSIE_REST", "NESSIE_REST", ts.URL, nil, nil},
		{"biglake_metastore", "BIGLAKE_METASTORE", "", nil, &stubCatalog{}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			scanner, err := NewScanner(ctx, ScanOptions{
				Database:   "db",
				Table:      "tbl",
				SourceType: tt.input,
				URI:        tt.uri,
				AwsConfig:  tt.aws,
			}, tt.cat)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer scanner.Close()

			if scanner.sourceType != tt.expected {
				t.Errorf("sourceType = %q, want %q", scanner.sourceType, tt.expected)
			}
		})
	}
}

// TestArrowUtility tests the Arrow utility functions
func TestArrowUtility(t *testing.T) {
	// Test Arrow utility creation
	t.Run("CreateArrowUtility", func(t *testing.T) {
		util := NewArrowUtility()
		if util == nil {
			t.Error("Failed to create ArrowUtility")
		}
	})

	t.Run("CreateArrowUtilityWithOptions", func(t *testing.T) {
		opts := &ArrowReaderOptions{
			BatchSize: 8192,
			ZeroCopy:  true,
		}
		util := NewArrowUtilityWithOptions(opts)
		if util == nil {
			t.Error("Failed to create ArrowUtility with options")
		}
	})

	t.Run("NewBytesSink", func(t *testing.T) {
		sink := NewBytesSink()
		if sink == nil {
			t.Error("Failed to create BytesSink")
		}

		// Test writing
		data := []byte("test data")
		n, err := sink.Write(data)
		if err != nil {
			t.Errorf("Failed to write to BytesSink: %v", err)
		}
		if n != len(data) {
			t.Errorf("Write returned wrong length: got %d, want %d", n, len(data))
		}

		// Test reading
		bytes := sink.Bytes()
		if string(bytes) != string(data) {
			t.Errorf("Bytes mismatch: got %v, want %v", bytes, data)
		}

		// Test close
		if err := sink.Close(); err != nil {
			t.Errorf("Failed to close BytesSink: %v", err)
		}
	})
}

// TestArrowSchemaInfo tests Arrow schema analysis
func TestArrowSchemaInfo(t *testing.T) {
	allocator := memory.NewGoAllocator()
	arrowUtil := NewArrowUtilityWithOptions(&ArrowReaderOptions{
		MemoryAllocator: allocator,
	})

	t.Run("AnalyzeSimpleSchema", func(t *testing.T) {
		// Create a simple schema
		fields := []arrow.Field{
			{Name: "id", Type: arrow.PrimitiveTypes.Int64},
			{Name: "name", Type: arrow.BinaryTypes.String},
			{Name: "age", Type: arrow.PrimitiveTypes.Int32},
		}
		schema := arrow.NewSchema(fields, nil)

		info := arrowUtil.AnalyzeArrowSchema(schema, 10, 1000, 102400)

		if info.NumFields != 3 {
			t.Errorf("Got wrong number of fields: got %d, want 3", info.NumFields)
		}

		if info.TotalRows != 1000 {
			t.Errorf("Got wrong total rows: got %d, want 1000", info.TotalRows)
		}

		if len(info.ColumnTypes) != 3 {
			t.Errorf("Got wrong column types count: got %d, want 3", len(info.ColumnTypes))
		}

		// Check column types (Arrow uses "utf8" for binary string type)
		if info.ColumnTypes["id"] != "int64" {
			t.Errorf("Got wrong type for id: got %v, want int64", info.ColumnTypes["id"])
		}
		// Note: Arrow represents string type as "utf8" (binary string)
		if info.ColumnTypes["name"] != "utf8" && info.ColumnTypes["name"] != "string" {
			t.Errorf("Got wrong type for name: got %v, want utf8 or string", info.ColumnTypes["name"])
		}
		if info.ColumnTypes["age"] != "int32" {
			t.Errorf("Got wrong type for age: got %v, want int32", info.ColumnTypes["age"])
		}
	})

	t.Run("AnalyzeNestedSchema", func(t *testing.T) {
		// Create a nested schema with list and struct
		fields := []arrow.Field{
			{Name: "id", Type: arrow.PrimitiveTypes.Int64},
			{Name: "tags", Type: arrow.ListOf(arrow.BinaryTypes.String)},
			{Name: "address", Type: arrow.StructOf(
				arrow.Field{Name: "street", Type: arrow.BinaryTypes.String},
				arrow.Field{Name: "city", Type: arrow.BinaryTypes.String},
			)},
		}
		schema := arrow.NewSchema(fields, nil)

		info := arrowUtil.AnalyzeArrowSchema(schema, 5, 500, 51200)

		if len(info.ListColumns) != 1 {
			t.Errorf("Got wrong number of list columns: got %d, want 1", len(info.ListColumns))
		}

		if len(info.StructColumns) != 1 {
			t.Errorf("Got wrong number of struct columns: got %d, want 1", len(info.StructColumns))
		}

		if len(info.NestedTypes) != 2 {
			t.Errorf("Got wrong number of nested types: got %d, want 2", len(info.NestedTypes))
		}
	})

	t.Run("AnalyzeDecimalSchema", func(t *testing.T) {
		// Create schema with decimal types
		fields := []arrow.Field{
			{Name: "id", Type: arrow.PrimitiveTypes.Int64},
			{Name: "amount", Type: &arrow.Decimal128Type{Precision: 10, Scale: 2}},
			{Name: "price", Type: &arrow.Decimal128Type{Precision: 18, Scale: 4}},
		}
		schema := arrow.NewSchema(fields, nil)

		info := arrowUtil.AnalyzeArrowSchema(schema, 10, 1000, 102400)

		if len(info.DecimalFields) != 2 {
			t.Errorf("Got wrong number of decimal fields: got %d, want 2", len(info.DecimalFields))
		}

		// Check decimal field names
		if !contains(info.DecimalFields, "amount") {
			t.Errorf("Decimal fields should contain 'amount'")
		}
		if !contains(info.DecimalFields, "price") {
			t.Errorf("Decimal fields should contain 'price'")
		}
	})
}

// TestArrowDataConversion tests Arrow to JSON conversion
func TestArrowDataConversion(t *testing.T) {
	allocator := memory.NewGoAllocator()
	arrowUtil := NewArrowUtilityWithOptions(&ArrowReaderOptions{
		MemoryAllocator: allocator,
	})

	t.Run("SimpleRecordBatch", func(t *testing.T) {
		// Create a simple record batch
		builder := array.NewRecordBuilder(allocator,
			arrow.NewSchema([]arrow.Field{
				{Name: "id", Type: arrow.PrimitiveTypes.Int64},
				{Name: "name", Type: arrow.BinaryTypes.String},
				{Name: "age", Type: arrow.PrimitiveTypes.Int32},
			}, nil))

		// Add some data
		for i := int64(0); i < 5; i++ {
			builder.Field(0).(*array.Int64Builder).Append(i)
			builder.Field(1).(*array.StringBuilder).Append(fmt.Sprintf("name_%d", i))
			builder.Field(2).(*array.Int32Builder).Append(int32(20 + i))
		}

		batch := builder.NewRecord()
		defer batch.Release()

		// Convert to JSON
		jsonLines, err := arrowUtil.ArrowToJson(batch)
		if err != nil {
			t.Fatalf("Failed to convert to JSON: %v", err)
		}

		if len(jsonLines) != 5 {
			t.Errorf("Got wrong number of JSON lines: got %d, want 5", len(jsonLines))
		}

		// Verify format
		for _, line := range jsonLines {
			if !isJSON(line) {
				t.Errorf("Invalid JSON: %s", line)
			}
			if !containsString(line, "\"id\"") {
				t.Errorf("JSON should contain 'id' field")
			}
		}
	})

	t.Run("NestedRecordBatch", func(t *testing.T) {
		// Create a nested record batch with list
		builder := array.NewRecordBuilder(allocator,
			arrow.NewSchema([]arrow.Field{
				{Name: "id", Type: arrow.PrimitiveTypes.Int64},
				{Name: "tags", Type: arrow.ListOf(arrow.BinaryTypes.String)},
			}, nil))

		// Add data with proper row alignment
		builder.Field(0).(*array.Int64Builder).Append(1)

		listBuilder1 := builder.Field(1).(*array.ListBuilder)
		stringBuilder1 := listBuilder1.ValueBuilder().(*array.StringBuilder)
		listBuilder1.Append(true)
		stringBuilder1.Append("tag1")
		stringBuilder1.Append("tag2")

		builder.Field(0).(*array.Int64Builder).Append(2)

		listBuilder2 := builder.Field(1).(*array.ListBuilder)
		listBuilder2.AppendNull()

		builder.Field(0).(*array.Int64Builder).Append(3)

		listBuilder3 := builder.Field(1).(*array.ListBuilder)
		stringBuilder3 := listBuilder3.ValueBuilder().(*array.StringBuilder)
		listBuilder3.Append(true)
		stringBuilder3.Append("tag3")

		batch := builder.NewRecord()
		defer batch.Release()

		// Convert to JSON
		jsonLines, err := arrowUtil.ArrowToJson(batch)
		if err != nil {
			t.Fatalf("Failed to convert nested batch to JSON: %v", err)
		}

		// Verify we got 3 rows
		if len(jsonLines) != 3 {
			t.Errorf("Expected 3 JSON lines, got %d", len(jsonLines))
		}

		// Verify list representation
		for _, line := range jsonLines {
			if !containsString(line, "\"tags\"") {
				t.Errorf("JSON should contain 'tags' field")
			}
		}
	})
}

// TestArrowIPC tests Arrow IPC format conversion
func TestArrowIPC(t *testing.T) {
	allocator := memory.NewGoAllocator()
	arrowUtil := NewArrowUtilityWithOptions(&ArrowReaderOptions{
		MemoryAllocator: allocator,
	})

	t.Run("ArrowToArrowIPCRoundTrip", func(t *testing.T) {
		// Create a record batch
		builder := array.NewRecordBuilder(allocator,
			arrow.NewSchema([]arrow.Field{
				{Name: "id", Type: arrow.PrimitiveTypes.Int64},
				{Name: "name", Type: arrow.BinaryTypes.String},
			}, nil))

		for i := int64(0); i < 5; i++ {
			builder.Field(0).(*array.Int64Builder).Append(i)
			builder.Field(1).(*array.StringBuilder).Append(fmt.Sprintf("name_%d", i))
		}

		batch := builder.NewRecord()
		defer batch.Release()

		// Convert to IPC
		ipcData, err := arrowUtil.ArrowToArrowIPC(batch.Schema(), []arrow.RecordBatch{batch})
		if err != nil {
			t.Fatalf("Failed to convert to IPC: %v", err)
		}

		if len(ipcData) == 0 {
			t.Error("IPC data is empty")
		}

		// Read back from IPC
		schema, batches, err := arrowUtil.ArrowFromArrowIPC(ipcData)
		if err != nil {
			t.Fatalf("Failed to read from IPC: %v", err)
		}

		if schema == nil {
			t.Error("Schema is nil after IPC round trip")
		}

		if len(batches) != 1 {
			t.Errorf("Got wrong number of batches from IPC: got %d, want 1", len(batches))
		}

		// Verify data
		readBatch := batches[0]
		if readBatch.NumRows() != batch.NumRows() {
			t.Errorf("Row count mismatch after IPC round trip: got %d, want %d", readBatch.NumRows(), batch.NumRows())
		}

		// Convert both to JSON for comparison
		originalJson, err := arrowUtil.ArrowToJson(batch)
		if err != nil {
			t.Fatalf("Failed to convert original to JSON: %v", err)
		}

		readJson, err := arrowUtil.ArrowToJson(readBatch)
		if err != nil {
			t.Fatalf("Failed to convert read batch to JSON: %v", err)
		}

		if len(originalJson) != len(readJson) {
			t.Errorf("JSON line count mismatch: got %d, want %d", len(readJson), len(originalJson))
		}
	})
}

// TestFormatSupport tests format compatibility
func TestFormatSupport(t *testing.T) {
	arrowUtil := NewArrowUtility()

	t.Run("FormatCompatibility", func(t *testing.T) {
		formats := []FileFormat{
			FormatParquet,
			FormatORC,
			FormatAvro,
			FormatArrow,
			FormatCSV,
		}

		for _, format := range formats {
			if !arrowUtil.FormatCompatible(format) {
				t.Errorf("Format %s should be compatible", format)
			}
		}

		// Unknown format should not be compatible
		if arrowUtil.FormatCompatible("unknown") {
			t.Error("Unknown format should not be compatible")
		}
	})

	t.Run("FormatCompression", func(t *testing.T) {
		compressions := map[FileFormat]string{
			FormatParquet: arrowUtil.GetFormatCompression(FormatParquet),
			FormatORC:     arrowUtil.GetFormatCompression(FormatORC),
			FormatAvro:    arrowUtil.GetFormatCompression(FormatAvro),
			FormatArrow:   arrowUtil.GetFormatCompression(FormatArrow),
		}

		expectedFormats := map[FileFormat]string{
			FormatParquet: "snappy,zstd,gzip",
			FormatORC:     "zstd,snappy,lzo",
			FormatAvro:    "snappy,deflate,bzip2",
			FormatArrow:   "lz4,zstd,none",
		}

		for format, expected := range expectedFormats {
			if compressions[format] != expected {
				t.Errorf("Compression for %s: got %s, want %s", format, compressions[format], expected)
			}
		}
	})
}

// Helper functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func isJSON(s string) bool {
	return len(s) > 0 && (strings.HasPrefix(s, "{") || strings.HasPrefix(s, "["))
}

// TestFilterPushdown tests filter expression creation
func TestFilterPushdown(t *testing.T) {
	ctx := go_context.Background()

	// Create mock AWS config (won't be used for this test)
	awsConfig := &aws.Config{
		Region: "us-east-1",
	}

	// Create scanner options
	opts := ScanOptions{
		Database:      "test_db",
		Table:         "test_table",
		CaseSensitive: true,
		AwsConfig:     awsConfig,
	}

	scanner, err := NewScanner(ctx, opts, nil)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}
	defer scanner.Close()

	// Test filter pushdown
	filterPushdown := NewFilterPushdown(nil, true)
	if filterPushdown == nil {
		t.Fatal("Failed to create filter pushdown")
	}

	// Test creating various filters
	tests := []struct {
		name     string
		filter   IcebergFilter
		notEmpty bool
	}{
		{
			name: "EqualFilter",
			filter: IcebergFilter{
				Op:    "=",
				Field: "status",
				Value: "active",
			},
			notEmpty: true,
		},
		{
			name: "GreaterThanFilter",
			filter: IcebergFilter{
				Op:    ">",
				Field: "age",
				Value: 30,
			},
			notEmpty: true,
		},
		{
			name: "AndFilter",
			filter: CreateAndFilter(
				CreateEqualFilter("status", "active"),
				CreateEqualFilter("age", 30),
			),
			notEmpty: true,
		},
		{
			name: "OrFilter",
			filter: CreateOrFilter(
				CreateEqualFilter("status", "active"),
				CreateEqualFilter("status", "pending"),
			),
			notEmpty: true,
		},
		{
			name:     "InFilter",
			filter:   CreateInFilter("category", "electronics", "books", "home"),
			notEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := filterPushdown.ConvertFilter(tt.filter)
			if err != nil {
				t.Errorf("Failed to convert filter: %v", err)
				return
			}

			if expr == nil && tt.notEmpty {
				t.Error("Expression is nil when it shouldn't be")
			}

			if expr != nil {
				t.Logf("Created expression: %v", expr)
			}
		})
	}

	// Test ApplyFilters
	filters := []IcebergFilter{
		CreateEqualFilter("status", "active"),
		CreateEqualFilter("age", 30),
	}

	if err := filterPushdown.ApplyFilters(filters); err != nil {
		t.Errorf("Failed to apply filters: %v", err)
	}

	expr := filterPushdown.GetExpression()
	if expr == nil {
		t.Error("GetExpression returned nil")
	} else {
		t.Logf("Combined expression: %v", expr)
	}
}

// TestScannerFilterLifecycle tests the full filter lifecycle
func TestScannerFilterLifecycle(t *testing.T) {
	ctx := go_context.Background()

	// Create mock AWS config
	awsConfig := &aws.Config{
		Region: "us-east-1",
	}

	// Create scanner
	opts := ScanOptions{
		Database:      "test_db",
		Table:         "test_table",
		CaseSensitive: true,
		AwsConfig:     awsConfig,
	}

	scanner, err := NewScanner(ctx, opts, nil)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}
	defer scanner.Close()

	// Test that filter pushdown is initialized when filters are provided
	filters := []IcebergFilter{
		CreateEqualFilter("status", "active"),
	}

	optsWithFilters := ScanOptions{
		Database:      "test_db",
		Table:         "test_table",
		CaseSensitive: true,
		AwsConfig:     awsConfig,
		Filters:       filters,
	}

	scannerWithFilters, err := NewScanner(ctx, optsWithFilters, nil)
	if err != nil {
		t.Fatalf("Failed to create scanner with filters: %v", err)
	}
	defer scannerWithFilters.Close()

	if scannerWithFilters.filterPushdown == nil {
		t.Error("Filter pushdown not initialized when filters provided")
	}

	t.Logf("Scanner with filters created successfully")
}

// TestNESSIESourceTypeRouting validates that NESSIE source types are correctly identified
func TestNESSIESourceTypeRouting(t *testing.T) {
	ctx := go_context.Background()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"defaults":{}, "overrides":{}}`)
	}))
	defer ts.Close()

	tests := []struct {
		sourceType string
		uri        string
		isNESSIE   bool
	}{
		{"NESSIE", ts.URL, true},
		{"nessie", ts.URL, true},
		{"NESSIE_REST", ts.URL, true},
		{"nessie_rest", ts.URL, true},
		{"AWS_GLUE", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.sourceType, func(t *testing.T) {
			var awsCfg *aws.Config
			if !tt.isNESSIE {
				awsCfg = &aws.Config{Region: "us-east-1"}
			}
			scanner, err := NewScanner(ctx, ScanOptions{
				Database:   "db",
				Table:      "tbl",
				SourceType: tt.sourceType,
				URI:        tt.uri,
				AwsConfig:  awsCfg,
			}, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer scanner.Close()

			normalizedType := scanner.sourceType
			isNESSIE := normalizedType == "NESSIE" || normalizedType == "NESSIE_REST"
			if isNESSIE != tt.isNESSIE {
				t.Errorf("sourceType %q: isNESSIE = %v, want %v", tt.sourceType, isNESSIE, tt.isNESSIE)
			}
		})
	}
}

/*
// Example 1: Read Iceberg table and convert to JSON using Arrow
func ExampleArrowIntegration() {
    ctx := go_context.Background()

    // Create AWS config
    awsConfig, err := NewAWSConfig(
        "access-key-id",
        "secret-access-key",
        "",
        "us-east-1",
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create scanner
    opts := ScanOptions{
        Database:  "my_database",
        Table:     "my_table",
        AwsConfig: awsConfig,
    }

    scanner, err := NewScanner(ctx, opts, nil)
    if err != nil {
        log.Fatal(err)
    }
    defer scanner.Close()

    scanner.LoadTable(ctx)

    // Create scan and reader
    scan, _ := scanner.Scan(ctx)
    reader, _ := NewReader(ctx, scan)
    defer reader.Close()

    // Create Arrow utility
    arrowUtil := NewArrowUtility()

    // Read data and convert to JSON
    iterator := NewIterator(reader)
    for iterator.Next() {
        currentRecord := reader.(*Reader).currentRecord

        // Convert Arrow record to JSON
        jsonLines, err := arrowUtil.ArrowToJson(currentRecord)
        if err != nil {
            log.Fatal(err)
        }

        for _, line := range jsonLines {
            fmt.Println(line)
        }
    }
}
*/

// This file contains examples of using Arrow integration for different file formats.

/*
// Example 1: Read Iceberg table and convert to JSON using Arrow
func ExampleArrowIntegration() {
    ctx := go_context.Background()

    // Create AWS config
    awsConfig, err := NewAWSConfig(
        "access-key-id",
        "secret-access-key",
        "",
        "us-east-1",
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create scanner
    opts := ScanOptions{
        Database:  "my_database",
        Table:     "my_table",
        AwsConfig: awsConfig,
    }

    scanner, err := NewScanner(ctx, opts, nil)
    if err != nil {
        log.Fatal(err)
    }
    defer scanner.Close()

    scanner.LoadTable(ctx)

    // Create scan and reader
    scan, _ := scanner.Scan(ctx)
    reader, _ := NewReader(ctx, scan)
    defer reader.Close()

    // Create Arrow utility
    arrowUtil := NewArrowUtility()

    // Read data and convert to JSON
    iterator := NewIterator(reader)
    for iterator.Next() {
        // Get current Arrow record
        scanner := scanner.(*Scanner)
        currentRecord := reader.(*Reader).currentRecord

        // Convert Arrow record to JSON
        jsonLines, err := arrowUtil.ArrowToJson(currentRecord)
        if err != nil {
            log.Fatal(err)
        }

        for _, line := range jsonLines {
            fmt.Println(line)
        }
    }
}

// Example 2: Analyze Arrow schema from Iceberg table
func ExampleArrowSchemaAnalysis() {
    ctx := go_context.Background()

    // Setup scanner (as in Example 1)
    // ...

    scanner.LoadTable(ctx)

    // Get Arrow schema
    schema := scanner.GetSchema().(*arrow.Schema)

    // Create Arrow utility with custom options
    arrowUtil := NewArrowUtilityWithOptions(&ArrowReaderOptions{
        BatchSize: 8192,
        ZeroCopy:  true,
    })

    // Create a scan to get schema info
    scan, _ := scanner.Scan(ctx)
    reader, _ := NewReader(ctx, scan)

    // Analyze schema
    schemaInfo := arrowUtil.AnalyzeArrowSchema(schema, 10, 1000, 102400)

    fmt.Println("Schema Analysis:")
    fmt.Printf("  Number of fields: %d\n", schemaInfo.NumFields)
    fmt.Printf("  Number of batches: %d\n", schemaInfo.NumBatches)
    fmt.Printf("  Total rows: %d\n", schemaInfo.TotalRows)

    // Check for nested types
    if len(schemaInfo.ListColumns) > 0 {
        fmt.Printf("  List columns: %v\n", schemaInfo.ListColumns)
    }
    if len(schemaInfo.StructColumns) > 0 {
        fmt.Printf("  Struct columns: %v\n", schemaInfo.StructColumns)
    }
    if len(schemaInfo.DecimalFields) > 0 {
        fmt.Printf("  Decimal fields: %v\n", schemaInfo.DecimalFields)
    }

    // Print column types
    fmt.Println("\nColumn Types:")
    for col, typ := range schemaInfo.ColumnTypes {
        fmt.Printf("  %s: %s\n", col, typ)
    }
}

// Example 3: Convert to different Arrow formats
func ExampleArrowFormatConversion() {
    ctx := go_context.Background()

    // Setup scanner and read data (as in previous examples)
    // ...

    // Collect RecordBatches
    arrowUtil := NewArrowUtility()

    var batches []arrow.RecordBatch
    reader := getReaderFromScanner(scanner) // Assume this function exists

    for reader.Next() {
        scanner := scanner.(*Scanner)
        rdr := reader.(*Reader)
        if rdr.currentRecord != nil {
            rdr.currentRecord.Retain()
            batches = append(batches, rdr.currentRecord)
        }
    }
    reader.Close()

    if len(batches) == 0 {
        log.Println("No data to convert")
        return
    }

    // Convert to Arrow IPC (binary format)
    schema := batches[0].Schema()
    ipcData, err := arrowUtil.ArrowToArrowIPC(schema, batches)
    if err != nil {
        log.Fatal(err)
    }

    // Save to file
    err := os.WriteFile("data.arrow.ipc", ipcData, 0644)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Converted %d batches to Arrow IPC format (%d bytes)\n",
        len(batches), len(ipcData))

    // Read back from IPC file
    ipcBytes, _ := os.ReadFile("data.arrow.ipc")
    readSchema, readBatches, err := arrowUtil.ArrowFromArrowIPC(ipcBytes)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Read back %d batches from IPC\n", len(readBatches))
}

// Example 4: Process Arrow data with column projection
func ExampleArrowColumnProjection() {
    ctx := go_context.Background()

    // Setup scanner (as in Example 1)
    // ...

    // Create Arrow utility with column filter
    neededColumns := map[string]bool{
        "id":    true,
        "name":  true,
        "status": true,
    }

    arrowUtil := NewArrowUtilityWithOptions(&ArrowReaderOptions{
        ColumnFilter: func(colName string) bool {
            return neededColumns[colName]
        },
    })

    scanner.LoadTable(ctx)
    scan, _ := scanner.Scan(ctx)
    reader, _ := NewReader(ctx, scan)

    // Process only selected columns
    iterator := NewIterator(reader)
    for iterator.Next() {
        scanner := scanner.(*Scanner)
        rdr := reader.(*Reader)
        currentRecord := rdr.currentRecord

        // Get record statistics
        stats := arrowUtil.GetRecordBatchStats(currentRecord)

        // Print null counts for each column
        fmt.Printf("Record batch Stats:\n")
        fmt.Printf("  Rows: %d\n", stats.NumRows)
        fmt.Printf("  Columns: %d\n", stats.NumColumns)

        for col, nullCount := range stats.NullValues {
            fmt.Printf("  %s: %d null values\n", col, nullCount)
        }
    }
}

// Example 5: Handle nested Arrow data
func ExampleArrowNestedData() {
    ctx := go_context.Background()

    // Setup scanner (as in Example 1)
    // ...

    scanner.LoadTable(ctx)
    scan, _ := scanner.Scan(ctx)
    reader, _ := NewReader(ctx, scan)

    arrowUtil := NewArrowUtility()

    iterator := NewIterator(reader)
    for iterator.Next() {
        scanner := scanner.(*Scanner)
        rdr := reader.(*Reader)
        currentRecord := rdr.currentRecord

        // Convert to JSON - will preserve nested structures
        jsonLines, err := arrowUtil.ArrowToJson(currentRecord)
        if err != nil {
            log.Fatal(err)
        }

        // Print JSON lines
        for _, line := range jsonLines {
            fmt.Println(line)
        }
    }
}

// Example 6: Format detection and compatibility check
func ExampleArrowFormatSupport() {
    arrowUtil := NewArrowUtility()

    // Check format compatibility
    formats := []FileFormat{
        FormatParquet,
        FormatORC,
        FormatAvro,
        FormatArrow,
        FormatCSV,
    }

    fmt.Println("Format Compatibility:")
    for _, format := range formats {
        if arrowUtil.FormatCompatible(format) {
            compression := arrowUtil.GetFormatCompression(format)
            fmt.Printf("  %s: Compatible (compressions: %s)\n", format, compression)
        } else {
            fmt.Printf("  %s: Not compatible\n", format)
        }
    }

    // Parse S3 URI to get file extension
    uri := "s3://my-bucket/path/to/data.parquet"
    _, key, err := ParseS3URI(uri)
    if err == nil {
        ext := filepath.Ext(key)
        format := FileFormat(strings.TrimPrefix(ext, "."))

        fmt.Printf("\nFile format from S3 URI: %s\n", format)
        fmt.Printf("  Compression options: %s\n",
            arrowUtil.GetFormatCompression(format))
    }
}

// Example 7: Streaming Arrow data to JSON
func ExampleArrowStreamingJSON() {
    ctx := go_context.Background()

    // Setup scanner (as in Example 1)
    // ...

    scanner.LoadTable(ctx)
    scan, _ := scanner.Scan(ctx)
    reader, _ := NewReader(ctx, scan)

    arrowUtil := NewArrowUtility()

    // Stream JSON to stdout
    iterator := NewIterator(reader)
    for iterator.Next() {
        scanner := scanner.(*Scanner)
        rdr := reader.(*Reader)
        currentRecord := rdr.currentRecord

        // Convert to JSON
        jsonLines, err := arrowUtil.ArrowToJson(currentRecord)
        if err != nil {
            log.Fatal(err)
        }

        // Write to stdout or file
        for _, line := range jsonLines {
            fmt.Println(line) // or write to file
        }
    }
    reader.Close()
}

// Example 8: Process Arrow data with memory management
func ExampleArrowMemoryManagement() {
    ctx := go_context.Background()

    // Setup scanner (as in Example 1)
    // ...

    // Create custom memory allocator for better control
    import "github.com/apache/arrow-go/v18/arrow/memory"

    allocator := memory.NewGoAllocator()
    arrowUtil := NewArrowUtilityWithOptions(&ArrowReaderOptions{
        MemoryAllocator: allocator,
        BatchSize:       4096,  // Smaller batches for memory efficiency
    })

    scanner.LoadTable(ctx)
    scan, _ := scanner.Scan(ctx)
    reader, _ := NewReader(ctx, scan)

    // Process data with manual memory management
    iterator := NewIterator(reader)
    for iterator.Next() {
        scanner := scanner.(*Scanner)
        rdr := reader.(*Reader)
        currentRecord := rdr.currentRecord

        // Retain the record if we need it longer
        currentRecord.Retain()
        defer currentRecord.Release()

        // Process data
        processRecordBatch(currentRecord, arrowUtil)
    }
    reader.Close()
    allocator.Release()
}

func processRecordBatch(batch arrow.RecordBatch, arrowUtil *ArrowUtility) {
    // Get statistics
    stats := arrowUtil.GetRecordBatchStats(batch)
    // Process based on statistics
    fmt.Printf("Processed batch with %d rows\n", stats.NumRows)
}

*/
