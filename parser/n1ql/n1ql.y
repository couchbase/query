%{
package n1ql

import "fmt"
import "math"
import "strings"
import "github.com/couchbase/clog"
import "github.com/couchbase/query/algebra"
import "github.com/couchbase/query/auth"
import "github.com/couchbase/query/datastore"
import "github.com/couchbase/query/errors"
import "github.com/couchbase/query/expression"
import "github.com/couchbase/query/expression/search"
import "github.com/couchbase/query/functions"
import "github.com/couchbase/query/functions/bridge"
import "github.com/couchbase/query/sequences"
import "github.com/couchbase/query/value"

func logDebugGrammar(format string, v ...interface{}) {
    clog.To("PARSER", format, v...)
}

type nameValueContext struct {
    name   string
    value  interface{}
    line   int
    column int
}

%}

%union {
s                string
u32              uint32
n                int64
f                float64
b                bool

ss               []string
expr             expression.Expression
exprs            expression.Expressions
subquery         *algebra.Subquery
whenTerm         *expression.WhenTerm
whenTerms        expression.WhenTerms
binding          *expression.Binding
bindings         expression.Bindings
with             expression.With
withs            expression.Withs
withclause       *algebra.WithClause
cyclecheck       *algebra.CycleCheck
dimensions       []expression.Bindings

node             algebra.Node
statement        algebra.Statement

setopType        algebra.SetOpType

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

functionName     functions.FunctionName
functionBody     functions.FunctionBody

identifier       *expression.Identifier

optimHintArr     []algebra.OptimHint
optimHints       *algebra.OptimHints

vpair            *nameValueContext
vpairs           []*nameValueContext
// token offset into the statement
tokOffset        int

// token location in the statement
line   int
column int
}

%token _ERROR_  // used by the scanner to flag errors
%token _INDEX_CONDITION
%token _INDEX_KEY
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
%token CACHE
%token CASE
%token CAST
%token CLUSTER
%token COLLATE
%token COLLECTION
%token COMMIT
%token COMMITTED
%token CONNECT
%token CONTINUE
%token _CORRELATED
%token _COVER
%token CREATE
%token CURRENT
%token CYCLE
%token DATABASE
%token DATASET
%token DATASTORE
%token DECLARE
%token DECREMENT
%token DEFAULT
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
%token FLATTEN_KEYS
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
%token LATERAL
%token LEFT
%token LET
%token LETTING
%token LEVEL
%token LIKE
%token ESCAPE
%token LIMIT
%token LSM
%token MAP
%token MAPPING
%token MATCHED
%token MATERIALIZED
%token MAXVALUE
%token MERGE
%token MINUS
%token MISSING
%token MINVALUE
%token NAMESPACE
%token NAMESPACE_ID
%token NEST
%token NEXT
%token NEXTVAL
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
%token PREV
%token PREVVAL
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
%token RECURSIVE
%token REDUCE
%token RENAME
%token REPLACE
%token RESPECT
%token RESTART
%token RESTRICT
%token RETURN
%token RETURNING
%token REVOKE
%token RIGHT
%token ROLE
%token ROLES
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
%token SEQUENCE
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
%token USERS
%token USING
%token VALIDATE
%token VALUE
%token VALUED
%token VALUES
%token VECTOR
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

%token INT NUM STR IDENT IDENT_ICASE NAMED_PARAM POSITIONAL_PARAM NEXT_PARAM OPTIM_HINTS
%token RANDOM_ELEMENT
%token LPAREN RPAREN
%token LBRACE RBRACE LBRACKET RBRACKET RBRACKET_ICASE
%token COMMA COLON

/* Precedence: lowest to highest */
%left           ORDER
%left           UNION INTERSECT EXCEPT
%left           JOIN NEST UNNEST FLATTEN INNER LEFT RIGHT
%left           OR
%left           AND
%right          NOT
%nonassoc       EQ DEQ NE
%nonassoc       LT GT LE GE
%nonassoc       LIKE
%nonassoc       ESCAPE
%nonassoc       BETWEEN
%nonassoc       IN WITHIN
%nonassoc       EXISTS
%nonassoc       IS                              /* IS NULL, IS MISSING, IS VALUED, IS NOT NULL, etc. */
%left           FROM                            /* IS [NOT] DISTINCT FROM */
%left           CONCAT
%left           PLUS MINUS
%left           STAR DIV MOD POW

/* Unary operators */
%right          _COVER _INDEX_KEY _INDEX_CONDITION
%left           ALL
%right          UMINUS
%left           DOT LBRACKET RBRACKET

/* Override precedence */
%left           LPAREN RPAREN


/* Types */
%type <s>                STR
%type <s>                IDENT IDENT_ICASE NAMESPACE_ID DEFAULT USER USERS permitted_identifiers SEQUENCE VECTOR
%type <identifier>       ident ident_icase
%type <s>                REPLACE
%type <s>                NAMED_PARAM
%type <f>                NUM
%type <n>                INT
%type <n>                POSITIONAL_PARAM NEXT_PARAM
%type <s>                OPTIM_HINTS
%type <expr>             literal construction_expr opt_execute_using object array
%type <expr>             param_expr
%type <pair>             member
%type <pairs>            members opt_members

%type <expr>             expr c_expr b_expr
%type <exprs>            exprs opt_exprs
%type <binding>          binding
%type <bindings>         bindings
%type <with>             with_term
%type <withs>            with_list

%type <s>                alias as_alias opt_as_alias variable opt_name opt_window_name

%type <expr>             case_expr simple_or_searched_case simple_case searched_case opt_else
%type <whenTerms>        when_thens

%type <expr>             collection_expr collection_cond collection_xform
%type <binding>          coll_binding
%type <bindings>         coll_bindings
%type <expr>             satisfies
%type <expr>             opt_when

%type <expr>             function_expr function_meta_expr
%type <identifier>       function_name

%type <functionName>     func_name long_func_name short_func_name
%type <ss>               opt_parm_list parameter_terms
%type <functionBody>     func_body
%type <expr>             opt_replace

%type <expr>             paren_expr
%type <subquery>         subquery_expr
%type <setopType>        setop

%type <fullselect>       fullselect
%type <subresult>        select_term select_terms
%type <subselect>        subselect
%type <subselect>        select_from
%type <subselect>        from_select
%type <fromTerm>         from_term from opt_from from_terms
%type <simpleFromTerm>   simple_from_term
%type <keyspaceTerm>     keyspace_term
%type <keyspacePath>     keyspace_path
%type <b>                opt_join_type opt_quantifier
%type <path>             path
%type <s>                namespace_term namespace_name path_part keyspace_name
%type <use>              opt_use opt_use_del_upd opt_use_merge use_options use_keys use_index join_hint
%type <joinHint>         use_hash_option
%type <expr>             on_keys on_key
%type <indexRefs>        index_refs
%type <indexRef>         index_ref
%type <bindings>         opt_let let
%type <withclause>       with
%type <expr>             opt_where where opt_filter
%type <group>            opt_group group
%type <expr>             opt_group_as
%type <bindings>         opt_letting letting
%type <expr>             opt_having having
%type <resultTerm>       project
%type <resultTerms>      projects
%type <projection>       projection
%type <order>            order_by opt_order_by
%type <sortTerm>         sort_term
%type <sortTerms>        sort_terms
%type <groupTerm>        group_term
%type <groupTerms>       group_terms
%type <expr>             limit opt_limit
%type <expr>             offset opt_offset
%type <expr>             dir opt_dir
%type <b>                opt_if_not_exists opt_if_exists
%type <b>                opt_vector
%type <statement>        stmt_body
%type <statement>        stmt advise explain explain_function prepare execute select_stmt dml_stmt ddl_stmt
%type <statement>        infer
%type <statement>        update_statistics
%type <statement>        insert upsert delete update merge
%type <statement>        index_stmt create_index drop_index alter_index build_index
%type <statement>        scope_stmt create_scope drop_scope
%type <statement>        transaction_stmt start_transaction commit_transaction rollback_transaction
%type <statement>        savepoint set_transaction_isolation
%type <statement>        collection_stmt create_collection drop_collection flush_collection
%type <statement>        role_stmt grant_role revoke_role
%type <statement>        user_stmt create_user alter_user drop_user
%type <statement>        group_stmt create_group alter_group drop_group
%type <statement>        function_stmt create_function drop_function execute_function
%type <statement>        bucket_stmt create_bucket alter_bucket drop_bucket

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
%type <mergeActions>     opt_merge_actions opt_merge_delete_insert
%type <mergeUpdate>      merge_update
%type <mergeDelete>      merge_delete
%type <mergeInsert>      merge_insert opt_merge_insert

%type <s>                index_name opt_index_name
%type <keyspaceRef>      simple_named_keyspace_ref named_keyspace_ref
%type <scopeRef>         named_scope_ref
%type <partitionTerm>    opt_index_partition
%type <indexType>        index_using opt_index_using
%type <expr>             index_term_expr opt_index_where
%type <indexKeyTerm>     index_term flatten_keys_expr
%type <indexKeyTerms>    index_terms flatten_keys_exprs opt_flatten_keys_exprs
%type <expr>             expr_input all_expr

%type <exprs>            update_stat_terms
%type <expr>             update_stat_term

%type <inferenceType>    opt_infer_using

%type <ss>               user_list groups
%type <keyspaceRefs>     keyspace_scope_list
%type <keyspaceRef>      keyspace_scope
%type <ss>               role_list group_role_list
%type <s>                role_name group_role_list_item
%type <s>                user group_name

%type <u32>              opt_ikattr ikattr
%type <expr>             opt_order_nulls
%type <b>                first_last

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

%type <optimHints>          hints_input opt_optim_hints
%type <optimHintArr>        optim_hints optim_hint
%type <ss>                  opt_hint_args hint_args

%type <val>                with_clause opt_with_clause opt_option_clause opt_def_with_clause

%type <exprs>              opt_exclude

%type <cyclecheck>         opt_cycle_clause
%type <statement>          sequence_stmt create_sequence drop_sequence alter_sequence
%type <keyspacePath>       sequence_full_name
%type <vpairs>             sequence_name_options
%type <vpair>              sequence_name_option
%type <s>                  opt_namespace_name sequence_object_name
%type <ss>                 sequence_next sequence_prev
%type <expr>               sequence_expr

%type <vpair>  start_with increment_by maxvalue minvalue cache cycle restart_with seq_alter_option seq_create_option sequence_with
%type <vpairs> seq_alter_options opt_seq_create_options
%type <vpair>  user_opt group_opt
%type <vpairs> user_opts group_opts

%type <expr> param_or_str

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
|
hints_input
{
    yylex.(*lexer).setOptimHints($1)
}
;

permitted_identifiers:
IDENT
|
DEFAULT
|
USER
|
USERS
|
SEQUENCE
|
VECTOR
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
explain_function
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
user_stmt
|
group_stmt
|
role_stmt
|
function_stmt
|
transaction_stmt
|
sequence_stmt
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

