# Schema for N1QL System Catalog

* Status: OPERATIVE
* Latest: [n1ql-system-catalog](https://github.com/couchbaselabs/query/blob/master/docs/n1ql-system-catalog.md)
* Modified: 2013-08-24

## Summary

This document presents a schema for a N1QL system catalog. This
catalog can be implemented concretely as stored data, or virtually via
code. This should be transparent to queries.

## Containment hierarchy

* **Site:** A database deployment, e.g. a server cluster, cloud
  service, or mobile installation. Analogous to a RDBMS instance.

* **Pool:** A namespace; a unit of authorization, resource allocation,
  and tenancy. Analogous to a RDBMS database or schema.

* **Bucket:** A set of documents, which are allowed to vary in
  structure; a unit of authorization and resource
  allocation. Analogous to a RDBMS table.

* **Index:** An index on a bucket. Analogous to a RDBMS index. These
  will include tree, view, fulltext, hash, and other indexes.

## Structure

The system catalog is a pool. The pool is called **sys_catalog.**

There is a bucket for each type of artifact. The bucket names are
plural (which is not recommended database practice) in order to avoid
coinciding with N1QL keywords.

The attributes below are the initial minimal set. Others are likely to
be added over time (maybe counts, maybe index uniqueness, etc.).

Other artifacts will also be added (e.g. database views, stored
procedures, built-in functions). Built-in functions will be added
shortly.

## Sites

The bucket is called **sites.** It typically contains a single
document.

* **id:** string
* **url:** string
* **name:** optional string

## Pools

The bucket is called **pools.**

* **id:** string
* **site_id:** string
* **name:** string

## Buckets

The bucket is called **buckets.**

* **id:** string
* **pool_id:** string
* **name:** string

## Indexes

The bucket is called **indexes.**

* **id:** string
* **bucket_id:** string
* **name:** string
* **index_key:** array of string
* **index_type:** string

## About this Document

### Document History

* 2013-08-22 - Initial version
* 2013-08-24 - Names
    * Lowercased names.
    * Changed cb\_catalog to sys\_catalog. 

### Open Issues

This meta-section records open issues in this document, and will
eventually disappear.

1. VIEWs would allow us to provide queryable objects like
   **all\_buckets** vs. **my\_buckets** vs. **all\_readable\_buckets.**

1. VIEWs would also allow us to support the SQL standard
   INFORMATION_SCHEMA and would make us usable by tools that already
   understand SQL standard system catalogs. NoSQL indeed.
