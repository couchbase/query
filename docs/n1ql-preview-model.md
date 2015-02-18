# N1QL from Couchbase&mdash;A Query Model Preview

by Gerald Sangudi, Marty Schoch, Steve Yen, and Sriram Melkote

* Status: DRAFT
* Latest: [n1ql-preview-model](https://github.com/couchbase/query/blob/master/docs/n1ql-preview-model.md)
* Modified: 2013-09-04

## Introduction

Couchbase is the leading next-generation, high-performance, scale-out,
document-oriented database. N1QL (pronounced "nickel") is a new query
language from Couchbase. N1QL aims to meet the query needs of
Couchbase users and applications at scale, and to offer a sound
abstraction for next-generation databases and their users.

This paper previews the N1QL query model, which underlies the query
language. It is written for readers who are familiar with database
concepts. This paper is a conceptual overview, and not a tutorial,
reference manual, or explication of syntax. It presents the overall
N1QL approach, and focuses on the subset of features available in a
preview release from Couchbase.

The following sections discuss N1QL in terms of the data model; the
query model; and the query language, which is a flavor and incarnation
of the query model.

## Data model

The data model, or how data is organized, is a defining characteristic
of any database. Couchbase, and any N1QL database, is both a key-value
and document-oriented database. Every data value is associated with an
immutable primary key. For data values that have structure and are not
opaque, those values are encoded as documents (JSON or equivalent),
with each document mapped to a primary key.

We start with the conceptual basis of the data model. In database
formalism, the N1QL data model is based on Non-First Normal Form, or
N1NF (hence the name N1QL). This model is a superset and
generalization of the relational model. The relational model requires
data normalization to First Normal Form (1NF) and advocates further
normalization to Third Normal Form (3NF). We examine the relational
model and its normalization principles, and then proceeed to the N1QL
data model.

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

Next, we discuss the principal relational normal forms. This
discussion serves as background for understanding the differences
between N1QL and the relational model. You may skim this discussion
and continue to the benefits and costs of the relational model, or you
may read it as an explanation or refresher.

#### First normal form

**First normal form (1NF)** requires that each attribute of every row
contain only a single, atomic value. An attribute cannot contain a
nested tuple, because a tuple is not atomic (it contains
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

* Store multi-valued data in multiple rows, creating a separate table
  if necessary; do not use multiple columns to store multi-valued data
* Identify each row with a unique primary key

#### Second normal form

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

**Second normal form (2NF)** requires that a table be in 1NF and not
contain any non-prime attribute that is dependent on a proper subset
of any candidate key.

Suppose that a customer can create only one shopping cart per instant
in time, so that *(Customer\_Id, Creation\_Time)* is a candidate key
for the *Shopping\_Cart* table. Now suppose that we stored
*Customer\_Birthdate* in the *Shopping\_Cart* table, in order to track
purchases by age group. Then *Shopping_Cart* would **not** be in 2NF,
because:

* *Customer\_Birthdate* is a non-prime attribute
* *Customer\_Birthdate* is dependent on *Customer\_Id*
* *Customer\_Id* is a proper subset of the candidate key
  *(Customer\_Id, Creation\_Time)*

If a customer has multiple shopping carts over time, and thus multiple
rows in *Shopping\_Cart*, it would be possible to modify
*Customer\_Birthdate* in one of those rows to be inconsistent with the
others. To satisfy 2NF, we would move *Customer\_Birthdate* from
*Shopping\_Cart* to the *Customer* table, where each customer would
have exactly one row.

Note that a 1NF table with no composite candidate keys is
automatically in 2NF; only composite candidate keys can have proper
subsets.

The practical rules for ensuring 2NF are:

* Ensure 1NF
* Remove subsets of attributes that repeat in multiple rows, and move
  them to a separate table
* Use foreign keys to connect related tables

#### Third normal form

**Superkeys** are used in defining third normal form. A superkey is
any set of attributes that forms a unique identifier for rows in a
table. Unlike a candidate key, a superkey does not need to be a
*minimal* set of attributes. Every candidate key is also a superkey.

**Third normal form (3NF)** requires that a table be in 2NF and that
every non-prime attribute be directly dependent on every superkey and
completely independent of every other non-key attribute. As Wikipedia
quotes Bill Kent: "Every non-key attribute must provide a fact about
the key, the whole key, and nothing but the key (so help me Codd)."

To obey 1NF, we created a *Customer\_Address* table and included the
attributes *Address\_Id*, *Customer\_Id*, *Street\_Address*, *City*,
*Zip*, and *State.* But this table contains a violation of 3NF. *Zip*
always determines *State*, and *Zip* and *State* are both non-primary
attributes. Therefore, the non-primary attribute *State* is dependent
on the non-key attribute *Zip*, which is a violation of 3NF. We would
need to create a separate *Zip* table with mappings from zip codes to
states.

The practical rules for ensuring 3NF are:

* Ensure 1NF and 2NF
* Remove attributes that are not directly and exclusively dependent on
  the primary key

#### Benefits of the relational model

The relational model and its normalization principles achieved several
benefits. Data duplication was minimized, which enhanced data
consistency and compactness of storage. Data updates could be granular
and authoritative. And a high degree of physical data independence was
ensured by the model: because data was normalized into discrete and
directly accessible tables, no single traversal path or object
composition was favored to the exclusion of others.

#### Costs of the relational model

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

Another cost of decomposition is the maintenance of referential
integrity. When a parent object is deleted, its child objects must be
deleted as well, so that no children are left orphaned. Referential
integrity is maintained using either foreign keys in the database or
logic in the application. Either way, it adds cost and complexity to
data modifications.

Fundamentally, the relational model did not distinguish composite
objects, which are ubiquitous in real-world data. The expense and
complexity of joins and referential integrity were the same for both
independent and dependent relationships. And the cost of object
traversal and assembly was the same for both the preponderant
traversal path and rarely used traversal paths.

Many relational systems eventually recognized these costs in the
relational model, and attempted to mitigate these costs by adding some
support for nested objects, multi-valued attributes, and other
features sometimes called "object-relational." But these additions
were outside the relational model, and the resulting combination
lacked the coherence and completeness of a data model designed from
inception to avoid these limitations. We now present such a data
model.

### N1QL data model and non-first normal form

The N!QL data model is non-first normal form (N1NF) with first-class
nesting and domain-oriented normalization. As N1NF, the N1QL data
model is also a proper superset and generalization of the relational
model. Let us examine each of its qualities.

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

In the *Customer* table, we add an attribute *Address*. This is a
multi-valued attribute, and each of its values is a tuple with the
attributes from the *Customer\_Address* table: *Address\_Id,
Street\_Address, City, Zip,* and *State.*

In the *Shopping\_Cart* table, we add an attribute *Line\_Item*. This
is a multi-valued attribute, and each of its values is a tuple with
attributes from the *Shopping\_Cart\_Line\_Item* table, including
*Line\_Item\_Id, Product\_Id,* and *Quantity.*

Now, *Customer* and *Shopping\_Cart* objects can be retrieved with or
without addresses and line items, respectively, and never requiring
joins. The choice in retrieval is simply whether or not to include the
*Address* and *Line\_Item* attributes.

#### First-class nesting

In the N1QL data model, nested tuples can be referenced and queried in
the same manner as top-level objects. We call this first-class
nesting. With first-class nesting, the N1QL data model combines the
benefits of N1NF and 1NF.

As discussed, N1NF provides natural modeling of object structure and
avoids artificial decompositions, joins, and referential integrity. In
our shopping cart example, this means embedding addresses and line
items in the *Customer* and *Shopping\_Cart* tables, respectively.

1NF does incur the costs of artificial decomposition, joins, and
referential integrity, but it offers at least one benefit. It allows
us to access dependent objects directly, without reference to the
corresponding parent objects. This is a form of physical data
independence. For example, if we needed to analyze the geographical
distribution of customer addresses, without referencing customers, we
could do so using only the *Customer\_Address* table. Likewise, if we
needed to analyze the distribution of products in line items, we could
do so using only the *Shopping\_Cart\_Line\_Item* and *Product*
tables.

With first-class nesting, the N1QL data model allows us to reference
nested objects. We can reference and query the *Customer.Address* and
*Shopping\_Cart.Line\_Item* attributes in the same manner as top-level
objects. As such, we can directly perform both computations enabled by
1NF above: analyzing the geographical distribution of customer
addresses, and analyzing the distribution of products in line items.

At the same time, the benefits of N1NF are retained&mdash;we can
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

In our shopping cart example, we have described how the N1QL data
model allows customer addresses and shopping cart line items to be
embedded in their respective parent objects. This does not introduce
denormalization or data duplication, because parent information is not
repeated.

Furthermore, *Customer* and *Product* information is not embedded in
*Shopping\_Cart*, because these are independent objects in the
semantics of the domain. A *Customer* exists independently of the
shopping carts he or she maintains, and a *Product* exists
independently of the shopping carts that reference it.

A N1QL application data model with domain-oriented normalization is
said to be in Domain Normal Form, or equivalently, in Business Normal
Form.

#### Structural flexibility

In addition to the benefits above, the N1QL data model supports
flexibility in document structure.

The underlying N1QL database may provide schema-less, open-schema, or
closed-schema document sets. As attributes are added to or removed
from a schema or specific documents, pre-existing data and queries
remain valid. This is a powerful benefit as applications and business
requirements evolve.

In our shopping cart example, we could add support for international
shipping by simply including additional attributes in those shipping
addresses: *Country*, *Postal\_Code.* Domestic addresses and queries
would be unaffected.

### Logical artifacts

The N1QL data model provides logical artifacts for constructing
specific data models and databases. These artifacts include documents,
fragments, buckets, and pools.

#### Documents

Documents are top-level objects. Each row in an independent relational
table would map to a document in the N1QL data model. In our shopping
cart example, every *Customer, Product, and Shopping\_Cart* object
would be a document.

Because Couchbase, and any N1QL database, is also a key-value
database, every document has a unique primary key, which can be used
to lookup and retrieve the document.

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
Customer.Address, Customer.Address.Zip, Product, Product.Unit\_Price,
Shopping\_Cart, Shopping\_Cart.Customer\_Id,
Shopping\_Cart.Line\_Item, Shopping\_Cart.Line\_Item.Quantity,* etc.

#### Buckets

Buckets are sets of documents. Buckets are analogous to relational
tables, except that N1QL does not require the documents in a given
bucket to have the same attributes or structure.

Like relational tables, buckets are the basic unit of collection and
querying. Every document is contained in a single bucket, and every
data-accessing query references one or more buckets.

Buckets should be used to organize data into logical collections. In
our shopping cart example, we created three separate buckets, for
customers, products, and shopping carts, respectively. These three
types of objects are logically distinct, and there are no use cases
that would treat them as one.

#### Pools

Pools are sets of buckets. Pools are analagous to a database or schema
in a relational database. Pools serve as namespaces, and can also
serve as units of data organization, access control, multi-tenancy,
and resource allocation.

## Query model

Just as the N1QL data model is a superset and generalization of the
relational data model, so too is the N1QL query model a superset and
generalization of the relational query model embodied in the SQL query
language.

The relational query model works with tables, which are 2-dimensional
and can be visualized as rectangles. The N1QL query model works with
both 2-dimensional data and nested objects of arbitrary depth, which
can be visualized as triangles. Thus the N1QL query model is said to
work with both rectangles and triangles.

In relational systems, rows are units of direct access&mdash;once a
row is obtained, its individual attributes are directly accessible for
use in expressions, joins, projections, and more. In N1QL, documents
are the analogous units of direct access&mdash;once a document is
obtained, its attributes, fragments, and metadata are accessible for
use in all aspects of query processing.

If relational queries express row-oriented processing, then N1QL
queries express fragment-oriented processing. Documents can be
considered a special case of top-level fragments that provide direct
and optimized access to all their contained fragments. To leverage
this optimized in-document access, the N1QL query model makes a
distinction between in-document joins (among fragments in the same
document) and cross-document joins (across documents).

The output of a relational query is a set of rows, just like a stored
relational table. This makes relational queries composable. The output
of a N1QL query is a set of documents, just like a stored N1QL
bucket. This makes N1QL queries composable as well.

We continue with the N1QL query model by enumerating the stages of the
query processing pipeline. Not surprisingly, the stages are the same
as in relational queries; but within each stage, the capabilities are
expanded to mirror the generalization of the N1QL data model.

### Pipeline

A query is a declarative specification of a selection, transformation,
and retrieval of data. The N1QL query model provides the following
pipeline stages for expressing queries. Most of these stages are
optional; the projecting stage is always required, and the sourcing
stage is required for accessing stored data.

#### Sourcing

In the sourcing stage, a data source is constructed from one or more
terms, which are logical artifacts in the database. If a term is a
bucket, it refers to all the documents in that bucket. If a term is an
attribute path within a bucket, it refers to all the fragments
reachable by traversing that attribute path within each document in
that bucket.

Terms can be combined using in-document and cross-document joins. The
output of the sourcing stage is a single data source, be it a single
term or the result of one or more joins.

#### Filtering

In the filtering stage, objects from the data source are filtered
using a filter expression.

N1QL queries provide additional expressions beyond the relational
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

N1QL queries provide additional aggregate expressions for constructing
arrays from expressions on the input objects in each group.

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

In addition to the projections of relational queries, N1QL queries
provide projection of arbitrary nested objects; construction of new
nested objects of arbitrary shape; array construction; metadata and
raw-value expressions; and all the additional expressions in N1QL.

#### De-duplicating

In the de-duplicating stage, duplicate result objects are removed, so
that each remaining result is unique.

#### Ordering

In the ordering stage, result objects are sorted based on expressions
and predefined collations. A list of ordering expressions can be
specified, with each expression specified to sort in ascending or
descending order.

Without an ordering stage, N1QL queries are not guaranteed to return
results in any particular order.

The ordering stage is essentially equivalent to the corresponding
relational stage, except for the availability of additional
expressions in N1QL.

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

The N1QL query language is a flavor and incarnation of the N1QL query
model. Given this paper's preview focus, only the preview features of
N1QL are highlighted here. In particular, cross-document joins,
subqueries, data modification, data definition, and compound
statements are omitted.

The salient features of N1QL queries include a SQL-like flavor; the
option to begin query statements with either SELECT or FROM; JSON
syntax for object and array expressions; attribute paths for
referencing fragments; a special syntax for in-document joins; and
additional expressions and functions from the N1QL query model.

This section is not a reference guide. The syntax presented here is
meant to illustrate the style and highlight some non-relational
features of the language.

### Statement format

The format of a N1QL query statement is:

    select-query-statement:

    SELECT [ DISTINCT ] select-list

    [ FROM from-term ]

    [ WHERE predicate ]

    [ GROUP BY expression, ... [ HAVING predicate ] ]

    [ ORDER BY ( expression [ ASC | DESC ] ), ... ]

    [ LIMIT integer-literal ]

    [ OFFSET integer-literal ]
  
Or, beginning with the FROM clause:

    from-query-statement:

    FROM from-term

    [ WHERE predicate ]

    [ GROUP BY expression, ... [ HAVING predicate ] ]

    SELECT [ DISTINCT ] select-list

    [ ORDER BY ( expression [ ASC | DESC ] ), ... ]

    [ LIMIT integer-literal ]

    [ OFFSET integer-literal ]

The *select-list* constructs result objects from specific expressions
and wildcards.

    select-list:

    ( '*' | path '.' '*' | expression [ [ AS ] alias ] ), ...

In the next syntax box, we only show the in-document version of
*from-term,* which is limited to a single bucket reference. The
recursive use of *from-term* means that OVER...IN clauses can be
chained.

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

As an example, suppose our *Customer* object has a *Rewards\_Number*
Attribute for handling various loyalty programs, including shipping
discounts. Our *Customer* bucket includes the following document:

    {
        name            : "William E. Coyote",
        rewards_number  : "ABC123XYZ",
        address         : [
                              {
                                  ship_to  : "Will Coyote",
                                  street   : "1 Universal Way",
                                  city     : "Wonderland",
                                  state    : "CA",
                                  zip      : 90210
                              },
                              {
                                  ship_to  : "Rod Runner",
                                  street   : "2 Water Ride",
                                  city     : "Magic",
                                  state    : "CA",
                                  zip      : 90211
                              }
                          ]
    }

We want to generate shipping labels for the customer. Each shipping
label contains a rewards number with the shipping address, so that the
correct shipping discount is applied. The query and results are:

    SELECT c.rewards_number, a.*
    FROM customer c OVER a IN c.address

    {
        rewards_number  : "ABC123XYZ",
        ship_to         : "Will Coyote",
        street          : "1 Universal Way",
        city            : "Wonderland",
        state           : "CA",
        zip             : 90210
    },
    {
        rewards_number  : "ABC123XYZ",
        ship_to         : "Rod Runner",
        street          : "2 Water Ride",
        city            : "Magic",
        state           : "CA",
        zip             : 90211
    }

### Non-relational expressions

Let us highlight some non-relational expressions in N1QL.

#### IS [ NOT ] MISSING

In the N1QL data model, documents in a bucket are not required to have
the same set of attributes. The IS MISSING and IS NOT MISSING
operators are provided to test whether an expression (typically
attribute or attribute path) is present in the processing context.

In N1QL, MISSING is not the same as NULL. A value that is explicitly
set to NULL is not MISSING.

Given the following data in the *Product* bucket:

    {
        sku             : "RURYRYR3T5",
        title           : "Comfy Recliner",
        fabric          : "Leather"
    },
    {
        sku             : "FHGHI5IG45",
        title           : "Wood Armchair",
        fabric          : null
    },
    {
        sku             : "O76OIU6IYO",
        title           : "Coffee Table",
        length_inches   : 48
    }

The following queries would produce the corresponding results.

    SELECT *
    FROM product
    WHERE fabric IS NOT MISSING

    {
        sku             : "RURYRYR3T5",
        title           : "Comfy Recliner",
        fabric          : "Leather"
    },
    {
        sku             : "FHGHI5IG45",
        title           : "Wood Armchair",
        fabric          : null
    }

And:

    SELECT *
    FROM product
    WHERE fabric IS MISSING

    {
        sku             : "O76OIU6IYO",
        title           : "Coffee Table",
        length_inches   : 48
    }

#### Arrays

Arrays can be subscripted to extract individual elements.

    expression[expression]

To extract customer name and last-positioned address:

    SELECT c.name, c.address[LENGTH(c.address) - 1] AS tail_address
    FROM customer c

    {
        name          : "William E. Coyote"
        tail_address  : {
                            ship_to  : "Rod Runner",
                            street   : "2 Water Ride",
                            city     : "Magic",
                            state    : "CA",
                            zip      : 90211
                        }
    }

In the FROM clause, any array subscripts must be constants. To extract
the first zip of every customer:

    SELECT zip
    FROM customer.address[0]

    {
        zip  : 90210
    }

#### Path expressions

Nested values of arbitrary depth can be referenced directly as
expressions. This is separate from the use of paths in the FROM
clause. To extract customer name and first-positioned zip code:

    SELECT c.name, c.address[0].zip
    FROM customer c

    {
        name  : "William E. Coyote",
        zip   : 90210
    }

#### Collection expressions

To leverage the multi-valued attributes of N1QL, a special set of
collection expressions is provided. In the syntax boxes below,
*collection* is a collection-valued subpath or expression.

The existential quantifier over collections tests whether any element
matches a predicate.

    ANY predicate OVER name IN collection END

Get customers who have any address in zip code 90210:

    SELECT name, rewards_number
    FROM customer c
    WHERE ANY a.zip = 90210 OVER a IN c.address END

    {
        name            : "William E. Coyote",
        rewards_number  : "ABC123XYZ"
    }

The universal quantifier over collections tests whether all elements
match a predicate.

    ALL predicate OVER name IN collection END

Get the names and address counts of customers who have no address
outside California:

    SELECT c.name, LENGTH(address) AS address_count
    FROM customer c
    WHERE ALL UPPER(a.state) = "CA" OVER a IN c.address END

    {
        name            : "William E. Coyote",
        address_count   : 2
    }

The selector over collections returns a single expression using a
collection and optional predicate.

    FIRST expression OVER name IN collection [ WHEN predicate ] END

Get customer name and any shipping address street in zip code 90211:

    SELECT name, FIRST UPPER(a.street) OVER a IN c.address WHEN a.zip = 90211 END AS street
    FROM customer c
    WHERE ANY a.zip = 90211 OVER a IN c.address END

    {
        name     : "William E. Coyote",
        street   : "2 WATER RIDE"
    }

The mapper over collections constructs a new expression array using a
collection and optional predicate. It is also called a comprehension.

    ARRAY expression OVER name IN collection [ WHEN predicate ] END

Get customer name and an array of all shipping address zip codes:

    SELECT name, ARRAY a.zip OVER a IN c.address END AS zips
    FROM customer c

    {
        name     : "William E. Coyote",
        zips     : [ 90210, 90211 ]
    }

#### Object expressions

A powerful feature of N1QL is the ability to construct arbitrary
object expressions. N1QL uses JSON syntax for this purpose. These
object expressions can be used in any expression context, but their
biggest impact is when projected in the SELECT clause to transform
result objects into arbitrary shapes as needed.

Arrays can be constructed over arbitrary expressions, including nested
arrays and objects.

    [ expression, ... ]

The following query extracts the rewards number as an array. This
could be useful if the client application or other data consumer
expects an array of rewards numbers per customer.

    SELECT name, [ rewards_number ] AS rewards_numbers
    FROM customer

    {
        name             : "William E. Coyote",
        rewards_numbers  : [ "ABC123XYZ" ]
    }

Objects can be constructed over arbitrary expressions, including
nested arrays and objects.

    { attribute : expression, ... }

To extract the customer name along with the first shipping label (this
time with the address further nested):

    SELECT name,
           {
               rewards_number : rewards_number,
               address        : address[0]
           } AS label
    FROM customer

    {
        name  : "William E. Coyote"
        label : {
                    rewards_number : "ABC123XYZ",
                    address        : {
                                         ship_to : "Will Coyote",
                                         street  : "1 Universal Way",
                                         city    : "Wonderland",
                                         state   : "CA",
                                         zip     : 90210
                                     }
                }
    }

#### Selected functions

* META(): Returns metadata of the current document, including its
  primary key

* VALUE(): Returns raw value of the current context

* BASE64_VALUE(): Returns base64 encoding of a raw value; this is
  useful for non-JSON values

* ARRAY_AGG(): Aggreagate function that constructs an array from an
  expression over the input objects to each group

## Conclusion

N1QL is a new query language from Couchbase. It aims to meet the query
needs of Couchbase users and applications at scale, and to offer a
sound abstraction for next-generation databases and their users. In
this paper, we have offered a preview of the N1QL query model.

We first discussed the N1QL data model. As background, we explored the
relational model and its principles, benefits and costs. We presented
the N1QL data model and identified its comparative benefits&mdash;
document-based access, intrinsic data modeling, first-class nesting,
domain-oriented normalization, and structural flexibility; in short,
preserving the benefits of the relational model while shedding its
model-induced costs.

We then described the N1QL query model by analogy to the
relational. The N1QL query model works with both rectangles and
triangles; is fragment-oriented; and is composable.  N1QL presents a
similar processing pipeline as relational, with additional
capabilities in various pipeline stages.

Finally, we previewed the N1QL query language and its syntax.  We
highlighted non-relational features and capabilities, including
in-document joins; array, path, collection, and object expressions;
and selected functions.

## About this document

### Document history

* 2013-08-19 - Initial version
* 2013-08-20 - Removed array slicing for now
* 2013-08-26 - Added END to collection expressions
* 2013-08-30 - Incorporated feedback, sans examples
* 2013-08-31 - Added remaining examples
* 2013-09-01 - Tweaks
    * Tweaked title and prose
    * Used singular names for nested attributes
* 2013-09-04 - Small edits
