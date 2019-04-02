package advise

import (
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/test/gsi"
	"testing"
)

func runStmt(mockServer *gsi.MockServer, q string) ([]interface{}, []errors.Error, errors.Error) {
	return gsi.RunStmt(mockServer, q)
}

func runMatch(filename string, prepared, explain bool, qc *gsi.MockServer, t *testing.T) {
	gsi.RunMatch(filename, prepared, explain, qc, t)
}

func start_cs() *gsi.MockServer {
	return gsi.Start_cs(true)
}
