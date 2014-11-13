#! /bin/bash

echo nex n1ql.nex
nex n1ql.nex
echo go tool yacc n1ql.y
go tool yacc n1ql.y
echo go build
go build
