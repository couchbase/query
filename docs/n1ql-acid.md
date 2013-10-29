# ACID Requirements for N1QL

* Status: DRAFT
* Latest: [n1ql-acid](https://github.com/couchbaselabs/query/blob/master/docs/n1ql-acid.md)
* Modified: 2013-10-29

## Summary

This document specifies the ACID requirements for N1QL. ACID refers to
Atomicity, Consistency, Isolation, and Durability. Where necessary,
this document also indicates ACID properties that are **not** required
by N1QL.

These ACID requirements are intended to inform the design of the
Indexing and Storage subsytems.

## N1QL background

[N1QL](https://github.com/couchbaselabs/query/blob/master/docs/) will
include the following features:

1. Predicate queries (SELECT ... FROM ... WHERE) and aggregates (GROUP
   BY)

1. Joins and subqueries using primary keys

1. DML statements with bounded cardinality (UPDATE / DELETE / MERGE
   ... WHERE with LIMIT or primary keys specified)

1. Transactions with bounded cardinality (BEGIN ... COMMIT / ROLLBACK)

1. Versioned reads within transactions (SELECT ... LIMIT ... FOR UPDATE)
   * These will behave excatly like UPDATEs with no modifications and
     no new sequence numbers

## ACID outline

1. ACID transactions for both data and indexes
   * Atomic success or failure
   * Index-data consistency within transactions
   * Isolated effects until committed
   * Once committed, durable or recoverable as defined for that
     environment (e.g. via persistence or replication)

1. Cursor stability for DML statements

1. Atomic DML statements (across both data and indexes)

1. Detection of index-data inconsistency
   * Committed data and committed index can have transient inconsistencies
   * These must detectable and reportable as errors / warnings /
     query engine post-processing / etc.

## About this Document

### Document History

* 2013-10-29 - Initial version

### Open Issues

This meta-section records open issues in this document, and will
eventually disappear.
