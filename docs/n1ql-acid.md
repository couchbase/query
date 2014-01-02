# ACID Requirements for N1QL

* Status: DRAFT
* Latest: [n1ql-acid](https://github.com/couchbaselabs/query/blob/master/docs/n1ql-acid.md)
* Modified: 2014-01-01

## Introduction

This document specifies the ACID requirements for N1QL. ACID refers to
Atomicity, Consistency, Isolation, and Durability. The purpose of
these requirements is to provide N1QL users and applications with a
well-defined programming model that balances robustness and usability
on the one hand, and performance and scale on the other.

Consistency in this context refers to database consistency, and not to
consistency among replicas (as in CAP theorem). Database consistency
refers to the correctness of data, i.e. that mutations will not
corrupt data or violate any rules defined in the database.

These ACID requirements are intended to inform the design of indexing
and transactions.

## N1QL background

N1QL will include the following features. Please see the [N1QL
specs](https://github.com/couchbaselabs/query/blob/master/docs)
for more details.

1. Predicate queries (SELECT ... FROM ... WHERE) and aggregates (GROUP
   BY)

1. Joins and subqueries using KEYS

1. DML statements with bounded cardinality (INSERT / UPDATE / DELETE /
   MERGE ... WHERE with LIMIT or KEYS specified)

1. Transactions (BEGIN ... COMMIT / ROLLBACK)

1. Versioned reads within transactions (SELECT ... LIMIT ... FOR
   UPDATE / FOR SHARE)

## Requirements summary

As stated above, the purpose of these requirements is to provide a
well-defined and balanced programming model.

### Eventual atomicity

* **Atomic transactions** - Transactions must be atomic in order to
  spare users the pain of implementing rollback logic.

* **Atomic statements** - DML statements must be atomic for the same
  reason as transactions. Furthermore, some mutations are not
  idempotent, e.g. *UPDATE b SET x = x + 1.* Completing or rolling
  back such a mutation would be difficult without atomic statements.

Atomicity is defined to be eventual, because the effects of a
statement or transaction may be visible at one node but not yet
visible at another.

### Database (not CAP) consistency

N1QL does not provide constraints for enforcing data correctness;
instead, N1QL provides an isolation model that allows users to
maintain data correctness.

### Stable isolation

* **Transaction overlay**

* **Stable scans**

* **Unique fetches**

### Eventual durability

Completed statements and transactions must be eventually durable. That
is, their effects are not required to be immediately visible to every
new query; however, they must eventually be visible to every new
query.

Furthermore, the effects of completed statements and transactions may
be visible at one node but not yet at another; however, they must
eventually be visible at every affected node.

If a query needs all the effects of a previous statement or
transaction to be completed, the query can provide a minimum stability
vector.

## ACID requirements

### Scans and fetches

Queries are executed by performing index scans and key-value fetches.

### Reads

The query engine performs two kinds of reads: index scans and
key-value fetches. Every scan or fetch should abort or fail rather
than produce results that do not satisfy the requirements below.

#### Index scans

Index scans include range scans and full scans for a given index. The
query engine may request a scan that covers several index
partitions. In that case, the indexer will combine the results from
the index partitions and provide the query engine with a unified
stream of results.

N1QL requires stable scans.

A stable scan is defined in terms of a version vector containing a
version (e.g. SeqNo) for each key-value data partition (not index
partition).

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

1. For every DML statement, all reads must (appear to) happen before
   all writes. That is, all the inputs to a statement must be read
   without being modified by the statement. This can be satisfied
   using stable scans.

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
* 2014-01-01 - Updated requirements based on discussions

### Open Issues

This meta-section records open issues in this document, and will
eventually disappear.
