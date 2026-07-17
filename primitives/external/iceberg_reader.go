//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of the
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package external

import (
	"bytes"
	go_context "context"
	"encoding/json"
	"fmt"
	stdio "io"
	"io/fs"
	"iter"
	"net/http"
	"strings"
	"sync"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/csv"
	"github.com/apache/arrow-go/v18/arrow/decimal"
	"github.com/apache/arrow-go/v18/arrow/extensions"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/parquet/variant"
	"github.com/apache/iceberg-go/io"
	"github.com/apache/iceberg-go/table"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/couchbase/query/logging"
	"github.com/google/uuid"
)

// Reader handles iteration over Iceberg table scan results using Arrow records
type Reader struct {
	ctx             go_context.Context
	scan            *table.Scan
	arrowSchema     *arrow.Schema
	recordIter      iter.Seq2[arrow.RecordBatch, error]
	currentRecord   arrow.RecordBatch
	currentRow      int
	schema          interface{}
	columnFilter    func(string) bool // Optional column filter function
	decimalToDouble bool              // When true, Decimal128/256 columns yield float64 instead of string
	lastError       error             // Last error encountered during iteration
	mu              sync.Mutex
	closed          bool
}

// NewReader creates a new reader for an Iceberg table using arrow records
func NewReader(ctx go_context.Context, scan *table.Scan) (*Reader, error) {
	if scan == nil {
		return nil, fmt.Errorf("scan cannot be nil")
	}

	reader := &Reader{
		ctx:    ctx,
		scan:   scan,
		closed: false,
	}

	schema, recordItr, err := scan.ToArrowRecords(ctx)
	if err != nil {
		logging.Errorf("Iceberg Reader: failed to get arrow records: %v", err)
		return nil, fmt.Errorf("failed to get arrow records: %w", err)
	}

	reader.arrowSchema = schema
	reader.recordIter = recordItr

	if schema != nil {
		logging.Infof("Iceberg Reader initialized with schema: %d fields", len(schema.Fields()))
	} else {
		logging.Warnf("Iceberg Reader initialized with nil schema")
	}

	return reader, nil
}

// SetColumnFilter sets a column filter function for the reader
// Columns for which the filter returns false will be excluded from results
func (r *Reader) SetColumnFilter(filter func(string) bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.columnFilter = filter
}

// SetDecimalToDouble toggles whether Decimal128/256 columns are returned as float64
// (true) or as their string representation (false, the default).
func (r *Reader) SetDecimalToDouble(b bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.decimalToDouble = b
}

// Next advances the reader to the next row
// Returns false when no more rows are available
func (r *Reader) Next() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		logging.Debugf("Iceberg Reader.Next(): reader is closed, returning false")
		return false
	}

	// Check if context is already canceled at Next() entry
	select {
	case <-r.ctx.Done():
		logging.Errorf("Iceberg Reader: context already canceled at Next() entry: %v", r.ctx.Err())
		return false
	default:
		// Context is not canceled, continue
	}

	// If we have a current record, try to advance within it
	if r.currentRecord != nil {
		if r.currentRow < int(r.currentRecord.NumRows()) {
			r.currentRow++
			totalRows := int(r.currentRecord.NumRows())

			// Debug logging for first and last rows in each record
			if r.currentRow == 1 || r.currentRow == totalRows {
				logging.Debugf("Iceberg Reader.Next(): advancing to row %d/%d in current record batch", r.currentRow, totalRows)
			}

			return true
		}
	}

	// Need to get next record batch
	logging.Debugf("Iceberg Reader.Next(): loading next record batch")
	if err := r.loadNextRecord(); err != nil {
		r.lastError = err // Save the error for GetError()
		if err == stdio.EOF {
			logging.Debugf("Iceberg Reader.Next(): reached end of data (EOF), no more records")
		} else {
			logging.Errorf("Iceberg Reader.Next(): error loading next record: %v", err)
		}
		r.closed = true
		return false
	}

	// Successfully loaded a new record batch, advance to first row
	if r.currentRecord != nil {
		r.currentRow++
		rowCount := int(r.currentRecord.NumRows())
		logging.Debugf("Iceberg Reader.Next(): loaded new record batch, advancing to first row (row %d/%d)", r.currentRow, rowCount)
		return true
	}

	// No more records available
	logging.Infof("Iceberg Reader.Next(): no more record batches available")
	return false
}

