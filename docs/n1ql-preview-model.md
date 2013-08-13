# The Couchbase Query Model&mdash;A Preview

* Status: DRAFT
* Latest: [n1ql-preview-model](https://github.com/couchbaselabs/query/blob/master/docs/n1ql-preview-model.md)
* Modified: 2013-08-12

## Introduction

Couchbase is the leading next-generation, high-performance, scale-out,
document-oriented database. The Couchbase query model aims to meet the
query needs of Couchbase users and applications at scale.

This paper previews the Couchbase approach to querying. It is written
for readers who are familiar with Couchbase and with database concepts
in general. This paper is a conceptual overview, and not a tutorial,
reference manual, or explication of syntax. Furthermore, it presents
the overall Couchbase approach, and not a specific feature set
associated with a product release or point in time.

The remaining sections discuss the Couchbase data model; the Couchbase
query model; Couchbase's new query language N1QL (pronounced
"nickel"), which is the first flavor and implementation of the
Couchbase query model; and query semantics at scale.

## Data model

The data model, or how data is organized, is a defining characteristic
of any database. Couchbase combines a key-value and document-oriented
database. Every data value in Couchbase is associated with an
immutable primary key. For data values that are documents or have
structure, those values are encoded as JSON documents, with each
document mapped to a primary key.

In this section, we start with the conceptual basis of the Couchbase
data model. In database formalism, the Couchbase data model is called
Non-First Normal Form, or N1NF. This model is a superset of the
relational data model, which requires data normalization to First
Normal Form (1NF) and advocates further normalization to Third Normal
Form (3NF). We summarize the relational model and its normalization
principles, and then proceeed to the Couchbase N1NF model and its
encoding in JSON documents.

### Relational model and normalization

In the relational model, data is organized into rows (tuples) and
tables (relations, or sets of tuples). A row has one or more
attributes, with each attribute containing a single atomic data
value. All the rows of a given table have the same attributes. If a
row does not have a value for a given attribute, that attribute is
assigned the special value NULL.

For illustration, we use an example data set from a shopping cart
application. The data set contains the following artifacts:

* *Customer* (identity, addresses, payment methods)
* *Product* (title, code, description, unit price)
* *Shopping Cart* (customer, shipping address, payment method, line items)

Let us review the principal relational normal forms.

* **First normal form (1NF)** requires that each attribute of every
  row contain only a single, atomic value. An attribute cannot contain
  a nested tuple, because a tuple is not atomic (it contains
  attributes). And an attribute cannot contain multiple atomic
  values. Examples of a valid attribute value would be a single
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

  The practical rules for ensuring 1NF are:

    * Store multi-valued data in multiple rows, creating a separate
      table if necessary; do not use multiple columns to store
      multi-valued data
    * Identify each row with a unique primary key

Intuitively, second and third normal form aim to prevent any piece of
information from being represented in more than one row in the
database. If the same piece of information is represented in more than
one row, it is possible to modify one row so that it becomes
inconsistent with the others; avoiding such inconsistenties is one of
the key goals of the relational model.

* **Candidate keys** are used in defining second and third normal
  form. A candidate key of a table is any minimal set of attributes
  whose values form a unique identifier for (rows in) the table.

* **Update anomalies** are the inconsistencies that arise when a table
  is not normalized, and one row with a piece of information is
  modified to be inconsistent with another row containing the same
  piece of information.

By definition, every attribute of every candidate key helps to
uniquely identify rows in a table. Such attributes, called prime
attributes, cannot cause inconsistencies&mdash;if the values of a
prime attribute are different in 2 different rows, then those 2 rows
represent separate pieces of information, because the prime attribute
is part of the unique identity of a row.

Second and third normal form are defined as restrictions on non-prime
attributes, which are those attributes that are not part of any
candidate key.

* **Second normal form (2NF)** requires that a table be in 1NF and not
  contain any non-prime attribute that is dependent on a proper subset
  of some candidate key.

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

* **Superkeys** are used in defining third normal form. A superkey is
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
and authoritative. And a high degree of physical indepedence was
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

Our example data set has three independent artifacts: *Customer,
Product, and Shopping Cart.* The relational model would force us to
introduce at least two additional tables: *Customer\_Address* and
*Shopping\_Cart\_Line\_Item.* Note that:

* *Customer\_Address* and *Shopping\_Cart\_Line\_Item* belong to
  *Customer* and *Shopping\_Cart*, respectively, and have no
  independent existence of their own.
* In applications, *Customer* is almost always loaded and displayed
  with its addresses, and *Shopping\_Cart* is almost always loaded and
  displayed with its line items. The expense and complexity of these
  joins is incurred on almost every query.
* These two additional tables, and the attendant joins, were
  introduced solely to satisfy the relational model. **There was no
  application, domain, or user impetus to perform this
  decomposition.**

A good indicator that a table has no independent existence and was
introduced to satisfy the relational model is if the table would be
defined with cascading delete on a single parent table if referential
integrity were in use. Such decomposition does not model the intrinsic
structure of the data.

The relational model did not recognize composite objects, which are
ubiquitous in real-world data. The expense and complexity of joins was
the same for both independent and dependent relationships. And the
cost of object traversal and assembly was the same for both the
intrinsic, preponderant traversal path, and alternate, occasional
traversal paths.

Many relational systems recognized these costs and their cause in the
relational model itself, and they attempted to mitigate this by adding
some support for nested objects, multi-valued attributes, and other
features sometimes called "object-relational." But these additions
were outside the relational model, and the resulting combination
lacked the coherence and completeness of a model designed from
inception to address these limitations. The next section presents that
model.

### The Couchbase model and non-first normal form

* generalized algebra / model (Garani)
* generalization / relaxation of relational model
* Business Normal Form, natural data modeling
* rectangles and triangles
* use cases
* good vs. bad normalization
* dependent vs. independent relationships
* schemaless to schemaful

### Documents and fragments

* documents and buckets (sets)
* document keys and key-value
* documents as special (top-level) fragments
* fragments as logical entities (need examples)

### JSON

* JSON as contemporary encoding of N1NF; cite others (XML, OODB, network / graph DB, see Garani)
* text and binary reprensentations possible
* user-friendly, compact, flexible, expressive, impedance match

## Query model

* generalization / relaxation of relational query semantics (SQL)
* single dataspace
* document boundaries as physical, not logical
* document as optimized access path
* fragments as first-class queryable objects; same as top-level documents
* fragment-oriented QL: paths, DML, vectors + scalars, etc.
* NULL vs. MISSING
* collections exprs: ANY / ALL / FIRST / comprehensions
* document JOINs: OVER
* cross-document JOINs (good vs. bad JOINs)

## Query language

* SQL-like flavor; others
* JSON expressions and return values
* paths
* FROM OVER
* ANY / ALL OVER
* DML upcoming
* fragment-oriented QL: paths, DML, vectors + scalars, etc.
* NULL vs. MISSING
* collections exprs: ANY / ALL / FIRST / comprehensions
* document JOINs: OVER
* cross-document JOINs (good vs. bad JOINs)

## Query semantics at scale

* fragment-oriented indexing
* scale-out, distribution, scatter-gather
* ACID semantics undergoing definition, design, tradeoffs
* deterministic vs. non-deterministic
    * persistence
    * consistency
* trade off failure rate vs. determinism

## Conclusion

## About this document

### Document history

* 2013-08-12 - Initial version

### Open issues

This meta-section records open issues in this document, and will
eventually disappear.
