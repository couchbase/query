#!/bin/bash

# Copyright 2014-Present Couchbase, Inc.
#
# Use of this software is governed by the Business Source License included in
# the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
# file, in accordance with the Business Source License, use of this software
# will be governed by the Apache License, Version 2.0, included in the file
# licenses/APL2.txt.

#
# To build the enterprise version, launch this  AS './build.sh -tags enterprise'
# To build the enterprise version with latest updates, launch this  AS './build.sh -u -tags enterprise'
# Add -s to fix standalone build issues. Keep indexer generated files in ~/devbld
# Note standalone build requires libraries from installed server, make sure installed server is
# compatible with source that is being built

PRODUCT_VERSION=${PRODUCT_VERSION:-"7.2.0-local_build"}
export PRODUCT_VERSION

args=""

enterprise=0
uflag=
sflag=0
fflag=1
while [ $# -gt 0 ]; do
  case $1 in
    -tags)
      shift
      [[ "$1" == "enterprise" ]] && enterprise=1
      args="$args -tags $1"
      ;;
    -u) uflag=-u ;;
    -s) sflag=1 ;;
    -S) sflag=2 ;;
    -nofmt) fflag=0 ;;
    *) args="$args $1" ;;
  esac
  shift
done

set -- $args

GIT=`which git`

cbranch=`$GIT rev-parse --abbrev-ref HEAD`
rbranch=`$GIT log -n 5 --pretty=format:"%D"|awk 'NF>0{p=$NF}END{print p}'`
defbranch="master"

