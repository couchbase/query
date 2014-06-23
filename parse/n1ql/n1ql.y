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
%left           JOIN NEST UNNEST LEFT
%left           OR
%left           AND
%right          NOT
%nonassoc       EQ NE
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
SELECT {
        logDebugGrammar("INPUT")
}
