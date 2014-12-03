# N1QL DP4 Feature Status

* Status: DRAFT
* Latest: [dp4-feature-status](https://github.com/couchbaselabs/query/blob/master/docs/dp4-feature-status.md)
* Modified: 2014-11-24

## Introduction

This document specifies the features of N1QL DP4 and their current
status for QE.

The document also indicates the status of features with respect to
Couchbase datastore and file-based datastore.

See the N1QL [SELECT
spec](https://github.com/couchbaselabs/query/blob/master/docs/n1ql-select.md),
[DDL
spec](https://github.com/couchbaselabs/query/blob/master/docs/n1ql-ddl.md),
and [DML
spec](https://github.com/couchbaselabs/query/blob/master/docs/n1ql-dml.md).

## DP4 Features

+ REST API

+ Expressions

    + Literals

    + Identifiers

    + Nested: field navigation, array indexing, array slicing

    + Case: simple case, searched case

    + Logical: AND, OR, NOT

    + Comparison: operators, BETWEEN, LIKE, IS [ VALUED, NULL, MISSING ]

    + Arithmetic: operators

    + Concatetion

    + Subqueries

    + EXISTS / IN / WITHIN

    + ANY / SOME / EVERY

    + ARRAY / FIRST / OBJECT

    + Construction: literal objects and arrays

    + Functions

        + Date functions

        + String functions

        + Number functions

        + Array functions

        + Object functions

        + JSON functions

        + Comparison functions

        + Conditional functions for unknowns

        + Conditional functions for numbers

        + Meta functions

        + Type checking functions

        + Type conversion functions

        + Aggregate functions

+ SELECT

    + DISTINCT / ALL

    + RAW

    + JOIN / NEST / UNNEST

    + UNION / INTERSECT / EXCEPT

    + LET / LETTING

    + Subqueries

    + GROUP BY / HAVING / LETTING

    + ORDER BY / LIMIT / OFFSET

+ EXPLAIN

+ PREPARE

+ CREATE INDEX

+ CREATE PRIMARY INDEX

+ DROP INDEX

+ DROP PRIMARY INDEX

+ ALTER INDEX

+ INSERT

+ UPSERT

+ DELETE

+ UPDATE

+ MERGE

## Feature Status

### REST API

The REST API is defined at [Query REST
API](https://docs.google.com/document/d/1Uyv4t06DNGq7TxJjGI_T_MbbEYf8Er-imC7yzTY0uZw/edit#heading=h.lfqenz86v2rl). It
is 80% implemented, and should be fully implemented and testable by
11/20/2014.

### Expressions

Expressions are implemented and testable, with the following
exceptions:

+ OBJECT is not yet implemented.

+ The following operators do not yet support the `name-var :` syntax
for ranging over attribute names.

    + WITHIN

    + ANY / SOME / EVERY

    + ARRAY / FIRST / OBJECT

### SELECT

SELECT is implemented and testable, with the following exceptions:

+ Index selection is not yet implemented, so a primary index is always
used.

+ LETTING without GROUP BY is not yet implemented.

### EXPLAIN

EXPLAIN is implemented, but the output format is still being
fine-tuned. It is ready for manual testing, but automated tests may
break if the format changes.

### PREPARE

PREPARE is implemented, but the output format is possibly subject 
to change. It is ready for manual testing, but automated tests may 
break if the format changes. PREPARE returns the query plan and
signature of the given statement.

### CREATE INDEX

CREATE INDEX is implemented for view indexes. Only a subset of
expressions is supported for the index expressions; other expressions
will generate a "not implemented" or "not supported" message.

Secondary indexes will be integrated in the coming weeks.

### CREATE PRIMARY INDEX

CREATE PRIMARY INDEX is implemented for view indexes.

Secondary indexes will be integrated in the coming weeks.

### DROP INDEX

DROP INDEX is implemented for view indexes.

Secondary indexes will be integrated in the coming weeks.

### DROP PRIMARY INDEX

DROP PRIMARY INDEX is implemented for view indexes.

Secondary indexes will be integrated in the coming weeks.

### ALTER INDEX

ALTER INDEX is currently not in scope for DP4 or Sherlock.

### INSERT

The syntax for INSERT has changed, so the implementation is no longer
identical to the spec.

The new syntax will be implemented by 11/28/2014.

### UPSERT

The syntax for UPSERT has changed, so the implementation is no longer
identical to the spec.

The new syntax will be implemented by 11/28/2014.

### DELETE

DELETE is implemented and testable for Couchbase and file-based
datastores.

### UPDATE

UPDATE is implemented and testable for Couchbase and file-based
datastores.

### MERGE

MERGE is implemented and testable for Couchbase and file-based
datastores. The current implementation is known to cause a crash See
[MB-12327](http://www.couchbase.com/issues/browse/MB-12327).

## About this Document

### Document History

* 2014-11-17 - Initial version
* 2014-11-24 - Delivery dates
    * New dates for implementing INSERT and UPSERT
