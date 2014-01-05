# ACID Requirements for N1QL

* Status: DRAFT
* Latest: [n1ql-acid](https://github.com/couchbaselabs/query/blob/master/docs/n1ql-acid.md)
* Modified: 2014-01-05

## Introduction

This document specifies the ACID requirements for N1QL. ACID refers to
Atomicity, Consistency, Isolation, and Durability. The purpose of
these requirements is to provide N1QL users and clients with a
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

1. Write statements with bounded cardinality (INSERT / UPDATE / DELETE
   / MERGE with LIMIT or KEYS specified)

1. Transactions (START TRANSACTION ... COMMIT / ROLLBACK)

1. Versioned reads within transactions (SELECT ... LIMIT ... FOR
   UPDATE / FOR SHARE)

## Requirements

Again, the purpose of these requirements is to provide a well-defined
and balanced programming model.

### Eventual atomicity

* **Atomic transactions** - By definition, transactions must be atomic.

* **Atomic statements** - Write statements must be atomic in order to
  avoid manual rollback. Furthermore, some write statements are not
  idempotent, e.g. *UPDATE b SET x = x + 1.* Completing or rolling
  back such a statement would be difficult without atomicity.

Atomicity is defined to be eventual, because the effects of a
statement or transaction may be visible at one node but not yet
visible at another.

### Database (not CAP) consistency

N1QL does not provide constraints for enforcing data correctness;
instead, N1QL provides an isolation model that allows users to
implement data correctness.

### Stable isolation

Isolation defines which data, versions, and mutations are visible to a
statement or transaction.

The N1QL isolation model (called *stable isolation*) is at the core of
these requirements. It has the following properties:

* **Statement-level** - It is statement-level, and not
  transaction-level.

* **Similar to snapshot isolation** - It is similar to, but not
  identical to, statement-level [snapshot
  isolation](http://en.wikipedia.org/wiki/Snapshot_isolation) (but it
  is not based on timestamps).

* **Stability vectors** - The statement-level "snapshot" is defined by
  obtaining a *stability vector*. The stability vector is a version
  vector containing a sequence number for every data (key-value)
  partition in the bucket.

* **Not point-in-time** - The stability vector does not imply any
  point-in-time correlation among the sequence numbers in the
  vector. It only defines a stable state against which queries can be
  executed.

* **At stable committed** - Like snapshot isolation, *stable
  isolation* only reads committed data. However, it reads the version
  of committed data specified exactly by the stability vector. If a
  new version of the data is committed after the stability vector is
  obtained but before the statement is executed, that new version is
  not read.

* **Stable scans** - Index scans are required to use stability
  vectors. A range scan or full scan may span several index
  partitions. In that case, all the index partitions must use the same
  stability vector, and the indexer must combine the results into a
  unified scan.  Stable scans ensure that an index scan reflects only
  one version of each document.

* **Unique fetches** - A statement fetches documents that are
  enumerated by index scans. The document versions read must be either
  committed or written by the current transaction.

    * A statement must fetch each document at most once, to avoid
      reading more than one version.

    * A statement must evaluate any WHERE clause predicates against
      each document, to account for any data-index inconsistency or
      lag.

    * A statement may also need to ensure that the fetched document
      satisfies a mimimum vector.

* **Transaction overlay** - Both scans and fetches are overlaid with
  writes by the current transaction.

    * Index scans will reflect both the stability vector and the
      current transaction's writes.

    * Document fetches will reflect both committed reads and the
      current transaction's writes.

* **Result vectors** - Transactions are able to produce *result
  vectors*, which are version vectors that would reflect all the
  (eventual) effects of the transaction.

  Read-your-own-writes can be implemented by using the result vector
  from one statement as the minimum vector for the next statement.

### Eventual durability

Completed statements and transactions must be eventually durable.
Their effects are not required to be immediately visible to every new
query; however, they must eventually be visible to every new query.

Furthermore, the effects of completed statements and transactions may
be visible at one node but not yet at another; however, they must
eventually be visible at every affected node.

If a query or client needs all the effects of a previous statement or
transaction to be visible, the query or client can provide a minimum
stability vector.

### Additional requirements for writes

N1QL write statements have additional requirements.

* For every write statement, all reads must (appear to) happen before
  all writes. That is, all the inputs to a statement must be read
  before being modified by the statement. This is satisfied by stable
  scans.

* All write statements are transactional.

    * If a write statement is not within an explicit transaction, it
      will behave like a single-statement transaction.

    * If a write statement is within an explicit transaction, it will
      behave atomically with respect to the rest of the transaction.

## About this Document

### Document History

* 2013-10-29 - Initial version
* 2013-11-21 - Updates
* 2014-01-05 - Updated requirements based on discussions

### Open Issues

This meta-section records open issues in this document, and will
eventually disappear.
