# AGENTS.md

This file provides guidance to AGENTS when working with code in this repository.

## What This Is

This is the **Couchbase N1QL Query Engine** — a multi-threaded SQL-like query server for Couchbase Server. It parses, plans, and executes N1QL (SQL for JSON) queries against Couchbase data.

## Build Commands

```bash
# Standard build
cd /Users/sitaram.vemulapalli/totoro/query/goproj/src/github.com/couchbase/query
./build.sh -tags enterprise

# Enterprise with dependency updates
./build.sh -u -tags enterprise

# Standalone build (links against installed Couchbase Server libraries)
./build.sh -s -tags enterprise

# Skip go fmt check
./build.sh -nofmt
```

The build script enforces `go fmt` — code that isn't formatted will fail the build. The main binary is `server/cbq-engine/`.

## Running Tests

```bash
# Unit tests for a single package
go test ./value/...
go test ./expression/...
go test ./execution/...

# All unit tests
go test ./...

# Clear test cache
go clean -testcache

# Integration tests (requires Couchbase Server with Data + Index services)
cd test/
./bucket_create.sh          # One-time setup
./runAlltest.sh             # Run all integration tests
./runAlltest.sh -v          # Verbose
./runSingleTest.sh <pkg>    # Single package integration test
./bucket_delete.sh          # Cleanup
```

Integration tests need Couchbase Server running at `127.0.0.1:8091` with credentials `Administrator:password`. Edit `test/multistore/json.go` to change these.

For CGO-dependent tests (enterprise features):
```bash
export CGO_CFLAGS="-I$GOPATH/src/github.com/couchbase/eventing-ee/evaluator/worker/include -I$GOPATH/src/github.com/couchbase/sigar/include"
export CGO_LDFLAGS="-L$GOPATH/lib"
export DYLD_LIBRARY_PATH=$GOPATH/lib:${DYLD_LIBRARY_PATH}  # macOS
```

## Query Processing Pipeline

The engine processes queries in three stages:

```
Text Query
    → [Parse]   algebra/parser/n1ql/n1ql.y → Algebra AST
    → [Prepare] planner/ + plannerbase/    → Executable Plan (with index selection)
    → [Execute] execution/                 → Results streamed via Go channels
```

## Key Packages

| Package | Purpose |
|---------|---------|
| `value/` | JSON value types (Boolean, Number, String, Null, Array, Object, Missing, Annotated). Primitive JSON values are Go primitives — no GC overhead. |
| `expression/` | All scalar expression evaluation: arithmetic, comparison, CASE, ANY/EVERY/ARRAY/FIRST, functions, identifiers, navigation. Used by both query and indexing. |
| `algebra/` | Full N1QL AST definitions — all statements (SELECT, INSERT, UPDATE, DELETE, UPSERT, MERGE, EXPLAIN, CREATE/DROP INDEX), aggregates, joins, subqueries. |
| `algebra/parser/n1ql/` | goyacc grammar (`n1ql.y`). Regenerate parser with `goyacc -l -p yy -o <output> n1ql.y` when grammar changes. |
| `plan/` | Executable plan operator structs (PrimaryScan, IndexScan, IntersectScan, Fetch, Join, Nest, Unnest, Filter, Group phases, Project, SendInsert/Update/Delete, Parallel, Sequence, etc.). Plans are built from algebra via visitor pattern. |
| `execution/` | Running instances of plan operators. Mirrors `plan/`. Go channels are used extensively for streaming and stop signaling. |
| `planner/` + `plannerbase/` | Converts algebra to plans; selects indexes; cost-based optimization. |
| `datastore/` | Interface to storage. Implementations: `datastore/couchbase/` (production), `datastore/file/` (tests), `datastore/mock/` (tests), `datastore/system/` (system catalog). All `Keyspace` mutation/scan methods (`Fetch`, `Insert`, `Update`, `Delete`, `Count`, `Stats`, `ExternalScan`) and `Datastore` transaction methods take a `QueryContext`. `QueryContext` carries credentials, transaction state, durability, RU/WU accounting, error reporting, request deadline, and logging. Use `datastore.NULL_QUERY_CONTEXT` or `datastore.MAJORITY_QUERY_CONTEXT` when no real request context is available. |
| `datastore/couchbase/iceberg/` | Apache Iceberg external collections via AWS Glue Catalog; supports Parquet/Avro/ORC/Arrow/CSV with filter pushdown. |
| `server/` | HTTP server and request handling. Entry point: `server/cbq-engine/main.go`. REST endpoints in `server/http/`. |
| `errors/` | All user-visible error codes and messages (SQL-style codes). All errors must come through this package. |
| `functions/` | User-defined function (UDF) support. |
| `prepareds/` | Prepared statement caching. |
| `semantics/` | Type analysis and semantic validation. |
| `rewrite/` | Query rewriting and optimization rules applied before planning. |
| `transactions/` | ACID transaction semantics. |
| `tenant/` | Multi-tenancy support. |

## Data Parallelism Model

The engine achieves data-parallelism via `plan.Parallel` operators. SELECT pipeline: Scan → **Parallelize** → Fetch/Join/Let/Where → GroupBy Initial/Intermediate → **Serialize** → GroupBy Final → **Parallelize** → Having → **Serialize** → OrderBy → **Parallelize** → Project → **Serialize** → Distinct/Offset/Limit.

DML statements have similar patterns with serialization before Limit and parallel mutation operators.

## Enterprise vs Community

Build tag `-tags enterprise` enables: eventing-ee (JavaScript UDFs), vector search, jemalloc allocator, and other enterprise-only features in `*_ee.go` files throughout the codebase.

## Parser Regeneration

If you modify `algebra/parser/n1ql/n1ql.y`:
```bash
goyacc -l -p yy -o algebra/parser/n1ql/n1ql.go algebra/parser/n1ql/n1ql.y
```

## Module Structure

`go.mod` uses local `replace` directives for all Couchbase internal dependencies (`cbauth`, `go-couchbase`, `indexing`, `query-ee`, `eventing-ee`, `n1fty`, `regulator`, etc.). These must be checked out as sibling repos under the same `GOPATH`.
