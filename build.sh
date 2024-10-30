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

DEF_VERSION="local_build"
if [ -f /opt/couchbase/VERSION.txt ]
then
  DEF_VERSION=`head -1 /opt/couchbase/VERSION.txt|sed 's/-.*/-local_build/'`
elif [ -f "/Applications/Couchbase Server.app/Contents/Resources/couchbase-core/VERSION.txt" ]
then
  DEF_VERSION=`head -1 "/Applications/Couchbase Server.app/Contents/Resources/couchbase-core/VERSION.txt"\
    |sed 's/-.*/-local_build/'`
fi

cwd1=`pwd`
PRODUCT_VERSION=${PRODUCT_VERSION:-$DEF_VERSION}
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
    cmd1="$GIT checkout $branch"
    res=`$cmd1 2>&1`
    if [[ $res =~ "did not match any file" ]]
    then
      # try refreshing the repo
      ($GIT pull 2>/dev/null 1>/dev/null)  # no need to report status
      res=`$cmd1 2>&1`
    fi
    if [[ ! $res =~ "is now at" && ! $res =~ "Switched to a new branch" ]]
    then
      report="${report}${D} -> ${cmd1}:\n${res}\n"
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
        $GIT clone $url $GOPATH/src/$path
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
    tools=`echo $ipath|awk -F/ '{print $(NF-2)}'`
    if [[ $tools == "tools-common" ]]; then
        tag=`echo $ipath|awk -F/ '{print $(NF-1)}'`
        gpath=`echo $ipath|awk -F/ '{print  $1 "/" $2 "/" $3}'`
        ver=$tag/$vers
        get_repo "$gpath" "$ver" "" "$ver" $ver
        return
    fi

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
        C=`grep -w "${grepo}" $file | grep -v module | grep -v replace | grep -v indirect| grep -v "${repo}-"`
        bpath=`echo $repo|awk -F/ '{print $1}'`
        if [ -z "$C" ]; then
            C=`grep -w "${grepo}" $file | grep -v module | grep -v replace | grep indirect| grep -v "${repo}-"`
        fi
        if [[ -z "$C" ]]
        then
            C="github.com/couchbase/${repo} $4"
        fi
        gpath=`echo $C| awk '{print $1}'`
	vers=`echo $C| awk '{print $2}'|awk -F- '{print $NF}'`
	if [[ $@ == "" ]]; then
	     # use go.mod version
             get_path_subpath_commit "$repo" "$gpath" "$vers" "" "$vers"
	else
             get_path_subpath_commit "$repo" "$gpath" "$vers" $@
	fi
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
    repo_by_gomod go.mod regulator "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod sigar "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod cbgt "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod cbft "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod hebrew "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod cbftx "" $cbranch $rbranch $defbranch
}

function DevStandaloneSetup {

    repo_setup

    ( dir=`echo $cwd1 |awk -F/ '{print  $(NF-4) "/" $(NF-3) "/" $(NF-2) "/" $(NF-1)}'`;
      cd $GOPATH/..;
      ln -s -f $dir/cbgt cbgt;
      ln -s -f $dir/cbft cbft;
      ln -s -f $dir/cbftx cbftx;
      ln -s -f $dir/hebrew hebrew;
      cd $cwd1)

    # indexer generated files
    if [[ -f ~/devbld/protoc-gen-go ]]
    then
            ln -sf ~/devbld/protoc-gen-go $GOPATH/bin
	    (cd $GOPATH/src/github.com/couchbase/indexing/secondary/protobuf/query; protoc -I. --plugin=protoc-gen-go=$GOPATH/bin//protoc-gen-go query.proto --go_out=`pwd`)
    fi

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
    if [[ ! -f ../eventing-ee/evaluator/impl/gen/parser/dynamic_config_schema.go ]]
    then
         (cd ../eventing-ee/evaluator/impl/gen/convertschema; go run generate.go  ../../parser/dynamic_config_schema.json DynamicConfigSchema ../parser/dynamic_config_schema.go)
    fi
    if [[ ! -f ../eventing-ee/evaluator/impl/v8wrapper/process_manager/gen/flatbuf/payload_generated.h ]]; then
	    (cd ../eventing-ee/evaluator/impl/v8wrapper/process_manager/gen; flatc -c -o flatbuf ../flatbuf/payload.fbs; flatc -g -o . ../flatbuf/payload.fbs)
    fi
}

# turn off go module for non repo sync build or standalone build
if [[ ( ! -d ../../../../../cbft && "$GOPATH" != "") || ( $sflag != 0) ]]; then
    export CGO_CFLAGS="-I$GOPATH/src/github.com/couchbase/eventing-ee/evaluator/worker/include -I$GOPATH/src/github.com/couchbase/sigar/include $CGO_FLAGS"
    export CGO_LDFLAGS="-L$GOPATH/lib $CGO_LDFLAGS"
    export LD_LIBRARY_PATH=$GOPATH/lib:${LD_LIBRARY_PATH}
    if [[ $sflag == 1 ]]; then
        DevStandaloneSetup
    fi
fi

cd $cwd1
echo cd parser/n1ql
cd parser/n1ql
./build.sh $*
cd ../..

if [[ ($fflag != 0) ]]
then
  echo go fmt ./...
  go fmt ./...
  if [[ $enterprise == 1 ]]; then
    (echo go fmt ../query-ee/...; cd ../query-ee; go fmt ./...)
  fi
fi

echo cd server/cbq-engine
cd server/cbq-engine
./build.sh $*
if [ $? -ne 0 ]
then
  exit 1
fi
cd ../..

echo cd shell/cbq
cd shell/cbq
./build.sh $*
if [ $? -ne 0 ]
then
  exit 1
fi
cd ../..

echo cd tutorial
cd tutorial
./build.sh $*
if [ $? -ne 0 ]
then
  exit 1
fi
cd ..

echo cd finderr
cd finderr
./build.sh $*
if [ $? -ne 0 ]
then
  exit 1
fi
cd ..

echo go install  $* ./...
go install $* ./...
