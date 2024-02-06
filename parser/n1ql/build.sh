#! /bin/bash

# Copyright 2014-Present Couchbase, Inc.
#
# Use of this software is governed by the Business Source License included in
# the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
# file, in accordance with the Business Source License, use of this software
# will be governed by the Apache License, Version 2.0, included in the file
# licenses/APL2.txt.

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
    cat << EOF > n1ql.nn.tmp
//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

EOF
    awk '/NEX_END_OF_LEXER_STRUCT/ { print "curOffset int\nreportError func(what string)" }
	 { print }' n1ql.nn.go >> n1ql.nn.tmp
    mv n1ql.nn.tmp n1ql.nn.go
    go fmt n1ql.nn.go
fi
echo goyacc n1ql.y
goyacc n1ql.y
echo go build $*
go build $*