function checkout_if_necessary {
  local current=`$GIT rev-parse --abbrev-ref HEAD 2>/dev/null`
  local commit=`$GIT log -n 1 --pretty=format:"%h" 2>/dev/null`

  if [[ -z $current ]]
  then
    # isn't a repo so can't check anything out
    return
  fi

  local report=""
  local res=""
  # if there is no subpath passed in then we'll report errors else remain silent
  report_errors=$1
  shift

  D=`echo ${PWD}|sed -E 's,.*github.com/couchbase/,,;s,.*golang.org,golang.org,'`
  #echo "checkout_if_necessary: [${PWD}] ${D} -> $@"

  while [[ $# > 0 ]]
  do
    branch=$1
    shift
    if [[ $branch == $current || $branch == $commit ]]
    then
      return
    fi
    cmd="$GIT checkout $branch"
    res=`$cmd 2>&1`
    if [[ $res =~ "did not match any file" ]]
    then
      # try refreshing the repo
      ($GIT pull 2>/dev/null 1>/dev/null)  # no need to report status
      res=`$cmd 2>&1`
    fi
    if [[ ! $res =~ "is now at" ]]
    then
      report="${report}${D} -> ${cmd}:\n${res}\n"
    else
      return  # success
    fi
  done
  if [[ -z $report_errors && -n $report ]]
  then
    echo -e "$report"
    echo
  fi
}

function get_repo {
    local path=$1
    local mcommit=$2
    local subpath=$3
    local scommit=$4
    shift
    shift
    shift
    shift

    #echo "get_repo: $path $mcommit $subpath $scommit $@"

    if [ ! -d $GOPATH/src/$path ]
    then
        if [[ $path =~ "github.com" ]]
        then
            url="git@${path/\//:}.git"
        else
            url="https://${path}.git"
        fi
        $GIT clone -b $mcommit $url $GOPATH/src/$path
    fi
    cd $GOPATH/src/$path
    checkout_if_necessary $subpath $mcommit $@
    if [[ $subpath != "" ]]
    then
        if [[ ! -d $subpath ]]
        then
            if [[ $path =~ "github.com" ]]
            then
                url="git@${path/\//:}.git"
            else
                url="https://${path}.git"
            fi
            $GIT clone -b $mcommit $url $subpath
        fi
        (cd $subpath; checkout_if_necessary "" $scommit $@)
    fi
    cd - >> /dev/null
}

function get_path_subpath_commit {
    #echo "get_path_subpath_commit: $@"
    local repo=$1
    local ipath=$2
    local vers=$3
    shift
    shift
    shift

    subpath=`echo $ipath|awk -F/ '{print $NF}'`
    if [[ $subpath == $repo ]]; then
        subpath=""
        path=$ipath
    else
        path=`echo $ipath|sed -E 's,/[^/]+$,,'`
    fi

    if [[ $vers == "" ]]; then
         commit=$1
    else
         commit=`echo $vers|awk -F- '{print $NF}'`
    fi

    if [[ $subpath != "" ]]; then
         mcommit=$1
    else
         mcommit=$commit
    fi

    get_repo "$path" "$mcommit" "$subpath" "$commit" $@

}


function repo_by_gomod {
    #echo "repo_by_gomod: $@"
    local file=$1
    local repo=$2
    local subbranch=$3
    shift
    shift
    shift

    C=`grep replace $file | grep "../${repo}" | grep -v "module" | grep -v "${repo}-"`
    if [[ -n "$C" ]]
    then
        opath=`echo $C| awk '{print $2}'`
        get_path_subpath_commit "$repo" "$opath" "" $@
    else
        grepo=$repo
        if [[ $subbranch != "" ]]; then
            grepo="${repo}/${subbranch}"
        fi
        C=`grep "${grepo}" $file | grep -v module | grep -v replace | grep -v indirect| grep -v "${repo}-"`
        if [[ -z "$C" ]]
        then
            C="../${repo} $4"
        fi
        gpath=`echo $C| awk '{print $1}'`
        vers=`echo $C| awk '{print $2}'`
        get_path_subpath_commit "$repo" "$gpath" "$vers" $@
    fi
}


function repo_setup {
    repo_by_gomod go.mod query "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod query-ee "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod indexing "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod go-couchbase "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod gomemcached "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod cbauth "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod godbc "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod goutils "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod go_json "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod gometa "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod eventing-ee "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod n1fty "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod cbgt "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod cbft "" $cbranch $rbranch $defbranch
    repo_by_gomod ../n1fty/go.mod bleve "" $defbranch
    repo_by_gomod ../n1fty/go.mod bleve "v2" $defbranch
    repo_by_gomod ../cbft/go.mod zapx "v11" $defbranch
    repo_by_gomod ../cbft/go.mod zapx "v12" $defbranch
    repo_by_gomod ../cbft/go.mod zapx "v13" $defbranch
    repo_by_gomod ../cbft/go.mod zapx "v14" $defbranch
    repo_by_gomod ../cbft/go.mod zapx "v15" $defbranch
    repo_by_gomod go.mod gocbcore-transactions "" $defbranch
    repo_by_gomod go.mod gocbcore "v10" $defbranch
    repo_by_gomod ../cbgt/go.mod gocbcore "v9" $defbranch
    repo_by_gomod go.mod x/net "" `go version |  awk -F'[. ]' '{print "release-branch." $3 "." $4}'` $defbranch
}

function DevStandaloneSetup {
    # curl fix match manifest
    (cd ../../couchbasedeps/go-curl; checkout_if_necessary "" 20161221-couchbase)

    repo_setup

    # indexer generated files
    if [[ (! -f ../indexing/secondary/protobuf/query/query.pb.go) ]]
    then
        base=
        if [[ -d ~/devbld ]]
        then
            base=~/devbld
        elif [[ -d ~/code/devbld ]]
        then
            base=~/code/devbld
        fi
        if [[ -n "${base}" ]]
        then
            if [[ -f $base/$cbranch.query.pb.go ]]
            then
                cp $base/$cbranch.query.pb.go ../indexing/secondary/protobuf/query/query.pb.go
            elif [[ -f $base/query.pb.go ]]
            then
                cp $base/query.pb.go ../indexing/secondary/protobuf/query/query.pb.go
            fi
        fi
    fi
    # eventing-ee generated files
    if [[ ( ! -L $GOPATH/lib ) ]]; then
    if [[ -d $GOPATH/lib ]]
    then
    rm -rf $GOPATH/lib
    fi
    if [[ "Linux" = `uname` ]]
    then
    ln -s /opt/couchbase/lib $GOPATH/lib
    elif [[ "Darwin" = `uname` ]]
    then
    ln -s "/Applications/Couchbase Server.app/Contents/Resources/couchbase-core/lib" $GOPATH/lib
    fi
    fi
    if [[ ! -f ../eventing-ee/evaluator/impl/gen/parser/global_config_schema.go ]]
    then
    (cd ../eventing-ee/evaluator/impl/gen/convertschema; go run generate.go  ../../parser/global_config_schema.json GlobalConfigSchema ../parser/global_config_schema.go)
    fi
}

# turn off go module for non repo sync build or standalone build
if [[ ( ! -d ../../../../../cbft && "$GOPATH" != "") || ( $sflag != 0) ]]; then
    export GO111MODULE=off
    export CGO_CFLAGS="-I$GOPATH/src/github.com/couchbase/eventing-ee/evaluator/worker/include $CGO_FLAGS"
    export CGO_LDFLAGS="-L$GOPATH/lib $CGO_LDFLAGS"
    export LD_LIBRARY_PATH=$GOPATH/lib:${LD_LIBRARY_PATH}
    cmd="go get $* $uflag -d -v ./..."
    echo $cmd
    $cmd
    if [[ $sflag == 1 ]]; then
        DevStandaloneSetup
        $cmd
    fi
fi

echo cd parser/n1ql
cd parser/n1ql
./build.sh $*
cd ../..

if [[ ($fflag != 0) ]]
then
  echo go fmt ./...
  go fmt ./...
  if [[ $enterprise == 1 ]]; then
    (echo go fmt ../query-ee/...; cd ../query-ee; export GO111MODULE=off; go fmt ./...)
  fi
fi

echo cd server/cbq-engine
cd server/cbq-engine
./build.sh $*
cd ../..

echo cd shell/cbq
cd shell/cbq
./build.sh $*
cd ../..

echo cd tutorial
cd tutorial
./build.sh $*
cd ..

echo go install  $* ./...
go install $* ./...
