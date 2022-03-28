# Copyright 2015-Present Couchbase, Inc.
#
# Use of this software is governed by the Business Source License included in
# the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
# file, in accordance with the Business Source License, use of this software will
# be governed by the Apache License, Version 2.0, included in the file
# licenses/APL.txt.

export NS_SERVER_CBAUTH_URL="http://localhost:8091/_cbauth"
export NS_SERVER_CBAUTH_USER="Administrator"
export NS_SERVER_CBAUTH_PWD="password"
export NS_SERVER_CBAUTH_RPC_URL="http://127.0.0.1:8091/cbauth-demo"

export CBAUTH_REVRPC_URL="http://Administrator:password@localhost:8091/query"
export LD_LIBRARY_PATH=${GOPATH}/lib:${LD_LIBRARY_PATH}

./cbq-engine "$@"