explain_function:
EXPLAIN FUNCTION func_name
{
    $$ = algebra.NewExplainFunction($3)
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
permitted_identifiers from_or_as
{
    $$ = $1
}
|
_invalid_case_insensitive_identifier from_or_as
{
    return yylex.(*lexer).FatalError("Prepared identifier must be case sensitive", $<line>1, $<column>1)
}
|
STR from_or_as
{
    $$ = $1
}
;

_invalid_case_insensitive_identifier:
IDENT_ICASE
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
EXECUTE expr opt_execute_using
{
    if id, ok := $2.(*expression.Identifier); ok {
        if id.CaseInsensitive() {
            return yylex.(*lexer).FatalError("Prepared identifier must be case sensitive", $<line>2, $<column>2)
        }
    }
    $$ = algebra.NewExecute($2, $3)
}
;

opt_execute_using:
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
INFER keyspace_collection simple_keyspace_ref opt_infer_using opt_with_clause
{
    $$ = algebra.NewInferKeyspace($3, $4, $5)
}
|
INFER keyspace_path opt_as_alias opt_infer_using opt_with_clause
{
    kr := algebra.NewKeyspaceRefFromPath($2, $3)
    $$ = algebra.NewInferKeyspace(kr, $4, $5)
}
|
INFER expr opt_infer_using opt_with_clause
{
    var pth *algebra.Path
    var err errors.Error

    switch other := $2.(type) {
    case *expression.Identifier:
        kr := algebra.NewKeyspaceRefWithContext(other.Identifier(), "", yylex.(*lexer).Namespace(), yylex.(*lexer).QueryContext())
        $$ = algebra.NewInferKeyspace(kr, $3, $4)
    case *expression.Field:
        p := other.Path()
        if len(p) == 2 {
        pth, err = algebra.NewPathFromElementsWithContext([]string{ p[0], p[1] }, yylex.(*lexer).Namespace(),
                                 yylex.(*lexer).QueryContext())
            if err != nil {
                return yylex.(*lexer).FatalError(err.Error(), $<line>2, $<column>2)
            }
        } else if len(p) == 3 {
            pth = algebra.NewPathLong(yylex.(*lexer).Namespace(), p[0], p[1], p[2])
        } else {
            yylex.(*lexer).FatalError("syntax error", $<line>2, $<column>2)
        }
        $$ = algebra.NewInferKeyspace(algebra.NewKeyspaceRefFromPath(pth, ""), $3, $4)
    default:
        $$ = algebra.NewInferExpression($2, $3, $4)
    }
}
;

keyspace_collection:
KEYSPACE
|
COLLECTION
;

opt_keyspace_collection:
/* empty */
{
}
|
keyspace_collection
;

opt_infer_using:
/* empty */
{
    $$ = datastore.INF_DEFAULT
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
bucket_stmt
|
scope_stmt
|
collection_stmt
;

user_stmt:
create_user
|
alter_user
|
drop_user
;

group_stmt:
create_group
|
alter_group
|
drop_group
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

bucket_stmt:
create_bucket
|
alter_bucket
|
drop_bucket
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
    $$ = algebra.NewSelect($1, nil, $2, nil, nil) /* OFFSET precedes LIMIT */
}
|
select_terms opt_order_by limit opt_offset
{
    $$ = algebra.NewSelect($1, nil, $2, $4, $3) /* OFFSET precedes LIMIT */
}
|
select_terms opt_order_by offset opt_limit
{
    $$ = algebra.NewSelect($1, nil, $2, $3, $4) /* OFFSET precedes LIMIT */
}
|
with select_terms opt_order_by
{
    $$ = algebra.NewSelect($2, $1, $3, nil, nil) /* OFFSET precedes LIMIT */
}
|
with select_terms opt_order_by limit opt_offset
{
    $$ = algebra.NewSelect($2, $1, $3, $5, $4) /* OFFSET precedes LIMIT */
}
|
with select_terms opt_order_by offset opt_limit
{
    $$ = algebra.NewSelect($2, $1, $3, $4, $5) /* OFFSET precedes LIMIT */
}
;

select_terms:
subselect
{
    $$ = $1
}
|
select_terms setop select_term
{
    $$ = algebra.NewSetOp($1, $3, $2)
    if $$ == nil {
       yylex.(*lexer).ErrorWithContext("Unexpected Set Operation",$<line>2,$<column>2)
    }
}
|
subquery_expr setop select_term
{
    $1.Select().SetUnderSetOp()
    left_term := algebra.NewSelectTerm($1.Select())
    $$ = algebra.NewSetOp(left_term, $3, $2)
    if $$ == nil {
       yylex.(*lexer).ErrorWithContext("Unexpected Set Operation",$<line>2,$<column>2)
    }
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
    // all current uses of select_term is under setop
    $1.Select().SetUnderSetOp()
    $$ = algebra.NewSelectTerm($1.Select())
}
;

subselect:
from_select
|
select_from
;

from_select:
from opt_let opt_where opt_group opt_window_clause SELECT opt_optim_hints projection
{
    $$ = algebra.NewSubselect($1, $2, $3, $4, $5, $8, $7)
}
;

select_from:
SELECT opt_optim_hints projection opt_from opt_let opt_where opt_group opt_window_clause
{
    $$ = algebra.NewSubselect($4, $5, $6, $7, $8, $3, $2)
}
;

setop:
UNION
{
    $$ = algebra.SETOP_UNION
}
|
UNION ALL
{
    $$ = algebra.SETOP_UNION_ALL
}
|
INTERSECT
{
    $$ = algebra.SETOP_INTERSECT
}
|
INTERSECT ALL
{
    $$ = algebra.SETOP_INTERSECT_ALL
}
|
EXCEPT
{
    $$ = algebra.SETOP_EXCEPT
}
|
EXCEPT ALL
{
    $$ = algebra.SETOP_EXCEPT_ALL
}
;

/*************************************************
 *
 * Optimizer Hints
 *
 *************************************************/
opt_optim_hints:
/* empty */
{
    $$ = nil
}
|
OPTIM_HINTS
{
    $$ = ParseOptimHints($1)
}
;

hints_input:
PLUS optim_hints
{
    $$ = algebra.NewOptimHints($2, false)
}
|
PLUS object
{
    hints := algebra.ParseObjectHints($2)
    $$ = algebra.NewOptimHints(hints, true)
}
;

optim_hints:
optim_hint
{
    $$ = $1
}
|
optim_hints optim_hint
{
    $$ = append($1, $2...)
}
;

optim_hint:
permitted_identifiers
{
    $$ = algebra.NewOptimHint($1, nil)
}
|
permitted_identifiers LPAREN opt_hint_args RPAREN
{
    $$ = algebra.NewOptimHint($1, $3)
}
|
INDEX LPAREN opt_hint_args RPAREN
{
    $$ = algebra.NewOptimHint("index", $3)
}
;

opt_hint_args:
/* empty */
{
    $$ = []string{}
}
|
hint_args
{
    $$ = $1
}
;

hint_args:
permitted_identifiers
{
    $$ = []string{$1}
}
|
permitted_identifiers DIV BUILD
{
    $$ = []string{$1 + "/BUILD"}
}
|
permitted_identifiers DIV PROBE
{
    $$ = []string{$1 + "/PROBE"}
}
|
hint_args permitted_identifiers
{
    $$ = append($1, $2)
}
;

/*************************************************
 *
 * Projection clause
 *
 *************************************************/

projection:
opt_quantifier projects opt_exclude
{
    $$ = algebra.NewProjection($1, $2, $3)
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

opt_exclude:
/* empty */
{
    $$ = nil
}
|
EXCLUDE exprs
{
    $$ = $2
}
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
    $$.Expression().ExprBase().SetErrorContext($<line>1,$<column>1)
}
|
expr DOT STAR
{
    switch e := $1.(type) {
      case *expression.All:
        if e.Distinct() {
          return yylex.(*lexer).FatalError("syntax error - DISTINCT out of place", $<line>1, $<column>1)
        } else {
          return yylex.(*lexer).FatalError("syntax error - ALL out of place", $<line>1, $<column>1)
        }
    }
    $$ = algebra.NewResultTerm($1, true, "")
    if $1 != nil {
        $$.Expression().ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
    }
}
|
expr opt_as_alias
{
    switch e := $1.(type) {
      case *expression.All:
        if e.Distinct() {
          return yylex.(*lexer).FatalError("syntax error - DISTINCT out of place", $<line>1, $<column>1)
        } else {
          return yylex.(*lexer).FatalError("syntax error - ALL out of place", $<line>1, $<column>1)
        }
    }
    $$ = algebra.NewResultTerm($1, false, $2)
    $$.Expression().ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
    if $2 != "" {
        $$.Expression().ExprBase().SetAliasErrorContext($<line>2,$<column>2)
    }
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
permitted_identifiers
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
FROM from_terms
{
    $$ = $2
}
;

from_terms:
from_term
{
    $$ = $1
}
|
from_terms COMMA from_term
{
    // enforce the RHS being a SimpleFromTerm here so we can produce a more meaningful error
    switch rterm := $3.(type) {
    case algebra.SimpleFromTerm:
        rterm.SetAnsiJoin()
        rterm.SetCommaJoin()
        $$ = algebra.NewAnsiJoin($1, false, rterm, nil)
    default:
        yylex.(*lexer).ErrorWithContext(
            fmt.Sprintf("Right side (%s) of a COMMA in a FROM clause must be a simple term or sub-query", $3.String()),
            $<line>3, $<column>3)
    }
}
|
from_terms COMMA LATERAL from_term
{
    // enforce the RHS being a SimpleFromTerm here so we can produce a more meaningful error
    switch rterm := $4.(type) {
    case algebra.SimpleFromTerm:
        rterm.SetAnsiJoin()
        rterm.SetCommaJoin()
        rterm.SetLateralJoin()
        $$ = algebra.NewAnsiJoin($1, false, rterm, nil)
    default:
        yylex.(*lexer).ErrorWithContext(
            fmt.Sprintf("Right side (%s) of a COMMA in a FROM clause must be a simple term or sub-query", $4.String()),
            $<line>4, $<column>4)
    }
}
;

from_term:
simple_from_term
{
    $$ = $1
}
|
from_term opt_join_type JOIN simple_from_term on_keys
{
    ksterm := algebra.GetKeyspaceTerm($4)
    if ksterm == nil {
        yylex.(*lexer).ErrorWithContext("JOIN must be done on a keyspace", $<line>4, $<column>4)
    } else {
        ksterm.SetJoinKeys($5)
        ksterm.SetValidateKeys($5.HasExprFlag(expression.EXPR_VALIDATE_KEYS))
    }
    $$ = algebra.NewJoin($1, $2, ksterm)
}
|
from_term opt_join_type JOIN LATERAL simple_from_term on_keys
{
    return yylex.(*lexer).FatalError(fmt.Sprintf("LATERAL cannot be specified in lookup join with ON KEYS clause (%s)",
      $5.Alias()),$<line>4,$<column>4)
}
|
from_term opt_join_type JOIN simple_from_term on_key FOR permitted_identifiers
{
    ksterm := algebra.GetKeyspaceTerm($4)
    if ksterm == nil {
        yylex.(*lexer).ErrorWithContext("JOIN must be done on a keyspace", $<line>4, $<column>4)
    } else {
        ksterm.SetIndexJoinNest()
        ksterm.SetJoinKeys($5)
        ksterm.SetValidateKeys($5.HasExprFlag(expression.EXPR_VALIDATE_KEYS))
    }
    $$ = algebra.NewIndexJoin($1, $2, ksterm, $7)
}
|
from_term opt_join_type JOIN LATERAL simple_from_term on_key FOR permitted_identifiers
{
    return yylex.(*lexer).FatalError(fmt.Sprintf("LATERAL cannot be specified in index join with ON KEY...FOR clause (%s)",
      $5.Alias()),$<line>4,$<column>4)
}
|
from_term opt_join_type NEST simple_from_term on_keys
{
    ksterm := algebra.GetKeyspaceTerm($4)
    if ksterm == nil {
        yylex.(*lexer).ErrorWithContext("NEST must be done on a keyspace", $<line>4, $<column>4)
    } else {
        ksterm.SetJoinKeys($5)
        ksterm.SetValidateKeys($5.HasExprFlag(expression.EXPR_VALIDATE_KEYS))
    }
    $$ = algebra.NewNest($1, $2, ksterm)
}
|
from_term opt_join_type NEST LATERAL simple_from_term on_keys
{
    return yylex.(*lexer).FatalError(fmt.Sprintf("LATERAL cannot be specified in lookup nest with ON KEYS clause (%s)",
      $5.Alias()),$<line>4,$<column>4)
}
|
from_term opt_join_type NEST simple_from_term on_key FOR permitted_identifiers
{
    ksterm := algebra.GetKeyspaceTerm($4)
    if ksterm == nil {
        yylex.(*lexer).ErrorWithContext("NEST must be done on a keyspace", $<line>4, $<column>4)
    } else {
        ksterm.SetIndexJoinNest()
        ksterm.SetJoinKeys($5)
        ksterm.SetValidateKeys($5.HasExprFlag(expression.EXPR_VALIDATE_KEYS))
    }
    $$ = algebra.NewIndexNest($1, $2, ksterm, $7)
}
|
from_term opt_join_type NEST LATERAL simple_from_term on_key FOR permitted_identifiers
{
    return yylex.(*lexer).FatalError(fmt.Sprintf("LATERAL cannot be specified in index nest with ON KEY...FOR clause (%s)",
      $5.Alias()),$<line>4,$<column>4)
}
|
from_term opt_join_type unnest expr opt_as_alias
{
    if $5 != "" {
        $4.ExprBase().SetAliasErrorContext($<line>5, $<column>5)
    }
    $$ = algebra.NewUnnest($1, $2, $4, $5)
}
|
from_term opt_join_type JOIN simple_from_term ON expr
{
    $4.SetAnsiJoin()
    $$ = algebra.NewAnsiJoin($1, $2, $4, $6)
}
|
from_term opt_join_type JOIN LATERAL simple_from_term ON expr
{
    $5.SetAnsiJoin()
    $5.SetLateralJoin()
    $$ = algebra.NewAnsiJoin($1, $2, $5, $7)
}
|
from_term opt_join_type NEST simple_from_term ON expr
{
    $4.SetAnsiNest()
    $$ = algebra.NewAnsiNest($1, $2, $4, $6)
}
|
from_term opt_join_type NEST LATERAL simple_from_term ON expr
{
    $5.SetAnsiNest()
    $5.SetLateralJoin()
    $$ = algebra.NewAnsiNest($1, $2, $5, $7)
}
|
simple_from_term RIGHT opt_outer JOIN simple_from_term ON expr
{
    $1.SetAnsiJoin()
    $$ = algebra.NewAnsiRightJoin($1, $5, $7)
}
|
simple_from_term RIGHT opt_outer JOIN LATERAL simple_from_term ON expr
{
    return yylex.(*lexer).FatalError(fmt.Sprintf("LATERAL cannot be specified in RIGHT OUTER JOIN (%s)",
      $6.Alias()),$<line>5,$<column>5)
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
    l := $<line>1
    c := $<column>1
    if $2 != "" {
        l = $<line>2
        c = $<column>2
        $1.ExprBase().SetAliasErrorContext(l, c)
    }
    isExpr := false
    switch other := $1.(type) {
        case *algebra.Subquery:
            if $2 == "" {
                return yylex.(*lexer).FatalError("Subquery in FROM clause must have an alias.", $<line>1, $<column>1)
            }
            if $3.Keys() != nil || $3.Indexes() != nil {
                return yylex.(*lexer).FatalError("FROM Subquery cannot have USE KEYS or USE INDEX.", $<line>3, $<column>3)
            }
            sqterm := algebra.NewSubqueryTerm(other.Select(), $2, $3.JoinHint())
            sqterm.SetErrorContext(l, c)
            $$ = sqterm
        case *expression.Identifier:
            if other.CaseInsensitive() {
                return yylex.(*lexer).FatalError("Keyspace term must be case sensitive", $<line>1, $<column>1)
            }
            ksterm := algebra.NewKeyspaceTermFromPath(algebra.NewPathWithContext(other.Alias(), yylex.(*lexer).Namespace(),
                yylex.(*lexer).QueryContext()), $2, $3.Keys(), $3.Indexes())
            ksterm.SetValidateKeys($3.ValidateKeys())
            ksterm.SetErrorContext($<line>1, $<column>1)
            $$ = algebra.NewExpressionTerm(other, $2, ksterm, other.Parenthesis() == false, $3.JoinHint())
        case *algebra.NamedParameter, *algebra.PositionalParameter:
            if $3.Indexes() == nil {
                if $3.Keys() != nil {
                    ksterm := algebra.NewKeyspaceTermFromExpression(other, $2, $3.Keys(), $3.Indexes(), $3.JoinHint())
                    ksterm.SetValidateKeys($3.ValidateKeys())
                    ksterm.SetErrorContext(l, c)
                    $$ = ksterm
                } else {
                    $$ = algebra.NewExpressionTerm(other, $2, nil, false, $3.JoinHint())
                }
            } else {
                return yylex.(*lexer).FatalError("FROM <placeholder> cannot have USE INDEX.", $<line>1, $<column>1)
            }
        case *expression.Field:
            path := other.Path()
            if len(path) == 2 {
                longPath, err := algebra.NewPathFromElementsWithContext([]string{ path[0], path[1] }, yylex.(*lexer).Namespace(),
                    yylex.(*lexer).QueryContext())
                if err != nil {
                    isExpr = true
                } else {
                    if cs := other.GetFirstCaseSensitivePathElement(); cs != nil {
                        l, c := cs.GetErrorContext()
                        return yylex.(*lexer).FatalError("Keyspace term must be case sensitive", l, c)
                    }
                    ksterm := algebra.NewKeyspaceTermFromPath(longPath, $2, $3.Keys(), $3.Indexes())
                    ksterm.SetFromTwoParts()
                    ksterm.SetValidateKeys($3.ValidateKeys())
                    ksterm.SetErrorContext($<line>1, $<column>1)
                    $$ = algebra.NewExpressionTerm(other, $2, ksterm, other.Parenthesis() == false, $3.JoinHint())
                }
            } else if len(path) == 3 {
                if cs := other.GetFirstCaseSensitivePathElement(); cs != nil {
                    l, c := cs.GetErrorContext()
                    return yylex.(*lexer).FatalError("Keyspace term must be case sensitive", l, c)
                }
                ksterm := algebra.NewKeyspaceTermFromPath(algebra.NewPathLong(yylex.(*lexer).Namespace(), path[0], path[1],
                    path[2]), $2, $3.Keys(), $3.Indexes())
                ksterm.SetValidateKeys($3.ValidateKeys())
                ksterm.SetErrorContext($<line>1, $<column>1)
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
            return yylex.(*lexer).FatalError("FROM Expression cannot have USE KEYS or USE INDEX.", $<line>1, $<column>1)
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
    ksterm.SetValidateKeys($3.ValidateKeys())
    ksterm.SetErrorContext($<line>1, $<column>1)
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
namespace_term path_part DOT path_part DOT keyspace_name
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

path_part:
permitted_identifiers
;

keyspace_name:
permitted_identifiers
{
  $$ = strings.TrimSpace($1)
  if $$ != $1 || $$ == "" {
    return yylex.(*lexer).FatalError(fmt.Sprintf("Invalid identifier '%v'", $1), $<line>1, $<column>1)
  }
}
|
_invalid_case_insensitive_identifier
{
    return yylex.(*lexer).FatalError("Keyspace term must be case sensitive", $<line>1, $<column>1)
}
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
|
opt_primary KEYS VALIDATE expr
{
    $$ = algebra.NewUse($4, nil, algebra.JOIN_HINT_NONE)
    $$.SetValidateKeys(true)
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
;

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
        yylex.Error("Keyspace reference cannot have join hint (USE HASH or USE NL) in DELETE or UPDATE statement" +
            yylex.(*lexer).ErrorContext())
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
|
ON opt_primary KEYS VALIDATE expr
{
    $$ = $5
    $$.SetExprFlag(expression.EXPR_VALIDATE_KEYS)
}
;

on_key:
ON opt_primary KEY expr
{
    $$ = $4
}
|
ON opt_primary KEY VALIDATE expr
{
    $$ = $5
    $$.SetExprFlag(expression.EXPR_VALIDATE_KEYS)
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
    $$.SetErrorContext($<line>1, $<column>1)
}
;

/*************************************************
 *
 * WITH clause
 *
 *************************************************/

with:
WITH with_list
{
    for _, with := range $2 {
        l, c := with.GetErrorContext()
        if with.Config()!=nil {
            return yylex.(*lexer).FatalError(fmt.Sprintf("cannot have OPTIONS "+
                "without RECURSIVE keyword as part of '%s'", with.Alias()), l, c)
        }

        if with.CycleFields()!=nil {
            return yylex.(*lexer).FatalError(fmt.Sprintf("cannot have CYCLE "+
                "without RECURSIVE keyword as part of '%s'", with.Alias()), l, c)
        }
    }
    $$ = algebra.NewWithClause(false, $2)
}
|
WITH RECURSIVE with_list
{
    $$ = algebra.NewWithClause(true, $3)
}
;

with_list:
with_term
{
    $$ = expression.Withs{$1}
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
alias AS paren_expr opt_cycle_clause opt_option_clause
{
    $$ = algebra.NewWith($1, $3,nil, false, $5, $4)
    $$.SetErrorContext($<line>1, $<column>1)
}
;

opt_option_clause:
{
    $$ = nil
}
|
OPTIONS object
{
    $$ = $2.Value()
    if $$ == nil {
        yylex.(*lexer).ErrorWithContext("OPTIONS value must be static", $<line>2, $<column>2)
    }
}
;

opt_cycle_clause:
{
    $$ = nil
}
|
/* non sql std for now */
CYCLE exprs RESTRICT
{
    $$ = algebra.NewCycleCheck($2)
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
    $$.ExprBase().SetErrorContext($<line>2,$<column>2)
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
GROUP BY group_terms opt_group_as opt_letting opt_having
{
    as := ""
    if $4 != nil {
        as = $4.Alias()
    }
    g := algebra.NewGroup($3, $5, $6, as)
    if $4 != nil {
        g.SetAsErrorContext($4.GetErrorContext())
    }
    $$ = g
}
|
letting
{
    $$ = algebra.NewGroup(nil, $1, nil, "")
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
    if $2 != "" {
        $1.ExprBase().SetAliasErrorContext($<line>2,$<column>2)
    }
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
    $$.ExprBase().SetErrorContext($<line>2,$<column>2)
}
;

opt_group_as:
/* empty */
{
    $$ = nil
}
|
GROUP AS permitted_identifiers
{
    $$ = expression.NewIdentifier($3)
    $$.ExprBase().SetErrorContext($<line>3, $<column>3)
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
    $$ = algebra.NewSortTerm($1, $2, $3)
    $$.Expression().ExprBase().SetErrorContext($<line>1,$<column>1)
}
;

opt_dir:
/* empty */
{
    $$ = nil
}
|
dir
;

dir:
param_expr
{
    $$ = $1
}
|
ASC
{
    $$ = expression.NewConstant(value.NewValue("asc"))
}
|
DESC
{
    $$ = expression.NewConstant(value.NewValue("desc"))
}
;

opt_order_nulls:
/* empty */
{
    $$ = nil
}
|
NULLS FIRST
{
    $$ = expression.NewConstant(value.NewValue("first"))
}
|
NULLS LAST
{
    $$ = expression.NewConstant(value.NewValue("last"))
}
|
NULLS param_expr
{
    $$ = $2
}
;

first_last:
FIRST { $$ = false }
|
LAST { $$ = true }
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
path_part DOT path_part opt_as_alias
{
    path, err := algebra.NewPathFromElementsWithContext([]string{$1,$3}, yylex.(*lexer).Namespace(), yylex.(*lexer).QueryContext())
    if err != nil {
      return yylex.(*lexer).FatalError(err.Error(), $<line>1, $<column>1)
    }
    $$ = algebra.NewKeyspaceRefFromPath(path, $4)
}
|
keyspace_path opt_as_alias
{
    $$ = algebra.NewKeyspaceRefFromPath($1, $2)
}
|
path_part DOT path_part DOT keyspace_name opt_as_alias
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
    if $2 != "" {
        $1.ExprBase().SetAliasErrorContext($<line>2,$<column>2)
    }
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
    $$ = algebra.NewProjection(false, $1, nil)
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
DELETE opt_optim_hints FROM keyspace_ref opt_use_del_upd opt_let opt_where limit opt_offset opt_returning /* LIMIT before OFFSET */
{
  $$ = algebra.NewDelete($4, $5.Keys(), $5.Indexes(), $7, $8, $9, $10, $2, $5.ValidateKeys(), $6)
}
|
DELETE opt_optim_hints FROM keyspace_ref opt_use_del_upd opt_let opt_where offset opt_limit opt_returning /* OFFSET before LIMIT */
{
  $$ = algebra.NewDelete($4, $5.Keys(), $5.Indexes(), $7, $9, $8, $10, $2, $5.ValidateKeys(), $6)
}
|
DELETE opt_optim_hints FROM keyspace_ref opt_use_del_upd opt_let opt_where opt_returning
{
  $$ = algebra.NewDelete($4, $5.Keys(), $5.Indexes(), $7, nil, nil, $8, $2, $5.ValidateKeys(), $6)
}
;


/*************************************************
 *
 * UPDATE
 *
 *************************************************/

update:
UPDATE opt_optim_hints keyspace_ref opt_use_del_upd opt_let set unset opt_where opt_limit opt_returning
{
    $$ = algebra.NewUpdate($3, $4.Keys(), $4.Indexes(), $6, $7, $8, $9, $10, $2, $4.ValidateKeys(), $5)
}
|
UPDATE opt_optim_hints keyspace_ref opt_use_del_upd opt_let set opt_where opt_limit opt_returning
{
    $$ = algebra.NewUpdate($3, $4.Keys(), $4.Indexes(), $6, nil, $7, $8, $9, $2, $4.ValidateKeys(), $5)
}
|
UPDATE opt_optim_hints keyspace_ref opt_use_del_upd opt_let unset opt_where opt_limit opt_returning
{
    $$ = algebra.NewUpdate($3, $4.Keys(), $4.Indexes(), nil, $6, $7, $8, $9, $2, $4.ValidateKeys(), $5)
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
         return yylex.(*lexer).FatalError(fmt.Sprintf("SET clause has invalid path %s", $3.String()), $<line>3, $<column>3)
    }
}
;

function_meta_expr:
function_name LPAREN opt_exprs RPAREN
{
    $$ = nil
    fname := $1.Identifier()
    f, ok := expression.GetFunction(fname)
    if ok && strings.ToLower(fname) == "meta" && len($3) >= f.MinArgs() && len($3) <= f.MaxArgs() {
         $$ = f.Constructor()($3...)
    } else {
         return yylex.(*lexer).FatalError(fmt.Sprintf("Invalid arguments to function %s", fname), $<line>3, $<column>3)
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
    $$.SetErrorContext($<line>1, $<column>1)
}
|
variable WITHIN expr
{
    $$ = expression.NewBinding("", $1, $3, true)
    $$.SetErrorContext($<line>1, $<column>1)
}
|
variable COLON variable IN expr
{
    $$ = expression.NewBinding($1, $3, $5, false)
    $$.SetErrorContext($<line>1, $<column>1)
}
|
variable COLON variable WITHIN expr
{
    $$ = expression.NewBinding($1, $3, $5, true)
    $$.SetErrorContext($<line>1, $<column>1)
}
;

variable:
permitted_identifiers
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
MERGE opt_optim_hints INTO simple_keyspace_ref opt_use_merge USING simple_from_term ON opt_key expr opt_let opt_merge_actions
opt_limit opt_returning
{
    switch other := $7.(type) {
    case *algebra.SubqueryTerm:
        source := algebra.NewMergeSourceSubquery(other)
        $$ = algebra.NewMerge($4, $5.Indexes(), source, $9, $10, $12, $13, $14, $2, $11)
    case *algebra.ExpressionTerm:
        source := algebra.NewMergeSourceExpression(other)
        $$ = algebra.NewMerge($4, $5.Indexes(), source, $9, $10, $12, $13, $14, $2, $11)
    case *algebra.KeyspaceTerm:
        source := algebra.NewMergeSourceFrom(other)
        $$ = algebra.NewMerge($4, $5.Indexes(), source, $9, $10, $12, $13, $14, $2, $11)
    default:
        yylex.(*lexer).ErrorWithContext("MERGE source term is UNKNOWN", $<line>7, $<column>7)
    }
}
;

opt_use_merge:
opt_use
{
    if $1.Keys() != nil {
        yylex.(*lexer).ErrorWithContext("Keyspace reference cannot have USE KEYS hint in MERGE statement", $<line>1, $<column>1)
    } else if $1.JoinHint() != algebra.JOIN_HINT_NONE {
        yylex.(*lexer).ErrorWithContext("Keyspace reference cannot have join hint (USE HASH or USE NL) in MERGE statement",
          $<line>1, $<column>1)
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

opt_merge_actions:
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
 * USERS
 *
 *************************************************/

create_user:
CREATE USER user user_opts
{
    var name, groups value.Value
    var password expression.Expression
    for _, v := range $4 {
        switch v.name {
        case "name":
            if name != nil { return yylex.(*lexer).FatalError("User attributes may only be specified once.", v.line, v.column) }
            name = value.NewValue(v.value.(string))
        case "password":
            if password != nil { return yylex.(*lexer).FatalError("User attributes may only be specified once.", v.line, v.column) }
            password = v.value.(expression.Expression)
        case "groups":
            if groups != nil { return yylex.(*lexer).FatalError("User attributes may only be specified once.", v.line, v.column) }
            sa := v.value.([]string)
            va := make(value.Values, len(sa))
            for i := range sa {
                va[i] = value.NewValue(sa[i])
            }
            groups = value.NewValue(va)
        }
    }
    $$ = algebra.NewCreateUser($3,password,name,groups)
}
;

alter_user:
ALTER USER user user_opts
{
    var name, groups value.Value
    var password expression.Expression
    for _, v := range $4 {
        switch v.name {
        case "name":
            if name != nil { return yylex.(*lexer).FatalError("User attributes may only be specified once.", v.line, v.column) }
            name = value.NewValue(v.value.(string))
        case "password":
            if password != nil { return yylex.(*lexer).FatalError("User attributes may only be specified once.", v.line, v.column) }
            password = v.value.(expression.Expression)
        case "groups":
            if groups != nil { return yylex.(*lexer).FatalError("User attributes may only be specified once.", v.line, v.column) }
            sa := v.value.([]string)
            va := make(value.Values, len(sa))
            for i := range sa {
                va[i] = value.NewValue(sa[i])
            }
            groups = value.NewValue(va)
        }
    }
    $$ = algebra.NewAlterUser($3,password,name,groups)
}
;

drop_user:
DROP USER user
{
    $$ = algebra.NewDropUser($3)
}
;

user_opts:
/* empty */
{
    $$ = nil
}
|
user_opts user_opt
{
    $$ = append($1, $2)
}
;

param_or_str:
param_expr
{
    $$ = $1
}
|
STR
{
    if len($1) < 6 {
        return yylex.(*lexer).FatalError("The password must be at least 6 characters long.", $<line>1, $<column>1)
    }
    $$ = expression.NewConstant(value.NewValue($1))
}
;

user_opt:
PASSWORD param_or_str
{
    $$ = &nameValueContext{"password", $2, $<line>1, $<column>1}
}
|
WITH STR
{
    $$ = &nameValueContext{"name", $2, $<line>1, $<column>1}
}
|
GROUPS groups
{
    var groups []string
    // de-duplicate groups list
    if len($2) > 0 {
        groups = make([]string, 0, len($2))
        for i := range $2 {
            found := false
            for j := range groups {
                if $2[i] == groups[j] {
                    found = true
                    break
                }
            }
            if !found {
                groups = append(groups, $2[i])
            }
        }
    }

    $$ = &nameValueContext{"groups", groups, $<line>1, $<column>1}
}
|
GROUP permitted_identifiers
{
    $$ = &nameValueContext{"groups", []string{$2}, $<line>1, $<column>1}
}
|
NO GROUPS
{
    $$ = &nameValueContext{"groups", []string{}, $<line>1, $<column>1}
}
;

groups:
permitted_identifiers
{
    $$ = append([]string(nil), $1)
}
|
groups COMMA permitted_identifiers
{
    $$ = append($1, $3)
}
;

/*************************************************
 *
 * GROUPS
 *
 *************************************************/

create_group:
CREATE GROUP group_name group_opts
{
    var desc, roles value.Value
    for _, v := range $4 {
        switch v.name {
        case "desc":
            if desc != nil { return yylex.(*lexer).FatalError("Group attributes may only be specified once.", v.line, v.column) }
            desc = value.NewValue(v.value.(string))
        case "roles":
            if roles != nil { return yylex.(*lexer).FatalError("Group attributes may only be specified once.", v.line, v.column) }
            sa := v.value.([]string)
            va := make(value.Values, len(sa))
            for i := range sa {
                va[i] = value.NewValue(sa[i])
            }
            roles = value.NewValue(va)
        }
    }
    $$ = algebra.NewCreateGroup($3,desc,roles)
}
;

alter_group:
ALTER GROUP group_name group_opts
{
    var desc, roles value.Value
    for _, v := range $4 {
        switch v.name {
        case "desc":
            if desc != nil { return yylex.(*lexer).FatalError("Group attributes may only be specified once.", v.line, v.column) }
            desc = value.NewValue(v.value.(string))
        case "roles":
            if roles != nil { return yylex.(*lexer).FatalError("Group attributes may only be specified once.", v.line, v.column) }
            sa := v.value.([]string)
            va := make(value.Values, len(sa))
            for i := range sa {
                va[i] = value.NewValue(sa[i])
            }
            roles = value.NewValue(va)
        }
    }
    $$ = algebra.NewAlterGroup($3,desc,roles)
}
;

drop_group:
DROP GROUP group_name
{
    $$ = algebra.NewDropGroup($3)
}
;

group_name:
permitted_identifiers
;

group_opts:
/* empty */
{
    $$ = nil
}
|
group_opts group_opt
{
    $$ = append($1, $2)
}
;

group_opt:
WITH STR
{
    $$ = &nameValueContext{"desc", $2, $<line>1, $<column>1}
}
|
ROLES group_role_list
{
    var roles []string
    // de-duplicate roles list
    if len($2) > 0 {
        roles = make([]string, 0, len($2))
        for i := range $2 {
            found := false
            for j := range roles {
                if $2[i] == roles[j] {
                    found = true
                    break
                }
            }
            if !found {
                roles = append(roles, $2[i])
            }
        }
    }

    $$ = &nameValueContext{"roles", roles, $<line>1, $<column>1}
}
|
NO ROLES
{
    $$ = &nameValueContext{"roles", []string{}, $<line>1, $<column>1}
}
|
ROLE group_role_list_item
{
    $$ = &nameValueContext{"roles", []string{$2}, $<line>1, $<column>1}
}
;

group_role_list:
group_role_list_item
{
    $$ = []string{ $1 }
}
|
group_role_list COMMA group_role_list_item
{
    $$ = append($1, $3)
}
;

group_role_list_item:
role_name
|
role_name ON keyspace_scope
{
    fn := $3.FullName()
    i := strings.Index(fn, ":")
    if i != -1 {
        // strip the namespace as endpoint doesn't accept it in the target information
        fn = fn[i+1:]
    }
    $$ = auth.AliasToRole($1) + "[" + strings.ReplaceAll(fn, ".", ":") + "]"
}
;

/*************************************************
 *
 * GRANT ROLE
 *
 *************************************************/

group_or_groups:
GROUP
|
GROUPS
;

user_users:
USER
|
USERS
;

grant_role:
GRANT role_list TO user_list
{
    $$ = algebra.NewGrantRole($2, nil, $4, false)
}
|
GRANT role_list ON keyspace_scope_list TO user_list
{
    $$ = algebra.NewGrantRole($2, $4, $6, false)
}
|
GRANT role_list TO user_users user_list
{
    $$ = algebra.NewGrantRole($2, nil, $5, false)
}
|
GRANT role_list ON keyspace_scope_list TO user_users user_list
{
    $$ = algebra.NewGrantRole($2, $4, $7, false)
}
|
GRANT role_list TO group_or_groups groups
{
    $$ = algebra.NewGrantRole($2, nil, $5, true)
}
|
GRANT role_list ON keyspace_scope_list TO group_or_groups groups
{
    $$ = algebra.NewGrantRole($2, $4, $7, true)
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
permitted_identifiers
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
path_part DOT path_part
{
    path, err := algebra.NewPathFromElementsWithContext([]string{ $1, $3 }, yylex.(*lexer).Namespace(),
                                 yylex.(*lexer).QueryContext())
    if err != nil {
        return yylex.(*lexer).FatalError(err.Error(), $<line>1, $<column>1)
    }
    $$ = algebra.NewKeyspaceRefFromPath(path, "")
}
|
namespace_name keyspace_name
{
    path := algebra.NewPathShort($1, $2)
    $$ = algebra.NewKeyspaceRefFromPath(path, "")
}
|
namespace_name path_part DOT path_part DOT keyspace_name
{
    path := algebra.NewPathLong($1, $2, $4, $6)
    $$ = algebra.NewKeyspaceRefFromPath(path, "")
}
|
path_part DOT path_part DOT keyspace_name
{
    path := algebra.NewPathLong(yylex.(*lexer).Namespace(), $1, $3, $5)
    $$ = algebra.NewKeyspaceRefFromPath(path, "")
}
|
namespace_name path_part DOT path_part
{
    path := algebra.NewPathScope($1, $2, $4)
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
permitted_identifiers
{
    $$ = $1
}
|
permitted_identifiers COLON permitted_identifiers
{
    $$ = $1 + ":" + $3
}
;

/*************************************************
 *
 * REVOKE ROLE
 *
 *************************************************/

revoke_role:
REVOKE role_list FROM user_list
{
    $$ = algebra.NewRevokeRole($2, nil, $4, false)
}
|
REVOKE role_list ON keyspace_scope_list FROM user_list
{
    $$ = algebra.NewRevokeRole($2, $4, $6, false)
}
|
REVOKE role_list FROM user_users user_list
{
    $$ = algebra.NewRevokeRole($2, nil, $5, false)
}
|
REVOKE role_list ON keyspace_scope_list FROM user_users user_list
{
    $$ = algebra.NewRevokeRole($2, $4, $7, false)
}
|
REVOKE role_list FROM group_or_groups groups
{
    $$ = algebra.NewRevokeRole($2, nil, $5, true)
}
|
REVOKE role_list ON keyspace_scope_list FROM group_or_groups groups
{
    $$ = algebra.NewRevokeRole($2, $4, $7, true)
}
;

/*************************************************
 *
 * CREATE BUCKET / DATABASE
 *
 *************************************************/

opt_def_with_clause:
opt_with_clause
{
    if $1 == nil {
        $$ = value.NewValue(map[string]interface{}{})
    } else {
        $$ = $1
    }
}
;

create_bucket:
CREATE BUCKET permitted_identifiers opt_if_not_exists opt_def_with_clause
{
    $$ = algebra.NewCreateBucket($3, $4, $5)
}
|
CREATE BUCKET IF NOT EXISTS permitted_identifiers opt_def_with_clause
{
    $$ = algebra.NewCreateBucket($6, false, $7)
}
|
CREATE DATABASE permitted_identifiers opt_if_not_exists opt_def_with_clause
{
    $$ = algebra.NewCreateBucket($3, $4, $5)
}
|
CREATE DATABASE IF NOT EXISTS permitted_identifiers opt_def_with_clause
{
    $$ = algebra.NewCreateBucket($6, false, $7)
}
;

/*************************************************
 *
 * ALTER BUCKET / DATABASE
 *
 *************************************************/

alter_bucket:
ALTER BUCKET permitted_identifiers with_clause
{
    $$ = algebra.NewAlterBucket($3, $4)
}
|
ALTER DATABASE permitted_identifiers with_clause
{
    $$ = algebra.NewAlterBucket($3, $4)
}
;

/*************************************************
 *
 * DROP BUCKET / DATABASE
 *
 *************************************************/

drop_bucket:
DROP BUCKET permitted_identifiers opt_if_exists
{
    $$ = algebra.NewDropBucket($3, $4)
}
|
DROP BUCKET IF EXISTS permitted_identifiers
{
    $$ = algebra.NewDropBucket($5, false)
}
|
DROP DATABASE permitted_identifiers opt_if_exists
{
    $$ = algebra.NewDropBucket($3, $4)
}
|
DROP DATABASE IF EXISTS permitted_identifiers
{
    $$ = algebra.NewDropBucket($5, false)
}
;

/*************************************************
 *
 * CREATE SCOPE
 *
 *************************************************/

create_scope:
CREATE SCOPE named_scope_ref opt_if_not_exists
{
    $$ = algebra.NewCreateScope($3, $4)
}
|
CREATE SCOPE IF NOT EXISTS named_scope_ref
{
    $$ = algebra.NewCreateScope($6, false)
}
;

/*************************************************
 *
 * DROP SCOPE
 *
 *************************************************/

drop_scope:
DROP SCOPE named_scope_ref opt_if_exists
{
    $$ = algebra.NewDropScope($3, $4)
}
|
DROP SCOPE IF EXISTS named_scope_ref
{
    $$ = algebra.NewDropScope($5, false)
}
;

/*************************************************
 *
 * CREATE COLLECTION
 *
 *************************************************/

create_collection:
CREATE COLLECTION named_keyspace_ref opt_if_not_exists opt_with_clause
{
    $$ = algebra.NewCreateCollection($3, $4, $5)
}
|
CREATE COLLECTION IF NOT EXISTS named_keyspace_ref opt_with_clause
{
    $$ = algebra.NewCreateCollection($6, false, $7)
}
;

/*************************************************
 *
 * DROP COLLECTION
 *
 *************************************************/

drop_collection:
DROP COLLECTION named_keyspace_ref opt_if_exists
{
    $$ = algebra.NewDropCollection($3, $4)
}
|
DROP COLLECTION IF EXISTS named_keyspace_ref
{
    $$ = algebra.NewDropCollection($5, false)
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
CREATE PRIMARY INDEX opt_if_not_exists ON named_keyspace_ref opt_index_partition opt_index_using opt_with_clause
{
    $$ = algebra.NewCreatePrimaryIndex("#primary", $6, $7, $8, $9, $4)
}
|
CREATE PRIMARY INDEX index_name opt_if_not_exists ON named_keyspace_ref opt_index_partition opt_index_using opt_with_clause
{
    $$ = algebra.NewCreatePrimaryIndex($4, $7, $8, $9, $10, $5)
}
|
CREATE PRIMARY INDEX IF NOT EXISTS index_name ON named_keyspace_ref opt_index_partition opt_index_using opt_with_clause
{
    $$ = algebra.NewCreatePrimaryIndex($7, $9, $10, $11, $12, false)
}
|
CREATE opt_vector INDEX index_name opt_if_not_exists
ON named_keyspace_ref LPAREN index_terms RPAREN opt_index_partition opt_index_where opt_index_using opt_with_clause
{
    $$ = algebra.NewCreateIndex($4, $7, $9, $11, $12, $13, $14, $5, $2)
}
|
CREATE opt_vector INDEX IF NOT EXISTS index_name
ON named_keyspace_ref LPAREN index_terms RPAREN opt_index_partition opt_index_where opt_index_using opt_with_clause
{
    $$ = algebra.NewCreateIndex($7, $9, $11, $13, $14, $15, $16, false, $2)
}
;

opt_vector:
/* empty */
{
    $$ = false
}
|
VECTOR
{
    $$ = true
}
;

index_name:
permitted_identifiers
;

opt_index_name:
{ $$ = "" }
|
index_name
;

opt_if_not_exists:
/* empty */
{
    $$ = true
}
|
IF NOT EXISTS
{
    $$ = false
}
;

named_keyspace_ref:
simple_named_keyspace_ref
|
namespace_name path_part
{
    path := algebra.NewPathShort($1, $2)
    $$ = algebra.NewKeyspaceRefFromPath(path, "")
}
|
path_part DOT path_part DOT keyspace_name
{
    path := algebra.NewPathLong(yylex.(*lexer).Namespace(), $1, $3, $5)
    $$ = algebra.NewKeyspaceRefFromPath(path, "")
}
|
path_part DOT keyspace_name
{
    path, err := algebra.NewPathFromElementsWithContext([]string{ $1, $3},
                    yylex.(*lexer).Namespace(), yylex.(*lexer).QueryContext())
    if err != nil {
        return yylex.(*lexer).FatalError(err.Error(), $<line>1, $<column>1)
    }
    $$ = algebra.NewKeyspaceRefFromPath(path, "")
}
;

simple_named_keyspace_ref:
keyspace_name
{
    $$ = algebra.NewKeyspaceRefWithContext($1, "", yylex.(*lexer).Namespace(), yylex.(*lexer).QueryContext())
}
|
namespace_name path_part DOT path_part DOT keyspace_name
{
    path := algebra.NewPathLong($1, $2, $4, $6)
    $$ = algebra.NewKeyspaceRefFromPath(path, "")
}
;

named_scope_ref:
namespace_name path_part DOT path_part
{
    path := algebra.NewPathScope($1, $2, $4)
    $$ = algebra.NewScopeRefFromPath(path, "")
}
|
path_part DOT path_part
{
    path := algebra.NewPathScope(yylex.(*lexer).Namespace(), $1, $3)
    $$ = algebra.NewScopeRefFromPath(path, "")
}
|
path_part
{
    path, err := algebra.NewPathScopeWithContext(yylex.(*lexer).Namespace(), $1, yylex.(*lexer).QueryContext())
    if err != nil {
        return yylex.(*lexer).FatalError(err.Error(), $<line>1, $<column>1)
    }
    $$ = algebra.NewScopeRefFromPath(path, "")
}
;

opt_index_partition:
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
;

index_term_expr:
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

all:
ALL
|
EACH
;

flatten_keys_expr:
expr opt_ikattr
{
   $$ = algebra.NewIndexKeyTerm($1, $2)
}
;

flatten_keys_exprs:
flatten_keys_expr
{
    $$ = algebra.IndexKeyTerms{$1}
}
|
flatten_keys_exprs COMMA flatten_keys_expr
{
    $$ = append($1, $3)
}
;

opt_flatten_keys_exprs:
/* empty */
{
    $$ = nil
}
|
flatten_keys_exprs
;

opt_index_where:
/* empty */
{
    $$ = nil
}
|
WHERE expr
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
       yylex.(*lexer).ErrorWithContext("Duplicate or Invalid index key attribute", $<line>2,$<column>2)
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
INCLUDE MISSING
{ $$ = algebra.IK_MISSING }
;


/*************************************************
 *
 * DROP INDEX
 *
 *************************************************/

drop_index:
DROP PRIMARY INDEX opt_if_exists ON named_keyspace_ref opt_index_using
{
    $$ = algebra.NewDropIndex($6, "#primary", $7, $4, true, false)
}
|
DROP PRIMARY INDEX index_name opt_if_exists ON named_keyspace_ref opt_index_using
{
    $$ = algebra.NewDropIndex($7, $4, $8, $5, true, false)
}
|
DROP PRIMARY INDEX IF EXISTS index_name ON named_keyspace_ref opt_index_using
{
    $$ = algebra.NewDropIndex($8, $6, $9, false, true, false)
}
|
DROP opt_vector INDEX simple_named_keyspace_ref DOT index_name opt_if_exists opt_index_using
{
    $$ = algebra.NewDropIndex($4, $6, $8, $7, false,$2)
}
|
DROP opt_vector INDEX IF EXISTS simple_named_keyspace_ref DOT index_name opt_index_using
{
    $$ = algebra.NewDropIndex($6, $8, $9, false, false,$2)
}
|
DROP opt_vector INDEX index_name opt_if_exists ON named_keyspace_ref opt_index_using
{
    $$ = algebra.NewDropIndex($7, $4, $8, $5, false,$2)
}
|
DROP opt_vector INDEX IF EXISTS index_name ON named_keyspace_ref opt_index_using
{
    $$ = algebra.NewDropIndex($8, $6, $9, false, false,$2)
}
;

opt_if_exists:
/* empty */
{
    $$ = true
}
|
IF EXISTS
{
    $$ = false
}
;

/*************************************************
 *
 * ALTER INDEX
 *
 *************************************************/

alter_index:
ALTER INDEX simple_named_keyspace_ref DOT index_name opt_index_using with_clause
{
    $$ = algebra.NewAlterIndex($3, $5, $6, $7)
}
|
ALTER INDEX index_name ON named_keyspace_ref opt_index_using with_clause
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
CREATE opt_replace FUNCTION opt_if_not_exists func_name
{
    if $5 != nil {
        // push function query context
        yylex.(*lexer).PushQueryContext($5.QueryContext())
    }
}
LPAREN opt_parm_list RPAREN opt_if_not_exists func_body
{
    if $5 != nil {
        yylex.(*lexer).PopQueryContext()
    }
    if $11 != nil {
        err := $11.SetVarNames($8)
        if err != nil {
            yylex.Error(err.Error()+yylex.(*lexer).ErrorContext())
        }
    }
    if $2.Value().Truth() && (!$10 || !$4) {
        return yylex.(*lexer).FatalError("syntax error - OR REPLACE and IF NOT EXISTS are mutually exclusive", $<line>2, $<column>2)
    } else if !$10 && !$4 {
        return yylex.(*lexer).FatalError("syntax error - specify IF NOT EXISTS only once", $<line>10, $<column>10)
    }
    $$ = algebra.NewCreateFunction($5, $11, $2.Value().Truth(), $10&&$4)
}
;

opt_replace:
/* empty */
{
    $$ = expression.FALSE_EXPR
}
|
OR REPLACE
{
    $$ = expression.TRUE_EXPR
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
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
    name, err := functionsBridge.NewFunctionName([]string{$1}, yylex.(*lexer).Namespace(), yylex.(*lexer).QueryContext())
    if err != nil {
        yylex.Error(err.Error()+yylex.(*lexer).ErrorContext())
    }
    $$ = name
}
|
path_part DOT path_part
{
    dummyPath, err := algebra.NewPathFromElementsWithContext([]string{ $1, $3},
        yylex.(*lexer).Namespace(), yylex.(*lexer).QueryContext())
    if err != nil {
        return yylex.(*lexer).FatalError(err.Error(), $<line>1, $<column>1)
    }
    name, err := functionsBridge.NewFunctionName(dummyPath.Parts(), yylex.(*lexer).Namespace(), yylex.(*lexer).QueryContext())
    if err != nil {
        yylex.Error(err.Error()+yylex.(*lexer).ErrorContext())
    }
    $$ = name
}
|
path_part DOT path_part DOT path_part
{
    name, err := functionsBridge.NewFunctionName([]string{yylex.(*lexer).Namespace(), $1, $3, $5},
        yylex.(*lexer).Namespace(), yylex.(*lexer).QueryContext())
    if err != nil {
        yylex.Error(err.Error()+yylex.(*lexer).ErrorContext())
    }
    $$ = name
}
;

long_func_name:
namespace_term keyspace_name
{
    name, err := functionsBridge.NewFunctionName([]string{$1, $2}, yylex.(*lexer).Namespace(), yylex.(*lexer).QueryContext())
    if err != nil {
        yylex.Error(err.Error()+yylex.(*lexer).ErrorContext())
    }
    $$ = name
}
|
namespace_term path_part DOT path_part DOT keyspace_name
{
    name, err := functionsBridge.NewFunctionName([]string{$1, $2, $4, $6},
        yylex.(*lexer).Namespace(), yylex.(*lexer).QueryContext())
    if err != nil {
        yylex.Error(err.Error()+yylex.(*lexer).ErrorContext())
    }
    $$ = name
}
;

opt_parm_list:
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
permitted_identifiers
{
    $$ = []string{$1}
}
|
parameter_terms COMMA permitted_identifiers
{
    $$ = append($1, string($3))
}
;

func_body:
LBRACE expr RBRACE
{
    body, err := functionsBridge.NewInlineBody($2, yylex.(*lexer).getSubString($<tokOffset>1,$<tokOffset>3))
    if err != nil {
        yylex.Error(err.Error()+yylex.(*lexer).ErrorContext())
    } else {
        $$ = body
    }
}
|
LANGUAGE INLINE AS expr
{
    body, err := functionsBridge.NewInlineBody($4, yylex.(*lexer).Remainder($<tokOffset>3))
    if err != nil {
        yylex.Error(err.Error()+yylex.(*lexer).ErrorContext())
    } else {
        $$ = body
    }
}
|
LANGUAGE JAVASCRIPT AS STR
{
    body, err := functionsBridge.NewJavascriptBody("","", $4)
    if err != nil {
        yylex.Error(err.Error()+yylex.(*lexer).ErrorContext())
    } else {
        $$ = body
    }
}
|
LANGUAGE JAVASCRIPT AS STR AT STR
{
    body, err := functionsBridge.NewJavascriptBody($6, $4, "")
    if err != nil {
        yylex.Error(err.Error()+yylex.(*lexer).ErrorContext())
    } else {
        $$ = body
    }
}
|
LANGUAGE GOLANG AS STR AT STR
{
    body, err := functionsBridge.NewGolangBody($6, $4)
    if err != nil {
        yylex.Error(err.Error()+yylex.(*lexer).ErrorContext())
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
DROP FUNCTION func_name opt_if_exists
{
    $$ = algebra.NewDropFunction($3, $4)
}
|
DROP FUNCTION IF EXISTS func_name
{
    $$ = algebra.NewDropFunction($5, false)
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
UPDATE STATISTICS opt_for named_keyspace_ref LPAREN update_stat_terms RPAREN opt_with_clause
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
UPDATE STATISTICS opt_for named_keyspace_ref INDEX LPAREN exprs RPAREN opt_index_using opt_with_clause
{
    $$ = algebra.NewUpdateStatisticsIndex($4, $7, $9, $10)
}
|
UPDATE STATISTICS opt_for named_keyspace_ref INDEX ALL opt_index_using opt_with_clause
{
    $$ = algebra.NewUpdateStatisticsIndexAll($4, $7, $8)
}
|
UPDATE STATISTICS FOR INDEX simple_named_keyspace_ref DOT index_name opt_index_using opt_with_clause
{
    $$ = algebra.NewUpdateStatisticsIndex($5, expression.Expressions{expression.NewIdentifier($7)}, $8, $9)
}
|
UPDATE STATISTICS FOR INDEX index_name ON named_keyspace_ref opt_index_using opt_with_clause
{
    $$ = algebra.NewUpdateStatisticsIndex($7, expression.Expressions{expression.NewIdentifier($5)}, $8, $9)
}
|
ANALYZE opt_keyspace_collection named_keyspace_ref LPAREN update_stat_terms RPAREN opt_with_clause
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
ANALYZE opt_keyspace_collection named_keyspace_ref INDEX LPAREN exprs RPAREN opt_index_using opt_with_clause
{
    $$ = algebra.NewUpdateStatisticsIndex($3, $6, $8, $9)
}
|
ANALYZE opt_keyspace_collection named_keyspace_ref INDEX ALL opt_index_using opt_with_clause
{
    $$ = algebra.NewUpdateStatisticsIndexAll($3, $6, $7)
}
|
ANALYZE INDEX simple_named_keyspace_ref DOT index_name opt_index_using opt_with_clause
{
    $$ = algebra.NewUpdateStatisticsIndex($3, expression.Expressions{expression.NewIdentifier($5)}, $6, $7)
}
|
ANALYZE INDEX index_name ON named_keyspace_ref opt_index_using opt_with_clause
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
permitted_identifiers
{
    $$ = expression.NewIdentifier($1)
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
}
|
path DOT permitted_identifiers
{
    fn := expression.NewFieldName($3, false)
    fn.ExprBase().SetErrorContext($<line>3,$<column>3)
    $$ = expression.NewField($1, fn)
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
}
|
path DOT ident_icase
{
    fn := expression.NewFieldName($3.Identifier(), true)
    fn.ExprBase().SetErrorContext($<line>3,$<column>3)
    field := expression.NewField($1, fn)
    field.SetCaseInsensitive(true)
    $$ = field
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
}
|
path DOT LBRACKET expr RBRACKET
{
    $$ = expression.NewField($1, $4)
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
}
|
path DOT LBRACKET expr RBRACKET_ICASE
{
    field := expression.NewField($1, $4)
    field.SetCaseInsensitive(true)
    $$ = field
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
}
|
path LBRACKET expr RBRACKET
{
    $$ = expression.NewElement($1, $3)
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
}
;


/*************************************************
 *
 * Expression
 *
 *************************************************/

ident:
permitted_identifiers
{
    $$ = expression.NewIdentifier($1)
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
}
;

ident_icase:
IDENT_ICASE
{
    $$ = expression.NewIdentifier($1)
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
    $$.SetCaseInsensitive(true)
}
;

expr:
c_expr
|
// relative path function call
expr DOT ident LPAREN opt_exprs RPAREN
{
    var path []string

    switch other := $1.(type) {
    case *expression.SequenceOperation:
        return yylex.(*lexer).FatalError("syntax error", $<line>4, $<column>4)
    case *expression.Identifier:
        path = []string { other.Alias(), $3.Identifier() }
        dummyPath, err := algebra.NewPathFromElementsWithContext([]string { other.Alias(), $3.Identifier() },
        yylex.(*lexer).Namespace(), yylex.(*lexer).QueryContext())
        if err != nil {
            return yylex.(*lexer).FatalError(err.Error(), $<line>1, $<column>1)
        }
        path = dummyPath.Parts()
    case *expression.Field:
        tempPath := other.Path()
        if len(tempPath) != 2 {
            return yylex.(*lexer).FatalError("syntax error", $<line>1, $<column>1)
        }
        path = append([]string { yylex.(*lexer).Namespace() }, append(tempPath, $3.Identifier())...)
    default:
        return yylex.(*lexer).FatalError("syntax error", $<line>1, $<column>1)
    }

    // NewFunctionName() cannot deal with 3 part names, and considers 2 parts as a global function
    // so we have to deal with this ourselves
    name, err := functionsBridge.NewFunctionName(path, yylex.(*lexer).Namespace(), yylex.(*lexer).QueryContext())
    if err != nil {
        return yylex.(*lexer).FatalError(err.Error(), $<line>1, $<column>1)
    }
    f := expression.GetUserDefinedFunction(name, yylex.(*lexer).UdfCheck())
    if f != nil {
        $$ = f.Constructor()($5...)
    } else {
        return yylex.(*lexer).FatalError(fmt.Sprintf("Invalid function %v", name.Key()), $<line>1, $<column>1)
    }
}
|
/* Nested */
expr DOT ident
{
    switch t := $1.(type) {
    case *expression.SequenceOperation:
        if !t.AddPart($3.Identifier()) {
            l, c := t.ExprBase().GetErrorContext()
            return yylex.(*lexer).FatalError("Invalid sequence name", l, c)
        }
        $$ = t
    default:
        fn := expression.NewFieldName($3.Identifier(), false)
        fn.ExprBase().SetErrorContext($<line>3,$<column>3)
        $$ = expression.NewField($1, fn)
        $$.ExprBase().SetErrorContext($3.ExprBase().GetErrorContext())
    }
}
|
expr DOT ident_icase
{
    switch t := $1.(type) {
    case *expression.SequenceOperation:
        l, c := t.ExprBase().GetErrorContext()
        return yylex.(*lexer).FatalError("Invalid sequence name", l, c)
    default:
        fn := expression.NewFieldName($3.Identifier(), true)
        fn.ExprBase().SetErrorContext($<line>3,$<column>3)
        field := expression.NewField($1, fn)
        field.SetCaseInsensitive(true)
        $$ = field
        $$.ExprBase().SetErrorContext($3.ExprBase().GetErrorContext())
    }
}
|
expr DOT LBRACKET expr RBRACKET
{
    switch $1.(type) {
    case *expression.SequenceOperation:
        return yylex.(*lexer).FatalError("syntax error", $<line>3, $<column>3)
    default:
        $$ = expression.NewField($1, $4)
        $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
    }
}
|
expr DOT LBRACKET expr RBRACKET_ICASE
{
    switch $1.(type) {
    case *expression.SequenceOperation:
        return yylex.(*lexer).FatalError("syntax error", $<line>3, $<column>3)
    default:
        field := expression.NewField($1, $4)
        field.SetCaseInsensitive(true)
        $$ = field
        $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
    }
}
|
expr LBRACKET RANDOM_ELEMENT RBRACKET
{
    switch $1.(type) {
    case *expression.SequenceOperation:
        return yylex.(*lexer).FatalError("syntax error", $<line>2, $<column>2)
    default:
        $$ = expression.NewRandomElement($1)
        $$.(*expression.RandomElement).SetOperator()
        $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
    }
}
|
expr LBRACKET expr RBRACKET
{
    switch $1.(type) {
    case *expression.SequenceOperation:
        return yylex.(*lexer).FatalError("syntax error", $<line>2, $<column>2)
    default:
        $$ = expression.NewElement($1, $3)
        $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
    }
}
|
expr LBRACKET expr COLON RBRACKET
{
    switch $1.(type) {
    case *expression.SequenceOperation:
        return yylex.(*lexer).FatalError("syntax error", $<line>2, $<column>2)
    default:
        $$ = expression.NewSlice($1, $3)
        $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
    }
}
|
expr LBRACKET expr COLON expr RBRACKET
{
    switch $1.(type) {
    case *expression.SequenceOperation:
        return yylex.(*lexer).FatalError("syntax error", $<line>2, $<column>2)
    default:
        $$ = expression.NewSlice($1, $3, $5)
        $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
    }
}
|
expr LBRACKET COLON expr RBRACKET
{
    switch $1.(type) {
    case *expression.SequenceOperation:
        return yylex.(*lexer).FatalError("syntax error", $<line>2, $<column>2)
    default:
        $$ = expression.NewSliceEnd($1, $4)
        $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
    }
}
|
expr LBRACKET COLON RBRACKET
{
    switch $1.(type) {
    case *expression.SequenceOperation:
        return yylex.(*lexer).FatalError("syntax error", $<line>2, $<column>2)
    default:
        $$ = expression.NewSlice($1)
        $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
    }
}
|
expr LBRACKET RBRACKET
{
    switch $1.(type) {
    case *expression.SequenceOperation:
        return yylex.(*lexer).FatalError("syntax error", $<line>2, $<column>2)
    default:
        $$ = expression.NewSlice($1)
        $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
    }
}
|
expr LBRACKET STAR RBRACKET
{
    switch $1.(type) {
    case *expression.SequenceOperation:
        return yylex.(*lexer).FatalError("syntax error", $<line>2, $<column>2)
    default:
        $$ = expression.NewArrayStar($1)
        $$.(*expression.ArrayStar).SetOperator()
        $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
    }
}
|
/* Arithmetic */
expr PLUS expr
{
    $$ = expression.NewAdd($1, $3)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr MINUS expr
{
    $$ = expression.NewSub($1, $3)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr STAR expr
{
    $$ = expression.NewMult($1, $3)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr DIV expr
{
    $$ = expression.NewDiv($1, $3)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr MOD expr
{
    $$ = expression.NewMod($1, $3)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr POW expr
{
    $$ = expression.NewPower($1, $3)
    $$.(*expression.Power).SetOperator()
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
/* Concat */
expr CONCAT expr
{
    $$ = expression.NewConcat($1, $3)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
/* Logical */
expr AND expr
{
    $$ = expression.NewAnd($1, $3)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr OR expr
{
    $$ = expression.NewOr($1, $3)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
NOT expr
{
    $$ = expression.NewNot($2)
    $$.ExprBase().SetErrorContext($2.ExprBase().GetErrorContext())
}
|
/* Comparison */
expr EQ expr
{
    $$ = expression.NewEq($1, $3)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr DEQ expr
{
    $$ = expression.NewEq($1, $3)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr NE expr
{
    $$ = expression.NewNE($1, $3)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr LT expr
{
    $$ = expression.NewLT($1, $3)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr GT expr
{
    $$ = expression.NewGT($1, $3)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr LE expr
{
    $$ = expression.NewLE($1, $3)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr GE expr
{
    $$ = expression.NewGE($1, $3)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr BETWEEN b_expr AND b_expr
{
    $$ = expression.NewBetween($1, $3, $5)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr NOT BETWEEN b_expr AND b_expr
{
    $$ = expression.NewNotBetween($1, $4, $6)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr LIKE expr ESCAPE expr
{
    $$ = expression.NewLike($1, $3, $5)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr LIKE expr
{
    $$ = expression.NewLike($1, $3, expression.DEFAULT_ESCAPE_EXPR)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr NOT LIKE expr ESCAPE expr
{
    $$ = expression.NewNotLike($1, $4, $6)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr NOT LIKE expr
{
    $$ = expression.NewNotLike($1, $4, expression.DEFAULT_ESCAPE_EXPR)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr IN expr
{
    $$ = expression.NewIn($1, $3)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr NOT IN expr
{
    $$ = expression.NewNotIn($1, $4)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr WITHIN expr
{
    $$ = expression.NewWithin($1, $3)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr NOT WITHIN expr
{
    $$ = expression.NewNotWithin($1, $4)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr IS NULL
{
    $$ = expression.NewIsNull($1)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr IS NOT NULL
{
    $$ = expression.NewIsNotNull($1)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr IS MISSING
{
    $$ = expression.NewIsMissing($1)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr IS NOT MISSING
{
    $$ = expression.NewIsNotMissing($1)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr IS valued
{
    $$ = expression.NewIsValued($1)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr IS NOT UNKNOWN
{
    $$ = expression.NewIsValued($1)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr IS NOT valued
{
    $$ = expression.NewIsNotValued($1)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr IS UNKNOWN
{
    $$ = expression.NewIsNotValued($1)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr IS DISTINCT FROM expr
{
    $$ = expression.NewIsDistinctFrom($1,$5)
    $$.(*expression.IsDistinctFrom).SetOperator()
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
expr IS NOT DISTINCT FROM expr
{
    $$ = expression.NewIsNotDistinctFrom($1,$6)
    $$.(*expression.IsNotDistinctFrom).SetOperator()
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
EXISTS expr
{
    $$ = expression.NewExists($2)
    $$.ExprBase().SetErrorContext($2.ExprBase().GetErrorContext())
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
sequence_expr
|
/* Construction */
construction_expr
|
/* Identifier */
permitted_identifiers
{
    $$ = expression.NewIdentifier($1)
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
}
|
/* Identifier */
IDENT_ICASE
{
    ident := expression.NewIdentifier($1)
    ident.SetCaseInsensitive(true)
    $$ = ident
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
}
|
/* Self */
SELF
{
    $$ = expression.NewSelf()
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
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
_COVER
{
    if yylex.(*lexer).parsingStatement() {
        yylex.(*lexer).ErrorWithContext("syntax error", $<line>1, $<column>1)
    }
}
LPAREN expr RPAREN
{
    $$ = expression.NewCover($4)
}
|
/* For index keys */
_INDEX_KEY
{
    if yylex.(*lexer).parsingStatement() {
        yylex.(*lexer).ErrorWithContext("syntax error", $<line>1, $<column>1)
    }
}
LPAREN expr RPAREN
{
    $$ = expression.NewIndexKey($4)
}
|
/* For index conditions */
_INDEX_CONDITION
{
    if yylex.(*lexer).parsingStatement() {
        yylex.(*lexer).ErrorWithContext("syntax error", $<line>1, $<column>1)
    }
}
LPAREN expr RPAREN
{
    $$ = expression.NewIndexCondition($4)
}
|
CURRENT USER
{
    $$ = expression.NewCurrentUser()
    $$.(*expression.CurrentUser).SetOperator()
    $$.ExprBase().SetErrorContext($<line>1, $<column>1)
}
;

b_expr:
c_expr
|
// relative path function call
b_expr DOT permitted_identifiers LPAREN opt_exprs RPAREN
{
    var path []string

    switch other := $1.(type) {
    case *expression.SequenceOperation:
        return yylex.(*lexer).FatalError("syntax error", $<line>4, $<column>4)
    case *expression.Identifier:
        path = []string { other.Alias(), $3 }
            dummyPath, err := algebra.NewPathFromElementsWithContext([]string { other.Alias(), $3 },
            yylex.(*lexer).Namespace(), yylex.(*lexer).QueryContext())
        if err != nil {
            return yylex.(*lexer).FatalError(err.Error(), $<line>1, $<column>1)
        }
        path = dummyPath.Parts()
    case *expression.Field:
        tempPath := other.Path()
        if len(tempPath) != 2 {
            return yylex.(*lexer).FatalError("syntax error", $<line>1, $<column>1)
        }
        path = append([]string { yylex.(*lexer).Namespace() }, append(tempPath, $3)...)
    default:
        return yylex.(*lexer).FatalError("syntax error", $<line>1, $<column>1)
    }

    // NewFunctionName() cannot deal with 3 part names, and considers 2 parts as a global function
    // so we have to deal with this ourselves
    name, err := functionsBridge.NewFunctionName(path, yylex.(*lexer).Namespace(), yylex.(*lexer).QueryContext())
    if err != nil {
        yylex.Error(err.Error()+yylex.(*lexer).ErrorContext())
    }
    f := expression.GetUserDefinedFunction(name, yylex.(*lexer).UdfCheck())
    if f != nil {
        $$ = f.Constructor()($5...)
    } else {
        return yylex.(*lexer).FatalError(fmt.Sprintf("Invalid function %v", name.Key()), $<line>1, $<column>1)
    }
}
|
/* Nested */
b_expr DOT permitted_identifiers
{
    switch t := $1.(type) {
    case *expression.SequenceOperation:
        if !t.AddPart($3) {
            l, c := t.ExprBase().GetErrorContext()
            return yylex.(*lexer).FatalError("Invalid sequence name", l, c)
        }
        $$ = t
    default:
        fn := expression.NewFieldName($3, false)
        fn.ExprBase().SetErrorContext($<line>3,$<column>3)
        $$ = expression.NewField($1, fn)
        $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
    }
}
|
b_expr DOT ident_icase
{
    switch t := $1.(type) {
    case *expression.SequenceOperation:
        l, c := t.ExprBase().GetErrorContext()
        return yylex.(*lexer).FatalError("Invalid sequence name", l, c)
    default:
        fn := expression.NewFieldName($3.Identifier(), true)
        fn.ExprBase().SetErrorContext($<line>3,$<column>3)
        field := expression.NewField($1, fn)
        field.SetCaseInsensitive(true)
        $$ = field
        $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
    }
}
|
b_expr DOT LBRACKET expr RBRACKET
{
    switch $1.(type) {
    case *expression.SequenceOperation:
        return yylex.(*lexer).FatalError("syntax error", $<line>2, $<column>2)
    default:
        $$ = expression.NewField($1, $4)
        $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
    }
}
|
b_expr DOT LBRACKET expr RBRACKET_ICASE
{
    switch $1.(type) {
    case *expression.SequenceOperation:
        return yylex.(*lexer).FatalError("syntax error", $<line>2, $<column>2)
    default:
        field := expression.NewField($1, $4)
        field.SetCaseInsensitive(true)
        $$ = field
        $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
    }
}
|
b_expr LBRACKET expr RBRACKET
{
    switch $1.(type) {
    case *expression.SequenceOperation:
        return yylex.(*lexer).FatalError("syntax error", $<line>2, $<column>2)
    default:
        $$ = expression.NewElement($1, $3)
        $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
    }
}
|
b_expr LBRACKET expr COLON RBRACKET
{
    switch $1.(type) {
    case *expression.SequenceOperation:
        return yylex.(*lexer).FatalError("syntax error", $<line>2, $<column>2)
    default:
        $$ = expression.NewSlice($1, $3)
        $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
    }
}
|
b_expr LBRACKET COLON expr RBRACKET
{
    switch $1.(type) {
    case *expression.SequenceOperation:
        return yylex.(*lexer).FatalError("syntax error", $<line>2, $<column>2)
    default:
        $$ = expression.NewSliceEnd($1, $4)
        $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
    }
}
|
b_expr LBRACKET expr COLON expr RBRACKET
{
    switch $1.(type) {
    case *expression.SequenceOperation:
        return yylex.(*lexer).FatalError("syntax error", $<line>2, $<column>2)
    default:
        $$ = expression.NewSlice($1, $3, $5)
        $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
    }
}
|
b_expr LBRACKET COLON RBRACKET
{
    switch $1.(type) {
    case *expression.SequenceOperation:
        return yylex.(*lexer).FatalError("syntax error", $<line>2, $<column>2)
    default:
        $$ = expression.NewSlice($1)
        $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
    }
}
|
b_expr LBRACKET STAR RBRACKET
{
    switch $1.(type) {
    case *expression.SequenceOperation:
        return yylex.(*lexer).FatalError("syntax error", $<line>2, $<column>2)
    default:
        $$ = expression.NewArrayStar($1)
        $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
    }
}
|
/* Arithmetic */
b_expr PLUS b_expr
{
    $$ = expression.NewAdd($1, $3)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
b_expr MINUS b_expr
{
    $$ = expression.NewSub($1, $3)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
b_expr STAR b_expr
{
    $$ = expression.NewMult($1, $3)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
b_expr DIV b_expr
{
    $$ = expression.NewDiv($1, $3)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
b_expr MOD b_expr
{
    $$ = expression.NewMod($1, $3)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
b_expr POW b_expr
{
    $$ = expression.NewPower($1, $3)
    $$.(*expression.Power).SetOperator()
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
}
|
/* Concat */
b_expr CONCAT b_expr
{
    $$ = expression.NewConcat($1, $3)
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
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
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
}
|
MISSING
{
    $$ = expression.MISSING_EXPR
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
}
|
FALSE
{
    $$ = expression.FALSE_EXPR
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
}
|
TRUE
{
    $$ = expression.TRUE_EXPR
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
}
|
NUM
{
    $$ = expression.NewConstant(value.NewValue($1))
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
}
|
INT
{
    $$ = expression.NewConstant(value.NewValue($1))
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
}
|
STR
{
    $$ = expression.NewConstant(value.NewValue($1))
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
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
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
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
expr opt_as_alias
{
    name := $2
    if name == "" {
      name = $1.Alias()
    }
    if name == "" {
        yylex.(*lexer).ErrorWithContext(fmt.Sprintf("Object member missing name or value: %s", $1.String()), $<line>1, $<column>1)
    }

    $$ = algebra.NewPair(expression.NewConstant(name), $1, nil)
}
;

array:
LBRACKET opt_exprs RBRACKET
{
    $$ = expression.NewArrayConstruct($2...)
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
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
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
}
|
POSITIONAL_PARAM
{
    p := int($1)
    if $1 > int64(p) {
        yylex.(*lexer).ErrorWithContext(fmt.Sprintf("Positional parameter out of range: $%v", $1), $<line>1, $<column>1)
    }

    $$ = algebra.NewPositionalParameter(p)
    yylex.(*lexer).countParam()
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
}
|
NEXT_PARAM
{
    n := yylex.(*lexer).nextParam()
    $$ = algebra.NewPositionalParameter(n)
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
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
    $$.ExprBase().SetErrorContext($1.ExprBase().GetErrorContext())
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
    $$.ExprBase().SetErrorContext($1[0].When.ExprBase().GetErrorContext())
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
FLATTEN_KEYS LPAREN opt_flatten_keys_exprs RPAREN
{
    $$ = nil

    fname := "flatten_keys"
    f, ok := expression.GetFunction(fname)
    if ok {
        if len($3) < f.MinArgs() || len($3) > f.MaxArgs() {
            return yylex.(*lexer).FatalError(fmt.Sprintf("Number of arguments to function %s must be between %d and %d.",
                                                          fname, f.MinArgs(), f.MaxArgs()), $<line>3, $<column>3)
        } else {
            $$ = f.Constructor()($3.Expressions()...)
            if fk, ok := $$.(*expression.FlattenKeys); ok {
                fk.SetAttributes($3.Attributes())
            }
        }
    } else {
        return yylex.(*lexer).FatalError(fmt.Sprintf("Invalid function %s.", fname), $<line>1, $<column>1)
    }
}
|
NTH_VALUE LPAREN exprs RPAREN opt_from_first_last opt_nulls_treatment window_function_details
{
    $$ = nil
    fname := "nth_value"
    f, ok := algebra.GetAggregate(fname, false, false, ($7 != nil))
    if ok {
        if len($3) < f.MinArgs() || len($3) > f.MaxArgs() {
            if f.MinArgs() == f.MaxArgs() {
                return yylex.(*lexer).FatalError(fmt.Sprintf("Number of arguments to function %s must be %d.",
                                                              fname, f.MaxArgs()), $<line>3, $<column>3)
            } else {
                return yylex.(*lexer).FatalError(fmt.Sprintf("Number of arguments to function %s must be between %d and %d.",
                                                              fname, f.MinArgs(), f.MaxArgs()), $<line>3, $<column>3)
            }
        } else {
            $$ = f.Constructor()($3...)
            if a, ok := $$.(algebra.Aggregate); ok {
                a.SetAggregateModifiers($5|$6, nil, $7)
            }
            if $3 != nil && len($3) > 0 {
                $$.ExprBase().SetErrorContext($3[0].ExprBase().GetErrorContext())
            }
        }
    } else {
        return yylex.(*lexer).FatalError(fmt.Sprintf("Invalid function %s.", fname), $<line>1, $<column>1)
    }
}
|
function_name LPAREN opt_exprs RPAREN opt_filter opt_nulls_treatment opt_window_function
{
    fname := $1.Identifier()
    $$ = nil
    f, ok := expression.GetFunction(fname)
    if !ok {
        f, ok = search.GetSearchFunction(fname)
    }
    if !ok || $7 != nil {
        f, ok = algebra.GetAggregate(fname, false, ($5 != nil), ($7 != nil))
    }

    if ok {
        if ($6 == algebra.AGGREGATE_RESPECTNULLS && !algebra.AggregateHasProperty(fname, algebra.AGGREGATE_WINDOW_RESPECTNULLS)) ||
           ($6 == algebra.AGGREGATE_IGNORENULLS && !algebra.AggregateHasProperty(fname, algebra.AGGREGATE_WINDOW_IGNORENULLS)) {
            return yylex.(*lexer).FatalError(fmt.Sprintf("RESPECT|IGNORE NULLS syntax is not valid for function %s.", fname),
                                             $<line>6, $<column>6)
        } else if ($5 != nil && !algebra.AggregateHasProperty(fname, algebra.AGGREGATE_ALLOWS_FILTER)) {
            return yylex.(*lexer).FatalError(fmt.Sprintf("FILTER clause syntax is not valid for function %s.", fname),
                                             $<line>5, $<column>5)
        } else if len($3) < f.MinArgs() || len($3) > f.MaxArgs() {
            if f.MinArgs() == f.MaxArgs() {
                return yylex.(*lexer).FatalError(fmt.Sprintf("Number of arguments to function %s must be %d.",
                                                              fname, f.MaxArgs()), $<line>3, $<column>3)
            } else {
                return yylex.(*lexer).FatalError(fmt.Sprintf("Number of arguments to function %s must be between %d and %d.",
                                                              fname, f.MinArgs(), f.MaxArgs()), $<line>3, $<column>3)
            }
        } else {
            $$ = f.Constructor()($3...)
            if a, ok := $$.(algebra.Aggregate); ok {
                a.SetAggregateModifiers($6, $5, $7)
            }
            $$.ExprBase().SetErrorContext($<line>1,$<column>1)
        }
    } else {
        var name functions.FunctionName
        var err errors.Error

        f = nil
        if $5 == nil && $6 == uint32(0) && $7 == nil {
            name, err = functionsBridge.NewFunctionName([]string{fname}, yylex.(*lexer).Namespace(), yylex.(*lexer).QueryContext())
            if err != nil {
                return yylex.(*lexer).FatalError(err.Error(), $<line>1, $<column>1)
            }
            f = expression.GetUserDefinedFunction(name, yylex.(*lexer).UdfCheck())
            if f != nil {
                $$ = f.Constructor()($3...)
            }
        }

        if f == nil {
            var msg string
            if name != nil {
                msg = fmt.Sprintf(" (resolving to %s)", name.Key())
            }
            return yylex.(*lexer).FatalError(fmt.Sprintf("Invalid function %s%s", fname, msg), $<line>1, $<column>1)
        }
    }
}
|
function_name LPAREN agg_quantifier expr RPAREN opt_filter opt_window_function
{
    fname := $1.Identifier()
    agg, ok := algebra.GetAggregate(fname, $3 == algebra.AGGREGATE_DISTINCT, ($6 != nil), ($7 != nil))
    if ok {
        $$ = agg.Constructor()($4)
        if a, ok := $$.(algebra.Aggregate); ok {
            a.SetAggregateModifiers($3, $6, $7)
        }
    } else {
        return yylex.(*lexer).FatalError(fmt.Sprintf("Invalid aggregate function %s.", fname), $<line>1, $<column>1)
    }
}
|
function_name LPAREN STAR RPAREN opt_filter opt_window_function
{
    fname := $1.Identifier()
    if strings.ToLower(fname) != "count" {
        return yylex.(*lexer).FatalError(fmt.Sprintf("Invalid aggregate function %s(*).", fname), $<line>1, $<column>1)
    } else {
        agg, ok := algebra.GetAggregate(fname, false, ($5 != nil), ($6 != nil))
        if ok {
            $$ = agg.Constructor()(nil)
            if a, ok := $$.(algebra.Aggregate); ok {
                a.SetAggregateModifiers(uint32(0), $5, $6)
            }
        } else {
            return yylex.(*lexer).FatalError(fmt.Sprintf("Invalid aggregate function %s.", fname), $<line>1, $<column>1)
        }
    }
}
|
long_func_name LPAREN opt_exprs RPAREN
{
    f := expression.GetUserDefinedFunction($1, yylex.(*lexer).UdfCheck())
    if f != nil {
        $$ = f.Constructor()($3...)
    } else {
        return yylex.(*lexer).FatalError(fmt.Sprintf("Invalid function %v", $1.Key()), $<line>1, $<column>1)
    }
}
;

function_name:
ident
|
// replace() needs special treatment because of the CREATE OR REPLACE FUNCTION statement
REPLACE
{
    $$ = expression.NewIdentifier($1)
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
}
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
    if $2 != nil && len($2) > 0 {
        $$.ExprBase().SetErrorContext($2[0].Expression().ExprBase().GetErrorContext())
    }
}
|
SOME coll_bindings satisfies END
{
    $$ = expression.NewAny($2, $3)
    if $2 != nil && len($2) > 0 {
        $$.ExprBase().SetErrorContext($2[0].Expression().ExprBase().GetErrorContext())
    }
}
|
EVERY coll_bindings satisfies END
{
    $$ = expression.NewEvery($2, $3)
    if $2 != nil && len($2) > 0 {
        $$.ExprBase().SetErrorContext($2[0].Expression().ExprBase().GetErrorContext())
    }
}
|
ANY AND EVERY coll_bindings satisfies END
{
    $$ = expression.NewAnyEvery($4, $5)
    if $4 != nil && len($4) > 0 {
        $$.ExprBase().SetErrorContext($4[0].Expression().ExprBase().GetErrorContext())
    }
}
|
SOME AND EVERY coll_bindings satisfies END
{
    $$ = expression.NewAnyEvery($4, $5)
    if $4 != nil && len($4) > 0 {
        $$.ExprBase().SetErrorContext($4[0].Expression().ExprBase().GetErrorContext())
    }
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
    $$.SetErrorContext($<line>1, $<column>1)
}
|
variable WITHIN expr
{
    $$ = expression.NewBinding("", $1, $3, true)
    $$.SetErrorContext($<line>1, $<column>1)
}
|
variable COLON variable IN expr
{
    $$ = expression.NewBinding($1, $3, $5, false)
    $$.SetErrorContext($<line>1, $<column>1)
}
|
variable COLON variable WITHIN expr
{
    $$ = expression.NewBinding($1, $3, $5, true)
    $$.SetErrorContext($<line>1, $<column>1)
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
    $$.ExprBase().SetErrorContext($2.ExprBase().GetErrorContext())
}
|
FIRST expr FOR coll_bindings opt_when END
{
    $$ = expression.NewFirst($2, $4, $5)
    $$.ExprBase().SetErrorContext($2.ExprBase().GetErrorContext())
}
|
OBJECT expr COLON expr FOR coll_bindings opt_when END
{
    $$ = expression.NewObject($2, $4, $6, $7)
    $$.ExprBase().SetErrorContext($2.ExprBase().GetErrorContext())
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
_CORRELATED
{
    if yylex.(*lexer).parsingStatement() {
        yylex.(*lexer).ErrorWithContext("syntax error", $<line>1, $<column>1)
    }
}
LPAREN fullselect RPAREN
{
    $$ = algebra.NewSubquery($4)
    err := $$.Select().CheckSetCorrelated()
    if err != nil {
        yylex.(*lexer).FatalError(fmt.Sprintf("Unexpected error in handling of CORRELATED subquery %v", err), $<line>3, $<column>3)
    }
    $$.ExprBase().SetErrorContext($<line>3,$<column>3)
}
|
LPAREN fullselect RPAREN
{
    $$ = algebra.NewSubquery($2)
    $$.ExprBase().SetErrorContext($<line>1,$<column>1)
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
permitted_identifiers AS window_specification
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
permitted_identifiers
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
OVER permitted_identifiers
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
permitted_identifiers
{
    $$ = $1
}
;

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

opt_with_clause:
/* empty */
{
    $$ = nil
}
|
with_clause
;
with_clause:
WITH expr
{
    $$ = $2.Value()
    if $$ == nil {
        return yylex.(*lexer).FatalError("WITH value must be static", $<line>2, $<column>2)
    }
}
;

/*************************************************
 *
 * Sequences
 *
 *************************************************/

opt_namespace_name:
{
    $$ = yylex.(*lexer).Namespace()
}
|
namespace_name
{
    $$ = $1
}
;

sequence_object_name:
permitted_identifiers
{
  $$ = strings.TrimSpace($1)
  if $$ != $1 || $$ == "" {
    return yylex.(*lexer).FatalError(fmt.Sprintf("Invalid identifier '%v'", $1), $<line>1, $<column>1)
  }
}
|
_invalid_case_insensitive_identifier
{
    return yylex.(*lexer).FatalError("Invalid sequence name", $<line>1, $<column>1)
}
;

sequence_full_name:
opt_namespace_name sequence_object_name
{
    p, err := algebra.NewVariablePathWithContext($2, $1, yylex.(*lexer).QueryContext())
    if err != nil {
        return yylex.(*lexer).FatalError(err.Error(), $<line>2, $<column>2)
    }
    if p == nil || p.Scope() == "" {
        return yylex.(*lexer).FatalError("Invalid sequence name", $<line>2, $<column>2)
    }
    $$ = p
}
|
opt_namespace_name path_part DOT path_part DOT sequence_object_name
{
    $$ = algebra.NewPathLong($1, $2, $4, $6)
}
|
opt_namespace_name path_part DOT sequence_object_name
{
    p, err := algebra.NewPathFromElementsWithContext([]string{$2, $4}, $1, yylex.(*lexer).QueryContext())
    if err != nil {
        return yylex.(*lexer).FatalError(err.Error(), $<line>2, $<column>2)
    }
    if p == nil || p.Scope() == "" {
        return yylex.(*lexer).FatalError("Invalid sequence name", $<line>2, $<column>2)
    }
    $$ = p
}
;

sequence_stmt:
create_sequence
|
drop_sequence
|
alter_sequence
;

create_sequence:
CREATE SEQUENCE sequence_name_options opt_seq_create_options
{
    failOnExists := true
    var name *algebra.Path
    for _, v := range $3 {
        if v.name == "" {
            if !failOnExists {
                return yylex.(*lexer).FatalError("syntax error - IF NOT EXISTS already specified", v.line, v.column)
            }
            failOnExists =false
        } else {
            if name != nil {
                return yylex.(*lexer).FatalError("syntax error - name already provided", v.line, v.column)
            }
            name = algebra.NewPathFromElements(algebra.ParsePath(v.name))
        }
    }
    if name == nil {
        return yylex.(*lexer).FatalError("syntax error - sequence name is required", $<line>3, $<column>3)
    }
    var with value.Value
    validate := func(m map[string]interface{}, v *nameValueContext) string {
        if v.name == "with" {
            if len(m) != 0 {
                return "WITH may not be used with other options"
            } else if v.value != nil {
                val := value.NewValue(v.value)
                if val.Type() == value.OBJECT {
                    m["with"] = val
                    return ""
                }
            }
        } else {
            if _, ok := m["with"]; ok {
              return "options may not be used with WITH clause"
            }
        }
        if _, ok := m[v.name]; ok {
            return "duplicate option"
        }
        if v.value == nil {
            return "invalid option value"
        }
        val := value.NewValue(v.value)
        if v.name == sequences.OPT_CYCLE && val.Type() == value.BOOLEAN {
            m[v.name] = val.Truth()
            return ""
        } else if val.Type() == value.NUMBER {
            if i, ok := value.IsIntValue(val); ok {
                  m[v.name] = i
                  return ""
            }
        }
        return "invalid option value"
    }
    if len($4) > 0 {
        m := make(map[string]interface{},len($4))
        for _, vp := range $4 {
            if err := validate(m, vp); err != "" {
                return yylex.(*lexer).FatalError("syntax error - " + err, vp.line, vp.column)
            }
        }
        if len(m) > 0 {
            if w, ok := m["with"]; ok {
                with = value.NewValue(w)
            } else {
                with = value.NewValue(m)
            }
        }
    }
    $$ = algebra.NewCreateSequence(name, failOnExists, with)
}
;

sequence_name_options:
sequence_name_option
{
    $$ = append([]*nameValueContext(nil), $1)
}
|
sequence_name_options sequence_name_option
{
    $$ = append($1, $2)
}
;

sequence_name_option:
IF NOT EXISTS
{
    $$ = &nameValueContext{"", nil, $<line>1, $<column>1}
}
|
sequence_full_name
{
    $$ = &nameValueContext{$1.SimpleString(), nil, $<line>1, $<column>1}
}
;

opt_seq_create_options:
/* empty */
{
    $$ = nil
}
|
opt_seq_create_options seq_create_option
{
    $$ = append($1, $2)
}
;

seq_create_option: sequence_with | start_with | increment_by | maxvalue | minvalue | cycle | cache ;

drop_sequence:
DROP SEQUENCE sequence_full_name opt_if_exists
{
    $$ = algebra.NewDropSequence($3, $4)
}
|
DROP SEQUENCE IF EXISTS sequence_full_name
{
    $$ = algebra.NewDropSequence($5, false)
}
;

alter_sequence:
ALTER SEQUENCE sequence_full_name with_clause
{
    if $4 == nil || $4.Type() != value.OBJECT || !$4.Truth() {
        return yylex.(*lexer).FatalError("syntax error - invalid options object", $<line>4, $<column>4)
    }
    $$ = algebra.NewAlterSequence($3, $4)
}
|
ALTER SEQUENCE sequence_full_name seq_alter_options
{
    if len($4) == 0 {
        return yylex.(*lexer).FatalError("syntax error - missing options", $<line>4, $<column>4)
    }
    var with value.Value
    validate := func(m map[string]interface{}, v *nameValueContext) string {
        if _, ok := m[v.name]; ok {
            return "duplicate option"
        }
        if v.value == nil {
            return "invalid option value"
        }
        val := value.NewValue(v.value)
        if v.name == sequences.OPT_CYCLE && val.Type() == value.BOOLEAN {
            m[v.name] = val.Truth()
            return ""
        } else if v.name == sequences.OPT_RESTART && val.Type() == value.NULL {
            m[v.name] = true
            return ""
        } else if val.Type() == value.NUMBER {
            if i, ok := value.IsIntValue(val); ok {
                  m[v.name] = i
                  return ""
            }
        }
        return "invalid option value"
    }
    m := make(map[string]interface{},len($4))
    for _, vp := range $4 {
        if err := validate(m, vp); err != "" {
            return yylex.(*lexer).FatalError("syntax error - " + err, vp.line, vp.column)
        }
    }
    if len(m) > 0 {
        with = value.NewValue(m)
    }
    $$ = algebra.NewAlterSequence($3, with)
}
;

seq_alter_options:
seq_alter_option
{
  $$ = append([]*nameValueContext(nil), $1)
}
|
seq_alter_options seq_alter_option
{
  $$ = append($1, $2)
}
;

seq_alter_option: restart_with | increment_by | maxvalue | minvalue | cycle | cache ;

sequence_with:
WITH expr
{
    v := $2.Value()
    if v == nil {
        return yylex.(*lexer).FatalError("WITH value must be static", $<line>2, $<column>2)
    }
    $$ = &nameValueContext{"with", v, $<line>1, $<column>1}
}
;

start_with:
START WITH expr
{
    v := $3.Value()
    if v == nil {
        return yylex.(*lexer).FatalError("Option value must be static", $<line>3, $<column>3)
    }
    $$ = &nameValueContext{sequences.OPT_START, v, $<line>1, $<column>1}
}
;

restart_with:
RESTART
{
    $$ = &nameValueContext{sequences.OPT_RESTART, value.NULL_VALUE, $<line>1, $<column>1}
}
|
RESTART WITH expr
{
    v := $3.Value()
    if v == nil {
        return yylex.(*lexer).FatalError("Option value must be static", $<line>3, $<column>3)
    }
    $$ = &nameValueContext{sequences.OPT_RESTART, v, $<line>1, $<column>1}
}
;

increment_by:
INCREMENT BY expr
{
    v := $3.Value()
    if v == nil {
        return yylex.(*lexer).FatalError("Option value must be static", $<line>3, $<column>3)
    }
    $$ = &nameValueContext{sequences.OPT_INCR, v, $<line>1, $<column>1}
}
;

maxvalue:
NO MAXVALUE
{
    $$ = &nameValueContext{sequences.OPT_MAX, value.NewValue(math.MaxInt64), $<line>1, $<column>1}
}
|
MAXVALUE expr
{
    v := $2.Value()
    if v == nil {
        return yylex.(*lexer).FatalError("Option value must be static", $<line>2, $<column>2)
    }
    $$ = &nameValueContext{sequences.OPT_MAX, v, $<line>1, $<column>1}
}
;

minvalue:
NO MINVALUE
{
    $$ = &nameValueContext{sequences.OPT_MIN, value.NewValue(math.MinInt64), $<line>1, $<column>1}
}
|
MINVALUE expr
{
    v := $2.Value()
    if v == nil {
        return yylex.(*lexer).FatalError("Option value must be static", $<line>2, $<column>2)
    }
    $$ = &nameValueContext{sequences.OPT_MIN, v, $<line>1, $<column>1}
}
;

cycle:
NO CYCLE
{
    $$ = &nameValueContext{sequences.OPT_CYCLE, value.FALSE_VALUE, $<line>1, $<column>1}
}
|
CYCLE
{
    $$ = &nameValueContext{sequences.OPT_CYCLE, value.TRUE_VALUE, $<line>1, $<column>1}
}
;

cache:
NO CACHE
{
    $$ = &nameValueContext{sequences.OPT_CACHE, value.ONE_VALUE, $<line>1, $<column>1}
}
|
CACHE expr
{
    v := $2.Value()
    if v == nil {
        return yylex.(*lexer).FatalError("Option value must be static", $<line>2, $<column>2)
    }
    $$ = &nameValueContext{sequences.OPT_CACHE, v, $<line>1, $<column>1}
}
;

sequence_next:
NEXTVAL FOR NAMESPACE_ID COLON permitted_identifiers
{
  $$ = []string{$3,$5}
}
|
NEXT VALUE FOR NAMESPACE_ID COLON permitted_identifiers
{
  $$ = []string{$4,$6}
}
|
NEXTVAL FOR permitted_identifiers
{
  $$ = []string{"",$3}
}
|
NEXT VALUE FOR permitted_identifiers
{
  $$ = []string{"",$4}
}
;

sequence_prev:
PREVVAL FOR NAMESPACE_ID COLON permitted_identifiers
{
  $$ = []string{$3,$5}
}
|
PREV VALUE FOR NAMESPACE_ID COLON permitted_identifiers
{
  $$ = []string{$4,$6}
}
|
PREVVAL FOR permitted_identifiers
{
  $$ = []string{"",$3}
}
|
PREV VALUE FOR permitted_identifiers
{
  $$ = []string{"",$4}
}
;

sequence_expr:
sequence_next
{
    var defs []string
    if $1[0] == "" {
        defs = algebra.ParseQueryContext(yylex.(*lexer).QueryContext())
        if defs[0] == "" {
            defs[0] = "default"
        }
    } else {
        defs = $1[:1]
    }
    s := expression.NewSequenceNext(defs...)
    s.ExprBase().SetErrorContext($<line>1, $<column>1)
    if !s.AddPart($1[1]) {
        return yylex.(*lexer).FatalError("Invalid sequence name", $<line>1, $<column>1)
    }
    $$ = s
}
|
sequence_prev
{
    var defs []string
    if $1[0] == "" {
        defs = algebra.ParseQueryContext(yylex.(*lexer).QueryContext())
        if defs[0] == "" {
            defs[0] = "default"
        }
    } else {
        defs = $1[:1]
    }
    s := expression.NewSequencePrev(defs...)
    s.ExprBase().SetErrorContext($<line>1, $<column>1)
    if !s.AddPart($1[1]) {
        return yylex.(*lexer).FatalError("Invalid sequence name", $<line>1, $<column>1)
    }
    $$ = s
}
;
