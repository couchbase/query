# N1QL Authentication

* Status: DRAFT
* Latest: [n1ql-authentication](https://github.com/couchbaselabs/query/blob/master/docs/n1ql-authentication.md)
* Modified: 2014-05-22

## Introduction

This document specifies how N1QL handles authentication, and more
specifically, how the N1QL query engine handles authentication.

N1QL is an abstract specification for accessing and querying a
document database. In that sense, it is analogous to SQL as an
abstract specification for accessing and querying a relational
database.

Because databases are intended to manage protected data and resources,
both N1QL and SQL need to support authentication and
authorization. The SQL approaches to authentication and authorization
are well established and highly successful. For N1QL, we intend to
learn from the approaches that have been successful in SQL and other
pervasive systems; we do not intend to incur limitations that have
already been avoided in earlier systems that are widely successful.

This document focuses on authentication. Authorization will be
discussed briefly, but a full N1QL authorization model will be
deferred beyond the N1QL GA 1.0 release.

## The successful authentication model

The authentication model that has been successful in SQL and other
systems is straightforward. This model is based on a few principles:

* Authentication involves an identity (e.g. username) and secret
  (e.g. password).

* The identity is not conflated with orthogonal concerns,
  e.g. resources and data.

The non-conflation accomplishes important things:

* A resource or unit of data can be shared across identities.

* An identity can access as many resources and data as desired per the
  functional requirements and security policies of the system.

* Rich authorization models can be provided in the system as mappings
  between identities and resources. Obviously, the very concept of
  authorization becomes meaningless if identities are conflated with
  resources.

This successful authentication model is supported by a large ecosystem
of standardized interfaces and APIs, e.g. JDBC, ODBC, HTTP Basic
Authentication, LDAP, and many others. Even more sophisticated
authentication systems like Kerberos are based on identities and
secrets.

## N1QL in context

Before presenting the N1QL authentication model, it is useful to
summarize the context in which a N1QL implementation operates.

N1QL sits between multiple application and end-user clients on the one
hand, and a back-end document database on the other. As an abstract
specification, N1QL supports both a variety of clients and a variety
of back ends.

This has the following implications:

* Authentication must originate with the application or end-user
  client.

* Because the back end manages data and resources that may be
  protected, authentication must pass through N1QL on to the back end.

## N1QL authentication model

The N1QL authentication model builds upon the successful
authentication model. For back ends that implement the successful
model, N1QL simply accepts an identity and a secret from each client,
and uses those to authenticate the client against the back end. For
purposes of this document, that specification is sufficient for such
back ends.

In the case of Couchbase server, the back end currently uses a
non-standard authentication model, which comes in two parts:

* A special "Administrator" credential has access to all other
  credentials.

* Buckets can be designated "SASL" buckets and configured with a
  password, which is then required for accessing data in the bucket.

When accessing Couchbase server as a back end, N1QL will map Couchbase
credentials onto the standard authentication model as follows, and
N1QL will perform the necessary logic to use the credential to access
data in Couchbase. Each type of Couchbase credential will be prefixed
with its type in order to simulate an identity:

* "Administrator" will have the identity "admin:Administrator", which
  will authenticate using the Administrator password.

* A SASL bucket will have the identity "bucket:bucket-name", where
  bucket-name is the name of the bucket. It will authenticate using
  the bucket's password.

If Couchbase adds other types of authentication in the future
(e.g. users), they will also be prefixed with their type when
authenticating through N1QL.

This enables N1QL to support current and future authentication types
in Couchbase, while supporting other back ends and clients using the
standard successful model.

N1QL will also support unauthenticated users by not authenticating to
the back end.

## Authorization

N1QL will not make authorization decisions; instead, it will perform
authentication and defer to the back end to either allow or reject the
requested access.

In the case of Couchbase Administrator, N1QL will use the
Administrator's credential to obtain bucket passwords. This is
consistent with the Administrator's current capabilities in Couchbase.

## About this Document

### Document History

* 2014-05-22 - Initial version

### Open Issues

This meta-section records open issues in this document, and will
eventually disappear.
