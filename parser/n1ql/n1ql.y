%{
package n1ql

import "fmt"
import "strings"
import "github.com/couchbaselabs/clog"
import "github.com/couchbaselabs/query/algebra"
import "github.com/couchbaselabs/query/datastore"
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

expr             expression.Expression
exprs            expression.Expressions
whenTerm         *expression.WhenTerm
whenTerms        expression.WhenTerms
binding          *expression.Binding
bindings         expression.Bindings

node             algebra.Node
statement        algebra.Statement

fullselect       *algebra.Select
subresult        algebra.Subresult
subselect        *algebra.Subselect
fromTerm         algebra.FromTerm
keyspaceTerm     *algebra.KeyspaceTerm
path             expression.Path
group            *algebra.Group
resultTerm       *algebra.ResultTerm
resultTerms      algebra.ResultTerms
projection       *algebra.Projection
order            *algebra.Order
sortTerm         *algebra.SortTerm
sortTerms        algebra.SortTerms

keyspaceRef      *algebra.KeyspaceRef

pairs            algebra.Pairs
set              *algebra.Set
unset            *algebra.Unset
setTerm          *algebra.SetTerm
setTerms         algebra.SetTerms
unsetTerm        *algebra.UnsetTerm
unsetTerms       algebra.UnsetTerms
updateFor        *algebra.UpdateFor
mergeActions     *algebra.MergeActions
mergeUpdate      *algebra.MergeUpdate
mergeDelete      *algebra.MergeDelete
mergeInsert      *algebra.MergeInsert

createIndex      *algebra.CreateIndex
dropIndex        *algebra.DropIndex
alterIndex       *algebra.AlterIndex
indexType        datastore.IndexType
}

%token ALL
%token ALTER
%token ANALYZE
%token AND
%token ANY
%token ARRAY
%token AS
%token ASC
%token BEGIN
%token BETWEEN
%token BINARY
%token BOOLEAN
%token BREAK
%token BUCKET
%token BY
%token CALL
%token CASE
%token CAST
%token CLUSTER
%token COLLATE
%token COLLECTION
%token COMMIT
%token CONNECT
%token CONTINUE
%token CREATE
%token DATABASE
%token DATASET
%token DATASTORE
%token DECLARE
%token DECREMENT
%token DELETE
%token DERIVED
%token DESC
%token DESCRIBE
%token DISTINCT
%token DO
%token DROP
%token EACH
%token ELEMENT
%token ELSE
%token END
%token EVERY
%token EXCEPT
%token EXCLUDE
%token EXECUTE
%token EXISTS
%token EXPLAIN
%token FALSE
%token FIRST
%token FLATTEN
%token FOR
%token FROM
%token FUNCTION
%token GRANT
%token GROUP
%token GSI
%token HAVING
%token IF
%token IN
%token INCLUDE
%token INCREMENT
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
%token KEYSPACE
%token LAST
%token LEFT
%token LET
%token LETTING
%token LIKE
%token LIMIT
%token LSM
%token MAP
%token MAPPING
%token MATCHED
%token MATERIALIZED
%token MERGE
%token MINUS
%token MISSING
%token NAMESPACE
%token NEST
%token NOT
%token NULL
%token NUMBER
%token OBJECT
%token OFFSET
%token ON
%token OPTION
%token OR
%token ORDER
%token OUTER
%token OVER
%token PARTITION
%token PASSWORD
%token PATH
%token POOL
%token PREPARE
%token PRIMARY
%token PRIVATE
%token PRIVILEGE
%token PROCEDURE
%token PUBLIC
%token RAW
%token REALM
%token REDUCE
%token RENAME
%token RETURN
%token RETURNING
%token REVOKE
%token RIGHT
%token ROLE
%token ROLLBACK
%token SATISFIES
%token SCHEMA
%token SELECT
%token SELF
%token SET
%token SHOW
%token SOME
%token START
%token STATISTICS
%token STRING
%token SYSTEM
%token THEN
%token TO
%token TRANSACTION
%token TRIGGER
%token TRUE
%token TRUNCATE
%token UNDER
%token UNION
%token UNIQUE
%token UNNEST
%token UNSET
%token UPDATE
%token UPSERT
%token USE
%token USER
%token USING
%token VALUE
%token VALUED
%token VALUES
%token VIEW
%token WHEN
%token WHERE
%token WHILE
%token WITH
%token WITHIN
%token WORK
%token XOR

