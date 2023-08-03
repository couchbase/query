# Copyright 2014-Present Couchbase, Inc.
#
# Use of this software is governed by the Business Source License included in
# the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
# file, in accordance with the Business Source License, use of this software
# will be governed by the Apache License, Version 2.0, included in the file
# licenses/APL2.txt.

if [ ../../parser/n1ql/n1ql.y -nt command/syntax_data.go ]
then
  echo "Generating syntax help data"
  # Test for options since not all platforms have all tools... in order of preference
  type realpath > /dev/null 2>&1
  if [ $? -eq 0 ]
  then
    dir=`realpath ../..`
  else
    type readlink > /dev/null 2>&1
    if [ $? -eq 0 ]
    then
      dir=`readlink -f ../..`
    else
      dir=`echo $PWD|sed 's,/shell/cbq,,'`
    fi
  fi
  command/build.sh $dir
fi
echo go build -ldflags "-X github.com/couchbase/query/shell/cbq/command.SHELL_VERSION=${PRODUCT_VERSION}"
go build -ldflags "-X github.com/couchbase/query/shell/cbq/command.SHELL_VERSION=${PRODUCT_VERSION}"