// loadNextRecord loads the next record batch from the iterator
func (r *Reader) loadNextRecord() error {
	if r.recordIter == nil {
		logging.Warnf("Iceberg Reader: record iterator is nil, returning EOF")
		return stdio.EOF
	}

	// Release current record if any
	if r.currentRecord != nil {
		logging.Debugf("Iceberg Reader: releasing current record batch with %d rows", r.currentRecord.NumRows())
		r.currentRecord.Release()
		r.currentRecord = nil
	}

	// Get next record from iterator - use range loop
	recordCount := 0
	logging.Debugf("Iceberg Reader: starting to iterate over record iterator")

	// Check if context is canceled before starting iteration
	select {
	case <-r.ctx.Done():
		logging.Errorf("Iceberg Reader: context canceled before record iteration: %v", r.ctx.Err())
		return fmt.Errorf("context canceled: %w", r.ctx.Err())
	default:
		// Continue
	}

	for rec, err := range r.recordIter {
		logging.Debugf("Iceberg Reader: got record from iterator, checking for error")
		if err != nil {
			if r.ctx.Err() != nil {
				return stdio.EOF
			}
			logging.Errorf("Iceberg Reader: error iterating records: %v", err)
			return err
		}
		r.currentRecord = rec
		r.currentRow = 0
		return nil
	}

	logging.Debugf("Iceberg Reader: end of iterator reached, processed %d record batches", recordCount)
	return stdio.EOF
}

// Row represents a single row from Iceberg table
type Row struct {
	data map[string]interface{}
}

// Data returns the row data as a map
func (r *Row) Data() (map[string]interface{}, error) {
	if r.data == nil {
		return make(map[string]interface{}), nil
	}
	return r.data, nil
}

// Bytes returns the row data as JSON bytes
func (r *Row) Bytes() ([]byte, error) {
	data, err := r.Data()
	if err != nil {
		return nil, err
	}
	return json.Marshal(data)
}

// GetCurrentRow converts the current arrow record row to a map
func (r *Reader) GetCurrentRow() (map[string]interface{}, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.currentRecord == nil {
		return nil, fmt.Errorf("no record available")
	}

	if r.currentRow <= 0 || r.currentRow > int(r.currentRecord.NumRows()) {
		return nil, fmt.Errorf("invalid row state: currentRow=%d (must be 1-%d), NumRows=%d",
			r.currentRow, r.currentRecord.NumRows(), r.currentRecord.NumRows())
	}

	// Use r.currentRow - 1 because currentRow uses 1-based indexing after increment, but array is 0-based
	rowIndex := r.currentRow - 1

	row := make(map[string]interface{})
	schema := r.arrowSchema

	logging.Debugf("Iceberg GetCurrentRow: reading row at array index %d (1-based currentRow=%d) from record with %d rows, %d columns",
		rowIndex, r.currentRow, r.currentRecord.NumRows(), r.currentRecord.NumCols())

	// Iterate over each column
	for i := 0; i < int(schema.NumFields()); i++ {
		field := schema.Field(i)

		// Apply column filter if configured
		if r.columnFilter != nil {
			if !r.columnFilter(field.Name) {
				continue
			}
		}

		col := r.currentRecord.Column(i)

		// Get the value at the correct row index
		if rowIndex < int(col.Len()) {
			if !col.IsNull(rowIndex) {
				row[field.Name] = r.getColumnValue(col, rowIndex)
			} else {
				row[field.Name] = nil
			}
		} else {
			logging.Warnf("Iceberg GetCurrentRow: rowIndex %d >= column.Len() %d, skipping column %s",
				rowIndex, col.Len(), field.Name)
			row[field.Name] = nil
		}
	}

	return row, nil
}

// getColumnValue extracts a value from an arrow array at a specific position
func (r *Reader) getColumnValue(col arrow.Array, pos int) interface{} {
	switch arr := col.(type) {
	case *array.Boolean:
		return arr.Value(pos)
	case *array.Int8:
		return arr.Value(pos)
	case *array.Int16:
		return arr.Value(pos)
	case *array.Int32:
		return arr.Value(pos)
	case *array.Int64:
		return arr.Value(pos)
	case *array.Uint8:
		return arr.Value(pos)
	case *array.Uint16:
		return arr.Value(pos)
	case *array.Uint32:
		return arr.Value(pos)
	case *array.Uint64:
		return arr.Value(pos)
	case *array.Float32:
		return arr.Value(pos)
	case *array.Float64:
		return arr.Value(pos)
	case *array.String:
		return arr.Value(pos)
	case *array.Binary:
		return arr.Value(pos)
	case *array.LargeString:
		return arr.Value(pos)
	case *array.LargeBinary:
		return arr.Value(pos)
	case *array.Decimal128:
		if r.decimalToDouble {
			t := arr.DataType().(*arrow.Decimal128Type)
			return arr.Value(pos).ToFloat64(t.Scale)
		}
		return arr.ValueStr(pos)
	case *array.Decimal256:
		if r.decimalToDouble {
			t := arr.DataType().(*arrow.Decimal256Type)
			return arr.Value(pos).ToFloat64(t.Scale)
		}
		return arr.ValueStr(pos)
	case *array.Timestamp:
		return int64(arr.Value(pos))
	case *array.Date32:
		return int32(arr.Value(pos))
	case *array.Date64:
		return int64(arr.Value(pos))
	case *array.Time32:
		return int32(arr.Value(pos))
	case *array.Time64:
		return int64(arr.Value(pos))
	case *array.Struct:
		return r.getStructValue(arr, pos)
	case *array.List:
		return r.getListValue(arr, pos)
	case *array.LargeList:
		return r.getLargeListValue(arr, pos)
	case *array.FixedSizeList:
		return r.getFixedSizeListValue(arr, pos)
	case *array.Map:
		return r.getMapValue(arr, pos)
	case *extensions.VariantArray:
		return getVariantValue(arr, pos, r.decimalToDouble)
	case *array.Null:
		return nil
	default:
		return fmt.Sprintf("%v", arr.GetOneForMarshal(pos))
	}
}