%token INT NUMBER STRING IDENTIFIER IDENTIFIER_ICASE NAMED_PARAM POSITIONAL_PARAM
%token LPAREN RPAREN
%token LBRACE RBRACE LBRACKET RBRACKET RBRACKET_ICASE
%token COMMA COLON

/* Precedence: lowest to highest */
%left           ORDER
%left           UNION INTERESECT EXCEPT
%left           JOIN NEST UNNEST FLATTEN INNER LEFT
%left           OR
%left           AND
%right          NOT
%nonassoc       EQ DEQ NE
%nonassoc       LT GT LE GE
%nonassoc       LIKE
%nonassoc       BETWEEN
%nonassoc       IN WITHIN
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
%type <s>                STRING
%type <s>                IDENTIFIER IDENTIFIER_ICASE
%type <s>                NAMED_PARAM
%type <f>                NUMBER
%type <n>                INT
%type <n>                POSITIONAL_PARAM
%type <expr>             literal construction_expr object array
%type <expr>             param_expr
%type <binding>          member
%type <bindings>         members opt_members

%type <expr>             expr c_expr b_expr
%type <exprs>            exprs opt_exprs
%type <binding>          binding
%type <bindings>         bindings

%type <s>                alias as_alias opt_as_alias variable

%type <expr>             case_expr simple_or_searched_case simple_case searched_case opt_else
%type <whenTerms>        when_thens

%type <expr>             collection_expr collection_cond collection_xform
%type <binding>          coll_binding
%type <bindings>         coll_bindings
%type <expr>             satisfies
%type <expr>             opt_when

%type <expr>             function_expr
%type <s>                function_name

%type <expr>             paren_or_subquery_expr paren_or_subquery

%type <fullselect>       fullselect
%type <subresult>        subselects
%type <subselect>        subselect
%type <subselect>        select_from
%type <subselect>        from_select
%type <fromTerm>         from_term from opt_from
%type <keyspaceTerm>     keyspace_term join_term
%type <b>                opt_join_type
%type <path>             path opt_subpath
%type <s>                namespace_name keyspace_name
%type <expr>             use_keys opt_use_keys on_keys
%type <bindings>         opt_let let
%type <expr>             opt_where where
%type <group>            opt_group group
%type <bindings>         opt_letting letting
%type <expr>             opt_having having
%type <resultTerm>       project
%type <resultTerms>      projects
%type <projection>       projection select_clause
%type <order>            order_by opt_order_by
%type <sortTerm>         sort_term
%type <sortTerms>        sort_terms
%type <expr>             limit opt_limit
%type <expr>             offset opt_offset
%type <b>                dir opt_dir

%type <statement>        stmt explain prepare execute select_stmt dml_stmt ddl_stmt
%type <statement>        insert upsert delete update merge
%type <statement>        index_stmt create_index drop_index alter_index

%type <keyspaceRef>      keyspace_ref
%type <pairs>            values values_list
%type <expr>             key_expr opt_value_expr
%type <projection>       returns returning opt_returning
%type <binding>          update_binding
%type <bindings>         update_bindings
%type <expr>             path_expr
%type <set>              set
%type <setTerm>          set_term
%type <setTerms>         set_terms
%type <unset>            unset
%type <unsetTerm>        unset_term
%type <unsetTerms>       unset_terms
%type <updateFor>        update_for opt_update_for
%type <binding>          update_binding
%type <bindings>         update_bindings
%type <mergeActions>     merge_actions opt_merge_delete_insert
%type <mergeUpdate>      merge_update
%type <mergeDelete>      merge_delete
%type <mergeInsert>      merge_insert opt_merge_insert

