%{
package n1ql

import "github.com/couchbaselabs/clog"
import "github.com/couchbaselabs/query/algebra"
import "github.com/couchbaselabs/query/expression"
import "github.com/couchbaselabs/query/value"

func logDebugGrammar(format string, v ...interface{}) {
    clog.To("PARSER", format, v...)
}
%}

%union {
s string
n int
f float64
b bool

expr        expression.Expression
exprs       expression.Expressions
whenTerm    *expression.WhenTerm
whenTerms   expression.WhenTerms
binding     *expression.Binding
bindings    expression.Bindings

node        algebra.Node
fullselect  *algebra.Select
subselect   *algebra.Subselect
fromTerm    algebra.FromTerm
bucketTerm  *algebra.BucketTerm
path        algebra.Path
groupBy     *algebra.GroupBy
resultTerm  algebra.ResultTerm
resultTerms algebra.ResultTerms
projection  *algebra.Projection
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
%token CLUSTER
%token COLLATE
%token CREATE
%token DATABASE
%token DATASET
%token DATATAG
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
%token REALM
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
%nonassoc       LT GT LE GE
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

/* Types */
%type <s>            STRING
%type <s>            IDENTIFIER
%type <f>            NUMBER
%type <n>            INT
%type <expr>         literal object array
%type <binding>      member
%type <bindings>     members optional_members

%type <expr>         expr c_expr b_expr
%type <exprs>        exprs optional_exprs
%type <binding>      binding
%type <bindings>     bindings

%type <s>            alias as_alias optional_as_alias variable

%type <expr>         case_expr simple_or_searched_case simple_case searched_case optional_else
%type <whenTerms>    when_thens

%type <expr>         collection_expr collection_cond collection_xform
%type <binding>      coll_binding
%type <bindings>     coll_bindings
%type <expr>         satisfies
%type <expr>         optional_when

%type <expr>         function_expr
%type <s>            function_name

%type <expr>         group_or_subquery_expr group_or_subquery

%type <fullselect>   fullselect
%type <subselect>    subselect
%type <subselect>    select_from
%type <subselect>    from_select
%type <fromTerm>     from_term from optional_from
%type <bucketTerm>   bucket_term
%type <b>            optional_join_type
%type <path>         path optional_subpath
%type <s>            pool_name bucket_name
%type <expr>         keys optional_keys
%type <bindings>     optional_let let
%type <expr>         optional_where where
%type <groupBy>      optional_group_by group_by
%type <bindings>     optional_letting letting
%type <expr>         optional_having having
%type <resultTerm>   project
%type <resultTerms>  projects
%type <projection>   projection select_clause
%type <sortTerm>     sort_term
%type <sortTerms>    sort_terms order_by optional_order_by
%type <expr>         limit optional_limit
%type <expr>         offset optional_offset
%type <b>            dir optional_dir

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
fullselect
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

fullselect:
subselects optional_order_by optional_limit optional_offset
{
  $$ = nil
}
;

subselects:
subselect
|
subselects UNION subselect
|
subselects UNION ALL subselect
;

subselect:
from_select
|
select_from
;

from_select:
from optional_let optional_where optional_group_by select_clause
{
  $$ = nil
}
;

select_from:
select_clause optional_from optional_let optional_where optional_group_by
{
  $$ = nil
}
;


/*************************************************
 *
 * SELECT clause
 *
 *************************************************/

select_clause:
SELECT
projection
{
  $$ = $2
}
;

projection:
projects
{
  $$ = algebra.NewProjection($1, false)
}
|
DISTINCT projects
{
  $$ = algebra.NewProjection($2, true)
}
|
ALL projects
{
  $$ = algebra.NewProjection($2, false)
}
|
RAW expr
{
  $$ = algebra.NewRawProjection($2, false)
}
|
DISTINCT RAW expr
{
  $$ = algebra.NewRawProjection($3, true)
}
;

projects:
project
{
  $$ = algebra.ResultTerms{$1}
}
|
projects COMMA project
{
  $$ = append($1, $3)
}
;

project:
STAR
{
  $$ = algebra.NewResultTerm(nil, true, "")
}
|
expr DOT STAR
{
  $$ = algebra.NewResultTerm($1, true, "")
}
|
expr optional_as_alias
{
  $$ = algebra.NewResultTerm($1, false, $2)
}
;

optional_as_alias:
/* empty */
{
  $$ = ""
}
|
as_alias
;

as_alias:
alias
|
AS alias
{
  $$ = $2
}
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
{
  $$ = nil
}
|
from
;

from:
FROM from_term
{
  $$ = $2
}
;

from_term:
bucket_term
{
  $$ = $1
}
|
from_term optional_join_type JOIN bucket_term
{
  $$ = algebra.NewJoin($1, $2, $4)
}
|
from_term optional_join_type NEST bucket_term
{
  $$ = algebra.NewNest($1, $2, $4)
}
|
from_term optional_join_type UNNEST path optional_as_alias
{
  $$ = algebra.NewUnnest($1, $2, $4, $5)
}
;

bucket_term:
pool_name COLON bucket_name optional_subpath optional_as_alias optional_keys
{
  $$ = algebra.NewBucketTerm($1, $3, $4, $5, $6)
}
|
bucket_name optional_subpath optional_as_alias optional_keys
{
  $$ = algebra.NewBucketTerm("". $1, $2, $3, $4)
}
;

pool_name:
IDENTIFIER
;

bucket_name:
IDENTIFIER
;

optional_subpath:
/* empty */
{
  $$ = nil
}
|
DOT path
{
  $$ = $2
}
;

optional_keys:
/* empty */
{
  $$ = nil
}
|
keys
;

keys:
KEYS expr
{
  $$ = $2
}
;

optional_join_type:
/* empty */
{
  $$ = false
}
|
INNER
{
  $$ = false
}
|
LEFT optional_outer
{
  $$ = true
}
;

optional_outer:
/* empty */
|
OUTER
;


/*************************************************
 *
 * LET clause
 *
 *************************************************/

optional_let:
/* empty */
{
  $$ = nil
}
|
let
;

let:
LET bindings
{
  $$ = $2
}
;

bindings:
binding
{
  $$ = algebra.Bindings{$1}
}
|
bindings COMMA binding
{
  $$ = append($1, $3)
}
;

binding:
alias EQ expr
{
  $$ = expression.NewBinding($1, $3)
}
;


/*************************************************
 *
 * WHERE clause
 *
 *************************************************/

optional_where:
/* empty */
{
  $$ = nil
}
|
where
;

where:
WHERE expr
{
  $$ = $2
}
;


/*************************************************
 *
 * GROUP BY clause
 *
 *************************************************/

optional_group_by:
/* empty */
{
  $$ = nil
}
|
group_by
;

group_by:
GROUP BY exprs optional_letting optional_having
{
  $$ = algebra.NewGroupBy($3, $4, $5)
}
;

exprs:
expr
{
  $$ = expression.Expressions{$1}
}
|
exprs COMMA expr
{
  $$ = append($1, $3)
}
;

optional_letting:
/* empty */
{
  $$ = nil
}
|
letting
;

letting:
LETTING bindings
{
  $$ = $2
}
;

optional_having:
/* empty */
{
  $$ = nil
}
|
having
;

having:
HAVING expr
{
  $$ = $2
}
;


/*************************************************
 *
 * ORDER BY clause
 *
 *************************************************/

optional_order_by:
/* empty */
{
  $$ = nil
}
|
order_by
;

order_by:
ORDER BY sort_terms
{
  $$ = $3
}
;

sort_terms:
sort_term
{
  $$ = algebra.SortTerms{$1}
}
|
sort_terms COMMA sort_term
{
  $$ = append($1, $3)
}
;

sort_term:
expr optional_dir
{
  $$ = algebra.NewSortTerm($1, $2)
}
;

optional_dir:
/* empty */
{
  $$ = false
}
|
dir
;

dir:
ASC
{
  $$ = false
}
|
DESC
{
  $$ = true
}
;


/*************************************************
 *
 * LIMIT clause
 *
 *************************************************/

optional_limit:
/* empty */
{
  $$ = nil
}
|
limit
;

limit:
LIMIT expr
{
  $$ = $2
}
;


/*************************************************
 *
 * OFFSET clause
 *
 *************************************************/

optional_offset:
/* empty */
{
  $$ = nil
}
|
offset
;

offset:
OFFSET expr
{
  $$ = $2
}
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
fullselect
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
{
  $$ = nil
}
|
WHEN expr
{
  $$ = $2
}
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
LPAREN fullselect RPAREN as_alias
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
{
  $$ = expression.NewIdentifier($1)
}
|
path DOT IDENTIFIER
{
  $$ = expression.NewField($1, expression.NewConstant(value.NewValue($3)))
}
|
path DOT LPAREN expr RPAREN
{
  $$ = expression.NewField($1, $4)
}
|
path LBRACKET expr RBRACKET
{
  $$ = expression.NewElement($1, $3)
}
;


/*************************************************
 *
 * Expression
 *
 *************************************************/

expr:
c_expr
|
/* Nested */
expr DOT IDENTIFIER
{
  $$ = expression.NewField($1, expression.NewConstant(value.NewValue($3)))
}
|
expr DOT LPAREN expr RPAREN
{
  $$ = expression.NewField($1, $4)
}
|
expr LBRACKET expr RBRACKET
{
  $$ = expression.NewElement($1, $3)
}
|
expr LBRACKET expr COLON RBRACKET
{
  $$ = expression.NewSlice($1, $3, nil)
}
|
expr LBRACKET expr COLON expr RBRACKET
{
  $$ = expression.NewSlice($1, $3, $5)
}
|
/* Arithmetic */
expr PLUS expr
{
  $$ = expression.NewAdd($1, $3)
}
|
expr MINUS expr
{
  $$ = expression.NewSubtract($1, $3)
}
|
expr STAR expr
{
  $$ = expression.NewMultiply($1, $3)
}
|
expr DIV expr
{
  $$ = expression.NewDivide($1, $3)
}
|
expr MOD expr
{
  $$ = expression.NewModulo($1, $3)
}
|
/* Concat */
expr CONCAT expr
{
  $$ = expression.NewConcat($1, $3)
}
|
/* Logical */
expr AND expr
{
  $$ = expression.NewAnd($1, $3)
}
|
expr OR expr
{
  $$ = expression.NewOr($1, $3)
}
|
NOT expr
{
  $$ = expression.NewNot($2)
}
|
/* Comparison */
expr EQ expr
{
  $$ = expression.NewEQ($1, $3)
}
|
expr DEQ expr
{
  $$ = expression.NewEQ($1, $3)
}
|
expr NE expr
{
  $$ = expression.NewNE($1, $3)
}
|
expr LT expr
{
  $$ = expression.NewLT($1, $3)
}
|
expr GT expr
{
  $$ = expression.NewGT($1, $3)
}
|
expr LE expr
{
  $$ = expression.NewLE($1, $3)
}
|
expr GE expr
{
  $$ = expression.NewGE($1, $3)
}
|
expr BETWEEN b_expr AND b_expr
{
  $$ = expression.NewBetween($1, $3, $5)
}
|
expr NOT BETWEEN b_expr AND b_expr
{
  $$ = expression.NewNotBetween($1, $4, $6)
}
|
expr LIKE expr
{
  $$ = expression.NewLike($1, $3)
}
|
expr NOT LIKE expr
{
  $$ = expression.NewNotLike($1, $4)
}
|
expr IN expr
{
  $$ = expression.NewIn($1, $3)
}
|
expr NOT IN expr
{
  $$ = expression.NewNotIn($1, $4)
}
|
expr IS NULL
{
  $$ = expression.NewIsNull($1)
}
|
expr IS NOT NULL
{
  $$ = expression.NewIsNotNull($1)
}
|
expr IS MISSING
{
  $$ = expression.NewIsMissing($1)
}
|
expr IS NOT MISSING
{
  $$ = expression.NewIsNotMissing($1)
}
|
expr IS VALUED
{
  $$ = expression.NewIsValued($1)
}
|
expr IS NOT VALUED
{
  $$ = expression.NewIsNotValued($1)
}
|
EXISTS expr
{
  $$ = expression.NewExists($2)
}
;

c_expr:
/* Literal */
literal
|
/* Identifier */
IDENTIFIER
{
  $$ = expression.NewIdentifier($1)
}
|
/* Function */
function_expr
|
/* Prefix */
MINUS expr %prec UMINUS
{
  $$ = expression.NewNegate($2)
}
|
/* Case */
case_expr
|
/* Collection */
collection_expr
|
/* Grouping and subquery */
group_or_subquery_expr
;

b_expr:
c_expr
|
/* Nested */
b_expr DOT IDENTIFIER
{
  $$ = expression.NewField($1, expression.NewConstant(value.NewValue($3)))
}
|
b_expr DOT LPAREN expr RPAREN
{
  $$ = expression.NewField($1, $4)
}
|
b_expr LBRACKET expr RBRACKET
{
  $$ = expression.NewElement($1, $3)
}
|
b_expr LBRACKET expr COLON RBRACKET
{
  $$ = expression.NewSlice($1, $3, nil)
}
|
b_expr LBRACKET expr COLON expr RBRACKET
{
  $$ = expression.NewSlice($1, $3, $5)
}
|
/* Arithmetic */
b_expr PLUS b_expr
{
  $$ = expression.NewAdd($1, $3)
}
|
b_expr MINUS b_expr
{
  $$ = expression.NewSubtract($1, $3)
}
|
b_expr STAR b_expr
{
  $$ = expression.NewMultiply($1, $3)
}
|
b_expr DIV b_expr
{
  $$ = expression.NewDivide($1, $3)
}
|
b_expr MOD b_expr
{
  $$ = expression.NewModulo($1, $3)
}
|
/* Concat */
b_expr CONCAT b_expr
{
  $$ = expression.NewConcat($1, $3)
}
;


/*************************************************
 *
 * Literal
 *
 *************************************************/

literal:
NULL
{
  $$ = expression.NULL_EXPR
}
|
FALSE
{
  $$ = expression.FALSE_EXPR
}
|
TRUE
{
  $$ = expression.TRUE_EXPR
}
|
NUMBER
{
  $$ = expression.NewConstant(value.NewValue($1))
}
|
INT
{
  $$ = expression.NewConstant(value.NewValue($1))
}
|
STRING
{
  $$ = expression.NewConstant(value.NewValue($1))
}
|
object
|
array
;

object:
LBRACE optional_members RBRACE
{
  $$ = expression.NewObjectLiteral($2)
}
;

optional_members:
/* empty */
{
  $$ = nil
}
|
members
;

members:
member
{
  $$ = expression.Bindings{$1}
}
|
members COMMA member
{
  $$ = append($1, $3)
}
;

member:
STRING COLON expr
{
  $$ = expression.NewBinding($1, $3)
}
;

array:
LBRACKET optional_exprs RBRACKET
{
  $$ = expression.NewArrayLiteral($2)
}
;

optional_exprs:
/* empty */
{
  $$ = nil
}
|
exprs
;


/*************************************************
 *
 * Case
 *
 *************************************************/

case_expr:
CASE simple_or_searched_case END
{
  $$ = $2
}
;

simple_or_searched_case:
simple_case
|
searched_case
;

simple_case:
expr when_thens optional_else
{
  $$ = expression.NewSimpleCase($1, $2, $3)
}
;

when_thens:
WHEN expr THEN expr
{
  $$ = expression.WhenTerms{&expression.WhenTerm{$2, $4}}
}
|
when_thens WHEN expr THEN expr
{
  $$ = append($1, &expression.WhenTerm{$3, $5})
}
;

searched_case:
when_thens
optional_else
{
  $$ = expression.NewSearchedCase($1, $2)
}
;

optional_else:
/* empty */
{
  $$ = nil
}
|
ELSE expr
{
  $$ = $2
}
;


/*************************************************
 *
 * Function
 *
 *************************************************/

function_expr:
function_name LPAREN optional_exprs RPAREN
{
  $$ = nil;
  agg, ok := algebra.GetAggregate($1, false);
  if ok {
    $$ = agg.Constructor()($3);
  } else {
    f, ok := expression.GetFunction($1);
    if ok {
      $$ = f.Constructor()($3)
    }
  }
}
|
function_name LPAREN DISTINCT exprs RPAREN
{
  $$ = nil;
  agg, ok := algebra.GetAggregate($1, true);
  if ok {
      $$ = agg.Constructor()($4)
  }
}
|
function_name LPAREN STAR RPAREN
{
  $$ = nil;
  agg, ok := algebra.GetAggregate($1, false);
  if ok {
      $$ = agg.Constructor()(nil)
  }
}
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
ANY coll_bindings satisfies END
{
  $$ = expression.NewAny($2, $3)
}
|
SOME coll_bindings satisfies END
{
  $$ = expression.NewAny($2, $3)
}
|
EVERY coll_bindings satisfies END
{
  $$ = expression.NewEvery($2, $3)
}
;

coll_bindings:
coll_binding
{
  $$ = expression.Bindings{$1}
}
|
coll_bindings COMMA coll_binding
{
  $$ = append($1, $3)
}
;

coll_binding:
variable IN expr
{
  $$ = expression.NewBinding($1, $3)
}
;

satisfies:
SATISFIES expr
{
  $$ = $2
}
;

collection_xform:
ARRAY expr FOR coll_bindings optional_when END
{
  $$ = expression.NewArray($2, $4, $5)
}
|
FIRST expr FOR coll_bindings optional_when END
{
  $$ = expression.NewFirst($2, $4, $5)
}
;


/*************************************************
 *
 * Grouping and subquery
 *
 *************************************************/

group_or_subquery_expr:
LPAREN group_or_subquery RPAREN
{
  $$ = $2
}
;

group_or_subquery:
expr
|
fullselect
{
  $$ = algebra.NewSubquery($1)
}
;
