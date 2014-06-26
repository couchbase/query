%{
package n1ql

import "github.com/couchbaselabs/clog"

func logDebugGrammar(format string, v ...interface{}) {
    clog.To("PARSER", format, v...)
}
%}

%union {
s string
n int
f float64
}

%token ALL
%token ALTER
%token AND
%token ANY
%token ARRAY
%token AS
%token ASC
%token BETWEEN
%token BUCKET
%token BY
%token CASE
%token CAST
%token COLLATE
%token CREATE
%token DATABASE
%token DELETE
%token DESC
%token DISTINCT
%token DROP
%token EACH
%token EXCEPT
%token EXISTS
%token ELSE
%token END
%token EVERY
%token EXISTS
%token EXPLAIN
%token FALSE
%token FIRST
%token FOR
%token FROM
%token GROUP
%token HAVING
%token IF
%token IN
%token INDEX
%token INLINE
%token INNER
%token INSERT
%token INTERSECT
%token INTO
%token IS
%token JOIN
%token KEY
%token KEYS
%token LEFT
%token LET
%token LETTING
%token LIKE
%token LIMIT
%token MATCHED
%token MERGE
%token MISSING
%token NEST
%token NOT
%token NULL
%token OFFSET
%token ON
%token OR
%token ORDER
%token OUTER
%token OVER
%token PARTITION
%token PATH
%token POOL
%token PRIMARY
%token RAW
%token RENAME
%token RETURNING
%token RIGHT
%token SATISFIES
%token SET
%token SOME
%token SELECT
%token THEN
%token TO
%token TRUE
%token UNDER
%token UNION
%token UNIQUE
%token UNNEST
%token UNSET
%token UPDATE
%token UPSERT
%token USING
%token VALUED
%token VALUES
%token VIEW
%token WHERE
%token WHEN
%token WITH
%token XOR

%token INT NUMBER IDENTIFIER STRING
%token LPAREN RPAREN
%token LBRACE RBRACE LBRACKET RBRACKET
%token COMMA COLON

/* Precedence: lowest to highest */
%left           UNION EXCEPT
%left           INTERSECT
%left           JOIN NEST UNNEST INNER LEFT
%left           OR
%left           AND
%right          NOT
%nonassoc       EQ DEQ NE
%nonassoc       LT GT LTE GTE
%nonassoc       LIKE
%nonassoc       BETWEEN
%nonassoc       IN
%nonassoc       EXISTS
%nonassoc       IS                              /* IS NULL, IS MISSING, IS VALUED, IS NOT NULL, etc. */
%left           CONCAT
%left           PLUS MINUS
%left           STAR DIV MOD

/* Unary operators */
%right          UMINUS
%left           DOT LBRACKET RBRACKET

/* Override precedence */
%left           LPAREN RPAREN

%start input

%%

input:
explain
|
stmt
;

explain:
EXPLAIN stmt
;

stmt:
select_stmt
|
dml_stmt
|
ddl_stmt
;

select_stmt:
select
;

dml_stmt:
insert
|
upsert
|
delete
|
update
|
merge
;

ddl_stmt:
index_stmt
;

index_stmt:
create_index
|
drop_index
|
alter_index
;

select:
subselects
optional_order_by
optional_limit
optional_offset
;

subselects:
subselect
|
subselects UNION subselect
|
subselects UNION ALL subselect
;

subselect:
select_from
|
from_select
;

select_from:
select_clause
optional_from
optional_let
optional_where
optional_group_by
;

from_select:
from
optional_let
optional_where
optional_group_by
select_clause
;


/*************************************************
 *
 * SELECT clause
 *
 *************************************************/

select_clause:
SELECT
projection
;

projection:
projects
|
DISTINCT projects
|
ALL projects
|
RAW expr
;

projects:
project
|
projects COMMA project
;

project:
STAR
|
expr DOT STAR
|
expr optional_as_alias
;

optional_as_alias:
/* empty */
|
as_alias
;

as_alias:
alias
|
AS alias
;

alias:
IDENTIFIER
;


/*************************************************
 *
 * FROM clause
 *
 *************************************************/

optional_from:
/* empty */
|
from
;

from:
FROM from_term
;

from_term:
from_path optional_as_alias optional_keys
|
from_term optional_join_type joiner
;

from_path:
IDENTIFIER
optional_from_subpath
;

optional_from_subpath:
/* empty */
|
COLON path
|
DOT path
;

optional_keys:
/* empty */
|
keys
;

keys:
KEYS expr
;

optional_join_type:
/* empty */
|
INNER
|
LEFT optional_outer
;

optional_outer:
/* empty */
|
OUTER
;

