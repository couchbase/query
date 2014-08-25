# N1QL Definition

* Status: DRAFT
* Latest: [n1ql-definition](https://github.com/couchbaselabs/query/blob/master/docs/n1ql-definition.md)
* Modified: 2014-08-25

## Begin outline

## 1 Introduction

### 1.1 What is N1QL?

Defnition, goals, motivation, history, name, etc.

### 1.2 Data in Couchbase

Couchbase as a document database

Modeling data with documents

Key-value datastore

CAS

### 1.3 N1QL queries and results

Queries

Results

Navigation

Transformation

### 1.4 N1QL and SQL

Language comparison

Data comparison

### 1.5 Key features of N1QL

SELECT

DML

Indexes

Primary key access

Aggregation

Joins

Nesting

Unnesting

Subqueries

Union, Intersect, and Except

### 1.6 Conventions 

## 2 Quick start

### 2.1 Installation

### 2.2 Running the query tutorial

### 2.3 Running the query server

### 2.4 Running the query shell

## 3 System information

### 3.1 Logical hierarchy

Namespaces (Pools)

Keyspaces (Buckets)

Keys and values

### 3.2 Querying the system catalog

Namespaces

Keyspaces

Indexes

## 4 Language structure

### 4.1 Statements

### 4.2 Expressions

### 4.3 Comments

### 4.4 Reserved words

## 5 Data types

### 5.1 Primitives

Booleans

Numbers

Strings

### 5.2 Arrays and objects

Arrays

Objects

### 5.3 Null and missing

Null

Missing

### 5.4 Collation

## 6 Literals

### 6.1 Booleans

### 6.2 Numbers

### 6.3 Strings

### 6.4 Nulls

## 7 Identifiers

### 7.1 Unescaped identifiers

### 7.2 Escaped identifiers

## 8 Operators

### 8.1 Arithmetic operators

\+ Add

\- Subtract

\* Multiply

/ Divide

% Modulo

\- Negagte

### 8.2 Collection operators

Any

Every

Array

First

Exists

In

Within

### 8.3 Comparison operators

= Equals

\!=, <> Not equals

< Less than

<= Less than or equals

\> Greater than

\>= Greater than or equals

Between

Like

Is Missing

Is Null

Is Valued

### 8.4 Conditional operators

Simple Case

Searched Case

### 8.5 Construction operators

Array construction

Object construction

### 8.6 Logical operators

And

Or

Not

### 8.7 Navigation operators

Field selection

Element selection

Slicing

### 8.8 String operators

|| Concatenation

### 8.9 Operator precedence

## 9 Functions

### 9.1 Aggregate functions

### 9.2 Array functions

### 9.3 Comparison functions

### 9.4 Conditional functions for numbers

### 9.5 Conditional functions for unknowns

### 9.6 Date functions

### 9.7 JSON functions

### 9.8 Meta and value functions

### 9.9 Numeric functions

### 9.10 Object functions

### 9.11 Pattern matching functions

### 9.12 String functions

### 9.13 Type functions

## 10 Subqueries

## 11 Boolean logic

### 11.1 Four-valued logic

### 11.2 Checking for null and missing

## 12 Statements

### 12.1 ALTER INDEX

### 12.2 CREATE INDEX

### 12.3 DELETE

### 12.4 DROP INDEX

### 12.5 EXPLAIN

### 12.6 INSERT

### 12.7 MERGE

### 12.8 SELECT

### 12.9 UPDATE

### 12.10 UPSERT

## 13 Programs

### 13.1 Query server

### 13.2 Query shell

### 13.3 Query tutorial

## 14 REST API

## A1 Release notes

## A2 Upgrading to DP4

## End outline

## About this document

### Document history

* 2014-08-24 - Initial checkin

* 2014-08-25 - Outline
    * Numbered outline
    * Filled in some content topics

### Open issues

This meta-section records open issues in this document, and will
eventually disappear.