// getVariantValue decodes a Parquet Variant (a binary-encoded superset of JSON) into
// native Go values (map[string]interface{}, []interface{}, string, int64, float64, bool,
// nil, ...) so it flows through the rest of the pipeline like any other JSON-shaped column.
func getVariantValue(arr *extensions.VariantArray, pos int, decimalToDouble bool) interface{} {
	if arr.IsNull(pos) {
		return nil
	}
	vv, err := arr.Value(pos)
	if err != nil {
		logging.Warnf("Iceberg getVariantValue: failed to decode variant at row %d: %v", pos, err)
		return nil
	}
	return decodeVariantScalar(vv, decimalToDouble)
}

// decodeVariantScalar recursively converts a decoded variant.Value into native Go values,
// mirroring the scalar conventions getColumnValue already uses for the same Arrow/Parquet
// logical types (timestamps/dates/times as epoch ints, decimals following decimalToDouble)
// so a value doesn't change shape depending on whether it arrived via a plain column or
// buried inside a Variant.
func decodeVariantScalar(vv variant.Value, decimalToDouble bool) interface{} {
	switch t := vv.Value().(type) {
	case nil, bool, int8, int16, int32, int64, float32, float64, string:
		return t
	case []byte:
		return t
	case arrow.Date32:
		return int32(t)
	case arrow.Timestamp:
		return int64(t)
	case arrow.Time64:
		return int64(t)
	case uuid.UUID:
		return t.String()
	case variant.DecimalValue[decimal.Decimal32]:
		return decodeVariantDecimal(t, decimalToDouble)
	case variant.DecimalValue[decimal.Decimal64]:
		return decodeVariantDecimal(t, decimalToDouble)
	case variant.DecimalValue[decimal.Decimal128]:
		return decodeVariantDecimal(t, decimalToDouble)
	case variant.ObjectValue:
		result := make(map[string]interface{}, t.NumElements())
		for key, fieldVal := range t.Values() {
			result[key] = decodeVariantScalar(fieldVal, decimalToDouble)
		}
		return result
	case variant.ArrayValue:
		result := make([]interface{}, 0, t.Len())
		for elemVal := range t.Values() {
			result = append(result, decodeVariantScalar(elemVal, decimalToDouble))
		}
		return result
	default:
		logging.Warnf("Iceberg decodeVariantScalar: unrecognized variant value type %T", t)
		return fmt.Sprintf("%v", t)
	}
}

func decodeVariantDecimal[T decimal.DecimalTypes](d variant.DecimalValue[T], decimalToDouble bool) interface{} {
	if decimalToDouble {
		return d.Value.ToFloat64(int32(d.Scale))
	}
	return d.Value.ToString(int32(d.Scale))
}

// getStructValue extracts a struct value
func (r *Reader) getStructValue(arr *array.Struct, pos int) map[string]interface{} {
	result := make(map[string]interface{})

	// Get the struct type to access field names
	dataType := arr.DataType()
	if structType, ok := dataType.(*arrow.StructType); ok {
		for i := 0; i < arr.NumField(); i++ {
			fieldArr := arr.Field(i)
			fieldName := structType.Field(i).Name

			if !fieldArr.IsNull(pos) {
				result[fieldName] = r.getColumnValue(fieldArr, pos)
			} else {
				result[fieldName] = nil
			}
		}
	}

	return result
}

// getListValue extracts a list value
func (r *Reader) getListValue(arr *array.List, pos int) []interface{} {
	list := arr.ListValues()
	if list == nil {
		return nil
	}

	start, end := arr.ValueOffsets(pos)
	listLen := int(end - start)
	result := make([]interface{}, listLen)
	for i := 0; i < listLen; i++ {
		if list.IsNull(int(start) + i) {
			result[i] = nil
		} else {
			result[i] = r.getColumnValue(list, int(start)+i)
		}
	}
	return result
}

