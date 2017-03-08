#! /bin/bash
NEX=github.com/couchbaselabs/nex

if [ n1ql.nex -nt n1ql.nn.go ]
then 
    echo nex n1ql.nex
    go get $NEX
    BACK=`pwd`
    cd $GOPATH/src/$NEX
    go build
    cd $BACK
    $GOPATH/src/$NEX/nex n1ql.nex
    awk '/NEX_END_OF_LEXER_STRUCT/ { print "curOffset int"; }
	 { print }' n1ql.nn.go > n1ql.nn.tmp
    mv n1ql.nn.tmp n1ql.nn.go
    go fmt n1ql.nn.go
fi
echo go tool yacc n1ql.y
go tool yacc n1ql.y
echo go build
go build