joiner:
JOIN from_path optional_as_alias optional_keys
|
NEST from_path optional_as_alias optional_keys
|
UNNEST from_path optional_as_alias
;


/*************************************************
 *
 * LET clause
 *
 *************************************************/

optional_let:
/* empty */
|
let
;

let:
LET bindings
;

bindings:
binding
|
bindings COMMA binding
;

binding:
alias EQ expr
;


/*************************************************
 *
 * WHERE clause
 *
 *************************************************/

optional_where:
/* empty */
|
where
;

where:
WHERE expr
;


/*************************************************
 *
 * GROUP BY clause
 *
 *************************************************/

optional_group_by:
/* empty */
|
group_by
;

group_by:
GROUP BY exprs optional_letting optional_having
;

exprs:
expr
|
exprs COMMA expr
;

optional_letting:
/* empty */
|
letting
;

letting:
LETTING bindings
;

optional_having:
/* empty */
|
having
;

having:
HAVING expr
;


/*************************************************
 *
 * ORDER BY clause
 *
 *************************************************/

optional_order_by:
/* empty */
|
order_by
;

order_by:
ORDER BY order_terms
;

order_terms:
order_term
|
order_terms COMMA order_term
;

order_term:
expr optional_dir
;

optional_dir:
/* empty */
|
dir
;

dir:
ASC
|
DESC
;


/*************************************************
 *
 * LIMIT clause
 *
 *************************************************/

optional_limit:
/* empty */
|
limit
;

limit:
LIMIT expr
;


/*************************************************
 *
 * OFFSET clause
 *
 *************************************************/

optional_offset:
/* empty */
|
offset
;

offset:
OFFSET expr
;


/*************************************************
 *
 * INSERT
 *
 *************************************************/

insert:
INSERT INTO bucket_ref
optional_key
source
optional_returning
;

bucket_ref:
bucket_spec optional_as_alias
;

bucket_spec:
pool_or_bucket_name optional_scoped_name
;

pool_or_bucket_name:
IDENTIFIER
;

optional_scoped_name:
/* empty */
|
COLON bucket_name
;

bucket_name:
IDENTIFIER
;

optional_key:
/* empty */
|
key
;

key:
KEY expr
;

source:
values
|
select
;

values:
VALUES exprs
;

optional_returning:
/* empty */
|
returning
;

returning:
RETURNING returns
;

returns:
projects
|
RAW expr
;


/*************************************************
 *
 * UPSERT
 *
 *************************************************/

upsert:
UPSERT INTO bucket_ref
optional_key
source
optional_returning
;


/*************************************************
 *
 * DELETE
 *
 *************************************************/

delete:
DELETE FROM bucket_ref optional_keys
optional_where
optional_limit
optional_returning
;


/*************************************************
 *
 * UPDATE
 *
 *************************************************/

update:
UPDATE bucket_ref optional_keys
set_unset
optional_where
optional_limit
optional_returning
;

set_unset:
set optional_unset
|
unset
;

set:
SET set_paths
;

set_paths:
set_path
|
set_paths COMMA set_path
;

set_path:
path EQ expr optional_update_for
;

optional_update_for:
/* empty */
|
update_for
;

update_for:
FOR update_bindings optional_when END
;

update_bindings:
update_binding
|
update_bindings COMMA update_binding
;

update_binding:
variable IN path
;

variable:
IDENTIFIER
;

optional_when:
/* empty */
|
WHEN expr
;

optional_unset:
/* empty */
|
unset
;

unset:
UNSET unset_paths
;

unset_paths:
unset_path
|
unset_paths COMMA unset_path
;

unset_path:
path optional_update_for
;


/*************************************************
 *
 * MERGE
 *
 *************************************************/

merge:
MERGE INTO bucket_ref
USING merge_source ON key
WHEN merge_actions
optional_limit
optional_returning
;

merge_source:
from_term
|
LPAREN select RPAREN as_alias
;

merge_actions:
MATCHED THEN merge_update_delete_insert
|
NOT MATCHED THEN merge_insert
;

merge_update_delete_insert:
merge_update
optional_merge_delete_insert
|
merge_delete
optional_merge_insert
;

optional_merge_delete_insert:
/* empty */
|
WHEN merge_delete_insert
;

merge_delete_insert:
MATCHED THEN merge_delete
optional_merge_insert
|
NOT MATCHED THEN merge_insert
;

optional_merge_insert:
/* empty */
|
WHEN NOT MATCHED THEN merge_insert
;

merge_update:
UPDATE
set_unset
optional_where
;

merge_delete:
DELETE
optional_where
;

merge_insert:
INSERT expr
optional_where
;


/*************************************************
 *
 * CREATE INDEX
 *
 *************************************************/