// getLargeListValue extracts a large list value
func (r *Reader) getLargeListValue(arr *array.LargeList, pos int) []interface{} {
	listValues := arr.ListValues()
	if listValues == nil {
		return nil
	}
	start, end := arr.ValueOffsets(pos)
	listLen := int(end - start)
	result := make([]interface{}, listLen)
	for i := 0; i < listLen; i++ {
		if listValues.IsNull(int(start) + i) {
			result[i] = nil
		} else {
			result[i] = r.getColumnValue(listValues, int(start)+i)
		}
	}
	return result
}

// getFixedSizeListValue extracts a fixed-size list value
func (r *Reader) getFixedSizeListValue(arr *array.FixedSizeList, pos int) []interface{} {
	listLen := arr.Len()
	result := make([]interface{}, listLen)
	itemArr := arr.ListValues()
	for i := 0; i < listLen; i++ {
		if itemArr.IsNull(pos*listLen + i) {
			result[i] = nil
		} else {
			result[i] = r.getColumnValue(itemArr, pos*listLen+i)
		}
	}
	return result
}

// getMapValue extracts a map value
func (r *Reader) getMapValue(arr *array.Map, pos int) map[string]interface{} {
	result := make(map[string]interface{})
	keyItems := arr.Keys()
	valueItems := arr.Items()
	keysStart, keysEnd := arr.ValueOffsets(pos)
	numEntries := int(keysEnd - keysStart)
	for i := 0; i < numEntries; i++ {
		keyIdx := int(keysStart) + i
		if keyItems.IsNull(keyIdx) {
			continue
		}
		keyValue := r.getColumnValue(keyItems, keyIdx)
		keyStr, ok := keyValue.(string)
		if !ok {
			keyStr = fmt.Sprintf("%v", keyValue)
		}
		if valueItems.IsNull(keyIdx) {
			result[keyStr] = nil
		} else {
			result[keyStr] = r.getColumnValue(valueItems, keyIdx)
		}
	}
	return result
}

// GetError returns any error from the reader
func (r *Reader) GetError() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastError
}
func (r *Reader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}

	r.closed = true

	if r.currentRecord != nil {
		r.currentRecord.Release()
		r.currentRecord = nil
	}

	return nil
}

// Iterator provides a simplified interface for iterating over scan results
type Iterator struct {
	reader *Reader
	err    error
}

// NewIterator creates a new iterator from a reader
func NewIterator(reader *Reader) *Iterator {
	return &Iterator{reader: reader}
}

// Next advances to the next row
func (it *Iterator) Next() bool {
	if it.err != nil && it.err != stdio.EOF {
		return false
	}

	// Reset EOF error since we're trying again
	if it.err == stdio.EOF {
		it.err = nil
	}

	hasNext := it.reader.Next()
	if !hasNext {
		// Check if the reader has an error
		if readerErr := it.reader.GetError(); readerErr != nil {
			// If it's EOF (from context cancellation workaround), don't treat as error
			if readerErr == stdio.EOF {
				logging.Infof("Iterator: reached end of available records (partial read due to context cancellation)")
				it.err = stdio.EOF
				return false
			}
			it.err = readerErr
		}
	}
	return hasNext
}

// Row returns the current row
func (it *Iterator) Row() (map[string]interface{}, error) {
	return it.reader.GetCurrentRow()
}

// Err returns any error encountered during iteration (excluding EOF)
func (it *Iterator) Err() error {
	if it.err != nil && it.err != stdio.EOF {
		return it.err
	}
	return nil
}

// Close closes the iterator
func (it *Iterator) Close() error {
	if it.reader != nil {
		return it.reader.Close()
	}
	return nil
}

// ToSlice reads all rows into a slice (use with caution for large datasets)
func (it *Iterator) ToSlice() ([]map[string]interface{}, error) {
	defer it.Close()

	var results []map[string]interface{}
	for it.Next() {
		row, err := it.Row()
		if err != nil {
			return nil, err
		}
		results = append(results, row)
	}

	if len(results) == 0 {
		logging.Warnf("Iceberg Iterator completed: no rows read")
	}

	if err := it.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// S3FS implements io.FileSystem for S3 access
type S3FS struct {
	client *s3.Client
	region string
	ctx    go_context.Context
}

// NewS3FS creates a new S3 filesystem
func NewS3FS(ctx go_context.Context, accessKeyID, secretAccessKey, sessionToken, region string) (*S3FS, error) {
	if accessKeyID == "" || secretAccessKey == "" || region == "" {
		return nil, fmt.Errorf("access key ID, secret access key, and region are required")
	}

	cfgProvider := credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, sessionToken)
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(cfgProvider),
		config.WithRegion(region),
		config.WithHTTPClient(&http.Client{Transport: icebergTransport()}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg)

	return &S3FS{
		client: client,
		region: region,
		ctx:    ctx,
	}, nil
}

