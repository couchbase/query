#!/bin/sh

export NS_SERVER_CBAUTH_URL="http://localhost:8091/_cbauth"
export NS_SERVER_CBAUTH_USER="Administrator"
export NS_SERVER_CBAUTH_PWD="password"
export NS_SERVER_CBAUTH_RPC_URL="http://127.0.0.1:8091/cbauth-demo"
export CBAUTH_REVRPC_URL="http://Administrator:password@localhost:8091/query"
export CBAUTH_TLS_CONFIG="{}"
export GSI_TEST=true
export GO111MODULE=off
