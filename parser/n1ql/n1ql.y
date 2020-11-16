%{
package n1ql

import "fmt"
import "strings"
import "github.com/couchbase/clog"
import "github.com/couchbase/query/algebra"
import "github.com/couchbase/query/datastore"
import "github.com/couchbase/query/errors"
import "github.com/couchbase/query/expression"
import "github.com/couchbase/query/expression/search"
import "github.com/couchbase/query/functions"
import "github.com/couchbase/query/functions/inline"
import "github.com/couchbase/query/functions/golang"
import "github.com/couchbase/query/functions/javascript"
import "github.com/couchbase/query/value"

func logDebugGrammar(format string, v ...interface{}) {
    clog.To("PARSER", format, v...)
}
%}

%union {
s string
u32 uint32
n int64
f float64
b bool

ss               []string
expr             expression.Expression
exprs            expression.Expressions
subquery         *algebra.Subquery
whenTerm         *expression.WhenTerm
whenTerms        expression.WhenTerms
binding          *expression.Binding
bindings         expression.Bindings
dimensions       []expression.Bindings

node             algebra.Node
statement        algebra.Statement

fullselect       *algebra.Select
subresult        algebra.Subresult
selectTerm       *algebra.SelectTerm
subselect        *algebra.Subselect
fromTerm         algebra.FromTerm
simpleFromTerm   algebra.SimpleFromTerm
keyspaceTerm     *algebra.KeyspaceTerm
keyspacePath     *algebra.Path
use              *algebra.Use
joinHint         algebra.JoinHint
indexRefs        algebra.IndexRefs
indexRef         *algebra.IndexRef
subqueryTerm     *algebra.SubqueryTerm
path             expression.Path
group            *algebra.Group
resultTerm       *algebra.ResultTerm
resultTerms      algebra.ResultTerms
projection       *algebra.Projection
order            *algebra.Order
sortTerm         *algebra.SortTerm
sortTerms        algebra.SortTerms
indexKeyTerm     *algebra.IndexKeyTerm
indexKeyTerms    algebra.IndexKeyTerms
partitionTerm   *algebra.IndexPartitionTerm
groupTerm       *algebra.GroupTerm
groupTerms       algebra.GroupTerms
windowTerm      *algebra.WindowTerm
windowTerms      algebra.WindowTerms
windowFrame     *algebra.WindowFrame
windowFrameExtents    algebra.WindowFrameExtents
windowFrameExtent *algebra.WindowFrameExtent

updStatistics    *algebra.UpdateStatistics

keyspaceRef      *algebra.KeyspaceRef
keyspaceRefs     []*algebra.KeyspaceRef
scopeRef         *algebra.ScopeRef

pair             *algebra.Pair
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

indexType        datastore.IndexType
inferenceType    datastore.InferenceType
val              value.Value

isolationLevel   datastore.IsolationLevel

functionName	 functions.FunctionName
functionBody     functions.FunctionBody

// token offset into the statement
tokOffset	 int
}

%token _ERROR_	// used by the scanner to flag errors
%token ADVISE
%token ALL
%token ALTER
%token ANALYZE
%token AND
%token ANY
%token ARRAY
%token AS
%token ASC
%token AT
%token BEGIN
%token BETWEEN
%token BINARY
%token BOOLEAN
%token BREAK
%token BUCKET
%token BUILD
%token BY
%token CALL
%token CASE
%token CAST
%token CLUSTER
%token COLLATE
%token COLLECTION
%token COMMIT
%token COMMITTED
%token CONNECT
%token CONTINUE
%token CORRELATED
%token COVER
%token CREATE
%token CURRENT
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
%token FETCH
%token FILTER
%token FIRST
%token FLATTEN
%token FLUSH
%token FOLLOWING
%token FOR
%token FORCE
%token FROM
%token FTS
%token FUNCTION
%token GOLANG
%token GRANT
%token GROUP
%token GROUPS
%token GSI
%token HASH
%token HAVING
%token IF
%token IGNORE
%token ILIKE
%token IN
%token INCLUDE
%token INCREMENT
%token INDEX
%token INFER
%token INLINE
%token INNER
%token INSERT
%token INTERSECT
%token INTO
%token IS
%token ISOLATION
%token JAVASCRIPT
%token JOIN
%token KEY
%token KEYS
%token KEYSPACE
%token KNOWN
%token LANGUAGE
%token LAST
%token LEFT
%token LET
%token LETTING
%token LEVEL
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
%token NAMESPACE_ID
%token NEST
%token NL
%token NO
%token NOT
%token NOT_A_TOKEN
%token NTH_VALUE
%token NULL
%token NULLS
%token NUMBER
%token OBJECT
%token OFFSET
%token ON
%token OPTION
%token OPTIONS
%token OR
%token ORDER
%token OTHERS
%token OUTER
%token OVER
%token PARSE
%token PARTITION
%token PASSWORD
%token PATH
%token POOL
%token PRECEDING
%token PREPARE
%token PRIMARY
%token PRIVATE
%token PRIVILEGE
%token PROBE
%token PROCEDURE
%token PUBLIC
%token RANGE
%token RAW
%token READ
%token REALM
%token REDUCE
%token RENAME
%token REPLACE
%token RESPECT
%token RETURN
%token RETURNING
%token REVOKE
%token RIGHT
%token ROLE
%token ROLLBACK
%token ROW
%token ROWS
%token SATISFIES
%token SAVEPOINT
%token SCHEMA
%token SCOPE
%token SELECT
%token SELF
%token SEMI
%token SET
%token SHOW
%token SOME
%token START
%token STATISTICS
%token STRING
%token SYSTEM
%token THEN
%token TIES
%token TO
%token TRAN
%token TRANSACTION
%token TRIGGER
%token TRUE
%token TRUNCATE
%token UNBOUNDED
%token UNDER
%token UNION
%token UNIQUE
%token UNKNOWN
%token UNNEST
%token UNSET
%token UPDATE
%token UPSERT
%token USE
%token USER
%token USING
%token VALIDATE
%token VALUE
%token VALUED
%token VALUES
%token VIA
%token VIEW
%token WHEN
%token WHERE
%token WHILE
%token WINDOW
%token WITH
%token WITHIN
%token WORK
%token XOR

%token INT NUM STR IDENT IDENT_ICASE NAMED_PARAM POSITIONAL_PARAM NEXT_PARAM
%token LPAREN RPAREN
%token LBRACE RBRACE LBRACKET RBRACKET RBRACKET_ICASE
%token COMMA COLON

/* Precedence: lowest to highest */
%left           ORDER
%left           UNION INTERESECT EXCEPT
%left           JOIN NEST UNNEST FLATTEN INNER LEFT RIGHT
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
%right          COVER
%left           ALL
%right          UMINUS
%left           DOT LBRACKET RBRACKET

/* Override precedence */
%left           LPAREN RPAREN
%left           NSCOLON


/* Types */
%type <s>                STR
%type <s>                IDENT IDENT_ICASE NAMESPACE_ID
%type <s>                REPLACE
%type <s>                NAMED_PARAM
%type <f>                NUM
%type <n>                INT
%type <n>                POSITIONAL_PARAM NEXT_PARAM
%type <expr>             literal construction_expr execute_using object array
%type <expr>             param_expr
%type <pair>             member
%type <pairs>            members opt_members

%type <expr>             expr c_expr b_expr
%type <exprs>            exprs opt_exprs
%type <binding>          binding with_term
%type <bindings>         bindings with_list

%type <s>                alias as_alias opt_as_alias variable opt_name opt_window_name

%type <expr>             case_expr simple_or_searched_case simple_case searched_case opt_else
%type <whenTerms>        when_thens

%type <expr>             collection_expr collection_cond collection_xform
%type <binding>          coll_binding
%type <bindings>         coll_bindings
%type <expr>             satisfies
%type <expr>             opt_when

%type <expr>             function_expr function_meta_expr
%type <s>                function_name

%type <functionName>	 func_name long_func_name short_func_name
%type <ss>		 parm_list parameter_terms
%type <functionBody>     func_body
%type <b>                opt_replace

%type <expr>             paren_expr
%type <subquery>         subquery_expr