// NewS3FSFromConfig creates an S3 filesystem from an AWS config
func NewS3FSFromConfig(cfg aws.Config) (*S3FS, error) {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Transport: icebergTransport()}
	}
	client := s3.NewFromConfig(cfg)

	// Extract region from config
	region := cfg.Region
	if region == "" {
		region = "us-east-1" // default
	}

	return &S3FS{
		client: client,
		region: region,
	}, nil
}

// Open opens a file from S3
func (s *S3FS) Open(location string) (io.File, error) {
	bucket, key, err := ParseS3URI(location)
	if err != nil {
		return nil, fmt.Errorf("invalid S3 URI: %w", err)
	}

	// Use range-based download for large files
	result, err := s.client.GetObject(s.ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from S3: %w", err)
	}

	// Wrap the response body to satisfy io.File interface
	return &s3File{
		ReadCloser: result.Body,
	}, nil
}

// s3File wraps the S3 response body to satisfy io.File
type s3File struct {
	stdio.ReadCloser
	size int64
}

func (f *s3File) Seek(offset int64, whence int) (int64, error) {
	return -1, fmt.Errorf("seek not supported")
}

func (f *s3File) ReadAt(p []byte, off int64) (n int, err error) {
	return 0, fmt.Errorf("readat not supported")
}

func (f *s3File) Stat() (fs.FileInfo, error) {
	return nil, fmt.Errorf("stat not supported")
}

// GetFileSize returns the size of a file in bytes
func (s *S3FS) GetFileSize(location string) (int64, error) {
	bucket, key, err := ParseS3URI(location)
	if err != nil {
		return 0, fmt.Errorf("invalid S3 URI: %w", err)
	}

	result, err := s.client.HeadObject(s.ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to head object from S3: %w", err)
	}

	if result.ContentLength == nil {
		return 0, fmt.Errorf("content length not available")
	}

	return *result.ContentLength, nil
}

// SetContext sets the context for S3 operations
func (s *S3FS) SetContext(ctx go_context.Context) {
	s.ctx = ctx
}

// FileFormat represents the different file formats supported by Iceberg
type FileFormat string

const (
	// FormatParquet represents Apache Parquet format
	FormatParquet FileFormat = "parquet"
	// FormatORC represents Apache ORC format
	FormatORC FileFormat = "orc"
	// FormatAvro represents Apache Avro format
	FormatAvro FileFormat = "avro"
	// FormatArrow represents Apache Arrow IPC format
	FormatArrow FileFormat = "arrow"
	// FormatCSV represents CSV format (less common)
	FormatCSV FileFormat = "csv"
)

// FormatInfo contains metadata about the file format
type FormatInfo struct {
	Format           FileFormat
	Compression      string
	CompressionLevel int
	Size             int64
	NumRows          int64
	Schema           *arrow.Schema
}

// ArrowReaderOptions configures the Arrow reader for different file formats
type ArrowReaderOptions struct {
	MemoryAllocator memory.Allocator
	CaseSensitive   bool
	ColumnFilter    func(string) bool // nil means include all
	BatchSize       int
	ZeroCopy        bool // Use zero-copy when possible
	DecimalToDouble bool // When true, Decimal128/256 columns yield float64 instead of string
}

// DefaultArrowReaderOptions returns default options for Arrow reading
func DefaultArrowReaderOptions() *ArrowReaderOptions {
	return &ArrowReaderOptions{
		MemoryAllocator: memory.DefaultAllocator,
		CaseSensitive:   true,
		BatchSize:       4096,
		ZeroCopy:        true,
	}
}

// ArrowUtility provides helper functions for Arrow data conversion
type ArrowUtility struct {
	allocator memory.Allocator
	options   *ArrowReaderOptions
}

// NewArrowUtility creates a new Arrow utility
func NewArrowUtility() *ArrowUtility {
	return &ArrowUtility{
		allocator: memory.DefaultAllocator,
		options:   DefaultArrowReaderOptions(),
	}
}

// NewArrowUtilityWithOptions creates a new Arrow utility with custom options
func NewArrowUtilityWithOptions(opts *ArrowReaderOptions) *ArrowUtility {
	if opts == nil {
		opts = DefaultArrowReaderOptions()
	}
	return &ArrowUtility{
		allocator: opts.MemoryAllocator,
		options:   opts,
	}
}

// ArrowSchemaInfo provides detailed information about an Arrow schema
type ArrowSchemaInfo struct {
	NumFields       int
	NumBatches      int
	TotalRows       int64
	TotalBytes      int64
	Compressed      bool
	ColumnTypes     map[string]string
	NestedTypes     []string
	ListColumns     []string
	StructColumns   []string
	DecimalFields   []string // Fields with decimal data type
	TimestampFields []string // Fields with timestamp data type
}