%type <s>                index_name
%type <keyspaceRef>      named_keyspace_ref
%type <expr>             index_partition
%type <indexType>        index_using
%type <s>                rename
%type <expr>             index_expr index_where
%type <exprs>            index_exprs

%start input

%%

input:
stmt
{
    yylex.(*lexer).setStatement($1)
}
|
expr
{
    yylex.(*lexer).setExpression($1)
}
;

stmt:
explain
|
prepare
|
execute
|
select_stmt
|
dml_stmt
|
ddl_stmt
;

explain:
EXPLAIN stmt
{
    $$ = algebra.NewExplain($2)
}
;

prepare:
PREPARE stmt
{
    $$ = algebra.NewPrepare($2)
}
;

execute:
EXECUTE object
{
    $$ = algebra.NewExecute($2)
}
;

select_stmt:
fullselect
{
    $$ = $1
}
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
subselects opt_order_by
{
    $$ = algebra.NewSelect($1, $2, nil, nil) /* OFFSET precedes LIMIT */
}
|
subselects opt_order_by limit opt_offset
{
    $$ = algebra.NewSelect($1, $2, $4, $3) /* OFFSET precedes LIMIT */
}
|
subselects opt_order_by offset opt_limit
{
    $$ = algebra.NewSelect($1, $2, $3, $4) /* OFFSET precedes LIMIT */
}
;

subselects:
subselect
{
    $$ = $1
}
|
subselects UNION subselect
{
    $$ = algebra.NewUnion($1, $3)
}
|
subselects UNION ALL subselect
{
    $$ = algebra.NewUnionAll($1, $4)
}
|
subselects INTERSECT subselect
{
    $$ = algebra.NewIntersect($1, $3)
}
|
subselects INTERSECT ALL subselect
{
    $$ = algebra.NewIntersectAll($1, $4)
}
|
subselects EXCEPT subselect
{
    $$ = algebra.NewExcept($1, $3)
}
|
subselects EXCEPT ALL subselect
{
    $$ = algebra.NewExceptAll($1, $4)
}
;

subselect:
from_select
|
select_from
;

from_select:
from opt_let opt_where opt_group select_clause
{
    $$ = algebra.NewSubselect($1, $2, $3, $4, $5)
}
;

select_from:
select_clause opt_from opt_let opt_where opt_group
{
    $$ = algebra.NewSubselect($2, $3, $4, $5, $1)
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
    $$ = algebra.NewProjection(false, $1)
}
|
DISTINCT projects
{
    $$ = algebra.NewProjection(true, $2)
}
|
ALL projects
{
    $$ = algebra.NewProjection(false, $2)
}
|
raw expr opt_as_alias
{
    $$ = algebra.NewRawProjection(false, $2, $3)
}
|
DISTINCT raw expr opt_as_alias
{
    $$ = algebra.NewRawProjection(true, $3, $4)
}
;

raw:
RAW
|
ELEMENT
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
expr opt_as_alias
{
    $$ = algebra.NewResultTerm($1, false, $2)
}
;

opt_as_alias:
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

opt_from:
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
keyspace_term
{
    $$ = $1
}
|
from_term opt_join_type JOIN join_term
{
    $$ = algebra.NewJoin($1, $2, $4)
}
|
from_term opt_join_type NEST join_term
{
    $$ = algebra.NewNest($1, $2, $4)
}
|
from_term opt_join_type unnest expr opt_as_alias
{
    $$ = algebra.NewUnnest($1, $2, $4, $5)
}
;

unnest:
UNNEST
|
FLATTEN
;

