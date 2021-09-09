# Copyright 2014-Present Couchbase, Inc.
#
# Use of this software is governed by the Business Source License included in
# the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
# file, in accordance with the Business Source License, use of this software will
# be governed by the Apache License, Version 2.0, included in the file
# licenses/APL.txt.

echo go build -ldflags "-X github.com/couchbase/query/util.VERSION=${PRODUCT_VERSION}" $*
go build -ldflags "-X github.com/couchbase/query/util.VERSION=${PRODUCT_VERSION}" $*