// AnalyzeArrowSchema analyzes an Arrow schema and returns detailed information
func (au *ArrowUtility) AnalyzeArrowSchema(schema *arrow.Schema, numBatches int, numRows int64, totalBytes int64) *ArrowSchemaInfo {
	info := &ArrowSchemaInfo{
		NumFields:   int(schema.NumFields()),
		NumBatches:  numBatches,
		TotalRows:   numRows,
		TotalBytes:  totalBytes,
		ColumnTypes: make(map[string]string),
	}

	// Analyze each field
	for i := 0; i < int(schema.NumFields()); i++ {
		field := schema.Field(i)
		name := field.Name
		dataType := field.Type

		// Record column type
		info.ColumnTypes[name] = dataType.String()

		// Check for nested types
		dataTypeID := dataType.ID()
		switch dataTypeID {
		case arrow.LIST:
			info.ListColumns = append(info.ListColumns, name)
			info.NestedTypes = append(info.NestedTypes, "LIST")
		case arrow.STRUCT:
			info.StructColumns = append(info.StructColumns, name)
			info.NestedTypes = append(info.NestedTypes, "STRUCT")
		case arrow.DECIMAL:
			info.DecimalFields = append(info.DecimalFields, name)
		case arrow.TIMESTAMP:
			info.TimestampFields = append(info.TimestampFields, name)
		}

		// Check nested types recursively
		au.analyzeNestedTypes(dataType, name, info)
	}

	return info
}

// analyzeNestedTypes recursively analyzes nested data types
func (au *ArrowUtility) analyzeNestedTypes(dataType arrow.DataType, path string, info *ArrowSchemaInfo) {
	switch dt := dataType.(type) {
	case *arrow.ListType:
		if elemType := dt.Elem(); elemType != nil {
			au.analyzeNestedTypes(elemType, path+"[]", info)
		}
	case *arrow.StructType:
		for i := 0; i < dt.NumFields(); i++ {
			field := dt.Field(i)
			au.analyzeNestedTypes(field.Type, path+"."+field.Name, info)
		}
	case *arrow.MapType:
		// Handle map types
		if keyType := dt.KeyType(); keyType != nil {
			au.analyzeNestedTypes(keyType, path+".key", info)
		}
		if itemType := dt.ItemType(); itemType != nil {
			au.analyzeNestedTypes(itemType, path+".value", info)
		}
	}
}

// RecordBatchStats provides statistics about an Arrow RecordBatch
type RecordBatchStats struct {
	NumRows     int64
	NumColumns  int
	Bytes       int64
	Compression string
	NullValues  map[string]int64
	ValueCounts map[string]int64
}

// GetRecordBatchStats returns statistics for a single RecordBatch
func (au *ArrowUtility) GetRecordBatchStats(batch arrow.RecordBatch) *RecordBatchStats {
	stats := &RecordBatchStats{
		NumRows:     batch.NumRows(),
		NumColumns:  int(batch.Schema().NumFields()),
		Bytes:       0,
		NullValues:  make(map[string]int64),
		ValueCounts: make(map[string]int64),
	}

	// Iterate over columns to gather statistics
	for i := 0; i < stats.NumColumns; i++ {
		col := batch.Column(i)
		fieldName := batch.Schema().Field(i).Name

		// Count null values
		nullCount := int64(0)
		for j := int64(0); j < batch.NumRows(); j++ {
			if col.IsNull(int(j)) {
				nullCount++
			}
		}

		stats.NullValues[fieldName] = nullCount
		stats.ValueCounts[fieldName] = batch.NumRows() - nullCount

		// Estimate bytes (rough approximation)
		stats.Bytes += int64(col.Len())
	}

	return stats
}

// ArrowToJson converts an Arrow RecordBatch to JSON format
func (au *ArrowUtility) ArrowToJson(batch arrow.RecordBatch) ([]string, error) {
	if batch == nil {
		return nil, fmt.Errorf("batch is nil")
	}

	schema := batch.Schema()
	result := make([]string, 0, int(batch.NumRows()))

	for i := int64(0); i < batch.NumRows(); i++ {
		row := make(map[string]interface{})

		for j := 0; j < int(schema.NumFields()); j++ {
			fieldName := schema.Field(j).Name

			// Apply column filter if configured
			if au.options != nil && au.options.ColumnFilter != nil {
				if !au.options.ColumnFilter(fieldName) {
					continue // Skip this column
				}
			}

			col := batch.Column(j)

			if !col.IsNull(int(i)) {
				value := au.getArrowValue(col, int(i))
				row[fieldName] = value
			} else {
				row[fieldName] = nil
			}
		}

		jsonBytes, err := json.Marshal(row)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal row %d: %w", i, err)
		}

		result = append(result, string(jsonBytes))
	}

	return result, nil
}

