#! /bin/bash

if [ n1ql.nex -nt n1ql.nn.go ]
then 
    echo nex n1ql.nex
    nex n1ql.nex
    awk '/NEX_END_OF_LEXER_STRUCT/ { print "curOffset int"; }
	 { print }' n1ql.nn.go > n1ql.nn.tmp
    mv n1ql.nn.tmp n1ql.nn.go
    go fmt n1ql.nn.go
fi
echo goyacc n1ql.y
goyacc n1ql.y
echo go build
go build