%type <fullselect>       fullselect
%type <subresult>        select_term select_terms
%type <subselect>        subselect
%type <subselect>        select_from
%type <subselect>        from_select
%type <fromTerm>         from_term from opt_from
%type <simpleFromTerm>   simple_from_term
%type <keyspaceTerm>     keyspace_term
%type <keyspacePath>     keyspace_path
%type <b>                opt_join_type opt_quantifier
%type <path>             path
%type <s>                namespace_term namespace_name bucket_name scope_name keyspace_name
%type <use>              opt_use opt_use_del_upd opt_use_merge use_options use_keys use_index join_hint
%type <joinHint>         use_hash_option
%type <expr>             on_keys on_key
%type <indexRefs>        index_refs
%type <indexRef>         index_ref
%type <bindings>         opt_let let opt_with
%type <expr>             opt_where where opt_filter
%type <group>            opt_group group
%type <bindings>         opt_letting letting
%type <expr>             opt_having having
%type <resultTerm>       project
%type <resultTerms>      projects
%type <projection>       projection select_clause
%type <order>            order_by opt_order_by
%type <sortTerm>         sort_term
%type <sortTerms>        sort_terms
%type <groupTerm>        group_term
%type <groupTerms>       group_terms
%type <expr>             limit opt_limit
%type <expr>             offset opt_offset
%type <b>                dir opt_dir

%type <statement>        stmt_body
%type <statement>        stmt advise explain prepare execute select_stmt dml_stmt ddl_stmt
%type <statement>        infer
%type <statement>        update_statistics
%type <statement>        insert upsert delete update merge
%type <statement>        index_stmt create_index drop_index alter_index build_index
%type <statement>        scope_stmt create_scope drop_scope
%type <statement>        transaction_stmt start_transaction commit_transaction rollback_transaction
%type <statement>        savepoint set_transaction_isolation
%type <statement>        collection_stmt create_collection drop_collection flush_collection
%type <statement>        role_stmt grant_role revoke_role
%type <statement>        function_stmt create_function drop_function execute_function

%type <keyspaceRef>      keyspace_ref simple_keyspace_ref
%type <pairs>            values values_list next_values
%type <expr>             key_expr_header value_expr_header options_expr_header
%type <pair>             key_val_expr key_val_options_expr key_val_options_expr_header

%type <projection>       returns returning opt_returning
%type <set>              set
%type <setTerm>          set_term
%type <setTerms>         set_terms
%type <unset>            unset
%type <unsetTerm>        unset_term
%type <unsetTerms>       unset_terms
%type <updateFor>        update_for opt_update_for
%type <binding>          update_binding
%type <bindings>         update_dimension
%type <dimensions>       update_dimensions
%type <b>                opt_key opt_force
%type <mergeActions>     merge_actions opt_merge_delete_insert
%type <mergeUpdate>      merge_update
%type <mergeDelete>      merge_delete
%type <mergeInsert>      merge_insert opt_merge_insert

%type <s>                index_name opt_primary_name opt_index_name
%type <keyspaceRef>      simple_named_keyspace_ref named_keyspace_ref
%type <scopeRef>         named_scope_ref
%type <partitionTerm>    index_partition
%type <indexType>        index_using opt_index_using
%type <val>              index_with opt_index_with
%type <expr>             index_term_expr index_expr index_where
%type <indexKeyTerm>     index_term
%type <indexKeyTerms>    index_terms
%type <expr>             expr_input all_expr

%type <exprs>            update_stat_terms
%type <expr>             update_stat_term

%type <inferenceType>    opt_infer_using
%type <val>              infer_ustat_with opt_infer_ustat_with

%type <ss>               user_list
%type <keyspaceRefs>     keyspace_scope_list
%type <keyspaceRef>      keyspace_scope
%type <ss>               role_list
%type <s>                role_name
%type <s>                user

%type <u32>              opt_order_nulls opt_ikattr ikattr
%type <b>                first_last nulls

%type <windowTerms>         opt_window_clause window_list
%type <windowTerm>          window_term window_specification window_function_details opt_window_function
%type <exprs>               opt_window_partition
%type <u32>                 window_frame_modifier window_frame_valexpr_modifier opt_window_frame_exclusion
%type <windowFrame>         opt_window_frame
%type <windowFrameExtents>  window_frame_extents
%type <windowFrameExtent>   window_frame_extent
%type <u32>                 opt_nulls_treatment nulls_treatment opt_from_first_last agg_quantifier

%type <isolationLevel>      opt_isolation_level isolation_level isolation_val
%type <s>                   opt_savepoint savepoint_name

%start input

%%

input:
stmt_body opt_trailer
{
    yylex.(*lexer).setStatement($1)
}
|
expr_input
{
    yylex.(*lexer).setExpression($1)
}
;

opt_trailer:
{
  /* nothing */
}
|
opt_trailer SEMI
;

stmt_body:
advise
|
explain
|
prepare
|
execute
|
stmt
;

stmt:
select_stmt
|
dml_stmt
|
ddl_stmt
|
infer
|
update_statistics
|
role_stmt
|
function_stmt
|
transaction_stmt
;

advise:
ADVISE opt_index stmt
{
    $$ = algebra.NewAdvise($3, yylex.(*lexer).Remainder($<tokOffset>1))
}
;

opt_index:
/* empty */
|
INDEX
{
    yylex.(*lexer).setOffset($<tokOffset>1)
}
;

explain:
EXPLAIN stmt
{
    $$ = algebra.NewExplain($2, yylex.(*lexer).Remainder($<tokOffset>1))
}
;

prepare:
PREPARE
{
    yylex.(*lexer).setOffset($<tokOffset>1)
}
opt_force opt_name stmt
{
    $$ = algebra.NewPrepare($4, $3, $5, yylex.(*lexer).getText(), yylex.(*lexer).getOffset())
}
;

opt_force:
/* empty */
{
    $$ = false
}
|
FORCE
{
    yylex.(*lexer).setOffset($<tokOffset>1)
    $$ = true
}
;

opt_name:
/* empty */
{
    $$ = ""
}
|
IDENT from_or_as
{
    $$ = $1
}
|
STR from_or_as
{
    $$ = $1
}
;

from_or_as:
FROM
{
    yylex.(*lexer).setOffset($<tokOffset>1)
}
|
AS
{
    yylex.(*lexer).setOffset($<tokOffset>1)
}
;

execute:
EXECUTE expr execute_using
{
    $$ = algebra.NewExecute($2, $3)
}
;

execute_using:
/* empty */
{
    $$ = nil
}
|
USING construction_expr
{
    $$ = $2
}
;

infer:
INFER opt_keyspace_collection simple_keyspace_ref opt_infer_using opt_infer_ustat_with
{
    $$ = algebra.NewInferKeyspace($3, $4, $5)
}
;

opt_keyspace_collection:
/* empty */
{
}
|
KEYSPACE
|
COLLECTION
;

opt_infer_using:
/* empty */
{
    $$ = datastore.INF_DEFAULT
}
;

opt_infer_ustat_with:
/* empty */
{
    $$ = nil
}
|
infer_ustat_with
;