// getArrowValue extracts a value from an Arrow array at a specific position
func (au *ArrowUtility) getArrowValue(arr arrow.Array, pos int) interface{} {
	switch a := arr.(type) {
	case *array.Boolean:
		return a.Value(pos)
	case *array.Int8:
		return a.Value(pos)
	case *array.Int16:
		return a.Value(pos)
	case *array.Int32:
		return a.Value(pos)
	case *array.Int64:
		return a.Value(pos)
	case *array.Uint8:
		return a.Value(pos)
	case *array.Uint16:
		return a.Value(pos)
	case *array.Uint32:
		return a.Value(pos)
	case *array.Uint64:
		return a.Value(pos)
	case *array.Float32:
		return a.Value(pos)
	case *array.Float64:
		return a.Value(pos)
	case *array.String:
		return a.Value(pos)
	case *array.Binary:
		return a.Value(pos)
	case *array.LargeString:
		return a.Value(pos)
	case *array.LargeBinary:
		return a.Value(pos)
	case *array.Decimal128:
		if au.options != nil && au.options.DecimalToDouble {
			t := a.DataType().(*arrow.Decimal128Type)
			return a.Value(pos).ToFloat64(t.Scale)
		}
		return a.ValueStr(pos)
	case *array.Decimal256:
		if au.options != nil && au.options.DecimalToDouble {
			t := a.DataType().(*arrow.Decimal256Type)
			return a.Value(pos).ToFloat64(t.Scale)
		}
		return a.ValueStr(pos)
	case *array.Timestamp:
		return int64(a.Value(pos))
	case *array.Date32:
		return int32(a.Value(pos))
	case *array.Date64:
		return int64(a.Value(pos))
	case *array.Time32:
		return int32(a.Value(pos))
	case *array.Time64:
		return int64(a.Value(pos))
	case *array.Struct:
		return au.getStructValue(a, pos)
	case *array.List:
		return au.getListValue(a, pos)
	case *array.LargeList:
		return au.getLargeListValue(a, pos)
	case *array.FixedSizeList:
		return au.getFixedSizeListValue(a, pos)
	case *array.Map:
		return au.getMapValue(a, pos)
	case *array.Null:
		return nil
	default:
		return fmt.Sprintf("%v", a.GetOneForMarshal(pos))
	}
}

// getStructValue extracts a struct value
func (au *ArrowUtility) getStructValue(arr *array.Struct, pos int) map[string]interface{} {
	result := make(map[string]interface{})
	dataType := arr.DataType()

	if structType, ok := dataType.(*arrow.StructType); ok {
		for i := 0; i < arr.NumField(); i++ {
			fieldArr := arr.Field(i)
			fieldName := structType.Field(i).Name

			if !fieldArr.IsNull(pos) {
				result[fieldName] = au.getArrowValue(fieldArr, pos)
			} else {
				result[fieldName] = nil
			}
		}
	}

	return result
}

// getListValue extracts a list value
func (au *ArrowUtility) getListValue(arr *array.List, pos int) []interface{} {
	listValues := arr.ListValues()
	if listValues == nil {
		return nil
	}

	start, end := arr.ValueOffsets(pos)
	listLen := int(end - start)
	result := make([]interface{}, listLen)

	for i := 0; i < listLen; i++ {
		if listValues.IsNull(int(start) + i) {
			result[i] = nil
		} else {
			result[i] = au.getArrowValue(listValues, int(start)+i)
		}
	}

	return result
}

// getLargeListValue extracts a large list value
func (au *ArrowUtility) getLargeListValue(arr *array.LargeList, pos int) []interface{} {
	listValues := arr.ListValues()
	if listValues == nil {
		return nil
	}

	start, end := arr.ValueOffsets(pos)
	listLen := int(end - start)
	result := make([]interface{}, listLen)

	for i := 0; i < listLen; i++ {
		if listValues.IsNull(int(start) + i) {
			result[i] = nil
		} else {
			result[i] = au.getArrowValue(listValues, int(start)+i)
		}
	}

	return result
}

// getFixedSizeListValue extracts a fixed-size list value
func (au *ArrowUtility) getFixedSizeListValue(arr *array.FixedSizeList, pos int) []interface{} {
	listLen := arr.Len()
	result := make([]interface{}, listLen)

	for i := 0; i < listLen; i++ {
		itemArr := arr.ListValues()
		if itemArr.IsNull(pos*listLen + i) {
			result[i] = nil
		} else {
			result[i] = au.getArrowValue(itemArr, pos*listLen+i)
		}
	}

	return result
}

