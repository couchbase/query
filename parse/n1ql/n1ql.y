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

input :
explain
|
stmt
;

explain :
EXPLAIN stmt
;

stmt :
select_stmt
|
dml_stmt
|
ddl_stmt
;

select_stmt :
select
;

dml_stmt :
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

ddl_stmt :
index_stmt
;

index_stmt :
create_index
|
drop_index
|
alter_index
;

select :
subselects
optional_order_by
optional_limit
optional_offset
;

subselects :
subselect
|
subselects UNION subselect
|
subselects UNION ALL subselect
;

subselect :
select_from
|
from_select
;

select_from :
select_clause
optional_from
optional_let
optional_where
optional_group_by
;

from_select :
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

select_clause :
SELECT
projection
;

projection :
projects
|
DISTINCT projects
|
ALL projects
|
RAW expr
;

projects :
project
|
projects COMMA project
;

project :
STAR
|
path DOT STAR
|
expr optional_as_alias
;

optional_as_alias :
/* empty */
|
as_alias
;

as_alias :
alias
|
AS alias
;

alias :
IDENTIFIER
;


/*************************************************
 *
 * FROM clause
 *
 *************************************************/

optional_from :
/* empty */
|
from
;

from :
FROM from_term
;

from_term :
from_path optional_as_alias optional_keys
|
from_term optional_join_type joiner
;

optional_keys :
/* empty */
|
keys
;

keys :
KEYS expr
;

optional_join_type :
/* empty */
|
INNER
|
LEFT optional_outer
;

optional_outer :
/* empty */
|
OUTER
;

joiner :
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

optional_let :
/* empty */
|
let
;

let :
LET bindings
;

bindings :
binding
|
bindings COMMA binding
;

binding :
alias EQ expr
;

/*************************************************
 *
 * WHERE clause
 *
 *************************************************/

optional_where :
/* empty */
|
where
;

where :
WHERE expr
;


/*************************************************
 *
 * GROUP BY clause
 *
 *************************************************/

optional_group_by :
/* empty */
|
group_by
;

group_by :
GROUP BY exprs optional_letting optional_having
;

exprs :
expr
|
exprs COMMA expr
;

optional_letting :
/* empty */
|
letting
;

letting :
LETTING bindings
;

optional_having :
/* empty */
|
having
;

having :
HAVING expr
;

/*************************************************
 *
 * ORDER BY clause
 *
 *************************************************/

optional_order_by :
/* empty */
|
order_by
;

order_by :
ORDER BY order_terms
;

order_terms :
order_term
|
order_terms COMMA order_term
;

order_term :
expr optional_dir
;

optional_dir :
/* empty */
|
dir
;

dir :
ASC
|
DESC
;

/*************************************************
 *
 * LIMIT clause
 *
 *************************************************/

optional_limit :
/* empty */
|
limit
;

limit :
LIMIT expr
;

/*************************************************
 *
 * OFFSET clause
 *
 *************************************************/

optional_offset :
/* empty */
|
offset
;

offset :
OFFSET expr
;
