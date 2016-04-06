#! /bin/bash

if [ n1ql.nex -nt n1ql.nn.go ]
then 
    echo nex n1ql.nex
    nex n1ql.nex
    go fmt n1ql.nn.go
fi
echo go tool yacc n1ql.y
go tool yacc n1ql.y
echo go build
go build
