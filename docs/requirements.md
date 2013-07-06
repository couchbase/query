# Query Language Requirements

## Summary

The Query Language is a human readable and writeable language to perform ad-hoc querying of data stored in Couchbase Server.

## Requirements

### Overview

This specification curretly only describes querying the database.  Future versions of this document may also include data modification.

The query language is expected to evolve over time, because of this:

* MUST support future extensibility, this may appear in different forms
    * reserved words in the query string, FROM
    * version numbers in intermeidate forms (Abstract Syntax Tree)


### Querying

Querying takes a query string and produces a result set.  The result set is a JSON array containing the result values.

The query is presumed to target a single bucket within Couchbase Server.  Because of this there is no specification of the bucket name in the query.  This also implies that there are no JOIN operations supported (neither inter or intra bucket).

Queries are composed of multiple optional parts.  These parts are logically evaluated in the following order (implementations may optimize the actual execution, but it should behave the same):

1.  If a filter expression was supplied, apply the filter expresion to each item in data set, and only keep the objects where the result is true
2.  If aggregate expressions are supplied, group all objects in the result set by evaluating the aggregate expressions, the result set becomes these groups
3.  If a having expression was supplied, apply the having expression to each item in the result set, and only keep the objects where the result is true
4.  If order expressions are supplied, order the items in the result set by evaluting these expressions
5.  If a skip value was supplied, discard this number of items in the result set, starting at index 0.
6.  If an limit value was supplied, truncate the result set to this size.
8.  Return the result set.

### Property Paths

A property path is a string that identifies a value within a JSON document.

* MUST support dotted notation refering to sub-elements within a document (address.city referes to the city value within the address value within the top level document)
* SHOULD support a way to refer to outter contexts when evaluated within a limited context (the element match operator described subsequently cretes a limited context)

### Expressions

* MUST support expressions that refer to literal values (boolean, number, string, array, object)
* MUST support expressions that refer to property paths
* MUST support comparisons using the following operators =,!=,<,<=,>,>=
* MUST support arithmetic operations using the following operators +,-,*,/,%
* MUST support combining expressions using boolean operators AND and OR
* MUST support an "element match" operator (described in Appendix 1)
* MUST support ANY and ALL qualifiers for the "element match" operator
* MUST support operator to determine if property exists in document (differentiating between non-existance and null values)
* SHOULD support string matching using * and ? patterns (LIKE clause)

### Filtering

* MUST return all documents when no filter expression is provided
* MUST return a subset of all documents when a filter expressions is supplied (returns only those documents where the filter expression evaluates to true)

### Aggregation

* MUST support optional grouping documents by 1 or more expressions
* MUST support the following aggregate functions returning data about the consituents of the group
    * min,max,avg,sum,count
* SHOULD support a group_concat operator that builds a JSON array of expressions evaluated on the documents within the group

When performing aggregate queries, group information (either group key information, or aggregate functions from the projection) is returned, not the documents comprising the group.

### Having

* MUST return all groups (from Aggregation query) when no having expression is provided
* MUST return a subset of all groups when a having expression is supplied (returns only those groups where the having expression evalutates to true)

### Ordering

* MAY return documents in any order when no order expressions are supplied
* MUST return documents ordered by the provided order expressions when they are supplied
    * order expressions are normal expressions with an optional ASCENDING or DESCENDING qualifier (default is ASCENDING when not specified)
    * when multiple order expresions are provided, each specifies a subsequent level of order (ie, the first order expression is the primary sort, the second is the a secondary sort within equal values of the primary sort, etc)

### Limiting

* MUST return all documents satisfying the rest of the query when no limit value is specified
* MUST support limiting the total number of documents returned with an integer limit value

### Skipping
* MUST not skip any documents in the result set when no skip value is specified
* MUST support skipping documents in the result set with an integer skip value

### Projection

* MUST support returning full objects from the resultset (either full documents or group information during an aggregate query)
* MUST support evaluating a expression (in the context of an object in the result set) the result of evlauting the expression replaces the object in the result set
* SHOULD support an additional function (NOT valid elsewhere in the query) that performs a document lookup within the same bucket by ID.
* SHOULD support a DISTINCT flag.  If the DISTINCT flag is specified, duplicate objects in the result set are removed.
* SHOULD support a function to lookup another document in the same bucket by ID

### Element Match

The element match operator consists of 3 pieces:

* property path which identifies an array within the document
* an expression which is evaluted on each item in the array (this expression is evaluated within the limited context of the object itself not the full document)
* an optional qualifier which specifies whether ANY or ALL items in the array must be true in order for the operator to return true

### Logical Operators

Logical operators are defined to only operate on values of the same type.  If the types differ, evaluation of the operator always returns false.  Comparison of two values of the same types is done as follows

* booleans - false < true
* numeric - normal arithmetic comparison
* strings - ICU collation (may be supplemented in the future)
* arrays - element by element comparison
* objects - member by member comparison (unreliable as order matters)

See Also: http://wiki.apache.org/couchdb/View_collation

### Arithmetic Operators

- Plus 
    - 2 numeric values, normal arithmetic addition
    - 2 strings, string concatenation
    - anything else, null
- Minus
    - 2 numeric values, normal arithmetic subtraction
    - anything else, null
- Multiplication
    - 2 numeric values, normal arithmetic subtraction
    - anything else, null
- Division
    - 2 numeric values, normal arithmetic subtraction
    - anything else, null

### Ordering

Ordering on a particular expression can involve comparing values of different types.  The order is defined to be as specified in View Collation http://wiki.apache.org/couchdb/View_collation with the addition that undefined (different from null) are sorted before null (first).

# Moved Out of Initial Scope

### Multi-Queries
* SHOULD support combining individual queries with UNION, UNION ALL, INTERSECT, EXCEPT

### Sub-Query
* SHOULD support sub-query in the projection

### Expressions
* MUST support operators and/or functions for geospatial queries (described in Appendix 1)
* MUST support common functions to operate on strings, dates, arrays, and objects (described in Appendix 1)

### Geospatial Operators/Functions

* function - distance between points
* function - point in bounding box,circle,sphere,polygon

### Date functions

* functions to extract date parts (year/month/day/etc) from standard date formats?
* functions to build dates out of parts into standard formats?

### Array functions

* length

### Object functions

* length (number of key/value pairs)

### String functions

* length
* trim (right/left/both)