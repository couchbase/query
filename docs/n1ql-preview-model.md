# The Couchbase Query Model&mdash;A Preview

by Gerald Sangudi

* Status: DRAFT
* Latest: [n1ql-preview-model](https://github.com/couchbaselabs/query/blob/master/docs/n1ql-preview-model.md)
* Modified: 2013-08-20

## Introduction

Couchbase is the leading next-generation, high-performance, scale-out,
document-oriented database. The Couchbase query model aims to meet the
query needs of Couchbase users and applications at scale.

This paper previews the Couchbase approach to querying. It is written
for readers who are familiar with database concepts. This paper is a
conceptual overview, and not a tutorial, reference manual, or
explication of syntax. It presents the overall Couchbase approach, and
not a specific feature set associated with a product release or point
in time.

The following sections discuss the Couchbase data model; the Couchbase
query model; and the new query language N1QL (pronounced "nickel"),
which is the first flavor and incarnation of the Couchbase query
model.

## Data model

The data model, or how data is organized, is a defining characteristic
of any database. Couchbase is both a key-value and document-oriented
database. Every data value in Couchbase is associated with an
immutable primary key. For data values that have structure and are not
opaque, those values are encoded as JSON documents, with each document
mapped to a primary key.

We start with the conceptual basis of the Couchbase data model. In
database formalism, the Couchbase data model is based on Non-First
Normal Form, or N1NF. This model is a superset and generalization of
the relational model, which requires data normalization to First
Normal Form (1NF) and advocates further normalization to Third Normal
Form (3NF). We examine the relational model and its normalization
principles, and then proceeed to the Couchbase N1NF model and its
encoding in JSON documents.

### Relational model and normalization

In the relational model, data is organized into rows (tuples) and
tables (relations, or sets of tuples). A row has one or more
attributes, with each attribute containing a single atomic data
value. All the rows of a given table have the same attributes. If a
row does not have a known value for a given attribute, that attribute
is assigned the special value NULL.

For illustration, we use an example data set from a shopping cart
application. The data set contains the following objects:

* *Customer* (identity, addresses, payment methods)
* *Product* (title, code, description, unit price)
* *Shopping Cart* (customer, shipping address, payment method, line items)

Let us consider the principal relational normal forms.

* **First normal form (1NF)** requires that each attribute of every
  row contain only a single, atomic value. An attribute cannot contain
  a nested tuple, because a tuple is not atomic (it contains
  attributes). And an attribute cannot contain multiple atomic
  values. Examples of a valid attribute value would include a single
  number, string, or date.

  Suppose we want to support multiple shipping addresses per
  customer. Because a 1NF attribute can only store a single value per
  row, we would be unable to store addresses in the *Customer* table,
  and would need to create a separate *Customer\_Address* table.

  Suppose also that we want to store multiple components per address,
  such as zip code and state, in order to analyze the geographical
  distribution of customers. Then the *Customer\_Address* table cannot
  simply contain a single *Address* attribute, because a 1NF attribute
  must be atomic and cannot be decomposable into components. Thus
  *Customer\_Address* would need to contain attributes such as
  *Address\_Id*, *Customer\_Id*, *Street\_Address*, *City*, *Zip*, and
  *State.*

  Similarly, multiple line items per shopping cart could not be stored
  in the *Shopping\_Cart* table. Instead, we would create a separate
  *Shopping\_Cart\_Line\_Item* table, with attributes including
  *Line\_Item\_Id, Shopping\_Cart\_Id, Product\_Id,* and *Quantity.*

  The practical rules for ensuring 1NF are:

    * Store multi-valued data in multiple rows, creating a separate
      table if necessary; do not use multiple columns to store
      multi-valued data
    * Identify each row with a unique primary key

Second and third normal form aim to prevent any piece of information
from being represented in more than one row in the database. If the
same piece of information is represented in more than one row, it is
possible to modify one row so that it becomes inconsistent with the
others; avoiding such inconsistenties is one of the key goals of the
relational model.

**Candidate keys** are used in defining second and third normal
form. A candidate key of a table is any minimal set of attributes
whose values form a unique identifier for (rows in) the table.

**Update anomalies** are the inconsistencies that arise when a table
is not normalized, and one row with a piece of information is modified
to be inconsistent with another row containing the same piece of
information.

By definition, every attribute of every candidate key helps to
uniquely identify rows in a table. Such attributes, called prime
attributes, cannot cause inconsistencies&mdash;if the values of a
prime attribute are different in two different rows, then those two
rows represent separate pieces of information, because the prime
attribute is part of the unique identity of a row.

Second and third normal form are defined as restrictions on non-prime
attributes, which are those attributes that are not part of any
candidate key.

* **Second normal form (2NF)** requires that a table be in 1NF and not
  contain any non-prime attribute that is dependent on a proper subset
  of any candidate key.

  Suppose that a customer can create only one shopping cart per
  instant in time, so that *(Customer\_Id, Creation\_Time)* is a
  candidate key for the *Shopping\_Cart* table. Now suppose that we
  stored *Customer\_Birthdate* in the *Shopping\_Cart* table, in order
  to track purchases by age group. Then *Shopping_Cart* would **not**
  be in 2NF, because:

      * *Customer\_Birthdate* is a non-prime attribute
      * *Customer\_Birthdate* is dependent on *Customer\_Id*
      * *Customer\_Id* is a proper subset of the candidate key
        *(Customer\_Id, Creation\_Time)*

  If a customer has multiple shopping carts over time, and thus
  multiple rows in *Shopping\_Cart*, it would be possible to modify
  *Customer\_Birthdate* in one of those rows to be inconsistent with
  the others. To satisfy 2NF, we would move *Customer\_Birthdate* from
  *Shopping\_Cart* to the *Customer* table, where each customer would
  have exactly one row.

  Note that a 1NF table with no composite candidate keys is
  automatically in 2NF; only composite candidate keys can have proper
  subsets.

  The practical rules for ensuring 2NF are:
    * Ensure 1NF
    * Remove subsets of attributes that repeat in multiple rows, and
      move them to a separate table
    * Use foreign keys to connect related tables

**Superkeys** are used in defining third normal form. A superkey is
any set of attributes that forms a unique identifier for rows in a
table. Unlike a candidate key, a superkey does not need to be a
*minimal* set of attributes. Every candidate key is also a superkey.

* **Third normal form (3NF)** requires that a table be in 2NF and that
  every non-prime attribute be directly dependent on every superkey
  and completely independent of every other non-key attribute. As
  Wikipedia quotes Bill Kent: "Every non-key attribute must provide a
  fact about the key, the whole key, and nothing but the key (so help
  me Codd)."

  To obey 1NF, we created a *Customer\_Address* table and included the
  attributes *Address\_Id*, *Customer\_Id*, *Street\_Address*, *City*,
  *Zip*, and *State.* But this table contains a violation of
  3NF. *Zip* always determines *State*, and *Zip* and *State* are both
  non-primary attributes. Therefore, the non-primary attribute *State*
  is dependent on the non-key attribute *Zip*, which is a violation of
  3NF. We would need to create a separate *Zip* table with mappings
  from zip codes to states.

  The practical rules for ensuring 3NF are:
    * Ensure 1NF and 2NF
    * Remove attributes that are not directly and exclusively
      dependent on the primary key

#### Benefits

The relational model and its normalization principles achieved several
benefits. Data duplication was minimized, which enhanced data
consistency and compactness of storage. Data updates could be granular
and authoritative. And a high degree of physical data independence was
ensured by the model: because data was normalized into discrete and
directly accessible tables, no single traversal path or object
composition was favored to the exclusion of others.

#### Costs

The costs of the relational model in performance and complexity arose
primarily from one cause: **The relational model did not model the
intrinsic structure of most data, and instead forced users to choose
between normalization, consistency, and mandatory joins on the one
hand; and denormalization, duplication, and update anomalies on the
other.**

Our example data set intrinsically has three independent tables:
*Customer, Product, and Shopping Cart.* The relational model would
force us to introduce at least two additional tables:
*Customer\_Address* and *Shopping\_Cart\_Line\_Item.* Note that:

* *Customer\_Address* and *Shopping\_Cart\_Line\_Item* belong to
  *Customer* and *Shopping\_Cart*, respectively, and have no
  independent existence of their own.
* In applications, *Customer* is almost always retrieved along with
  its addresses, and *Shopping\_Cart* is almost always retrieved along
  with its line items. The expense and complexity of these joins is
  incurred on almost every query.
* These two additional tables, and the attendant joins, were
  introduced solely to satisfy the relational model. **There was no
  application, domain, or user impetus for this decomposition.**

A good indicator that a table has no independent existence and was
introduced to satisfy the relational model is if the table would be
defined with cascading delete on a single parent table under
referential integrity. Such decomposition of constituent parts does
not model the intrinsic structure of data.

The relational model did not recognize composite objects, which are
ubiquitous in real-world data. The expense and complexity of joins was
the same for both independent and dependent relationships. And the
cost of object traversal and assembly was the same for both the
default traversal path and rarely used traversal paths.

Many relational systems eventually recognized these costs in the
relational model, and attempted to mitigate these costs by adding some
support for nested objects, multi-valued attributes, and other
features sometimes called "object-relational." But these additions
were outside the relational model, and the resulting combination
lacked the coherence and completeness of a data model designed from
inception to avoid these limitations. The next section presents that
data model.

### Couchbase data model and non-first normal form

The Couchbase data model is non-first normal form (N1NF) with
first-class nesting and domain-oriented normalization. As N1NF, the
Couchbase data model is also a proper superset and generalization of
the relational model. Let us examine each of its qualities.

#### Non-first normal form (N1NF)

Non-first normal form (N1NF) generalizes the two main constraints of
first-normal form (1NF).

* Attributes may contain tuples as values, i.e. attribute values are
  not required to be atomic. This is called nesting.
* Attributes may contain multiple values, i.e. attribute are not
  required to be single-valued.

These two qualities provide the ability to naturally model the
structure of real-world data and objects. Dependent and component
objects are modeled as nested tuples. Multi-valued attributes are
modeled directly.

Returning to our shopping cart example, we can now remove the
artificial decomposition and joins required by the relational
model. We can embed the *Customer\_Address* and
*Shopping\_Cart\_Line\_Item* data directly in the *Customer* and
*Shopping\_Cart* tables, respectively.

In the *Customer* table, we add an attribute *Addresses*. This is a
multi-valued attribute, and each of its values is a tuple with the
attributes from the *Customer\_Address* table: *Address\_Id,
Street\_Address, City, Zip,* and *State.*

In the *Shopping\_Cart* table, we add an attribute *Line\_Items*. This
is a multi-valued attribute, and each of its values is a tuple with
attributes from the *Shopping\_Cart\_Line\_Item* table, including
*Line\_Item\_Id, Product\_Id,* and *Quantity.*

Now, *Customer* and *Shopping\_Cart* objects can be retrieved with or
without addresses and line items, respectively, and never requiring
joins. The choice in retrieval is simply whether or not to include the
*Addresses* and *Line\_Items* attributes.

#### First-class nesting

In the Couchbase data model, nested tuples can be referenced and
queried in the same manner as top-level objects. We call this
first-class nesting. With first-class nesting, the Couchbase data
model combines the benefits of N1NF and 1NF.

As discussed, N1NF provides natural modeling of object structure and
avoids artificial decompositions and joins. In our shopping cart
example, this means embedding address and line items in the *Customer*
and *Shopping\_Cart* tables, respectively.

1NF does incur the costs of artificial decomposition and joins, but it
offers at least one benefit. It allows us to access dependent objects
directly, without reference to the corresponding parent objects. This
is a form of physical data independence. For example, if we needed to
analyze the geographical distribution of customer addresses, without
referencing customers, we could do so using only the
*Customer\_Address* table. Likewise, if we needed to analyze the
distribution of products in line items, we could do so using only the
*Shopping\_Cart\_Line\_Item* and *Product* tables.

With first-class nesting, the Couchbase data model allows us to
reference nested objects. We can reference and query the
*Customer.Addresses* and *Shopping\_Cart.Line\_Items* attributes in
the same manner as top-level objects. As such, we can directly perform
both computations enabled by 1NF above: analyzing the geographical
distribution of customer addresses, and analyzing the distribution of
products in line items.

At the same time, the benefits of N1NF are retainedY&mdash;we can
retrieve customers with their addresses and shopping carts with their
line items, all without any joins.

#### Domain-oriented normalization

Domain-oriented normalization is the normalization of data based only
on domain semantics for object independence, and not on any
constraints imposed by the data model. It separates natural,
beneficial normalization from artificial, detractive normalization.

Domain-oriented normalization can be used to achieve the same data
consistency, data de-duplication, and anomaly avoidance as the
relational normal forms.

In our shopping cart example, we have described how the Couchbase data
model allows customer addresses and shopping cart line items to be
embedded in their respective parent objects. This does not introduce
denormalization or data duplication, because parent information is not
repeated.

Furthermore, *Customer* and *Product* information is not embedded in
*Shopping\_Cart*, because these are independent objects in the
semantics of the domain. A *Customer* exists independently of the
shopping carts he or she maintains, and a *Product* exists
independently of the shopping carts that reference it.

A Couchbase application data model with domain-oriented normalization
is said to be in Domain Normal Form (or Business Normal Form).

### Logical artifacts

The Couchbase data model provides logical artifacts for constructing
specific data models and databases. These artifacts include documents,
buckets, and fragments.

#### Documents

Documents are top-level objects. Each row in an independent relational
table would map to a document in the Couchbase data model. In our
shopping cart example, every *Customer, Product, and Shopping\_Cart*
object would be a document.

Because Couchbase is also a key-value database, every document has a
unique primary key, which can be used to lookup and retrieve the
document.

#### Buckets

Buckets are sets of documents. Buckets are analogous to relational
tables, except that the documents in a given bucket are not required
to have the same attributes or structure.

Like tables, buckets are the basic unit of collection and
querying. Every document is contained in a single bucket, and every
data-accessing query references one or more buckets.

Buckets should be used to organize data into logical collections. In
our shopping cart example, we created three separate buckets, for
customers, products, and shopping carts, respectively. These three
types of objects are logically distinct, and there are no scenarios or
queries that would treat them as one.

#### Fragments

Fragments are nested values within documents. Trivially, a document
can be considered a "top-level" fragment.

Every attribute value in a document is a fragment. This includes both
atomic-valued and tuple-valued attributes, and both single-valued and
multi-valued attributes.

With first-class nesting, every attribute path, or a chain of
attribute traversals, is queryable. Hence, every fragment is
retrievable by query.

In our shopping cart example, every bucket or attribute path would
reference a set of fragments: *Customer, Customer.Name,
Customer.Addresses, Customer.Addresses.Zip, Product,
Product.UnitPrice, Shopping\_Cart, Shopping\_Cart.Customer\_Id,
Shopping\_Cart.Line\_Items, Shopping\_Cart.Line\_Items.Quantity,* etc.

### JSON

JSON, or JavaScript Object Notation, is the physical text-based
encoding used by Couchbase. Although it originated in a non-database
context, JSON turns out to be a notation for encoding N1NF data. It
provides objects, attributes, atomic values, NULL, nested objects
(tuples), and array-valued (multi-valued) attributes.

## Query model

Just as the Couchbase data model is a superset and generalization of
the relational data model, so too is the Couchbase query model a
superset and generalization of the relational query model embodied in
the SQL query language.

The relational query model works with tables, which are 2-dimensional
and can be visualized as rectangles. The Couchbase query model works
with both 2-dimensional data and nested objects of arbitrary depth,
which can be visualized as triangles. Thus the Couchbase query model
is said to work with both rectangles and triangles.

In relational systems, rows are units of direct access&mdash;once a
row is obtained, its individual attributes are directly accessible for
use in expressions, joins, projections, and more. In Couchbase,
documents are the analogous units of direct access&mdash;once a
document is obtained, its attributes, fragments, and metadata are
accessible for use in all aspects of query processing.

If relational queries express row-oriented processing, then Couchbase
queries express fragment-oriented processing. Documents can be
considered a special case of top-level fragments that provide direct
and optimized access to all their contained fragments. To highlight
this optimized in-document access, the Couchbase query model makes a
distinction between in-document joins (among fragments in the same
document) and cross-document joins (across documents).

The output of a relational query is a set of rows, just like a stored
relational table. This makes relational queries composable. The output
of a Couchbase query is a set of documents, just like a stored
Couchbase bucket. This makes Couchbase queries composable as well.

We now present the Couchbase query model by enumerating the stages of
the query processing pipeline. Not surprisingly, the stages are the
same as in relational queries; but within each stage, the capabilities
are expanded to mirror the generalization of the Couchbase data model.

### Pipeline

A query is a declarative specification of a selection, transformation,
and retrieval of data. The Couchbase query model provides the
following pipeline stages for expressing queries. Most of these stages
are optional; the projecting stage is always required, and the
sourcing stage is required for accessing stored data.

#### Sourcing

In the sourcing stage, a data source is constructed from one or more
terms, which are logical artifacts in the database. If a term is a
bucket, it refers to all the documents in that bucket. If the term is
an attribute path within a bucket, it refers to all the fragments
reachable by traversing that attribute path within each document in
that bucket.

Terms can be combined using in-document and cross-document joins. The
output of the sourcing stage is a single data source, be it a single
term or the result of one or more joins.

#### Filtering

In the filtering stage, objects from the data source are filtered
using a filter expression.

Couchbase queries provide additional expressions beyond the relational
ones&mdash;testing for missing attributes (in addition to NULL
testing); expressions involving arrays (multi-valued attributes);
object constructors; and more.

#### Grouping and aggregating

In the grouping and aggregating stage, input objects are consolidated
into groups based on zero or more attribute expressions (zero to group
all input objects together). A single output object is generated for
each group.

Within each group, aggregate expressions can be generated and output
over the input objects in the group. These aggregate expressions can
include counts, sums, statistical measures, and arbitrary combinations
of these.

Couchbase queries provide additional aggregate expressions for
constructing arrays from expressions on the input objects in each
group.

#### Group filtering

The group filtering stage has, as a prerequisite, the grouping and
aggregating stage. The group filtering stage receives the output of
the grouping and aggregating stage and filters those groups based on a
filter expression. The filter expression can involve the aggregate
expressions generated in the grouping and aggregating stage.

#### Projecting

In the projecting stage, result objects are defined. These result
objects can contain arbitrary expressions on the objects from the
preceding stage.

In addition to the attribute and expression projections of relational
queries, Couchbase queries provide projection of arbitrary nested
objects; construction of new nested objects of arbitrary shape; array
construction; metadata and raw-value expressions; and all the
additional expressions in Couchbase queries.

#### Ordering

In the ordering stage, result objects are sorted based on expressions
and predefined collations. A list of ordering expressions can be
specified, with each expression specified to sort in ascending or
descending order.

Without an ordering stage, Couchbase queries are not guaranteed to
return results in any particular order.

The ordering stage is essentially equivalent to the corresponding
relational stage, except for the availability of additional
expressions in Couchbase.

#### Paginating

Paginating is provided via two stages, skipping and limiting.
Together, they provide the ability to specify a "page" of results for
retrieval. Paginating is mostly useful in conjunction with ordering.

The skipping stage skips and discards a fixed number of result objects
from the beginning of the result list. The limiting stage limits the
number of result objects to a fixed maximum. Skipping is applied
before limiting.

The skipping and limiting stages are equivalent to the corresponding
relational stages.

## Query language

We now introduce the query language N1QL (pronounced "nickel") as the
first flavor and incarnation of the Couchbase query model. To state
the obvious, the name N1QL is an homage to N1NF. Given this paper's
focus on queries, only the query features of N1QL are highlighted
here, while data modification features are not included.

The salient features of N1QL queries include a SQL-like flavor; the
option to begin query statements with either SELECT or FROM; JSON
syntax for object constructors and expressions; attribute paths for
referencing fragments; a special syntax for in-document joins; and
additional expressions and functions from the Couchbase query model.

This section is not a complete or authoritative reference. The syntax
presented here is meant to illustrate the style and highlight some
non-relational features of the language.

### Statement format

The format of a N1QL query statement is:

    select-query-statement:

    SELECT select-list

    [ FROM from-term ]

    [ WHERE predicate ]

    [ GROUP BY expression, ... [ HAVING predicate, ... ] ]

    [ ORDER BY expression [ ASC | DESC ], ... ]

    [ LIMIT integer-constant ]

    [ OFFSET integer-constant ]
  
or, beginning with the FROM clause:

    from-query-statement:

    FROM from-term

    [ WHERE predicate ]

    [ GROUP BY expression, ... [ HAVING predicate, ... ] ]

    SELECT select-list

    [ ORDER BY expression [ ASC | DESC ], ... ]

    [ LIMIT integer-constant ]

    [ OFFSET integer-constant ]

The *select-list* constructs result objects from specific expressions
and wildcards.

    select-list:

    ( '*' | path '.' '*' | expression [ [ AS ] alias ] ), ...

In the next syntax box, we show the in-document version of
*from-term,* which is limited to a single bucket reference, and leave
the cross-document version to a future paper. The recursive use of
*from-term* means that OVER...IN clauses can be chained.

    from-term:

    path [ [ AS ] alias ]
    |
    from-term OVER name IN subpath

In *from-term, path* begins with a bucket reference optionally
followed by a chain of attribute names and literal array
subscripts. *Subpath* is a *path* that begins with the *alias, name,*
or trailing identifier of a preceding term.

### In-document joins

An in-document join creates new source objects from the cross-product
of its left- and right-hand side fragments. Each new source object
contains one nested value from the left-hand-side fragment (before
OVER) and one nested value from the right-hand-side fragment.

When the left-hand-side fragment is a parent or ancestor of the
right-hand-side fragment, the in-document join amounts to an unnesting
of the right-hand-side fragment.

Examples.

### Non-relational expressions

We now highlight some non-relational expressions in N1QL.

#### IS [ NOT ] MISSING

In the Couchbase data model, documents in a bucket are not required to
have the same set of attributes. The IS [ NOT ] MISSING operator is
provided to test whether an expressing (typically attribute or
attribute path) is present in the processing context.

Examples.

#### Path expressions

Nested values of arbitrary depth can be referenced directly as
expressions, separately from the use of attribute paths in the FROM
clause.

Examples.

#### Arrays

Array concatenation (same syntax as strings).

    expression || expression

Examples.

Array length (same syntax as strings).

    LENGTH(expression)

Examples.

Array subscripting.

    expression[expression]

Examples.

#### Collection expressions

To leverage the multi-valued attributes of the Couchbase N1NF data
model, a special set of collection expressions are provided. In the
boxes below, collection is an array-valued subpath or expression.

The existential quantifier over arrays tests whether any array element
matches a predicate.

    ANY predicate OVER name IN collection END

Examples.

The universal quantifier over arrays tests whether all array elements
match a predicate.

    ALL predicate OVER name IN collection END

Examples.

The selector over arrays returns a single expression using an array
and optional predicate.

    FIRST expression OVER name IN collection [ WHEN predicate ] END

Examples.

The mapper over arrays constructs a new expression array using an
array and optional predicate.

    ARRAY expression OVER name IN collection [ WHEN predicate ] END

Examples.

#### Object expressions

A powerful feature of N1QL is the ability to construct arbitrary
object expressions. N1QL uses JSON syntax for this purpose. These
object expressions can be used in any expression context in N1QL, but
their biggest impact is when projected in the SELECT clause to
transform result objects into arbitrary shapes as needed.

Array constructors.

    [ expression, ... ]

Examples.

Object constructors.

    { name : expression, ... }

Examples.

#### Functions

* META(): Returns metadata of the current document, including primary key

* VALUE(): Returns raw value of the current context

* BASE64_VALUE(): Returns nase64 enconding of raw value; this is
  useful for non-JSON values

* ARRAY_AGG(): An aggreagate function that constructs an array from an
  expression over the input objects to each group

## Conclusion

This paper has previewed the Couchbase query model.

Beginning with the data model, we reviewed the relational model and
its normalization principles. We reviewed non-first normal form (N1NF)
as a superset and generalization of the relational model. We presented
the Couchbase data model as N1NF with first-class nesting and
domain-oriented normalization. We discussed domain-oriented
normalization as preserving beneficial, intrinsic normalization while
discarding detractive, artificial normalization. We noted the logical
artifacts and the JSON encoding of the Couchbase data model.

We presented the Couchbase query model by analogy to the relational
query model, and by enumerating and describing the stages of the
Couchbase query processing pipeline.

Finally, we introduced the query language N1QL as the first flavor and
incarnation of the Couchbase query model. We diagrammed the query
statement format, and highlighted some non-relational features of the
language.

## About this document

### Document history

* 2013-08-19 - Initial version
* 2013-08-20 - Removed array slicing

### Open issues

This meta-section records open issues in this document, and will
eventually disappear.
