The generated parser is already included this repo.

These instructions are only required for developers modifying the
parser.

### Building Parser

1.  Install nex - https://github.com/blynn/nex
2.  Install go's yacc tool - http://golang.org/cmd/yacc/
3.  Run build.sh

### Adding a keyword

Declare it with `%token` in n1ql.y and add it to the `keywords` map in
keywords.go. Do not add a rule for it to n1ql.nex: keywords are matched by the
identifier rule.
