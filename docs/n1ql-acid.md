# ACID Requirements for N1QL

* Status: DRAFT
* Latest: [n1ql-acid](https://github.com/couchbaselabs/query/blob/master/docs/n1ql-acid.md)
* Modified: 2013-11-21

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

## ACID requirements

### Reads

The query engine performs 2 kinds of reads: index scans and key-value
fetches. Every scan or fetch should abort or fail rather than produce
results that don't satisfy the semantics below.

#### Index scans

1. Stateless scan. Scan committed entries, with no additional
   requirements. This should be the fastest scan possible. It may or
   may not be used, but it should be provided, if only for baseline
   performance comparison.

1. Rolling scan. Scan committed entries based on a start vector. The
   start vector is acquired by asking each index node for a start
   value. Each index node then performs its scan to reflect that start
   value or later.
   * A rolling scan is required to prevent reading a new value
     followed by an older value for the same entry. This can happen,
     for example, if two separate index replicas are scanned by
     subqueries within a query.
   * Rolling scans (and fetches) can also be used to implement
     read-your-own-writes.

1. Stable scan. Scan committed entries based on a stability
   vector. The stability vector is acquired by asking each index node
   for a stability value. Each index node then performs its scan to
   reflect that stability value exactly. This may require MVCC and
   some history.
   * A stable scan is required for the following query: find all
     duplicates. Every value returned must have been a duplicate at
     some point. So if a value is not a duplicate, but is scanned,
     deleted, reinserted, and scanned again, it should **not** be
     returned.
   * If stable scans are performant, every non-transactional query
     might use them. This would produce more meaningful results.

1. Transactional scan. Stable scan overlaid with entries modified or
held (SELECT FOR UPDATE) by the current transaction.

#### Key-value fetches

1. Stateless fetch.

1. Transactional fetch. Stateless fetch overlaid with entries modified
   or held (SELECT FOR UPDATE) by the current transaction.

1. Question: Is stable fetch required in order to make stable scan
   useful?

### Writes

All writes by the query engine (DML statements) require strong ACID
semantics.

1. For a given statement, all reads must (appear to) happen before all
   writes. That is, all the inputs to a statement must be read without
   being modified by the statement. This can be satisfied using stable
   scans.

1. All DML statements are transactional.
   * If a DML statement is not within an explicit transaction, it will
     behave like a single-statement transaction.
   * If a DML statement is within an explicit transaction, it will
     behave atomically with respect to the rest of the
     transaction. This might require two-level transaction support by
     the transaction manager.

1. Transactional scans (see above).

1. Distributed atomicity
   * Immediate atomicity within each index node (and data node?)
   * Eventual atomicity across nodes

1. Index-data consistency for data modified by the current statement
   (or held by the current transaction)

1. Distributed isolation, commit, and rollback
   * Immediate commit and rollback with each index node (and data node?)
   * Eventual commit and rollback across nodes (partial isolation)

1. Distributed durability
   * N1QL does not define durability, but leaves it to the underlying
     database (e.g. persistence, replica count, etc.)
   * Eventual durability: each node can achieve durability at a
     different point in time, but the transaction state (committed or
     rolled back) must eventually be reflected in "durable" form
   * May require post-processing by garbage collectors and transaction
     background post-processors

## About this Document

### Document History

* 2013-10-29 - Initial version
* 2013-11-21 - Updates

### Open Issues

This meta-section records open issues in this document, and will
eventually disappear.
