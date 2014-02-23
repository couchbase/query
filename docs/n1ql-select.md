# N1QL Query Language&mdash;SELECT

* Status: DRAFT
* Latest: [n1ql-select](https://github.com/couchbaselabs/query/blob/master/docs/n1ql-select.md)
* Modified: 2014-02-22

## Introduction

N1QL ("nickel") is the query language from Couchbase. N1QL aims to
meet the query needs of distributed document-oriented databases. This
document specifies the syntax and semantics of the SELECT statement in
N1QL.

*N1QL* stands for Non-1st Query Language. The name reflects the fact
that the Couchbase document-oriented data model is based on [Non-1st
Normal Form
(N1NF)](http://en.wikipedia.org/wiki/Database_normalization#Non-first_normal_form_.28NF.C2.B2_or_N1NF.29).

## SELECT syntax

The syntax of the SELECT statement is as follows.

### SELECT statement

*select:*

![](diagram/select.png)

In N1QL, SELECT statements can begin with either SELECT or FROM. The
behavior is the same in either case.

*select-core:*

![](diagram/select-core.png)

*select-from-core:*

![](diagram/select-from-core.png)

*from-select-core:*

![](diagram/from-select-core.png)

### SELECT clause

*select-clause:*

![](diagram/select-clause.png)

*result-expr:*

![](diagram/result-expr.png)

*path:*

![](diagram/path.png)

*alias:*

![](diagram/alias.png)

### FROM clause

*from-clause:*

![](diagram/from-clause.png)

*from-term:*

![](diagram/from-term.png)

*from-path:*

![](diagram/from-path.png)

*pool-name:*

![](diagram/pool-name.png)

*keys-clause:*

![](diagram/keys-clause.png)

*join-clause:*

![](diagram/join-clause.png)

*join-type:*

![](diagram/join-type.png)

*nest-clause:*

![](diagram/nest-clause.png)

*unnest-clause:*

![](diagram/unnest-clause.png)

### LET clause

*let-clause:*

![](diagram/let-clause.png)

### WHERE clause

*where-clause:*

![](diagram/where-clause.png)

*cond:*

![](diagram/cond.png)

### GROUP BY clause

*group-by-clause:*

![](diagram/group-by-clause.png)

*letting-clause:*

![](diagram/letting-clause.png)

*having-clause:*

![](diagram/having-clause.png)

### ORDER BY clause

*order-by-clause:*

![](diagram/order-by-clause.png)

*ordering-term:*

![](diagram/ordering-term.png)

### LIMIT clause

*limit-clause:*

![](diagram/limit-clause.png)

### OFFSET clause

*offset-clause:*

![](diagram/offset-clause.png)

## SELECT processing

The behavior of a SELECT query is best understood as a sequence of
steps. Output objects from each step become input objects to the next
step. The result of a SELECT is an array containing zero or more
result objects.

1.  Data sourcing - the data source in the FROM clause describes which
    objects become the input source for the query

1.  Filtering - result objects are filtered by the WHERE clause, if
    present

1.  Result object generation - result objects are generated from GROUP
    BY and HAVING clauses and the result expression list

1.  Duplicate removal - if DISTINCT is specified, duplicate result
    objects are removed

1.  Ordering - if ORDER BY is specified, the results are sorted
    according to the ordering terms

1.  Offsetting - if OFFSET is specified, the specified number of
    results are skipped

1.  Limiting - if LIMIT is specified, the results are limited to the
    given number

## FROM clause

The FROM clause defines the data sources and input objects for the
query.

Every FROM clause specifies one or more buckets. The first bucket is
called the primary bucket.

*from-clause:*

![](diagram/from-clause.png)

*from-term:*

![](diagram/from-term.png)

*from-path:*

![](diagram/from-path.png)

*pool-name:*

![](diagram/pool-name.png)

*keys-clause:*

![](diagram/keys-clause.png)

*join-clause:*

![](diagram/join-clause.png)

*join-type:*

![](diagram/join-type.png)

*nest-clause:*

![](diagram/nest-clause.png)

*unnest-clause:*

![](diagram/unnest-clause.png)

The following sections discuss various elements of the FROM
clause. These elements can be combined.

### Omitted

If the FROM clause is omitted, the data source is equivalent to an
array containing a single empty object. This allows you to evaluate
expressions that do not depend on stored data.

Evaluating an expression:

        SELECT 10 + 20

=>

        [ { "$1" : 30 } ]

Counting the number of inputs:

        SELECT COUNT(*) AS input_count

=>

        [ { "input_count" : 1 } ]

Getting the input contents:

        SELECT *

=>

        [ { } ]

### Buckets

The simplest type of FROM clause specifies a bucket:

        SELECT * FROM customer

This returns every value in the *customer* bucket.

The bucket name can be prefixed with an optional pool name:

        SELECT * FROM main:customer

This queries the *customer* bucket in the *main* pool.

If the pool name is omitted, the default pool in the current session
is used.

### Keys

Specific primary keys within a bucket can be specified. Only values
having those primary keys will be included as inputs to the query.

To specify a single key:

        SELECT * FROM customer KEYS "acme-uuid-1234-5678"

To specify multiple keys:

        SELECT * FROM customer KEYS [ "acme-uuid-1234-5678", "roadster-uuid-4321-8765" ]

In the FROM clause of a subquery, KEYS is mandatory for the primary
bucket.

### Nested paths

Nested paths within buckets can be specified. For each document in the
bucket, the path is evaluated and its value becomes an input the
query. For a given document, if any element of the path is NULL or
MISSING, that document is skipped and does not contribute any inputs
to the query.

If some customer documents contain a *primary\_contact* object, the
following query can retrieve them:

        SELECT * FROM customer.primary_contact

=>

        [
            { "name" : "John Smith", "phone" : "+1-650-555-1234", "address" : { ... } },
            { "name" : "Jane Brown", "phone" : "+1-650-555-5678", "address" : { ... } }
        ]

Nested paths can have arbitrary depth and can include array
subscripts.

        SELECT * FROM customer.primary_contact.address

=>

        [
            { "street" : "101 Main St.", "zip" : "94040" },
            { "street" : "3500 Wilshire Blvd.", "zip" : "90210" }
        ]

### Joins

Joins allow you to create new input objects by combining two or more
source objects. For example, if our *customer* objects were:

        {
            "name": ...,
            "primary_contact": ...,
            "address": [ ... ]
        }

And our *invoice* objects were:

        {
            "customer_key": ...,
            "invoice_date": ...,
            "invoice_item_keys": [ ... ],
            "total": ...
        }

And the FROM clause was:

        FROM invoice inv JOIN customer cust KEYS inv.customer_key

Then each joined object would be:

        {
            "inv" : {
                "customer_key": ...,
                "invoice_date": ...,
                "invoice_item_keys": [ ... ],
                "total": ...
            },
            "cust" : {
                "name": ...,
                "primary_contact": ...,
                "address": [ ... ]
            }
        }

If our *invoice_item* objects were:

        {
            "invoice_key": ...,
            "product_key": ...,
            "unit_price": ...,
            "quantity": ...,
            "item_subtotal": ...
        }

And the FROM clause was:

        FROM invoice JOIN invoice_item item KEYS invoice.invoice_item_keys

Then our joined objects would be:

        {
            "invoice" : {
                "customer_key": ...,
                "invoice_date": ...,
                "invoice_item_keys": [ ... ],
                "total": ...
            },
            "item" : {
                "invoice_key": ...,
                "product_key": ...,
                "unit_price": ...,
                "quantity": ...,
                "item_subtotal": ...
            }
        },
        {
            "invoice" : {
                "customer_key": ...,
                "invoice_date": ...,
                "invoice_item_keys": [ ... ],
                "total": ...
            },
            "item" : {
                "invoice_key": ...,
                "product_key": ...,
                "unit_price": ...,
                "quantity": ...,
                "item_subtotal": ...
            }
        },
        ...

KEYS is required after each JOIN. It specifies the primary keys for
the second bucket in the join.

Joins can be chained.

By default, an INNER join is performed. This means that for each
joined object produced, both the left and right hand source objects
must be non-missing and non-null.

If LEFT or LEFT OUTER is specified, then a left outer join is
performed. At least one joined object is produced for each left hand
source object. If the right hand source object is NULL or MISSING,
then the joined object's right-hand side value is also NULL or MISSING
(omitted), respectively.

### Unnests

If a document or object contains a nested array, UNNEST conceptually
performs a join of the nested array with its parent object. Each
resulting joined object becomes an input to the query.

If some customer documents contain an array of addresses under the
*address* field, the following query retrieves each nested address
along with the parent customer's name.

        SELECT c.name, a.* FROM customer c UNNEST c.address a

=>

        [
            { "name" : "Acme Inc.", "street" : "101 Main St.", "zip" : "94040" },
            { "name" : "Acme Inc.", "street" : "300 Broadway", "zip" : "10011" },
            { "name" : "Roadster Corp.", "street" : "3500 Wilshire Blvd.", "zip" : "90210" },
            { "name" : "Roadster Corp.", "street" : "4120 Alamo Dr.", "zip" : "75019" }
        ]

The first path element after each UNNEST must reference some preceding
path.

Unnests can be chained.

By default, an INNER unnest is performed. This means that for each
result object produced, both the left and right hand source objects
must be non-missing and non-null.

If LEFT or LEFT OUTER is specified, then a left outer unnest is
performed. At least one result object is produced for each left hand
source object. If the right hand source object is NULL, MISSING,
empty, or a non-array value, then the result object's right-hand side
value is MISSING (omitted).

### Nests

Nesting is conceptually the inverse of unnesting. Nesting performs a
join across two buckets. But instead of producing a cross-product of
the left and right hand inputs, a single result is produced for each
left hand input, while the corresponding right hand inputs are
collected into an array and nested as a single array-valued field in
the result object.

Recall our *invoice* objects:

        {
            "customer_key": ...,
            "invoice_date": ...,
            "invoice_item_keys": [ ... ],
            "total": ...
        }

And our *invoice_item* objects:

        {
            "invoice_key": ...,
            "product_key": ...,
            "unit_price": ...,
            "quantity": ...,
            "item_subtotal": ...
        }

If the FROM clause was:

        FROM invoice inv NEST invoice_item items KEYS inv.invoice_item_keys

The results would be:

        {
            "invoice" : {
                "customer_key": ...,
                "invoice_date": ...,
                "invoice_item_keys": [ ... ],
                "total": ...
            },
            "items" : [
                {
                    "invoice_key": ...,
                    "product_key": ...,
                    "unit_price": ...,
                    "quantity": ...,
                    "item_subtotal": ...
                },
                {
                    "invoice_key": ...,
                    "product_key": ...,
                    "unit_price": ...,
                    "quantity": ...,
                    "item_subtotal": ...
                }
            ]
        },
        {
            "invoice" : {
                "customer_key": ...,
                "invoice_date": ...,
                "invoice_item_keys": [ ... ],
                "total": ...
            },
            "items" : [
                {
                    "invoice_key": ...,
                    "product_key": ...,
                    "unit_price": ...,
                    "quantity": ...,
                    "item_subtotal": ...
                },
                {
                    "invoice_key": ...,
                    "product_key": ...,
                    "unit_price": ...,
                    "quantity": ...,
                    "item_subtotal": ...
                }
            ]
        },
        ...

Nests can be chained with other nests, joins, and unnests.

By default, an INNER nest is performed. This means that for each
result object produced, both the left and right hand source objects
must be non-missing and non-null.

If there is no matching right hand source object, then the right hand
source object is as follows:

* If the KEYS expression evaluates to MISSING, the right hand value
  is also MISSING
* If the KEYS expression evaluates to NULL, the right hand value is
  MISSING
* If the KEYS expression evaluates to an array, the right hand value
  is an empty array
* If the KEYS expression evaluates to a non-array value, the right
  hand value is an empty array

If LEFT or LEFT OUTER is specified, then a left outer nest is
performed. One result object is produced for each left hand source
object.

The right hand result of NEST is always an array or MISSING.

### Arrays

If an array occurs along a path, the array may be subscripted to
select one element.

Array values - for each customer, the entire *address* array is
selected:

        SELECT VALUE() FROM customer.address

=>

        [
            {
                "$1": [
                          { "street" : "101 Main St.", "zip" : "94040" },
                          { "street" : "300 Broadway", "zip" : "10011" }
                      ]
            },
            {
                "$1": [
                          { "street" : "3500 Wilshire Blvd.", "zip" : "90210" },
                          { "street" : "4120 Alamo Dr.", "zip" : "75019" }
                      ]
            }
        ]

Subscripting - for each customer, the first element of the *address*
array is selected:

        SELECT * FROM customer.address[0]

=>

        [
            { "street" : "101 Main St.", "zip" : "94040" },
            { "street" : "3500 Wilshire Blvd.", "zip" : "90210" }
        ]

## WHERE clause

*where-clause:*

![](diagram/where-clause.png)

*cond:*

![](diagram/cond.png)

If a WHERE clause is specified, the input objects are filtered
accordingly. The WHERE condition is evaluated for each input object,
and only objects evaluating to TRUE are retained.

## GROUP BY clause

*group-by-clause:*

![](diagram/group-by-clause.png)

*letting-clause:*

![](diagram/letting-clause.png)

*having-clause:*

![](diagram/having-clause.png)

### HAVING

## SELECT clause

*select-clause:*

![](diagram/select-clause.png)

*result-expr:*

![](diagram/result-expr.png)

*path:*

![](diagram/path.png)

*alias:*

![](diagram/alias.png)

### DISTINCT

## ORDER BY clause

*order-by-clause:*

![](diagram/order-by-clause.png)

*ordering-term:*

![](diagram/ordering-term.png)

If no ORDER BY clause is specified, the order in which the result
objects are returned is undefined.

If an ORDER BY clause is specified, the order of items in the result
array is determined by the ordering expressions.  Objects are first
sorted by the left-most expression in the list.  Any items with the
same sort value are then sorted by the next expression in the list.
This is repeated to break tie sort values until the end of the
expression list is reached.  The order of objects with the same sort
value for each sort expression is undefined.

As ORDER BY expressions can evaluate to any JSON value, we define an
ordering when comparing values of different types.  The following list
describes the order by type (from lowest to highest):

* missing value
* null
* false
* true
* number
* string (string comparison is done using a raw byte collation of UTF8
  encoded strings)
* array (element by element comparison is performed until the end of
  the shorter array; if all the elements so far are equal, then the
  longer array sorts after)
* object (larger objects sort after; for objects of equal length,
  key/value by key/value comparison is performed; keys are examined in
  sorted order using the normal ordering for strings)

## OFFSET clause

*offset-clause:*

![](diagram/offset-clause.png)

An OFFSET clause specifies a number of objects to be skipped. If a
LIMIT clause is also present, the OFFSET is applied prior to the
LIMIT.  The OFFSET value must be a non-negative integer.

## LIMIT clause

*limit-clause:*

![](diagram/limit-clause.png)

A LIMIT clause imposes an upper bound on the number of objects
returned by the SELECT statement.  The LIMIT value must be a
non-negative integer.

## Expressions

*expr:*

![](diagram/expr.png)

### Literal value

The specification for literal values can be found in Appendix.

### Identifier

*identifier:*

![](diagram/identifier.png)

An identifier can either be escaped or unescaped.  Unescaped
identifiers cannot support the full range of idenfiers allowed in a
JSON document, but do support the most common ones with a simpler
syntax.  Escaped identifiers are surrounded with backticks and support
all identifiers allowed in JSON.  Using the backtick character within
an escaped identifier can be accomplised by using two consecutive
backtick characters.

Keywords cannot be escaped; therefore, escaped identifiers can overlap
with keywords.

*unescaped-identifier:*

![](diagram/unescaped-identifier.png)

*escaped-identifier:*

![](diagram/escaped-identifier.png)

An identifier is a symbolic reference to a particular value in the
current context.

If the current context is the document:

    {
        "name": "n1ql"
    }

Then the identifier *name* would evaluate to the value n1ql.

#### Case-sensitivity of identifiers

Identifiers in N1QL are **case-sensitive.**

### Nested

nested-expr:

![](diagram/nested-expr.png)

Nested expressions support using the dot (`.`) operator to access
fields nested inside of other objects as well as using the bracket
notation (`[position]`) to access elements inside an array or object.

Consider the following object:

    {
      "address": {
        "city": "Mountain View"
      },
      "revisions": [2013]
    }

 The expression `address.city` evalutes to the value `"Mountain View"`.

 The expression `revisions[0]` evaluates to the value `2013`.

### Case

*case-expr:*

![](diagram/case-expr.png)

*simple-case-expr:*

![](diagram/simple-case-expr.png)

Simple case expressions allow for conditional matching within an
expression.  The first WHEN expression is evaluated.  If it is equal
to the search expression, the result of this expression is the THEN
expression.  If not, subsequent WHEN clauses are evaluated in the same
manner.  If none of the WHEN expressions is equal to the search
expression, then the result of the CASE expression is the ELSE
expression.  If no ELSE expression was provided, the result is NULL.

*searched-case-expr:*

![](diagram/searched-case-expr.png)

Searched case expressions allow for conditional logic within an
expression.  The first WHEN expression is evaluated.  If TRUE, the
result of this expression is the THEN expression.  If not, subsequent
WHEN clauses are evaluated in the same manner.  If none of the WHEN
clauses evaluate to TRUE, then the result of the expression is the
ELSE expression.  If no ELSE expression was provided, the result is
NULL.

### Logical

*logical-term:*

![](diagram/logical-term.png)

Logical terms allow for combining other expressions using boolean
logic.  Standard AND, OR and NOT operators are supported.

### Comparison

*comparison-term:*

![](diagram/comparison-term.png)

Comparison terms allow for comparing two expressions.  Standard
"equal", "not equal", "greater than", "greater than or equal", "less
than", and "less than or equal" are supported.

For equal (= and ==) and not equal (!= and <>) two forms are supported
to aid in compatibility with other query languages.

If either operand in a comparison is MISSING, the result is MISSING.
Next, if either operand in a comparison is NULL, the result is NULL.
Otherwise the remaining rules for comparing values are followed.

In N1QL a comparison operator implicitly requires that both operands
be of the same type.  If the operands are of different types it always
evaluates to FALSE.

String comparison is done using a raw byte collation of UTF8 encoded
strings (sometimes referred to as binary, C, or memcmp).  This
collation is **case sensitive**.  Case insensitive comparisons can be
performed using UPPER() or LOWER() functions.

#### LIKE

The LIKE operator allows for wildcard matching of string values.  The
right-hand side of the operator is a pattern, optionally containg '%'
and '\_' wildcard characters.  Percent (%) matches any string of zero
or more characters; underscore (\_) matches any single character.

The wildcards can be escaped by preceding them with a backslash
(\\). Backslash itself can also be escaped by preceding it with
another backslash.

### Arithmetic

*arithmetic-term:*

![](diagram/arithmetic-term.png)

Arithemetic terms allow for performing basic arithmetic within an
expression.  The standard addition, subtraction, multiplication,
division, and modulo operators are supported.  Additionally, a
negation operator will change the sign of the expression.

These arithmetic operators only operate on numbers. If either operand
is not a number, it will evaluate to NULL.

### Concatenation

*concatenation-term:*

![](diagram/concatenation-term.png)

If both operands are strings, the `||` operator concatenates these
strings.  Otherwise the expression evaluates to NULL.

### Function

*function-call:*

![](diagram/function-call.png)

*function-name:*

![](diagram/function-name.png)

Function names are case-insensitive.  See Appendices for the supported
functions.

### Subquery

*subquery-expr:*

![](diagram/subquery-expr.png)

Subquery expressions return an array that is the result of evaluating
the subquery.

In the FROM clause of a subquery, KEYS is mandatory for the primary
bucket.

### Collection

*collection-expr:*

![](diagram/collection-expr.png)

*exists-expr:*

![](diagram/exists-expr.png)

EXISTS evaluates to TRUE if the value is an array and contains at
least one element.

*in-expr:*

![](diagram/in-expr.png)

IN evaluates to TRUE if the right-hand-side value is an array and
contains the left-hand-side value.

*collection-cond:*

Collection predicates allow you to test a boolean condition over the
elements of a collection.

![](diagram/collection-cond.png)

*variable:*

![](diagram/variable.png)

*collection-xform:*

Collection transforms allow you to map and filter the elements of a
collection.

![](diagram/collection-xform.png)

*array-expr:*

![](diagram/array-expr.png)

*first-expr:*

![](diagram/first-expr.png)

## Boolean interpretation

Some contexts require values to be interpreted as booleans. For
example:

* WHERE clause
* HAVING clause
* WHEN clause

The following rules apply:

* MISSING, NULL, and false are false
* numbers +0, -0, and NaN are false
* empty strings, arrays, and objects are false
* all other values are true

## Appendix - Identifier scoping / ambiguity

Identifiers include bucket names, fields within documents, and
aliases. The following rules apply.

* FROM - Aliases in the FROM clause create new names that may
  be referred to anywhere in the query
* LET - Aliases in the LET clause create new names that may
  be referred to anywhere in the query
* LETTING - Aliases in the LETTING clause create new names that may be
  referred to in the HAVING, SELECT, and ORDER BY clauses
* SELECT - Aliases in the projection create new names that
  may be referred to in the SELECT and ORDER BY clauses
* FOR - Aliases in a collection expression create names that
  are local to that collection expression

When an alias collides with a bucket or field name in the same scope,
the identifier always refers to the alias. This allows for consistent
behavior in scenarios where an identifier only collides in some
documents.

The left-most portion of a dotted identifier may refer to the name of
the datasource.  For example:

    SELECT beer.name FROM beer

In this query `beer.name` is simply a more formal way of expressing
`name`.

## Appendix - Operator precedence

The following operators are supported by N1QL.  The list is ordered
from highest to lowest precedence.

* CASE/WHEN/THEN/ELSE/END
* . 
* [] 
* \- (unary)
* \* / %
* \+ \- (binary)
* IS VALUED, IS NULL, IS MISSING
* IS NOT VALUED, IS NOT NULL, IS NOT MISSING
* LIKE
* < <= > >=
* =, ==, <>, !=
* NOT
* AND
* OR

Parentheses allow for grouping expressions to override the order of
operations. They have the highest precedence.

## Appendix - Four-valued logic

In N1QL boolean propositions can evaluate to NULL or MISSING.  The
following table describes how these values relate to the logical
operators:

<table>
  <tr>
        <th>A</th>
        <th>B</th>
        <th>A and B</th>
        <th>A or B</th>
  </tr>
  <tr>
        <td>FALSE</td>
        <td>FALSE</td>
        <td>FALSE</td>
        <td>FALSE</td>
  </tr>
   <tr>
        <td>TRUE</td>
        <td>TRUE</td>
        <td>TRUE</td>
        <td>TRUE</td>
  </tr>
   <tr>
        <td>FALSE</td>
        <td>TRUE</td>
        <td>FALSE</td>
        <td>TRUE</td>
  </tr>
  <tr>
        <td>FALSE</td>
        <td>NULL</td>
        <td>FALSE</td>
        <td>NULL</td>
  </tr>
  <tr>
        <td>TRUE</td>
        <td>NULL</td>
        <td>NULL</td>
        <td>TRUE</td>
  </tr>
  <tr>
        <td>NULL</td>
        <td>NULL</td>
        <td>NULL</td>
        <td>NULL</td>
  </tr>
  <tr>
        <td>FALSE</td>
        <td>MISSING</td>
        <td>FALSE</td>
        <td>MISSING</td>
  </tr>
  <tr>
        <td>TRUE</td>
        <td>MISSING</td>
        <td>MISSING</td>
        <td>TRUE</td>
  </tr>
  <tr>
        <td>NULL</td>
        <td>MISSING</td>
        <td>MISSING</td>
        <td>NULL</td>
  </tr>
  <tr>
        <td>MISSING</td>
        <td>MISSING</td>
        <td>MISSING</td>
        <td>MISSING</td>
  </tr>
</table>

<table>
        <tr>
                <th>A</th>
                <th>not A</th>
        </tr>
        <tr>
                <td>FALSE</td>
                <td>TRUE</td>
        </tr>
        <tr>
                <td>TRUE</td>
                <td>FALSE</td>
        </tr>
        <tr>
                <td>NULL</td>
                <td>NULL</td>
        </tr>
        <tr>
                <td>MISSING</td>
                <td>MISSING</td>
        </tr>
</table>

#### Comparing NULL and MISSING values

<table>
        <tr>
                <th>Operator</th>
                <th>Non-NULL Value</th>
                <th>NULL</th>
                <th>MISSING</th>
        </tr>
    <tr>
        <td>IS NULL</td>
        <td>FALSE</td>
        <td>TRUE</td>
        <td>MISSING</td>
    </tr>
    <tr>
        <td>IS NOT NULL</td>
        <td>TRUE</td>
        <td>FALSE</td>
        <td>MISSING</td>
    </tr>
    <tr>
        <td>IS MISSING</td>
        <td>FALSE</td>
        <td>FALSE</td>
        <td>TRUE</td>
    </tr>
    <tr>
        <td>IS NOT MISSING</td>
        <td>TRUE</td>
        <td>TRUE</td>
        <td>FALSE</td>
    </tr>
    <tr>
        <td>IS VALUED</td>
        <td>TRUE</td>
        <td>FALSE</td>
        <td>FALSE</td>
    </tr>
    <tr>
        <td>IS NOT VALUED</td>
        <td>FALSE</td>
        <td>TRUE</td>
        <td>TRUE</td>
    </tr>
</table>

## Appendix - Literal JSON values

The following rules are the same as defined by
[json.org](http://json.org/) with two changes:

1. In standard JSON arrays and objects only contain nested values.  In
   N1QL, literal arrays and objects can contain nested expressions.
1. In standard JSON "true", "false" and "null" are case-sensitive.  In
   N1QL, to be consistent with other keywords, they are defined to be
   case-insensitive.

*literal-value:*

![](diagram/literal-value.png)

*object:*

![](diagram/object.png)

*members:*

![](diagram/members.png)

*pair:*

![](diagram/pair.png)

*array:*

![](diagram/array.png)

*elements:*

![](diagram/elements.png)

*string:*

![](diagram/string.png)

*chars:*

![](diagram/chars.png)

*char:*

![](diagram/char.png)

*number:*

![](diagram/number.png)

*int:*

![](diagram/int.png)

*uint:*

![](diagram/uint.png)

*frac:*

![](diagram/frac.png)

*exp:*

![](diagram/exp.png)

*digits:*

![](diagram/digits.png)

*non-zero-digit:*

![](diagram/non-zero-digit.png)

*digit:*

![](diagram/digit.png)

*e:*

![](diagram/e.png)

*hex-digit:*

![](diagram/hex-digit.png)

## Appendix - Comments

N1QL supports both block comments and line comments.

block-comment:

![](diagram/block-comment.png)

line-comment:

![](diagram/line-comment.png)

## Appendix - Scalar functions

Function names are case in-sensitive. Scalar functions return MISSING
if any argument is MISSING, and then NULL if any argument is NULL or
not of the required type.

### Date functions

**CLOCK\_NOW\_MILLIS()** - system clock at function evaluation time,
as UNIX milliseconds; varies during a query.

**CLOCK\_NOW\_STR()** - system clock at function evaluation time, as a
string in ISO 8601 / RFC 3339 format; varies during a query.

**DATE\_PART\_MILLIS(expr, part)** - date part as an integer. The date
expr is a number representing UNIX milliseconds, and part is one of
the following date part strings.

* **"millenium"**
* **"century"**
* **"decade"** - year / 10
* **"year"**
* **"quarter"** - 1 to 4
* **"month"** - 1 to 12
* **"day"** - 1 to 31
* **"hour"** - 0 to 23
* **"minute"** - 0 to 59
* **"second"** - 0 to 59
* **"millisecond"** - 0 to 999
* **"week"** - 1 to 53; ceil(day\_of\_year / 7.0)
* **"day\_of\_year", "doy"** - 1 to 366
* **"day\_of\_week", "dow"** - 0 to 6
* **"iso\_week"** - 1 to 53; use with "iso_year"
* **"iso\_year"** - use with "iso_week"
* **"iso\_dow"** - 1 to 7
* **"timezone"** - offset from UTC in seconds
* **"timezone\_hour"** - hour component of timezone offset
* **"timezone\_minute"** - minute component of timezone offset

**DATE\_PART\_STR(expr, part)** - date part as an integer. The date
expr is a string in a supported format, and part is one of the
supported date part strings.

**DATE\_TRUNC\_MILLIS(expr, part)** - truncates UNIX timestamp so that
the given date part string is the least significant.

**DATE\_TRUNC\_STR(expr, part)** - truncates ISO 8601 timestamp so
that the given date part string is the least significant.

**MILLIS\_TO\_STR(expr)** - converts UNIX milliseconds to string in
ISO 8601 format.

**NOW\_MILLIS()** - statement timestamp as UNIX milliseconds; does not
vary during a query.

**NOW\_STR()** - statement timestamp as a string in ISO 8601 / RFC
3339 format; does not vary during a query.

**STR\_TO\_MILLIS(expr)** - converts date in ISO 8601 format to UNIX
milliseconds.

### String functions

**CONTAINS(expr, substr)** - true if the string contains the
substring.

**INITCAP(expr)** - converts the string so that the first letter of
each word is uppercase and every other letter is lowercase.

**LENGTH(expr)** - length of the string value.

**LOWER(expr)** - lowercase of the string value.

**LTRIM(expr)** - string with all beginning whitespace removed.

**POSITION(expr, substr)** - the first position of the substring
within the string, or -1. The position is 0-based.

**REMOVE(expr, substr)** - string with all occurences of *substr*
removed.

**REPEAT(expr, count)** - string formed by repeating *expr* *count*
times.

**REPLACE(expr, substr1, substr2)** - string with all occurences of
*substr1* replaced with *substr2.*

**REVERSE(expr)** - string with characters in reverse order.

**RTRIM(expr)** - string with all ending whitespace removed.

**SUBSTR(expr, position)** - returns the substring from the integer
*position* to the end of the string. The position is 0-based, i.e. the
*first position is 0. If *position* is negative, it is counted from
*the end of the string; -1 is the last position in the string.

**SUBSTR(expr, position, length)** - returns the substring of the
given *length* from the integer *position* to the end of the
string. The position is 0-based, i.e. the first position is 0. If
*position* is negative, it is counted from the end of the string; -1
is the last position in the string.

**TRIM(expr)** - string with all beginning and ending whitespace
removed.

**UPPER(expr)** - uppercase of the string value.

### Number functions

**ABS(expr)** - absolute value of the number.

**CEIL(expr)** - smallest integer not less than the number.

**FLOOR(expr)** - largest integer not greater than the number.

**RANDOM()** -

**RANDOM(expr)** -

**ROUND(expr)** - rounds the number to the nearest integer; same as
ROUND(value, 0).

**ROUND(expr, digits)** - rounds the value to the given number of
integer digits to the right of the decimal point (left if digits is
negative).

**SIGN(expr)** - -1, 0, or 1 for negative, zero, or positive numbers
respectively.

**TRUNC(expr)** - truncates the number towards zero; same as
TRUNC(value, 0).

**TRUNC(expr, digits)** - truncates the number to the given number of
integer digits to the right of the decimal point (left if digits is
negative).

### Array functions

**ARRAY\_ADD(expr, value)** - new array with *value* appended, if
*value* is not already present; else unmodified input array.

**ARRAY\_APPEND(expr, value)** - new array with *value* appended.

**ARRAY\_CONCAT(expr1, expr2)** - new array with the concatenation of
the input arrays.

**ARRAY_CONTAINS(expr, value)** - true if the array contains *value.*

**ARRAY_DISTINCT(expr)** - new array with distinct elements of input
array.

**ARRAY\_IFNULL(expr)** - return the first non-NULL value in the
array, or NULL.

**ARRAY\_LENGTH(expr)** - number of elements in the array.

**ARRAY\_MAX(expr)** - largest non-NULL, non-MISSING array element, in
N1QL collation order.

**ARRAY\_MIN(expr)** - smallest non-NULL, non-MISSING array element,
in N1QL collation order.

**ARRAY_POSITION(expr, value)** - the first position of *value* within
the array, or -1. The position is 0-based.

**ARRAY\_PREPEND(value, expr)** - new array with *value* prepended.

**ARRAY\_REMOVE(expr, value)** - new array with all occurences of
*value* removed.

**ARRAY\_REPEAT(value, count)** - new array with *value* repeated
*count* times.

**ARRAY\_REPLACE(expr, value1, value2)** - new array with all
occurences of *value1* replaced with *value2.*

**ARRAY\_REVERSE(expr)** - new array with all elements
in reverse order.

**ARRAY\_SORT(expr)** - new array with elements sorted in N1QL
collation order.

### Object functions

**OBJECT\_LENGTH(expr)** - returns the number of key-value pairs in
the object.

**OBJECT\_KEYS(expr)** - returns an array containing the keys of the
object, in N1QL collation order.

**OBJECT\_VALUES(expr)** - returns an array containing the values of
the object, in N1QL collation order of the corresponding keys.

### JSON functions

**POLY\_LENGTH(expr)** - length of the value after evaluating the
expression.  The exact meaning of length depends on the type of the
value:

* MISSING - MISSING
* NULL - NULL
* string - the length of the string
* array - the number of elements in the array
* object - the number of key/value pairs in the object
* any other value - NULL

**SIZE\_JSON(expr)** - returns the number of bytes in an uncompressed
JSON encoding of the value. The exact size is
implementation-dependent. Always returns an integer, and never MISSING
or NULL; returns 0 for MISSING.

### Comparison functions

**GREATEST(expr1, expr2, ...)** - largest non-NULL, non-MISSING value
if the values are of the same type; otherwise NULL.

**LEAST(expr1, expr2, ...)** - smallest non-NULL, non-MISSING value if
the values are of the same type; otherwise NULL.

### Conditional functions for unknowns

**IFMISSING(expr1, expr2, ...)** - returns the first non-MISSING
value.

**IFMISSINGORNULL(expr1, expr2, ...)** - returns the first non-NULL,
non-MISSING value.

**IFNULL(expr1, expr2, ...)** - returns the first non-NULL value. Note
that this function may return MISSING.

**MISSINGIF(value1, value2)** - if value1 = value 2, returns MISSING;
otherwise, value1.

**NULLIF(value1, value2)** - if value1 = value 2, returns NULL,
otherwise value1

### Conditional functions for numbers

**IFINF(expr1, expr2, ...)** -

**IFNAN(expr1, expr2, ...)** -

**IFNANORINF(expr1, expr2, ...)** -

**IFNEGINF(expr1, expr2, ...)** -

**IFPOSINF(expr1, expr2, ...)** -

**FIRSTNUM(expr1, expr2, ...)** -

**NANIF(expr1, expr2)** -

**NEGINFIF(expr1, expr2)** -

**POSINFIF(expr1, expr2)** -

### Meta and value functions

**BASE64_VALUE()** -

**BASE64_VALUE(expr)** -

**META()** - returns the meta data for the primary document in the
current context.

**META(expr)** -

**VALUE()** -

**VALUE(expr)** -

### Type checking functions

**IS_ARRAY(expr)** - true if expr is an array; else false.

**IS_ATOM(expr)** - true if expr is a boolean, number, or
string; else false.

**IS_BOOL(expr)** - true if expr is a boolean; else false.

**IS_NUM(expr)** - true if expr is a number; else false.

**IS_OBJ(expr)** - true if expr is an object; else false.

**IS_STR(expr)** - true if expr is a string; else false.

**TYPE_NAME(expr)** - one of the following strings, based on the value
of expr:

* **"missing"**
* **"null"**
* **"not_json"**
* **"boolean"**
* **"number"**
* **"string"**
* **"array"**
* **"object"**

### Type casting functions

**TO_ARRAY(expr)** - array as follows:

* MISSING is MISSING
* NULL is NULL
* arrays are themselves
* all other values are wrapped in an array

**TO_ATOM(expr)** - atomic value as follows:

* MISSING is MISSING
* NULL is NULL
* arrays of length 1 are the result of TOATOM() on their single element
* objects of length 1 are the result of TOATOM() on their single value
* booleans, numbers, and strings are themselves
* all other values are NULL

**TO_BOOL(expr)** - boolean as follows:

* MISSING is MISSING
* NULL is NULL
* false is false
* numbers +0, -0, and NaN are false
* empty strings, arrays, and objects are false
* all other values are true

**TO_NUM(expr)** - number as follows:

* MISSING is MISSING
* NULL is NULL
* false is 0
* true is 1
* numbers are themselves
* strings that parse as numbers are those numbers
* all other values are NULL

**TO_STR(expr)** - string as follows:

* MISSING is MISSING
* NULL is NULL
* false is "false"
* true is "true"
* numbers are their string representation
* strings are themselves
* all other values are NULL

## Appendix - Aggregate functions

Aggregate functions can only be used in LETTING, HAVING, SELECT, and
ORDER BY clauses.  When aggregate functions are used in expressions in
these clauses, the query will operate as an aggregate query.

If there is no input row for the group, COUNT functions return 0. All
other aggregate functions return NULL.

**ARRAY_AGG(expr)** - array of the values in the group, including
NULLs.

**ARRAY_AGG(DISTINCT expr)** - array of the distinct values in the
group, including NULLs.

**AVG(expr)** - arithmetic mean (average) of all the numeric,
non-NULL, non-MISSING values in the group.

**AVG(DISTINCT expr)** - arithmetic mean (average) of all the
distinct numeric, non-NULL, non-MISSING values in the group.

**COUNT(*)** - count of all the input rows for the group, regardless
of value.

**COUNT(expr)** - count of all the non-NULL, non-MISSING values in
the group.

**COUNT(DISTINCT expr)** - count of all the distinct non-NULL,
non-MISSING values in the group.

**MAX(expr)** - maximum non-NULL, non-MISSING value in the group,
according to N1QL collation.

**MIN(expr)** - minimum non-NULL, non-MISSING value in the group,
according to N1QL collation.

**SUM(expr)** - arithmetic sum of all the numeric, non-NULL,
non-MISSING values in the group.

**SUM(DISTINCT expr)** - arithmetic sum of all the distinct numeric,
non-NULL, non-MISSING values in the group.

## Appendix - Key / reserved words

The following keywords are reserved and cannot be used in document
property paths without escaping.  All keywords are case-insensitive.

Keywords cannot be escaped; therefore, escaped identifiers can overlap
with keywords.

* ALL
* ALTER
* AND
* ANY
* AS
* ASC
* BETWEEN
* BUCKET
* BY
* CASE
* CAST
* COLLATE
* CREATE
* DATABASE
* DELETE
* DESC
* DISTINCT
* DROP
* EACH
* ELSE
* END
* EXCEPT
* EXISTS
* EXPLAIN
* FALSE
* FROM
* GROUP
* HAVING
* IF
* IN
* INLINE
* INSERT
* INTERSECT
* INTO
* IS
* JOIN
* LIKE
* LIMIT
* MISSING
* NOT
* NULL
* OFFSET
* ON
* OR
* ORDER
* OVER
* PATH
* SELECT
* THEN
* TRUE
* UNION
* UNIQUE
* UPDATE
* VALUED
* WHEN
* WHERE
* XOR

## Appendix - Sample projections

For the following examples consider a bucket containing the following
document with ID "n1ql-2013"

    {
      "name": "N1QL",
      "address": {
        "city": "Mountain View"
      },
      "revisions": [2013]
    }

### Selecting the whole document

`SELECT *`

    {
      "name": "N1QL",
      "address": {
        "city": "Mountain View"
      },
      "revisions": [2013]
    }

### Selecting individual field

`SELECT name`

    {
        "name": "N1QL"
    }

### Selecting a more complex expression

`SELECT revsions[0] - 13`

    {
        "revisions[0]-13": 2000
    }

### Selecting a more complex expression with custom identifier

`SELECT revsions[0] - 13 AS modified_revision`

    {
        "modified_revision": 2000
    }

### Selecting the whole document and adding meta-data

`SELECT *, META()`

    {
      "name": "N1QL",
      "address": {
        "city": "Mountain View"
      },
      "revisions": [2013],
      "meta": {
        "id": "n1ql-2013",
        "cas": "8BADF00DDEADBEEF",
        "flags": 0,
        "expiration": 0
      }
    }

### Selecting the whole document and adding meta-data with custom identifer (to avoid any collisions)

`SELECT *, META() AS custom_meta_field`

    {
      "name": "N1QL",
      "address": {
        "city": "Mountain View"
      },
      "revisions": [2013],
      "custom_meta_field": {
        "id": "n1ql-2013",
        "cas": "8BADF00DDEADBEEF",
        "flags": 0,
        "expiration": 0
      }
    }

### Building a complex object using literal JSON

SELECT {"thename": name} AS custom_obj

    {
      "custom_obj": {
        "thename": "N1QL"
      }
    }

## Appendix - Syntax changes for Beta / GA

#### FROM ... OVER => FROM ... UNNEST

* Replaced FROM ... OVER with FROM ... UNNEST

#### ANY / ALL ... OVER => ANY / EVERY ... SATISFIES

* Replaced ANY / ALL ... OVER with ANY / EVERY ... SATISFIES

#### KEYS Clause

* Added KEYS clause to FROM clause

#### JOINs

* Added JOINs based on primary keys

#### Subqueries

* Added subqueries based on primary keys

#### CASE expressions

* Added a second form of CASE expression

#### Array functions and slicing

* Added array slicing

* Added ARRAY_CONCAT(), ARRAY_LENGTH(), ARRAY_APPEND(),
  ARRAY_PREPEND(), and other array functions

## About this document

The
[grammar](https://github.com/couchbaselabs/query/blob/master/docs/n1ql-select.ebnf)
forming the basis of this document is written in a [W3C dialect of
EBNF](http://www.w3.org/TR/REC-xml/#sec-notation).

This grammar has not yet been converted to an actual implementation,
ambiguities and conflicts may still be present.

Diagrams were generated by [Railroad Diagram
Generator](http://bottlecaps.de/rr/ui/) ![](diagram/.png)

### Document history

* 2013-07-06 - Initial branching off from UNQL spec
    * Added joins, subqueries, and pools
    * Allowed scalar values along path joins
    * Generalized collection expressions
    * Added comprehensions
    * Added EXISTS, IN, NOT IN
    * Hid UNIQUE, which is supported as a synonym for DISTINCT (to reduce the spec slightly)
    * Added BETWEEN, NOT BETWEEN
    * Made AS optional everywhere
    * Generalized concatenation to include arrays
    * Added DIV for integer division
    * Detached OFFSET from LIMIT
* 2013-07-06 - Cosmetics
    * Fixed some spelling
    * Clarified some prose
    * Added (non-)description of joins
* 2013-07-08 - Grammar
    * Clarified that escaped identifiers can overlap with keywords
    * Streamlined grammar for functions and result-exprs
    * Added to open issues
* 2013-07-13 - Comments
    * Added Appendix on comments
* 2013-07-15 - Added Open Issue
    * Added open issue on default aliases in results / projections
* 2013-07-16 - Case-sensitivity and rounding
    * Specified syntax for case-sensitivity of field names
    * Specified behavior of ROUND() and TRUNC() functions
* 2013-07-17 - Case-sensitivity
    * Renamed default case-sensitivity to nearest-case matching
    * On duplicate matches, no match is made and warnings are generated
* 2013-07-19 - cond
    * Added cond to EBNF diagrams
* 2013-07-23 - JOIN result objects
    * Added Appendix on JOIN result objects
* 2013-12-03 - Target syntax
    * Updated syntax targeting beta / production release
    * KEY joins and subqueries
    * Updated syntax for FROM UNNEST
    * Updated syntax for collection expressions
    * BETWEEN operator
    * [*] operator for array traversal
    * Handle NaN and +/- infinity
    * Date/Time features
* 2013-12-10 - Syntax
    * Array expansion
* 2013-12-15 - Beta / GA deltas
    * Added appendix on syntax changes for Beta / GA
* 2014-01-01 - NEST
    * Added NEST
    * Added join-type to UNNEST
* 2014-01-02 - NEST result format
    * Changed NEST result format to be consistent with UNNEST and JOIN
* 2014-01-07 - NEST result format
    * Changed result format for inner nests
* 2014-01-21 - Collection expressions
    * Per customer requirement, extend collection expressions to
      multiple collections
    * Customer requirement: If you have a property that is an array of
      subdocuments like the children property in your examples, it
      looks easy to find the documents where there is a child with the
      gender equal to female and the age greater than 12 say for
      example.  Now suppose the data is stored differently.  There are
      now two properties in the document, each being an array, one for
      the list of children genders and one for the list of children
      ages.  In this case the gender on line one corresponds to the
      age on line one.  How would you search the document such that
      the lines matched up?  How do you make sure the line that is
      female is also the line where the age is greater than 12?  Now
      take that one step further and put the two properties into two
      separate documents.  You still want to find the documents where
      the search criteria are true on the same line in each document.
      How do you do that?  We need to be able to relate multiple
      properties together on a line by line basis and they may be
      stored separately.
* 2014-02-06 - Misc
    * Restored IS [ NOT ] VALUED
    * Removed MAP() and REDUCE() for now
    * Syntax fixes for traversing multiple collections
* 2014-02-10 - Function call syntax
    * Removed COUNT(path.*), kept COUNT(*)
    * Removed COUNT(DISTINCT *), kept COUNT(DISTINCT expr, ...)
* 2014-02-13 - SELECT clause
    * Omit result expressions to return raw value
* 2014-02-14 - Collation, LET, LETTING
    * Clarified collation spec strings, arrays, and objects
    * Added LET and LETTING clauses
* 2014-02-16 - SELECT RAW
    * Added SELECT RAW
* 2014-02-17 - KEY / KEYS
    * Cleaned up usage of KEY and kEYS.
* 2014-02-18 - SELECT list
    * Require SELECT list
* 2013-12-18 - Array expansion
    * Removed array expansion for now.
* 2013-12-22 - Functions
    * Expanded set of functions.

### Open issues

This meta-section records open issues in this document, and will
eventually disappear.