infer_ustat_with:
WITH expr
{
    $$ = $2.Value()
    if $$ == nil {
	yylex.Error("WITH value must be static.")
    }
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
|
scope_stmt
|
collection_stmt
;

role_stmt:
grant_role
|
revoke_role
;

index_stmt:
create_index
|
drop_index
|
alter_index
|
build_index
;

scope_stmt:
create_scope
|
drop_scope
;

collection_stmt:
create_collection
|
drop_collection
|
flush_collection
;

function_stmt:
create_function
|
drop_function
|
execute_function
;

transaction_stmt:
start_transaction
|
commit_transaction
|
rollback_transaction
|
savepoint
|
set_transaction_isolation
;

fullselect:
select_terms opt_order_by
{
    $$ = algebra.NewSelect($1, $2, nil, nil) /* OFFSET precedes LIMIT */
}
|
select_terms opt_order_by limit opt_offset
{
    $$ = algebra.NewSelect($1, $2, $4, $3) /* OFFSET precedes LIMIT */
}
|
select_terms opt_order_by offset opt_limit
{
    $$ = algebra.NewSelect($1, $2, $3, $4) /* OFFSET precedes LIMIT */
}
;

select_terms:
subselect
{
    $$ = $1
}
|
select_terms UNION select_term
{
    $$ = algebra.NewUnion($1, $3)
}
|
select_terms UNION ALL select_term
{
    $$ = algebra.NewUnionAll($1, $4)
}
|
select_terms INTERSECT select_term
{
    $$ = algebra.NewIntersect($1, $3)
}
|
select_terms INTERSECT ALL select_term
{
    $$ = algebra.NewIntersectAll($1, $4)
}
|
select_terms EXCEPT select_term
{
    $$ = algebra.NewExcept($1, $3)
}
|
select_terms EXCEPT ALL select_term
{
    $$ = algebra.NewExceptAll($1, $4)
}
|
subquery_expr UNION select_term
{
    left_term := algebra.NewSelectTerm($1.Select())
    $$ = algebra.NewUnion(left_term, $3)
}
|
subquery_expr UNION ALL select_term
{
    left_term := algebra.NewSelectTerm($1.Select())
    $$ = algebra.NewUnionAll(left_term, $4)
}
|
subquery_expr INTERSECT select_term
{
    left_term := algebra.NewSelectTerm($1.Select())
    $$ = algebra.NewIntersect(left_term, $3)
}
|
subquery_expr INTERSECT ALL select_term
{
    left_term := algebra.NewSelectTerm($1.Select())
    $$ = algebra.NewIntersectAll(left_term, $4)
}
|
subquery_expr EXCEPT select_term
{
    left_term := algebra.NewSelectTerm($1.Select())
    $$ = algebra.NewExcept(left_term, $3)
}
|
subquery_expr EXCEPT ALL select_term
{
    left_term := algebra.NewSelectTerm($1.Select())
    $$ = algebra.NewExceptAll(left_term, $4)
}
;

select_term:
subselect
{
    $$ = $1
}
|
subquery_expr
{
    $$ = algebra.NewSelectTerm($1.Select())
}
;

subselect:
from_select
|
select_from
;

from_select:
opt_with from opt_let opt_where opt_group opt_window_clause select_clause
{
    $$ = algebra.NewSubselect($1, $2, $3, $4, $5, $6, $7)
}
;

select_from:
opt_with select_clause opt_from opt_let opt_where opt_group opt_window_clause
{
    $$ = algebra.NewSubselect($1, $3, $4, $5, $6, $7, $2)
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
opt_quantifier projects
{
    $$ = algebra.NewProjection($1, $2)
}
|
opt_quantifier raw expr opt_as_alias
{
    $$ = algebra.NewRawProjection($1, $3, $4)
}
;

opt_quantifier:
/* empty */
{ $$ = false }
|
ALL
{ $$ = false }
|
DISTINCT
{ $$ = true }
;

raw:
RAW
|
ELEMENT
|
VALUE
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
    $$ = algebra.NewResultTerm(expression.SELF, true, "")
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
IDENT
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
simple_from_term
{
    if $1 != nil && $1.JoinHint() != algebra.JOIN_HINT_NONE {
        yylex.Error(fmt.Sprintf("Join hint (USE HASH or USE NL) cannot be specified on the first from term %s", $1.Alias()))
    }
    $$ = $1
}
|
from_term opt_join_type JOIN simple_from_term on_keys
{
    ksterm := algebra.GetKeyspaceTerm($4)
    if ksterm == nil {
        yylex.Error("JOIN must be done on a keyspace.")
    } else {
        ksterm.SetJoinKeys($5)
    }
    $$ = algebra.NewJoin($1, $2, ksterm)
}
|
from_term opt_join_type JOIN simple_from_term on_key FOR IDENT
{
    ksterm := algebra.GetKeyspaceTerm($4)
    if ksterm == nil {
        yylex.Error("JOIN must be done on a keyspace.")
    } else {
        ksterm.SetIndexJoinNest()
        ksterm.SetJoinKeys($5)
    }
    $$ = algebra.NewIndexJoin($1, $2, ksterm, $7)
}
|
from_term opt_join_type NEST simple_from_term on_keys
{
    ksterm := algebra.GetKeyspaceTerm($4)
    if ksterm == nil {
        yylex.Error("NEST must be done on a keyspace.")
    } else {
        ksterm.SetJoinKeys($5)
    }
    $$ = algebra.NewNest($1, $2, ksterm)
}
|
from_term opt_join_type NEST simple_from_term on_key FOR IDENT
{
    ksterm := algebra.GetKeyspaceTerm($4)
    if ksterm == nil {
        yylex.Error("NEST must be done on a keyspace.")
    } else {
        ksterm.SetIndexJoinNest()
        ksterm.SetJoinKeys($5)
    }
    $$ = algebra.NewIndexNest($1, $2, ksterm, $7)
}
|
from_term opt_join_type unnest expr opt_as_alias
{
    $$ = algebra.NewUnnest($1, $2, $4, $5)
}
|
from_term opt_join_type JOIN simple_from_term ON expr
{
    $4.SetAnsiJoin()
    $$ = algebra.NewAnsiJoin($1, $2, $4, $6)
}
|
from_term opt_join_type NEST simple_from_term ON expr
{
    $4.SetAnsiNest()
    $$ = algebra.NewAnsiNest($1, $2, $4, $6)
}
|
simple_from_term RIGHT opt_outer JOIN simple_from_term ON expr
{
    $1.SetAnsiJoin()
    $$ = algebra.NewAnsiRightJoin($1, $5, $7)
}
;

simple_from_term:
keyspace_term
{
    $$ = $1
}
|
expr opt_as_alias opt_use
{
    isExpr := false
    switch other := $1.(type) {
        case *algebra.Subquery:
            if $2 == "" {
                 return yylex.(*lexer).FatalError("Subquery in FROM clause must have an alias.")
            }
            if $3.Keys() != nil || $3.Indexes() != nil {
                 return yylex.(*lexer).FatalError("FROM Subquery cannot have USE KEYS or USE INDEX.")
            }
            $$ = algebra.NewSubqueryTerm(other.Select(), $2, $3.JoinHint())
        case *expression.Identifier:
            ksterm := algebra.NewKeyspaceTermFromPath(algebra.NewPathWithContext(other.Alias(), yylex.(*lexer).Namespace(), yylex.(*lexer).QueryContext()),
						      $2, $3.Keys(), $3.Indexes())
            $$ = algebra.NewExpressionTerm(other, $2, ksterm, other.Parenthesis() == false, $3.JoinHint())
        case *algebra.NamedParameter, *algebra.PositionalParameter:
            if $3.Indexes() == nil {
                   if $3.Keys() != nil {
                        $$ = algebra.NewKeyspaceTermFromExpression(other, $2, $3.Keys(), $3.Indexes(), $3.JoinHint())
                   } else {
                        $$ = algebra.NewExpressionTerm(other, $2, nil, false, $3.JoinHint())
                   }
            } else {
                   return yylex.(*lexer).FatalError("FROM <placeholder> cannot have USE INDEX.")
            }
        case *expression.Field:
	    path := other.Path()
	    if len(path) == 3 {
                ksterm := algebra.NewKeyspaceTermFromPath(algebra.NewPathLong(yylex.(*lexer).Namespace(), path[0], path[1], path[2]),
							$2, $3.Keys(), $3.Indexes())
                  $$ = algebra.NewExpressionTerm(other, $2, ksterm, other.Parenthesis() == false, $3.JoinHint())
	    } else {
		isExpr = true
            }
        default:
	    isExpr = true
    }
    if isExpr {
        if $3.Keys() == nil && $3.Indexes() == nil {
            $$ = algebra.NewExpressionTerm($1, $2, nil, false, $3.JoinHint())
        } else {
            return yylex.(*lexer).FatalError("FROM Expression cannot have USE KEYS or USE INDEX.")
        }
    }
}
;

unnest:
UNNEST
|
FLATTEN
;

keyspace_term:
keyspace_path opt_as_alias opt_use
{
    ksterm := algebra.NewKeyspaceTermFromPath($1, $2, $3.Keys(), $3.Indexes())
    if $3.JoinHint() != algebra.JOIN_HINT_NONE {
        ksterm.SetJoinHint($3.JoinHint())
    }
    $$ = ksterm
}
;

keyspace_path:
namespace_term keyspace_name
{
    $$ = algebra.NewPathShort($1, $2)
}
|
namespace_term bucket_name DOT scope_name DOT keyspace_name
{
    $$ = algebra.NewPathLong($1, $2, $4, $6)
}
;

/* for namespaces we have to have a rule with the namespace followed by the colon
   to resolve most shift reduce / reduce reduce conflicts
*/
namespace_term:
namespace_name
|
SYSTEM COLON
{
    $$ = datastore.SYSTEM_NAMESPACE
}
;

namespace_name:
NAMESPACE_ID COLON
{
    $$ = $1
}
;

bucket_name:
IDENT
;

scope_name:
IDENT
;

keyspace_name:
IDENT
;

opt_use:
/* empty */
{
    $$ = algebra.EMPTY_USE
}
|
USE use_options
{
    $$ = $2
}
;

use_options:
use_keys
|
use_index
|
join_hint
|
use_index join_hint
{
    $1.SetJoinHint($2.JoinHint())
    $$ = $1
}
|
join_hint use_index
{
    $1.SetIndexes($2.Indexes())
    $$ = $1
}
|
use_keys join_hint
{
    $1.SetJoinHint($2.JoinHint())
    $$ = $1
}
|
join_hint use_keys
{
    $1.SetKeys($2.Keys())
    $$ = $1
}
;

use_keys:
opt_primary KEYS expr
{
    $$ = algebra.NewUse($3, nil, algebra.JOIN_HINT_NONE)
}
;

use_index:
INDEX LPAREN index_refs RPAREN
{
    $$ = algebra.NewUse(nil, $3, algebra.JOIN_HINT_NONE)
}
;

join_hint:
HASH LPAREN use_hash_option RPAREN
{
    $$ = algebra.NewUse(nil, nil, $3)
}
|
NL
{
    $$ = algebra.NewUse(nil, nil, algebra.USE_NL)
}
;

opt_primary:
/* empty */
{
}
|
PRIMARY
;

index_refs:
index_ref
{
    $$ = algebra.IndexRefs{$1}
}
|
index_refs COMMA index_ref
{
    $$ = append($1, $3)
}
;

index_ref:
opt_index_name opt_index_using
{
    $$ = algebra.NewIndexRef($1, $2)
}

use_hash_option:
BUILD
{
    $$ = algebra.USE_HASH_BUILD
}
|
PROBE
{
    $$ = algebra.USE_HASH_PROBE
}
;

opt_use_del_upd:
opt_use
{
    if $1.JoinHint() != algebra.JOIN_HINT_NONE {
        yylex.Error("Keyspace reference cannot have join hint (USE HASH or USE NL) in DELETE or UPDATE statement")
    }
    $$ = $1
}
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

on_key:
ON opt_primary KEY expr
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
    $$ = expression.NewSimpleBinding($1, $3)
}
;

/*************************************************
 *
 * WITH clause
 *
 *************************************************/

opt_with:
/* empty */
{ $$ = nil }
|
WITH with_list
{
    $$ = $2
}
;

with_list:
with_term
{
    $$ = expression.Bindings{$1}
}
|
with_list COMMA with_term
{
    $$ = append($1, $3)
}
;

with_term:

/* we want expressions in parentesheses, but don't want to be
   forced to have subquery expressions in nested parentheses
 */
alias AS paren_expr
{
    $$ = expression.NewSimpleBinding($1, $3)
    $$.SetStatic(true)
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
GROUP BY group_terms opt_letting opt_having
{
    $$ = algebra.NewGroup($3, $4, $5)
}
|
letting
{
    $$ = algebra.NewGroup(nil, $1, nil)
}
;

group_terms:
group_term
{
    $$ = algebra.GroupTerms{$1}
}
|
group_terms COMMA group_term
{
    $$ = append($1, $3)
}
;

group_term:
expr opt_as_alias
{
    $$ = algebra.NewGroupTerm($1, $2)
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
expr opt_dir opt_order_nulls
{
    $$ = algebra.NewSortTerm($1, $2, algebra.NewOrderNullsPos($2,$3))
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

opt_order_nulls:
/* empty */
{
    $$ = algebra.NewOrderNulls(true,false,false)
}
|
nulls first_last
{
    $$ = algebra.NewOrderNulls(false, $1,$2)
}
;

first_last:
FIRST { $$ = false }
|
LAST { $$ = true }
;

nulls:
NULLS { $$ = true }
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
INSERT INTO keyspace_ref LPAREN key_val_options_expr_header RPAREN fullselect opt_returning
{
    $$ = algebra.NewInsertSelect($3, $5.Key(), $5.Value(), $5.Options(), $7, $8)
}
;

simple_keyspace_ref:
keyspace_name opt_as_alias
{
    $$ = algebra.NewKeyspaceRefWithContext($1, $2, yylex.(*lexer).Namespace(), yylex.(*lexer).QueryContext())
}
|
keyspace_path opt_as_alias
{
    $$ = algebra.NewKeyspaceRefFromPath($1, $2)
}
|
bucket_name DOT scope_name DOT keyspace_name  opt_as_alias
{
    path := algebra.NewPathLong(yylex.(*lexer).Namespace(), $1, $3, $5)
    $$ = algebra.NewKeyspaceRefFromPath(path, $6)
}
;

keyspace_ref:
simple_keyspace_ref
{
    $$ = $1
}
|
param_expr opt_as_alias
{
    $$ = algebra.NewKeyspaceRefFromExpression($1, $2)
}
;

opt_values_header:
/* empty */
|
LPAREN opt_primary KEY COMMA VALUE RPAREN
|
LPAREN opt_primary KEY COMMA VALUE COMMA OPTIONS RPAREN
;

key:
opt_primary KEY
;

values_list:
values
|
values_list COMMA next_values
{
    $$ = append($1, $3...)
}
;

values:
VALUES key_val_expr
{
    $$ = algebra.Pairs{$2}
}
|
VALUES key_val_options_expr
{
    $$ = algebra.Pairs{$2}
}
;

next_values:
values
|
key_val_expr
{
    $$ = algebra.Pairs{$1}
}
|
key_val_options_expr
{
    $$ = algebra.Pairs{$1}
}
;

key_val_expr:
LPAREN expr COMMA expr RPAREN
{
    $$ = algebra.NewPair($2, $4, nil)
}
;

key_val_options_expr:
LPAREN expr COMMA expr COMMA expr RPAREN
{
    $$ = algebra.NewPair($2, $4, $6)
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

key_expr_header:
key expr
{
    $$ = $2
}
;

value_expr_header:
VALUE expr
{
    $$ = $2
}
;

options_expr_header:
OPTIONS expr
{
    $$ = $2
}
;

key_val_options_expr_header:
key_expr_header
{
    $$ = algebra.NewPair($1, nil, nil)
}
|
key_expr_header COMMA value_expr_header
{
    $$ = algebra.NewPair($1, $3, nil)
}
|
key_expr_header COMMA value_expr_header COMMA options_expr_header
{
    $$ = algebra.NewPair($1, $3, $5)
}
|
key_expr_header COMMA options_expr_header
{
    $$ = algebra.NewPair($1, nil, $3)
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
UPSERT INTO keyspace_ref LPAREN key_val_options_expr_header RPAREN fullselect opt_returning
{
    $$ = algebra.NewUpsertSelect($3, $5.Key(), $5.Value(), $5.Options(), $7, $8)
}
;


/*************************************************
 *
 * DELETE
 *
 *************************************************/

delete:
DELETE FROM keyspace_ref opt_use_del_upd opt_where opt_limit opt_returning
{
    $$ = algebra.NewDelete($3, $4.Keys(), $4.Indexes(), $5, $6, $7)
}
;


/*************************************************
 *
 * UPDATE
 *
 *************************************************/

update:
UPDATE keyspace_ref opt_use_del_upd set unset opt_where opt_limit opt_returning
{
    $$ = algebra.NewUpdate($2, $3.Keys(), $3.Indexes(), $4, $5, $6, $7, $8)
}
|
UPDATE keyspace_ref opt_use_del_upd set opt_where opt_limit opt_returning
{
    $$ = algebra.NewUpdate($2, $3.Keys(), $3.Indexes(), $4, nil, $5, $6, $7)
}
|
UPDATE keyspace_ref opt_use_del_upd unset opt_where opt_limit opt_returning
{
    $$ = algebra.NewUpdate($2, $3.Keys(), $3.Indexes(), nil, $4, $5, $6, $7)
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
     $$ = algebra.NewSetTerm($1, $3, $4, nil)
}
|
function_meta_expr DOT path EQ expr
{
    $$ = nil
    if $1 != nil && algebra.IsValidMetaMutatePath($3){
         $$ = algebra.NewSetTerm($3, $5, nil, $1)
    } else if $1 != nil {
         return yylex.(*lexer).FatalError(fmt.Sprintf("SET clause has invalid path %s",  $3.String()))
    }
}
;

function_meta_expr:
function_name LPAREN opt_exprs RPAREN
{
    $$ = nil
    f, ok := expression.GetFunction($1)
    if ok && strings.ToLower($1) == "meta" && len($3) >= f.MinArgs() && len($3) <= f.MaxArgs() {
         $$ = f.Constructor()($3...)
    } else {
         return yylex.(*lexer).FatalError(fmt.Sprintf("SET clause has invalid path %s", $1))
    }
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
update_dimensions opt_when END
{
    $$ = algebra.NewUpdateFor($1, $2)
}
;

update_dimensions:
FOR update_dimension
{
    $$ = []expression.Bindings{$2}
}
|
update_dimensions FOR update_dimension
{
    dims := make([]expression.Bindings, 0, 1+len($1))
    dims = append(dims, $3)
    $$ = append(dims, $1...)
}
;

update_dimension:
update_binding
{
    $$ = expression.Bindings{$1}
}
|
update_dimension COMMA update_binding
{
    $$ = append($1, $3)
}
;

update_binding:
variable IN expr
{
    $$ = expression.NewSimpleBinding($1, $3)
}
|
variable WITHIN expr
{
    $$ = expression.NewBinding("", $1, $3, true)
}
|
variable COLON variable IN expr
{
    $$ = expression.NewBinding($1, $3, $5, false)
}
|
variable COLON variable WITHIN expr
{
    $$ = expression.NewBinding($1, $3, $5, true)
}
;

variable:
IDENT
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
MERGE INTO simple_keyspace_ref opt_use_merge USING simple_from_term ON opt_key expr merge_actions opt_limit opt_returning
{
     switch other := $6.(type) {
         case *algebra.SubqueryTerm:
              source := algebra.NewMergeSourceSubquery(other)
              $$ = algebra.NewMerge($3, $4.Indexes(), source, $8, $9, $10, $11, $12)
         case *algebra.ExpressionTerm:
              source := algebra.NewMergeSourceExpression(other)
              $$ = algebra.NewMerge($3, $4.Indexes(), source, $8, $9, $10, $11, $12)
         case *algebra.KeyspaceTerm:
              source := algebra.NewMergeSourceFrom(other)
              $$ = algebra.NewMerge($3, $4.Indexes(), source, $8, $9, $10, $11, $12)
         default:
	      yylex.Error("MERGE source term is UNKNOWN.")
     }
}
;

opt_use_merge:
opt_use
{
    if $1.Keys() != nil {
        yylex.Error("Keyspace reference cannot have USE KEYS hint in MERGE statement.")
    } else if $1.JoinHint() != algebra.JOIN_HINT_NONE {
        yylex.Error("Keyspace reference cannot have join hint (USE HASH or USE NL)in MERGE statement.")
    }
    $$ = $1
}
;

opt_key:
/* empty */
{
    $$ = false
}
|
key
{
    $$ = true
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
    $$ = algebra.NewMergeInsert(nil, $1, nil, $2)
}
|
key_val_expr opt_where
{
    $$ = algebra.NewMergeInsert($1.Key(), $1.Value(), nil, $2)
}
|
key_val_options_expr opt_where
{
    $$ = algebra.NewMergeInsert($1.Key(), $1.Value(), $1.Options(), $2)
}
|
LPAREN key_val_options_expr_header RPAREN opt_where
{
    $$ = algebra.NewMergeInsert($2.Key(), $2.Value(), $2.Options(), $4)
}
;

/*************************************************
 *
 * GRANT ROLE
 *
 *************************************************/

grant_role:
GRANT role_list TO user_list
{
	$$ = algebra.NewGrantRole($2, nil, $4)
}
|
GRANT role_list ON keyspace_scope_list TO user_list
{
	$$ = algebra.NewGrantRole($2, $4, $6)
}
;

role_list:
role_name
{
	$$ = []string{ $1 }
}
|
role_list COMMA role_name
{
	$$ = append($1, $3)
}
;

role_name:
IDENT
{
	$$ = $1
}
|
SELECT
{
	$$ = "select"
}
|
INSERT
{
	$$ = "insert"
}
|
UPDATE
{
	$$ = "update"
}
|
DELETE
{
	$$ = "delete"
}
;

keyspace_scope_list:
keyspace_scope
{
	$$ = []*algebra.KeyspaceRef{ $1 }
}
|
keyspace_scope_list COMMA keyspace_scope
{
	$$ = append($1, $3)
}
;

keyspace_scope:
keyspace_name
{
    $$ = algebra.NewKeyspaceRefWithContext($1, "", yylex.(*lexer).Namespace(), yylex.(*lexer).QueryContext())
}
|
namespace_name keyspace_name
{
    path := algebra.NewPathShort($1, $2)
    $$ = algebra.NewKeyspaceRefFromPath(path, "")
}
|
namespace_name bucket_name DOT scope_name DOT keyspace_name
{
    path := algebra.NewPathLong($1, $2, $4, $6)
    $$ = algebra.NewKeyspaceRefFromPath(path, "")
}
|
bucket_name DOT scope_name DOT keyspace_name
{
    path := algebra.NewPathLong(yylex.(*lexer).Namespace(), $1, $3, $5)
    $$ = algebra.NewKeyspaceRefFromPath(path, "")
}
|
namespace_name bucket_name DOT scope_name
{
    path := algebra.NewPathScope($1, $2, $4)
    $$ = algebra.NewKeyspaceRefFromPath(path, "")
}
|
bucket_name DOT scope_name
{
    path := algebra.NewPathScope(yylex.(*lexer).Namespace(), $1, $3)
    $$ = algebra.NewKeyspaceRefFromPath(path, "")
}
;

user_list:
user
{
	$$ = []string{ $1 }
}
|
user_list COMMA user
{
	$$ = append($1, $3)
}
;

user:
IDENT
{
	$$ = $1
}
|
IDENT COLON IDENT
{
	$$ = $1 + ":" + $3
}

/*************************************************
 *
 * REVOKE ROLE
 *
 *************************************************/

revoke_role:
REVOKE role_list FROM user_list
{
	$$ = algebra.NewRevokeRole($2, nil, $4)
}
|
REVOKE role_list ON keyspace_scope_list FROM user_list
{
	$$ = algebra.NewRevokeRole($2, $4, $6)
}
;

/*************************************************
 *
 * CREATE SCOPE
 *
 *************************************************/

create_scope:
CREATE SCOPE named_scope_ref
{
    $$ = algebra.NewCreateScope($3)
}
;

/*************************************************
 *
 * DROP SCOPE
 *
 *************************************************/

drop_scope:
DROP SCOPE named_scope_ref
{
    $$ = algebra.NewDropScope($3)
}
;

/*************************************************
 *
 * CREATE COLLECTION
 *
 *************************************************/

create_collection:
CREATE COLLECTION named_keyspace_ref
{
    $$ = algebra.NewCreateCollection($3)
}
;

/*************************************************
 *
 * DROP COLLECTION
 *
 *************************************************/

drop_collection:
DROP COLLECTION named_keyspace_ref
{
    $$ = algebra.NewDropCollection($3)
}
;

/*************************************************
 *
 * FLUSH COLLECTION
 *
 *************************************************/

flush_collection:
flush_or_truncate COLLECTION named_keyspace_ref
{
    $$ = algebra.NewFlushCollection($3)
}
;

flush_or_truncate:
FLUSH
|
TRUNCATE
;

/*************************************************
 *
 * CREATE INDEX
 *
 *************************************************/

create_index:
CREATE PRIMARY INDEX opt_primary_name ON named_keyspace_ref index_partition opt_index_using opt_index_with
{
    $$ = algebra.NewCreatePrimaryIndex($4, $6, $7, $8, $9)
}
|
CREATE INDEX index_name ON named_keyspace_ref LPAREN index_terms RPAREN index_partition index_where opt_index_using opt_index_with
{
    $$ = algebra.NewCreateIndex($3, $5, $7, $9, $10, $11, $12)
}
;

opt_primary_name:
/* empty */
{
    $$ = "#primary"
}
|
index_name
;

index_name:
IDENT
;

opt_index_name:
{ $$ = "" }
|
index_name
;

named_keyspace_ref:
simple_named_keyspace_ref
|
namespace_name bucket_name
{
    path := algebra.NewPathShort($1, $2)
    $$ = algebra.NewKeyspaceRefFromPath(path, "")
}
|
bucket_name DOT scope_name DOT keyspace_name
{
    path := algebra.NewPathLong(yylex.(*lexer).Namespace(), $1, $3, $5)
    $$ = algebra.NewKeyspaceRefFromPath(path, "")
}
;

simple_named_keyspace_ref:
keyspace_name
{
    $$ = algebra.NewKeyspaceRefWithContext($1, "", yylex.(*lexer).Namespace(), yylex.(*lexer).QueryContext())
}
|
namespace_name bucket_name DOT scope_name DOT keyspace_name
{
    path := algebra.NewPathLong($1, $2, $4, $6)
    $$ = algebra.NewKeyspaceRefFromPath(path, "")
}
;

named_scope_ref:
namespace_name bucket_name DOT scope_name
{
    path := algebra.NewPathScope($1, $2, $4)
    $$ = algebra.NewScopeRefFromPath(path, "")
}
|
bucket_name DOT scope_name
{
    path := algebra.NewPathScope(yylex.(*lexer).Namespace(), $1, $3)
    $$ = algebra.NewScopeRefFromPath(path, "")
}
;

index_partition:
/* empty */
{
    $$ = nil
}
|
PARTITION BY HASH LPAREN exprs RPAREN
{
    $$ = algebra.NewIndexPartitionTerm(datastore.HASH_PARTITION,$5)
}
;

opt_index_using:
/* empty */
{
    $$ = datastore.DEFAULT
}
|
index_using
;

index_using:
USING VIEW
{
    $$ = datastore.VIEW
}
|
USING GSI
{
    $$ = datastore.GSI
}
|
USING FTS
{
    $$ = datastore.FTS
}
;

opt_index_with:
/* empty */
{
    $$ = nil
}
|
index_with
;

index_with:
WITH expr
{
    $$ = $2.Value()
    if $$ == nil {
	yylex.Error("WITH value must be static.")
    }
}
;

index_terms:
index_term
{
    $$ = algebra.IndexKeyTerms{$1}
}
|
index_terms COMMA index_term
{
    $$ = append($1, $3)
}
;

index_term:
index_term_expr opt_ikattr
{
   $$ = algebra.NewIndexKeyTerm($1, $2)
}

index_term_expr:
index_expr
|
all index_expr
{
    $$ = expression.NewAll($2, false)
}
|
all DISTINCT index_expr
{
    $$ = expression.NewAll($3, true)
}
|
DISTINCT index_expr
{
    $$ = expression.NewAll($2, true)
}
;

index_expr:
expr
{
    exp := $1
    if exp != nil && (!exp.Indexable() || exp.Value() != nil) {
        yylex.Error(fmt.Sprintf("Expression not indexable: %s", exp.String()))
    }

    $$ = exp
}

all:
ALL
|
EACH
;

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

opt_ikattr:
/* empty */
{ $$ = algebra.IK_NONE }
|
ikattr
{ $$ = $1 }
|
ikattr ikattr
{
   attr, valid := algebra.NewIndexKeyTermAttributes($1,$2)
   if !valid {
       yylex.Error("Duplicate or Invalid index key attribute")
   }
   $$ = attr
}
;


ikattr:
ASC
{ $$ = algebra.IK_ASC }
|
DESC
{ $$ = algebra.IK_DESC }
|
MISSING
{ $$ = algebra.IK_MISSING }
;


/*************************************************
 *
 * DROP INDEX
 *
 *************************************************/

drop_index:
DROP PRIMARY INDEX ON named_keyspace_ref opt_index_using
{
    $$ = algebra.NewDropIndex($5, "#primary", $6)
}
|
DROP INDEX simple_named_keyspace_ref DOT index_name opt_index_using
{
    $$ = algebra.NewDropIndex($3, $5, $6)
}
|
DROP INDEX index_name ON named_keyspace_ref opt_index_using
{
    $$ = algebra.NewDropIndex($5, $3, $6)
}
;

/*************************************************
 *
 * ALTER INDEX
 *
 *************************************************/

alter_index:
ALTER INDEX simple_named_keyspace_ref DOT index_name opt_index_using index_with
{
    $$ = algebra.NewAlterIndex($3, $5, $6, $7)
}
|
ALTER INDEX index_name ON named_keyspace_ref opt_index_using index_with
{
    $$ = algebra.NewAlterIndex($5, $3, $6, $7)
}
;

/*************************************************
 *
 * BUILD INDEX
 *
 *************************************************/

build_index:
BUILD INDEX ON named_keyspace_ref LPAREN exprs RPAREN opt_index_using
{
    $$ = algebra.NewBuildIndexes($4, $8, $6...)
}
;

/*************************************************
 *
 * CREATE FUNCTION
 *
 *************************************************/

create_function:
CREATE opt_replace FUNCTION func_name LPAREN parm_list RPAREN func_body
{
    if $8 != nil {
	err := $8.SetVarNames($6)
	if err != nil {
		yylex.Error(err.Error())
    	}
    }
    $$ = algebra.NewCreateFunction($4, $8, $2)
}
;

opt_replace:
/* empty */
{
    $$ = false
}
|
OR REPLACE
{
    $$ = true
}
;

func_name:
short_func_name
|
long_func_name
;

short_func_name:
keyspace_name
{
    name, err := functions.Constructor([]string{$1}, yylex.(*lexer).Namespace(), yylex.(*lexer).QueryContext())
    if err != nil {
	yylex.Error(err.Error())
    }
    $$ = name
}
;

long_func_name:
namespace_term keyspace_name
{
    name, err := functions.Constructor([]string{$1, $2}, yylex.(*lexer).Namespace(), yylex.(*lexer).QueryContext())
    if err != nil {
	yylex.Error(err.Error())
    }
    $$ = name
}
|
namespace_term bucket_name DOT scope_name DOT keyspace_name
{
    name, err := functions.Constructor([]string{$1, $2, $4, $6}, yylex.(*lexer).Namespace(), yylex.(*lexer).QueryContext())
    if err != nil {
	yylex.Error(err.Error())
    }
    $$ = name
}
;

parm_list:
/* empty */
{
    $$ = []string{}
}
|
DOT DOT DOT
{
    $$ = nil
}
|
parameter_terms
;

parameter_terms:
IDENT
{
    $$ = []string{$1}
}
|
parameter_terms COMMA IDENT
{
    $$ = append($1, string($3))
}
;

func_body:
LBRACE expr RBRACE
{
    body, err := inline.NewInlineBody($2)
    if err != nil {
	yylex.Error(err.Error())
    } else {
        $$ = body
    }
}
|
LANGUAGE INLINE AS expr
{
    body, err := inline.NewInlineBody($4)
    if err != nil {
	yylex.Error(err.Error())
    } else {
        $$ = body
    }
}
|
LANGUAGE GOLANG AS STR AT STR
{
    body, err := golang.NewGolangBody($6, $4)
    if err != nil {
        yylex.Error(err.Error())
    } else {
        $$ = body
    }
}
|
LANGUAGE JAVASCRIPT AS STR AT STR
{
    body, err := javascript.NewJavascriptBody($6, $4)
    if err != nil {
        yylex.Error(err.Error())
    } else {
        $$ = body
    }
}
;

/*************************************************
 *
 * DROP FUNCTION
 *
 *************************************************/

drop_function:
DROP FUNCTION func_name
{
    $$ = algebra.NewDropFunction($3)
}
;

/*************************************************
 *
 * EXECUTE FUNCTION
 *
 *************************************************/

execute_function:
EXECUTE FUNCTION func_name LPAREN opt_exprs RPAREN
{
    $$ = algebra.NewExecuteFunction($3, $5)
}
;

/*************************************************
 *
 * UPDATE STATISTICS
 *
 *************************************************/

update_statistics:
UPDATE STATISTICS opt_for named_keyspace_ref LPAREN update_stat_terms RPAREN opt_infer_ustat_with
{
    $$ = algebra.NewUpdateStatistics($4, $6, $8)
}
|
UPDATE STATISTICS opt_for named_keyspace_ref DELETE LPAREN update_stat_terms RPAREN
{
    $$ = algebra.NewUpdateStatisticsDelete($4, $7)
}
|
UPDATE STATISTICS opt_for named_keyspace_ref DELETE ALL
{
    $$ = algebra.NewUpdateStatisticsDelete($4, nil)
}
|
UPDATE STATISTICS opt_for named_keyspace_ref INDEX LPAREN exprs RPAREN opt_index_using opt_infer_ustat_with
{
    $$ = algebra.NewUpdateStatisticsIndex($4, $7, $9, $10)
}
|
UPDATE STATISTICS FOR INDEX simple_named_keyspace_ref DOT index_name opt_index_using opt_infer_ustat_with
{
    $$ = algebra.NewUpdateStatisticsIndex($5, expression.Expressions{expression.NewIdentifier($7)}, $8, $9)
}
|
UPDATE STATISTICS FOR INDEX index_name ON named_keyspace_ref opt_index_using opt_infer_ustat_with
{
    $$ = algebra.NewUpdateStatisticsIndex($7, expression.Expressions{expression.NewIdentifier($5)}, $8, $9)
}
|
ANALYZE opt_keyspace_collection named_keyspace_ref LPAREN update_stat_terms RPAREN opt_infer_ustat_with
{
    $$ = algebra.NewUpdateStatistics($3, $5, $7)
}
|
ANALYZE opt_keyspace_collection named_keyspace_ref DELETE STATISTICS LPAREN update_stat_terms RPAREN
{
    $$ = algebra.NewUpdateStatisticsDelete($3, $7)
}
|
ANALYZE opt_keyspace_collection named_keyspace_ref DELETE STATISTICS
{
    $$ = algebra.NewUpdateStatisticsDelete($3, nil)
}
|
ANALYZE opt_keyspace_collection named_keyspace_ref INDEX LPAREN exprs RPAREN opt_index_using opt_infer_ustat_with
{
    $$ = algebra.NewUpdateStatisticsIndex($3, $6, $8, $9)
}
|
ANALYZE INDEX simple_named_keyspace_ref DOT index_name opt_index_using opt_infer_ustat_with
{
    $$ = algebra.NewUpdateStatisticsIndex($3, expression.Expressions{expression.NewIdentifier($5)}, $6, $7)
}
|
ANALYZE INDEX index_name ON named_keyspace_ref opt_index_using opt_infer_ustat_with
{
    $$ = algebra.NewUpdateStatisticsIndex($5, expression.Expressions{expression.NewIdentifier($3)}, $6, $7)
}
;

opt_for:
/* empty */
|
FOR
;

update_stat_terms:
update_stat_term
{
    $$ = expression.Expressions{$1}
}
|
update_stat_terms COMMA update_stat_term
{
    $$ = append($1, $3)
}
;

update_stat_term:
index_term_expr
;

/*************************************************
 *
 * Path
 *
 *************************************************/

path:
IDENT
{
    $$ = expression.NewIdentifier($1)
}
|
path DOT IDENT
{
    $$ = expression.NewField($1, expression.NewFieldName($3, false))
}
|
path DOT IDENT_ICASE
{
    field := expression.NewField($1, expression.NewFieldName($3, true))
    field.SetCaseInsensitive(true)
    $$ = field
}
|
path DOT LBRACKET expr RBRACKET
{
    $$ = expression.NewField($1, $4)
}
|
path DOT LBRACKET expr RBRACKET_ICASE
{
    field := expression.NewField($1, $4)
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
expr DOT IDENT
{
    $$ = expression.NewField($1, expression.NewFieldName($3, false))
}
|
expr DOT IDENT_ICASE
{
    field := expression.NewField($1, expression.NewFieldName($3, true))
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
expr LBRACKET STAR RBRACKET
{
    $$ = expression.NewArrayStar($1)
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
expr IS valued
{
    $$ = expression.NewIsValued($1)
}
|
expr IS NOT valued
{
    $$ = expression.NewIsNotValued($1)
}
|
EXISTS expr
{
    $$ = expression.NewExists($2)
}
;

valued:
VALUED
|
KNOWN
;

c_expr:
/* Literal */
literal
|
/* Construction */
construction_expr
|
/* Identifier */
IDENT
{
    $$ = expression.NewIdentifier($1)
}
|
/* Identifier */
IDENT_ICASE
{
    ident := expression.NewIdentifier($1)
    ident.SetCaseInsensitive(true)
    $$ = ident
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
paren_expr
|
/* For covering indexes */
COVER
{
   if yylex.(*lexer).parsingStatement() {
	yylex.Error("syntax error")
   }
}
LPAREN expr RPAREN
{
    $$ = expression.NewCover($4)
}
;

b_expr:
c_expr
|
/* Nested */
b_expr DOT IDENT
{
    $$ = expression.NewField($1, expression.NewFieldName($3, false))
}
|
b_expr DOT IDENT_ICASE
{
    field := expression.NewField($1, expression.NewFieldName($3, true))
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
b_expr LBRACKET STAR RBRACKET
{
    $$ = expression.NewArrayStar($1)
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
NUM
{
    $$ = expression.NewConstant(value.NewValue($1))
}
|
INT
{
    $$ = expression.NewConstant(value.NewValue($1))
}
|
STR
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
    $$ = expression.NewObjectConstruct(algebra.MapPairs($2))
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
    $$ = algebra.Pairs{$1}
}
|
members COMMA member
{
    $$ = append($1, $3)
}
;

member:
expr COLON expr
{
    $$ = algebra.NewPair($1, $3, nil)
}
|
expr
{
    name := $1.Alias()
    if name == "" {
        yylex.Error(fmt.Sprintf("Object member missing name or value: %s", $1.String()))
    }

    $$ = algebra.NewPair(expression.NewConstant(name), $1, nil)
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

/*************************************************
 *
 * Parameter
 *
 *************************************************/

param_expr:
NAMED_PARAM
{
    $$ = algebra.NewNamedParameter($1)
    yylex.(*lexer).countParam()
}
|
POSITIONAL_PARAM
{
    p := int($1)
    if $1 > int64(p) {
        yylex.Error(fmt.Sprintf("Positional parameter out of range: $%v.", $1));
    }

    $$ = algebra.NewPositionalParameter(p)
    yylex.(*lexer).countParam()
}
|
NEXT_PARAM
{
    n := yylex.(*lexer).nextParam()
    $$ = algebra.NewPositionalParameter(n)
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
 * NTH_VALUE(expr,n) [FROM FIRST|LAST] [RESPECT|IGNORE NULLS] OVER(....)
 *   requires special handling due to FROM (avoid conflict with query FROM)
 *   example: SELECT SUM(c1) FROM default WHERE ...
 *************************************************/

function_expr:
NTH_VALUE LPAREN exprs RPAREN opt_from_first_last opt_nulls_treatment window_function_details
{
    $$ = nil
    fname := "nth_value"
    f, ok := algebra.GetAggregate(fname, false, false, ($7 != nil))
    if ok {
        if len($3) < f.MinArgs() || len($3) > f.MaxArgs() {
             if f.MinArgs() == f.MaxArgs() {
                   yylex.Error(fmt.Sprintf("Number of arguments to function %s must be %d.", fname, f.MaxArgs()))
             } else {
                   yylex.Error(fmt.Sprintf("Number of arguments to function %s must be between %d and %d.", fname, f.MinArgs(), f.MaxArgs()))
            }
        } else {
            $$ = f.Constructor()($3...)
            if a, ok := $$.(algebra.Aggregate); ok {
                 a.SetAggregateModifiers($5|$6, nil, $7)
            }
        }
    } else {
        return yylex.(*lexer).FatalError(fmt.Sprintf("Invalid function %s", fname))
    }
}
|
function_name LPAREN opt_exprs RPAREN opt_filter opt_nulls_treatment opt_window_function
{
    $$ = nil
    f, ok := expression.GetFunction($1)
    if !ok {
        f, ok = search.GetSearchFunction($1)
    }
    if !ok || $7 != nil {
        f, ok = algebra.GetAggregate($1, false, ($5 != nil), ($7 != nil))
    }

    if ok {
        if ($6 == algebra.AGGREGATE_RESPECTNULLS && !algebra.AggregateHasProperty($1, algebra.AGGREGATE_WINDOW_RESPECTNULLS)) ||
           ($6 == algebra.AGGREGATE_IGNORENULLS && !algebra.AggregateHasProperty($1, algebra.AGGREGATE_WINDOW_IGNORENULLS)) {
            yylex.Error(fmt.Sprintf("RESPECT|IGNORE NULLS syntax is not valid for function %s.", $1))
        } else if ($5 != nil && !algebra.AggregateHasProperty($1, algebra.AGGREGATE_ALLOWS_FILTER)) {
            yylex.Error(fmt.Sprintf("FILTER caluse syntax is not valid for function %s.", $1))
        } else if len($3) < f.MinArgs() || len($3) > f.MaxArgs() {
             if f.MinArgs() == f.MaxArgs() {
                   yylex.Error(fmt.Sprintf("Number of arguments to function %s must be %d.", $1, f.MaxArgs()))
             } else {
                   yylex.Error(fmt.Sprintf("Number of arguments to function %s must be between %d and %d.", $1, f.MinArgs(), f.MaxArgs()))
            }
        } else {
            $$ = f.Constructor()($3...)
            if a, ok := $$.(algebra.Aggregate); ok {
                 a.SetAggregateModifiers($6, $5, $7)
            }
        }
    } else {
	var name functions.FunctionName
	var err errors.Error

        f = nil
        if $5 == nil && $6 == uint32(0) && $7 == nil {
	     name, err = functions.Constructor([]string{$1}, yylex.(*lexer).Namespace(), yylex.(*lexer).QueryContext())
	     if err != nil {
	         return yylex.(*lexer).FatalError(err.Error())
	     }
	     f = expression.GetUserDefinedFunction(name)
        }

	if f != nil {
		$$ = f.Constructor()($3...)
	} else {
		return yylex.(*lexer).FatalError(fmt.Sprintf("Invalid function %s (resolving to %s)", $1, name.Key()))
	}
    }
}
|
function_name LPAREN agg_quantifier expr RPAREN opt_filter opt_window_function
{
    agg, ok := algebra.GetAggregate($1, $3 == algebra.AGGREGATE_DISTINCT, ($6 != nil), ($7 != nil))
    if ok {
        $$ = agg.Constructor()($4)
        if a, ok := $$.(algebra.Aggregate); ok {
             a.SetAggregateModifiers($3, $6, $7)
        }
    } else {
        yylex.Error(fmt.Sprintf("Invalid aggregate function %s.", $1))
    }
}
|
function_name LPAREN STAR RPAREN opt_filter opt_window_function
{
    if strings.ToLower($1) != "count" {
        yylex.Error(fmt.Sprintf("Invalid aggregate function %s(*).", $1))
    } else {
        agg, ok := algebra.GetAggregate($1, false, ($5 != nil), ($6 != nil))
        if ok {
            $$ = agg.Constructor()(nil)
            if a, ok := $$.(algebra.Aggregate); ok {
                 a.SetAggregateModifiers(uint32(0), $5, $6)
            }
        } else {
            yylex.Error(fmt.Sprintf("Invalid aggregate function %s.", $1))
        }
    }
}
|
long_func_name LPAREN opt_exprs RPAREN
{
	f := expression.GetUserDefinedFunction($1)
	if f != nil {
		$$ = f.Constructor()($3...)
	} else {
		return yylex.(*lexer).FatalError(fmt.Sprintf("Invalid function %v", $1.Key()))
	}
}
;

function_name:
IDENT
|
// replace() needs special treatment because of the CREATE OR REPLACE FUNCTION statement
REPLACE
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
|
ANY AND EVERY coll_bindings satisfies END
{
    $$ = expression.NewAnyEvery($4, $5)
}
|
SOME AND EVERY coll_bindings satisfies END
{
    $$ = expression.NewAnyEvery($4, $5)
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
    $$ = expression.NewSimpleBinding($1, $3)
}
|
variable WITHIN expr
{
    $$ = expression.NewBinding("", $1, $3, true)
}
|
variable COLON variable IN expr
{
    $$ = expression.NewBinding($1, $3, $5, false)
}
|
variable COLON variable WITHIN expr
{
    $$ = expression.NewBinding($1, $3, $5, true)
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
|
OBJECT expr COLON expr FOR coll_bindings opt_when END
{
    $$ = expression.NewObject($2, $4, $6, $7)
}
;


/*************************************************
 *
 * Parentheses and subquery
 *
 *************************************************/

paren_expr:
LPAREN expr RPAREN
{
    switch other := $2.(type) {
         case *expression.Identifier:
              other.SetParenthesis(true)
              $$ = other
         case *expression.Field:
              other.SetParenthesis(true)
              $$ = other
         default:
              $$ = other
    }
}
|
LPAREN all_expr RPAREN
{
    $$ = $2
}
|
subquery_expr
{
    $$ = $1
}
;

subquery_expr:
CORRELATED
{
    if yylex.(*lexer).parsingStatement() {
        yylex.Error("syntax error")
    }
}
LPAREN fullselect RPAREN
{
    $$ = algebra.NewSubquery($4)
    $$.Select().SetCorrelated()
}
|
LPAREN fullselect RPAREN
{
    $$ = algebra.NewSubquery($2)
}
;

/*************************************************
 *
 * Top-level expression input / parsing.
 *
 *************************************************/

expr_input:
expr
|
all_expr
;

all_expr:
all expr
{
    $$ = expression.NewAll($2, false)
}
|
all DISTINCT expr
{
    $$ = expression.NewAll($3, true)
}
|
DISTINCT expr
{
    $$ = expression.NewAll($2, true)
}
;

/*************************************************
 *
 * WINDOW clause
 *
 *************************************************/

opt_window_clause:
/* empty */
{ $$ = nil }
|
WINDOW window_list
{
    $$ = $2
}
;

window_list:
window_term
{
    $$ = algebra.WindowTerms{$1}
}
|
window_list COMMA window_term
{
    $$ = append($1, $3)
}
;

window_term:
IDENT AS window_specification
{
    $$ = $3
    $$.SetAsWindowName($1)
}
;

window_specification:
LPAREN opt_window_name opt_window_partition opt_order_by opt_window_frame RPAREN
{
    $$ = algebra.NewWindowTerm($2,$3,$4,$5, false)
}
;

opt_window_name:
/* empty */
{ $$ = "" }
|
IDENT
;

opt_window_partition:
/* empty */
{ $$ = nil }
|
PARTITION BY exprs
{ $$ = $3 }
;

opt_window_frame:
/* empty */
{
    $$ = nil
}
|
window_frame_modifier window_frame_extents opt_window_frame_exclusion
{
    $$ = algebra.NewWindowFrame($1|$3, $2)
}
;

window_frame_modifier:
ROWS
{
    $$ = algebra.WINDOW_FRAME_ROWS
}
|
RANGE
{
    $$ = algebra.WINDOW_FRAME_RANGE
}
|
GROUPS
{
    $$ = algebra.WINDOW_FRAME_GROUPS
}
;

opt_window_frame_exclusion:
/* empty */
{
     $$ = uint32(0)
}
|
EXCLUDE NO OTHERS
{
     $$ = uint32(0)
}
|
EXCLUDE CURRENT ROW
{
     $$ = algebra.WINDOW_FRAME_EXCLUDE_CURRENT_ROW
}
|
EXCLUDE TIES
{
     $$ = algebra.WINDOW_FRAME_EXCLUDE_TIES
}
|
EXCLUDE GROUP
{
     $$ = algebra.WINDOW_FRAME_EXCLUDE_GROUP
}
;

window_frame_extents:
window_frame_extent
{
    $$ = algebra.WindowFrameExtents{$1}
}
|
BETWEEN window_frame_extent AND window_frame_extent
{
    $$ = algebra.WindowFrameExtents{$2, $4}
}
;

window_frame_extent:
UNBOUNDED PRECEDING
{
    $$ = algebra.NewWindowFrameExtent(nil, algebra.WINDOW_FRAME_UNBOUNDED_PRECEDING)
}
|
UNBOUNDED FOLLOWING
{
    $$ = algebra.NewWindowFrameExtent(nil, algebra.WINDOW_FRAME_UNBOUNDED_FOLLOWING)
}
|
CURRENT ROW
{
    $$ = algebra.NewWindowFrameExtent(nil, algebra.WINDOW_FRAME_CURRENT_ROW)
}
|
expr window_frame_valexpr_modifier
{
    $$ = algebra.NewWindowFrameExtent($1, $2)
}
;

window_frame_valexpr_modifier:
PRECEDING
{
    $$ = algebra.WINDOW_FRAME_VALUE_PRECEDING
}
|
FOLLOWING
{
    $$ = algebra.WINDOW_FRAME_VALUE_FOLLOWING
}
;

opt_nulls_treatment:
/* empty */
{ $$ = uint32(0) }
|
nulls_treatment
{ $$ = $1 }
;

nulls_treatment:
RESPECT NULLS
{ $$ = algebra.AGGREGATE_RESPECTNULLS }
|
IGNORE NULLS
{ $$ = algebra.AGGREGATE_IGNORENULLS }
;

opt_from_first_last:
/* empty */
{ $$ = uint32(0) }
|
FROM first_last
{
    if $2 {
         $$ = algebra.AGGREGATE_FROMLAST
    } else {
         $$ = algebra.AGGREGATE_FROMFIRST
    }
}
;

agg_quantifier:
ALL
{
   $$ = uint32(0)
}
|
DISTINCT
{
   $$ = algebra.AGGREGATE_DISTINCT
}
;

opt_filter:
/* empty */
{ $$ = nil }
|
FILTER LPAREN where RPAREN
{ $$ = $3 }
;

opt_window_function:
/* empty */
{ $$ = nil }
|
window_function_details
{ $$ = $1 }
;

window_function_details:
OVER IDENT
{
    $$ = algebra.NewWindowTerm($2,nil,nil, nil, true)
}
|
OVER window_specification
{
    $$ = $2
}
;

/*************************************************
 *
 * <START|BEGIN> <TRANSACTION | TRAN | WORK> [ISOLATION LEVEL READ COMMITED]
 *
 *************************************************/

start_transaction:
start_or_begin transaction opt_isolation_level
{
    $$ = algebra.NewStartTransaction($3)
}
;

commit_transaction:
COMMIT opt_transaction
{
    $$ = algebra.NewCommitTransaction()
}
;

rollback_transaction:
ROLLBACK opt_transaction opt_savepoint
{
    $$ = algebra.NewRollbackTransaction($3)
}
;

start_or_begin:
START
|
BEGIN
;

opt_transaction:
/* empty */
{}
|
transaction
;

transaction:
TRAN
|
TRANSACTION
|
WORK
;

opt_savepoint:
/* empty */
{
    $$ = ""
}
|
TO SAVEPOINT savepoint_name
{
    $$ = $3
}
;

savepoint_name:
IDENT
{
    $$ = $1
}

opt_isolation_level:
/* empty */
{
    $$ = datastore.IL_READ_COMMITTED
}
|
isolation_level
{
    $$ = $1
}
;

isolation_level:
ISOLATION LEVEL isolation_val
{
    $$ = $3
}
;

isolation_val:
READ COMMITTED
{
    $$ = datastore.IL_READ_COMMITTED
}
;

set_transaction_isolation:
SET TRANSACTION isolation_level
{
    $$ = algebra.NewTransactionIsolation($3)
}
;

savepoint:
SAVEPOINT savepoint_name
{
    $$ = algebra.NewSavepoint($2)
}
;