keyspace_term:
keyspace_name opt_subpath opt_as_alias opt_use_keys
{
    $$ = algebra.NewKeyspaceTerm("", $1, $2, $3, $4)
}
|
namespace_name COLON keyspace_name opt_subpath opt_as_alias opt_use_keys
{
    $$ = algebra.NewKeyspaceTerm($1, $3, $4, $5, $6)
}
|
SYSTEM COLON keyspace_name opt_subpath opt_as_alias opt_use_keys
{
    $$ = algebra.NewKeyspaceTerm("#system", $3, $4, $5, $6)
}
;

join_term:
keyspace_name opt_subpath opt_as_alias on_keys
{
    $$ = algebra.NewKeyspaceTerm("", $1, $2, $3, $4)
}
|
namespace_name COLON keyspace_name opt_subpath opt_as_alias on_keys
{
    $$ = algebra.NewKeyspaceTerm($1, $3, $4, $5, $6)
}
|
SYSTEM COLON keyspace_name opt_subpath opt_as_alias on_keys
{
    $$ = algebra.NewKeyspaceTerm("#system", $3, $4, $5, $6)
}
;

namespace_name:
IDENTIFIER
;

keyspace_name:
IDENTIFIER
;

opt_subpath:
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

opt_use_keys:
/* empty */
{
    $$ = nil
}
|
use_keys
;

use_keys:
USE opt_primary KEYS expr
{
    $$ = $4
}
;

opt_primary:
/* empty */
{
}
|
PRIMARY
;

opt_join_type:
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
LEFT opt_outer
{
    $$ = true
}
;

opt_outer:
/* empty */
|
OUTER
;

on_keys:
ON opt_primary KEYS expr
{
    $$ = $4
}
;


/*************************************************
 *
 * LET clause
 *
 *************************************************/

opt_let:
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
    $$ = expression.Bindings{$1}
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

opt_where:
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

opt_group:
/* empty */
{
    $$ = nil
}
|
group
;

