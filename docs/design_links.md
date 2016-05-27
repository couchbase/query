# External Design Links for Query

* Status: DRAFT
* Latest: [design_links](https://github.com/couchbase/query/blob/master/docs/design_links.md)
* Modified: 2015-01-23

## Introduction

This document provides links to external design documents; links are provided under relevant heading.

## Core Query Engine

* [N1QL Query API](http://goo.gl/ezpmVx) - design document for query REST API
* [Prepared Statements](http://goo.gl/T8l7nd) - design document for prepared statements
* [encoded_plan](https://goo.gl/aeGjlj)- design document for REST API encoded_plan
* [N1QL Error Handling](http://goo.gl/IzZA0y) - design document for error handling
* [N1QL - JDBC Type Mapping](http://goo.gl/akKrBe) - design document for mapping N1QL types to JDBC

## Clustering Management

* [Overview of Clustering Management Survey](http://goo.gl/gid7LX) - identifies the high level concerns in clustered systems
* [Cluster Management Survey](http://goo.gl/gid7LX) - a survey of best practices and common policies in clustered distributed systems
* [Cluster Management Topics](http://goo.gl/RFa2Yb) - a design document for Clustering Management in N1QL
* [N1QL Cluster Management API](http://goo.gl/yKZ6v5) - design document for the Clustering API in N1QL
* [N1QL Stats for Sherlock](http://goo.gl/ZlVeag) - spec of N1QL stats for Sherlock release

## Shell

* [Command Line Shell Survey](http://goo.gl/ZStXN7) - a survey of open source command line shells; identifies features and functionality
* [Command Line Shell Architecture Summary](http://goo.gl/SFwRWq) - describes workings of open source command line shells; identifies best practices in design and functionality
* [Command Line Shell Design](https://goo.gl/2G1sa8) - design guide for N1QL command line shell

## N1QL Design Documents

* [Push Query Offset, Limit to the Indexer](https://docs.google.com/a/couchbase.com/document/d/1pCvrLGPJwfczYX_yPxV6aVp0QL1RnBW7VW3jOsgnJxM/edit?usp=sharing) - describes when to push the query offset, limit to indexer
* [Optimize the query with order, offdset, limit to exploit index ordering](https://docs.google.com/a/couchbase.com/document/d/1wfRY7bVshnZ1woexoaLUnDU9y2aSitHPgdYJXesAgUg/edit?usp=sharing) - describes when query uses index order
* [Optimize the query with count(expr) to exploit index scan count](https://docs.google.com/a/couchbase.com/document/d/1FXPRr-lCshSpo97kIMShVRi5IAaO7fnd1BihVm5aO24/edit?usp=sharing) - describes when query uses index scan count

## N1QL Presentations

* [Performance Improvements](https://docs.google.com/a/couchbase.com/presentation/d/14K74FEJlD3gY_0ViKDuMBLYEYwQa5GMg9Sb7ABZ_eO4/edit?usp=sharing) - describes order, limit, count performance improvements
