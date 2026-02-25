#!/bin/bash

# Copyright 2026-Present Couchbase, Inc.
#
# Use of this software is governed by the Business Source License included in
# the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
# file, in accordance with the Business Source License, use of this software will
# be governed by the Apache License, Version 2.0, included in the file
# licenses/APL.txt.

function get_openssl_version {
    local openssl_bin=""
    if [[ "Linux" = `uname` ]]; then
        openssl_bin="/opt/couchbase/bin/openssl"
    elif [[ "Darwin" = `uname` ]]; then
        openssl_bin="/Applications/Couchbase Server.app/Contents/Resources/couchbase-core/bin/openssl"
    fi

    if [[ -f "$openssl_bin" ]]; then
        local version=$("$openssl_bin" version | awk '{print $2}')
        echo "openssl-${version}"
    else
        echo "Error: OpenSSL binary not found at $openssl_bin" >&2
        exit 1
    fi
}