create_index:
CREATE INDEX index_name
ON bucket_spec LPAREN exprs RPAREN
optional_partition
optional_using
;

index_name:
IDENTIFIER
;

optional_partition:
/* empty */
|
partition
;

partition:
PARTITION BY exprs
;

optional_using:
/* empty */
|
using
;

using:
USING VIEW
;


/*************************************************
 *
 * DROP INDEX
 *
 *************************************************/

drop_index:
DROP INDEX bucket_spec DOT index_name
;


/*************************************************
 *
 * ALTER INDEX
 *
 *************************************************/

alter_index:
ALTER INDEX bucket_spec DOT index_name RENAME TO index_name
;


/*************************************************
 *
 * Path
 *
 *************************************************/

path:
IDENTIFIER
|
path DOT IDENTIFIER
|
path DOT LPAREN expr RPAREN
|
path LBRACKET expr RBRACKET
;


/*************************************************
 *
 * Expression
 *
 *************************************************/

expr:
c_expr
|
/* Logical */
expr AND expr
|
expr OR expr
|
/* Comparison */
expr EQ expr
|
expr DEQ expr
|
expr NE expr
|
expr LT expr
|
expr GT expr
|
expr LTE expr
|
expr GTE expr
|
expr LIKE expr
|
expr NOT LIKE expr
|
expr is
|
/* Arithmetic */
expr PLUS expr
|
expr MINUS expr
|
expr STAR expr
|
expr DIV expr
|
expr MOD expr
|
/* Concat */
expr CONCAT expr
|
/* In */
expr IN expr
|
expr NOT IN expr
;

c_expr:
/* Literal */
literal
|
/* Identifier */
IDENTIFIER
|
/* Function */
function_expr
|
/* Case */
case_expr
|
/* Collection */
collection_expr
|
/* Grouping and subquery */
group_or_subquery_expr
|
/* Prefix */
prefix_expr
|
/* Nested */
expr DOT IDENTIFIER
|
expr DOT LPAREN expr RPAREN
|
expr LBRACKET expr RBRACKET
|
expr LBRACKET expr COLON RBRACKET
|
expr LBRACKET expr COLON expr RBRACKET
;


/*************************************************
 *
 * Literal
 *
 *************************************************/

literal:
NULL
|
FALSE
|
TRUE
|
NUMBER
|
STRING
|
object
|
array
;

object:
LBRACE optional_members RBRACE
;

optional_members:
/* empty */
|
members
;

members:
pair
|
members COMMA pair
;

pair:
STRING COMMA expr
;

array:
LBRACKET optional_exprs RBRACKET
;

optional_exprs:
/* empty */
|
exprs
;


/*************************************************
 *
 * Case
 *
 *************************************************/

case_expr:
CASE simple_or_searched_case optional_else END
;

simple_or_searched_case:
simple_case
|
searched_case
;

simple_case:
expr when_thens
;

when_thens:
WHEN expr THEN expr
|
when_thens WHEN expr THEN expr
;

searched_case:
when_thens
;

optional_else:
/* empty */
|
ELSE expr
;


/*************************************************
 *
 * Prefix
 *
 *************************************************/

prefix_expr:
MINUS expr %prec UMINUS
|
NOT expr
|
EXISTS expr
;

optional_not:
/* empty */
|
NOT
;

/*
between:
BETWEEN b_expr AND b_expr
;
*/

is:
IS optional_not null_missing_valued
;

null_missing_valued:
NULL
|
MISSING
|
VALUED
;

/*************************************************
 *
 * Function
 *
 *************************************************/

function_expr:
function_name LPAREN RPAREN
|
function_name LPAREN exprs RPAREN
|
function_name LPAREN DISTINCT exprs RPAREN
|
function_name LPAREN STAR RPAREN
;

function_name:
IDENTIFIER
;


/*************************************************
 *
 * Collection
 *
 *************************************************/

collection_expr:
collection_cond
|
collection_xform
;

collection_cond:
ANY coll_cond
|
SOME coll_cond
|
EVERY coll_cond
;

coll_cond:
coll_bindings optional_satisfies END
;

coll_bindings:
coll_binding
|
coll_bindings COMMA coll_binding
;

coll_binding:
variable IN expr
;

optional_satisfies:
/* empty */
|
SATISFIES expr
;

collection_xform:
ARRAY coll_xform
|
FIRST coll_xform
;

coll_xform:
expr FOR coll_bindings optional_when END
;


/*************************************************
 *
 * Grouping and subquery
 *
 *************************************************/

group_or_subquery_expr:
LPAREN group_or_subquery RPAREN
;

group_or_subquery:
expr
|
select
;