group:
GROUP BY exprs opt_letting opt_having
{
    $$ = algebra.NewGroup($3, $4, $5)
}
|
letting
{
    $$ = algebra.NewGroup(nil, $1, nil)
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

opt_letting:
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

opt_having:
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

opt_order_by:
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
    $$ = algebra.NewOrder($3)
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
expr opt_dir
{
    $$ = algebra.NewSortTerm($1, $2)
}
;

opt_dir:
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

opt_limit:
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

opt_offset:
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
INSERT INTO keyspace_ref opt_values_header values_list opt_returning
{
    $$ = algebra.NewInsertValues($3, $5, $6)
}
|
INSERT INTO keyspace_ref LPAREN key_expr opt_value_expr RPAREN fullselect opt_returning
{
    $$ = algebra.NewInsertSelect($3, $5, $6, $8, $9)
}
;

keyspace_ref:
namespace_name COLON keyspace_name opt_as_alias
{
    $$ = algebra.NewKeyspaceRef($1, $3, $4)
}
|
keyspace_name opt_as_alias
{
    $$ = algebra.NewKeyspaceRef("", $1, $2)
}
;

opt_values_header:
/* empty */
|
LPAREN KEY COMMA VALUE RPAREN
|
LPAREN PRIMARY KEY COMMA VALUE RPAREN
;

key:
KEY
|
PRIMARY KEY
;

values_list:
values
|
values_list COMMA values
{
    $$ = append($1, $3...)
}
;

values:
VALUES LPAREN expr COMMA expr RPAREN
{
    $$ = algebra.Pairs{&algebra.Pair{Key: $3, Value: $5}}
}
;

opt_returning:
/* empty */
{
    $$ = nil
}
|
returning
;

returning:
RETURNING returns
{
    $$ = $2
}
;

returns:
projects
{
    $$ = algebra.NewProjection(false, $1)
}
|
raw expr
{
    $$ = algebra.NewRawProjection(false, $2, "")
}
;

key_expr:
key expr
{
    $$ = $2
}
;

opt_value_expr:
/* empty */
{
    $$ = nil
}
|
COMMA VALUE expr
{
    $$ = $3
}
;


/*************************************************
 *
 * UPSERT
 *
 *************************************************/

upsert:
UPSERT INTO keyspace_ref opt_values_header values_list opt_returning
{
    $$ = algebra.NewUpsertValues($3, $5, $6)
}
|
UPSERT INTO keyspace_ref LPAREN key_expr opt_value_expr RPAREN fullselect opt_returning
{
    $$ = algebra.NewUpsertSelect($3, $5, $6, $8, $9)
}
;


/*************************************************
 *
 * DELETE
 *
 *************************************************/

delete:
DELETE FROM keyspace_ref opt_use_keys opt_where opt_limit opt_returning
{
    $$ = algebra.NewDelete($3, $4, $5, $6, $7)
}
;


/*************************************************
 *
 * UPDATE
 *
 *************************************************/

update:
UPDATE keyspace_ref opt_use_keys set unset opt_where opt_limit opt_returning
{
    $$ = algebra.NewUpdate($2, $3, $4, $5, $6, $7, $8)
}
|
UPDATE keyspace_ref opt_use_keys set opt_where opt_limit opt_returning
{
    $$ = algebra.NewUpdate($2, $3, $4, nil, $5, $6, $7)
}
|
UPDATE keyspace_ref opt_use_keys unset opt_where opt_limit opt_returning
{
    $$ = algebra.NewUpdate($2, $3, nil, $4, $5, $6, $7)
}
;

set:
SET set_terms
{
    $$ = algebra.NewSet($2)
}
;

set_terms:
set_term
{
    $$ = algebra.SetTerms{$1}
}
|
set_terms COMMA set_term
{
    $$ = append($1, $3)
}
;

set_term:
path EQ expr opt_update_for
{
    $$ = algebra.NewSetTerm($1, $3, $4)
}
;

opt_update_for:
/* empty */
{
    $$ = nil
}
|
update_for
;

update_for:
FOR update_bindings opt_when END
{
    $$ = algebra.NewUpdateFor($2, $3)
}
;

update_bindings:
update_binding
{
    $$ = expression.Bindings{$1}
}
|
update_bindings COMMA update_binding
{
    $$ = append($1, $3)
}
;

update_binding:
variable IN path_expr
{
    $$ = expression.NewBinding($1, $3)
}
|
variable WITHIN path_expr
{
    $$ = expression.NewDescendantBinding($1, $3)
}
;

variable:
IDENTIFIER
;

path_expr:
path
{
    $$ = $1
}
;

opt_when:
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

unset:
UNSET unset_terms
{
    $$ = algebra.NewUnset($2)
}
;

unset_terms:
unset_term
{
    $$ = algebra.UnsetTerms{$1}
}
|
unset_terms COMMA unset_term
{
    $$ = append($1, $3)
}
;

unset_term:
path opt_update_for
{
    $$ = algebra.NewUnsetTerm($1, $2)
}
;


/*************************************************
 *
 * MERGE
 *
 *************************************************/

merge:
MERGE INTO keyspace_ref USING keyspace_term ON key_expr merge_actions opt_limit opt_returning
{
    source := algebra.NewMergeSourceFrom($5, "")
    $$ = algebra.NewMerge($3, source, $7, $8, $9, $10)
}
|
MERGE INTO keyspace_ref USING LPAREN fullselect RPAREN as_alias ON key_expr merge_actions opt_limit opt_returning
{
    source := algebra.NewMergeSourceSelect($6, $8)
    $$ = algebra.NewMerge($3, source, $10, $11, $12, $13)
}
;

merge_actions:
/* empty */
{
    $$ = algebra.NewMergeActions(nil, nil, nil)
}
|
WHEN MATCHED THEN UPDATE merge_update opt_merge_delete_insert
{
    $$ = algebra.NewMergeActions($5, $6.Delete(), $6.Insert())
}
|
WHEN MATCHED THEN DELETE merge_delete opt_merge_insert
{
    $$ = algebra.NewMergeActions(nil, $5, $6)
}
|
WHEN NOT MATCHED THEN INSERT merge_insert
{
    $$ = algebra.NewMergeActions(nil, nil, $6)
}
;

opt_merge_delete_insert:
/* empty */
{
    $$ = algebra.NewMergeActions(nil, nil, nil)
}
|
WHEN MATCHED THEN DELETE merge_delete opt_merge_insert
{
    $$ = algebra.NewMergeActions(nil, $5, $6)
}
|
WHEN NOT MATCHED THEN INSERT merge_insert
{
    $$ = algebra.NewMergeActions(nil, nil, $6)
}
;

opt_merge_insert:
/* empty */
{
    $$ = nil
}
|
WHEN NOT MATCHED THEN INSERT merge_insert
{
    $$ = $6
}
;

merge_update:
set opt_where
{
    $$ = algebra.NewMergeUpdate($1, nil, $2)
}
|
set unset opt_where
{
    $$ = algebra.NewMergeUpdate($1, $2, $3)
}
|
unset opt_where
{
    $$ = algebra.NewMergeUpdate(nil, $1, $2)
}
;

merge_delete:
opt_where
{
    $$ = algebra.NewMergeDelete($1)
}
;

merge_insert:
expr opt_where
{
    $$ = algebra.NewMergeInsert($1, $2)
}
;


/*************************************************
 *
 * CREATE INDEX
 *
 *************************************************/

create_index:
CREATE PRIMARY INDEX ON named_keyspace_ref index_using
{
  $$ = algebra.NewCreatePrimaryIndex($5, $6)
}
|
CREATE INDEX index_name ON named_keyspace_ref LPAREN index_exprs RPAREN index_partition index_where index_using
{
  $$ = algebra.NewCreateIndex($3, $5, $7, $9, $10, $11)
}
;

index_name:
IDENTIFIER
;

named_keyspace_ref:
keyspace_name
{
    $$ = algebra.NewKeyspaceRef("", $1, "")
}
|
namespace_name COLON keyspace_name
{
    $$ = algebra.NewKeyspaceRef($1, $3, "")
}
;

index_partition:
/* empty */
{
    $$ = nil
}
|
PARTITION BY expr
{
    $$ = $3
}
;

index_using:
/* empty */
{
    $$ = datastore.VIEW
}
|
USING VIEW
{
    $$ = datastore.VIEW
}
|
USING LSM
{
    $$ = datastore.GSI
}
|
USING GSI
{
    $$ = datastore.GSI
}
;

index_exprs:
index_expr
{
    $$ = expression.Expressions{$1}
}
|
index_exprs COMMA index_expr
{
    $$ = append($1, $3)
}
;

index_expr:
expr
{
    exp := $1
    if !exp.Indexable() || exp.Value() != nil {
        yylex.Error(fmt.Sprintf("Expression not indexable."))
    }

    $$ = exp
}

index_where:
/* empty */
{
    $$ = nil
}
|
WHERE index_expr
{
    $$ = $2
}
;


/*************************************************
 *
 * DROP INDEX
 *
 *************************************************/

drop_index:
DROP PRIMARY INDEX ON named_keyspace_ref
{
    $$ = algebra.NewDropIndex($5, "#primary") 
}
|
DROP INDEX named_keyspace_ref DOT index_name
{
    $$ = algebra.NewDropIndex($3, $5)
}
;

/*************************************************
 *
 * ALTER INDEX
 *
 *************************************************/

alter_index:
ALTER INDEX named_keyspace_ref DOT index_name rename
{
    $$ = algebra.NewAlterIndex($3, $5, $6)
}

rename:
/* empty */
{
    $$ = ""
}
|
RENAME TO index_name
{
    $$ = $3
}
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
    $$ = expression.NewField($1, expression.NewFieldName($3))
}
|
path DOT IDENTIFIER_ICASE
{
    field := expression.NewField($1, expression.NewFieldName($3))
    field.SetCaseInsensitive(true)
    $$ = field
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
    $$ = expression.NewField($1, expression.NewFieldName($3))
}
|
expr DOT IDENTIFIER_ICASE
{
    field := expression.NewField($1, expression.NewFieldName($3))
    field.SetCaseInsensitive(true)
    $$ = field
}
|
expr DOT LBRACKET expr RBRACKET
{
    $$ = expression.NewField($1, $4)
}
|
expr DOT LBRACKET expr RBRACKET_ICASE
{
    field := expression.NewField($1, $4)
    field.SetCaseInsensitive(true)
    $$ = field
}
|
expr LBRACKET expr RBRACKET
{
    $$ = expression.NewElement($1, $3)
}
|
expr LBRACKET expr COLON RBRACKET
{
    $$ = expression.NewSlice($1, $3)
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
    $$ = expression.NewSub($1, $3)
}
|
expr STAR expr
{
    $$ = expression.NewMult($1, $3)
}
|
expr DIV expr
{
    $$ = expression.NewDiv($1, $3)
}
|
expr MOD expr
{
    $$ = expression.NewMod($1, $3)
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
    $$ = expression.NewEq($1, $3)
}
|
expr DEQ expr
{
    $$ = expression.NewEq($1, $3)
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
expr WITHIN expr
{
    $$ = expression.NewWithin($1, $3)
}
|
expr NOT WITHIN expr
{
    $$ = expression.NewNotWithin($1, $4)
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
expr IS BOOLEAN
{
    $$ = expression.NewIsBoolean($1)
}
|
expr IS NOT BOOLEAN
{
    $$ = expression.NewNot(expression.NewIsBoolean($1))
}
|
expr IS NUMBER
{
    $$ = expression.NewIsNumber($1)
}
|
expr IS NOT NUMBER
{
    $$ = expression.NewNot(expression.NewIsNumber($1))
}
|
expr IS STRING
{
    $$ = expression.NewIsString($1)
}
|
expr IS NOT STRING
{
    $$ = expression.NewNot(expression.NewIsString($1))
}
|
expr IS ARRAY
{
    $$ = expression.NewIsArray($1)
}
|
expr IS NOT ARRAY
{
    $$ = expression.NewNot(expression.NewIsArray($1))
}
|
expr IS OBJECT
{
    $$ = expression.NewIsObject($1)
}
|
expr IS NOT OBJECT
{
    $$ = expression.NewNot(expression.NewIsObject($1))
}
|
expr IS BINARY
{
    $$ = expression.NewIsBinary($1)
}
|
expr IS NOT BINARY
{
    $$ = expression.NewNot(expression.NewIsBinary($1))
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
/* Construction */
construction_expr
|
/* Identifier */
IDENTIFIER
{
    $$ = expression.NewIdentifier($1)
}
|
/* Self */
SELF
{
    $$ = expression.NewSelf()
}
|
/* Parameter */
param_expr
|
/* Function */
function_expr
|
/* Prefix */
MINUS expr %prec UMINUS
{
    $$ = expression.NewNeg($2)
}
|
/* Case */
case_expr
|
/* Collection */
collection_expr
|
/* Grouping and subquery */
paren_or_subquery_expr
;

b_expr:
c_expr
|
/* Nested */
b_expr DOT IDENTIFIER
{
    $$ = expression.NewField($1, expression.NewFieldName($3))
}
|
b_expr DOT IDENTIFIER_ICASE
{
    field := expression.NewField($1, expression.NewFieldName($3))
    field.SetCaseInsensitive(true)
    $$ = field
}
|
b_expr DOT LBRACKET expr RBRACKET
{
    $$ = expression.NewField($1, $4)
}
|
b_expr DOT LBRACKET expr RBRACKET_ICASE
{
    field := expression.NewField($1, $4)
    field.SetCaseInsensitive(true)
    $$ = field
}
|
b_expr LBRACKET expr RBRACKET
{
    $$ = expression.NewElement($1, $3)
}
|
b_expr LBRACKET expr COLON RBRACKET
{
    $$ = expression.NewSlice($1, $3)
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
    $$ = expression.NewSub($1, $3)
}
|
b_expr STAR b_expr
{
    $$ = expression.NewMult($1, $3)
}
|
b_expr DIV b_expr
{
    $$ = expression.NewDiv($1, $3)
}
|
b_expr MOD b_expr
{
    $$ = expression.NewMod($1, $3)
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
MISSING
{
    $$ = expression.MISSING_EXPR
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
;


/*************************************************
 *
 * Construction
 *
 *************************************************/

construction_expr:
object
|
array
;

object:
LBRACE opt_members RBRACE
{
    $$ = expression.NewObjectConstruct($2)
}
;

opt_members:
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
LBRACKET opt_exprs RBRACKET
{
    $$ = expression.NewArrayConstruct($2...)
}
;

opt_exprs:
/* empty */
{
    $$ = nil
}
|
exprs
;


/*************************************************
 *
 * Parameter
 *
 *************************************************/

param_expr:
NAMED_PARAM
{
    $$ = algebra.NewNamedParameter($1)
}
|
POSITIONAL_PARAM
{
    $$ = algebra.NewPositionalParameter($1)
}
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
expr when_thens opt_else
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
opt_else
{
    $$ = expression.NewSearchedCase($1, $2)
}
;

opt_else:
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
function_name LPAREN opt_exprs RPAREN
{
    $$ = nil;
    f, ok := expression.GetFunction($1);
    if !ok && yylex.(*lexer).parsingStatement() {
        f, ok = algebra.GetAggregate($1, false);
    }

    if ok {
        if len($3) < f.MinArgs() || len($3) > f.MaxArgs() {
            yylex.Error(fmt.Sprintf("Wrong number of arguments to function %s.", $1));
        } else {
            $$ = f.Constructor()($3...);
        }
    } else {
        yylex.Error(fmt.Sprintf("Invalid function %s.", $1));
    }
}
|
function_name LPAREN DISTINCT expr RPAREN
{
    $$ = nil;
    if !yylex.(*lexer).parsingStatement() {
        yylex.Error("Cannot use aggregate as an inline expression.");
    } else {
        agg, ok := algebra.GetAggregate($1, true);
        if ok {
            $$ = agg.Constructor()($4);
        } else {
            yylex.Error(fmt.Sprintf("Invalid aggregate function %s.", $1));
        }
    }
}
|
function_name LPAREN STAR RPAREN
{
    $$ = nil;
    if !yylex.(*lexer).parsingStatement() {
        yylex.Error("Cannot use aggregate as an inline expression.");
    } else {
        if strings.ToLower($1) != "count" {
            yylex.Error(fmt.Sprintf("Invalid aggregate function %s(*).", $1));
        } else {
            agg, ok := algebra.GetAggregate($1, false);
            if ok {
                $$ = agg.Constructor()(nil);
            } else {
                yylex.Error(fmt.Sprintf("Invalid aggregate function %s.", $1));
            }
        }
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
|
variable WITHIN expr
{
    $$ = expression.NewDescendantBinding($1, $3)
}
;

satisfies:
SATISFIES expr
{
    $$ = $2
}
;

collection_xform:
ARRAY expr FOR coll_bindings opt_when END
{
    $$ = expression.NewArray($2, $4, $5)
}
|
FIRST expr FOR coll_bindings opt_when END
{
    $$ = expression.NewFirst($2, $4, $5)
}
;


/*************************************************
 *
 * Parentheses and subquery
 *
 *************************************************/

paren_or_subquery_expr:
LPAREN paren_or_subquery RPAREN
{
    $$ = $2
}
;

paren_or_subquery:
expr
|
fullselect
{
    $$ = nil;
    if yylex.(*lexer).parsingStatement() {
        $$ = algebra.NewSubquery($1);
    } else {
        yylex.Error("Cannot use subquery as an inline expression.");
    }
}
;