// getMapValue extracts a map value
func (au *ArrowUtility) getMapValue(arr *array.Map, pos int) map[string]interface{} {
	result := make(map[string]interface{})

	keyItems := arr.Keys()
	valueItems := arr.Items()

	keysStart, keysEnd := arr.ValueOffsets(pos)
	numEntries := int(keysEnd - keysStart)

	for i := 0; i < numEntries; i++ {
		keyIdx := int(keysStart) + i

		if keyItems.IsNull(keyIdx) {
			continue
		}

		keyValue := au.getArrowValue(keyItems, keyIdx)
		keyStr, ok := keyValue.(string)
		if !ok {
			// Convert non-string keys to string
			keyStr = fmt.Sprintf("%v", keyValue)
		}

		if valueItems.IsNull(keyIdx) {
			result[keyStr] = nil
		} else {
			result[keyStr] = au.getArrowValue(valueItems, keyIdx)
		}
	}

	return result
}

// ArrowToCSV converts Arrow RecordBatches to CSV format
func (au *ArrowUtility) ArrowToCSV(schema *arrow.Schema, batches []arrow.RecordBatch) (string, error) {
	if len(batches) == 0 {
		return "", nil
	}

	// Create CSV writer
	builder := strings.Builder{}
	allocator := memory.NewGoAllocator()
	csvWriter := csv.NewWriter(&builder, schema, csv.WithAllocator(allocator))

	// Write all batches
	for _, batch := range batches {
		if err := csvWriter.Write(batch); err != nil {
			return "", fmt.Errorf("failed to write batch to CSV: %w", err)
		}
	}

	// Flush the writer
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return "", fmt.Errorf("CSV writer error: %w", err)
	}

	return builder.String(), nil
}

// ArrowToArrowIPC converts Arrow RecordBatches to Arrow IPC format
func (au *ArrowUtility) ArrowToArrowIPC(schema *arrow.Schema, batches []arrow.RecordBatch) ([]byte, error) {
	if len(batches) == 0 {
		return nil, nil
	}

	// Create a memory allocator
	allocator := memory.NewGoAllocator()

	// Create IPC Writer
	sink := NewBytesSink()
	ipcWriter := ipc.NewWriter(sink, ipc.WithAllocator(allocator))

	// Write all batches
	for _, batch := range batches {
		if err := ipcWriter.Write(batch); err != nil {
			return nil, fmt.Errorf("failed to write batch to IPC: %w", err)
		}
	}

	// Close the writer
	if err := ipcWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close IPC writer: %w", err)
	}

	return sink.Bytes(), nil
}

// BytesSink implements the ipc.FileWriter interface for writing to bytes
type BytesSink struct {
	bytes []byte
}

// NewBytesSink creates a new BytesSink
func NewBytesSink() *BytesSink {
	return &BytesSink{
		bytes: make([]byte, 0),
	}
}

// Write implements the io.Writer interface
func (bs *BytesSink) Write(p []byte) (n int, err error) {
	bs.bytes = append(bs.bytes, p...)
	return len(p), nil
}

// Close closes the sink
func (bs *BytesSink) Close() error {
	return nil
}

// Bytes returns the accumulated bytes
func (bs *BytesSink) Bytes() []byte {
	return bs.bytes
}

// ArrowFromArrowIPC reads Arrow IPC format back to RecordBatches
func (au *ArrowUtility) ArrowFromArrowIPC(data []byte) (*arrow.Schema, []arrow.RecordBatch, error) {
	allocator := memory.NewGoAllocator()

	// Create a reader from bytes
	reader := bytes.NewReader(data)

	// Read the IPC stream
	ipcReader, err := ipc.NewReader(reader, ipc.WithAllocator(allocator))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create IPC reader: %w", err)
	}
	defer ipcReader.Release()

	// Read all record batches
	batches := make([]arrow.RecordBatch, 0)
	for ipcReader.Next() {
		record := ipcReader.Record()
		record.Retain()
		batches = append(batches, record)
	}

	if err := ipcReader.Err(); err != nil {
		return nil, nil, fmt.Errorf("IPC reader error: %w", err)
	}

	schema := ipcReader.Schema()
	return schema, batches, nil
}

// FormatCompatible checks if a file format is compatible with Arrow
func (au *ArrowUtility) FormatCompatible(format FileFormat) bool {
	switch format {
	case FormatParquet, FormatORC, FormatAvro, FormatArrow, FormatCSV:
		return true
	default:
		return false
	}
}

// GetFormatCompression returns the typical compression for a format
func (au *ArrowUtility) GetFormatCompression(format FileFormat) string {
	switch format {
	case FormatParquet:
		return "snappy,zstd,gzip"
	case FormatORC:
		return "zstd,snappy,lzo"
	case FormatAvro:
		return "snappy,deflate,bzip2"
	case FormatArrow:
		return "lz4,zstd,none"
	default:
		return "unknown"
	}
}
