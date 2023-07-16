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
  command/build.sh `realpath ../..`
fi
echo go build -ldflags "-X github.com/couchbase/query/shell/cbq/command.SHELL_VERSION=${PRODUCT_VERSION}"
go build -ldflags "-X github.com/couchbase/query/shell/cbq/command.SHELL_VERSION=${PRODUCT_VERSION}"
