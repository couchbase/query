# N1QL DP4 Syntax Changes

* Status: DRAFT
* Latest: [dp4-syntax-changes](https://github.com/couchbaselabs/query/blob/master/docs/dp4-syntax-changes.md)
* Modified: 2014-10-27

## Introduction

This document specifies the syntax changes between N1QL DP3 and
DP4. It is meant to be useful to the QE team, the documentation team,
and other consumers of N1QL specs.


## List of syntax changes

+ __pool__ renamed to __namespace__

+ __bucket__ renamed to __keyspace__

+ __:pool-name.bucket-name__ changed to __namespace-name:keyspace-name__ (moved the colon to the middle)

+ FROM ... KEYS changing to FROM ... USE [ PRIMARY ] KEYS

+ JOIN / NEST ... KEYS changing to JOIN / NEST ... ON [ PRIMARY ] KEYS


## List of additions

+ UNION [ ALL ]

+ EXCEPT [ ALL ]

+ INTERSECT [ ALL ]

+ SELECT [ DISTINCT ] RAW

+ LET

+ LETTING

+ Subqueries

+ EXISTS

+ a [ NOT ] IN b

+ a [ NOT ] WITHIN b

+ New functions

## About this Document

### Document History

* 2014-10-27 - Initial version.
