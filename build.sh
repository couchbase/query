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

PRODUCT_VERSION=${PRODUCT_VERSION:-"7.1.0-local_build"}
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
    -nofmt) fflag=0 ;;
    *) args="$args $1" ;;
  esac
  shift
done

set -- $args

cbranch=`git rev-parse --abbrev-ref HEAD`
defbranch="master"


get_repo() {
     local path=$1
     local mcommit=$2
     local subpath=$3
     local scommit=$4

     #echo "$path" "$mcommit" "$subpath" "$scommit"

     cd $GOPATH/src/$path
     abranch=`git branch | awk '{print $2}'`
     if [[ $abranch != $mcommit ]]; then
         git checkout $mcommit
     fi
     if [[ $subpath != "" ]]
     then
         if [[ ! -d $subpath ]]
         then
            url="https://"$path
            git clone -b $mcommit $url $subpath
         fi
         (cd $subpath; git checkout $scommit)
     fi
     cd - >> /dev/null
}

get_path_subpath_commit() {
    dirs=`echo $3 | tr "\/" "\n"`
    declare -i l=0
    for d in $dirs
    do
       subpath=$d
       l+=1
    done

    if [[ $subpath == $1 ]]; then
         subpath=""
         path=$3
    else
         declare -i i=0
         for d in $dirs
         do
            if [[ $i == 0 ]]; then
                path=$d
            elif [[ $i -ne $l-1 ]]; then
                path=${path}"/"${d}
            fi
            i+=1
         done
    fi

    if [[ $4 == "" ]]; then
         commit=$2
    else
          versions=`echo $4 | tr "-" "\n"`
          for v in $versions
          do
             commit=$v
          done
    fi

    if [[ $subpath != "" ]]; then
         mcommit=$2
    else
         mcommit=$commit
    fi

    get_repo "$path" "$mcommit" "$subpath" "$commit"

}


repo_by_gomod() {
     local file=$1
     local repo=$2
     local branch=$3
     local subbranch=$4

     C=`grep replace $file | grep ../$repo | grep -v module | grep -v "${repo}-"`
     if [[ $C != "" ]]; then
           opath=`echo $C| awk '{print $2}'`
            get_path_subpath_commit "$repo" "$branch" "$opath"  ""
     else
           grepo=$repo
           if [[ $subbranch != "" ]]; then
               grepo="${repo}/${subbranch}"
           fi
           C=`grep $grepo $file | grep -v module | grep -v replace | grep -v indirect| grep -v "${repo}-"`
           if [[ $C == "" ]]; then
               return
           fi
           gpath=`echo $C| awk '{print $1}'`
           vers=`echo $C| awk '{print $2}'`
           get_path_subpath_commit "$repo" "$branch" "$gpath"  "$vers"
     fi
}


repo_setup() {
    repo_by_gomod go.mod query $cbranch
    repo_by_gomod go.mod query-ee $cbranch
    repo_by_gomod go.mod indexing $cbranch
    repo_by_gomod go.mod go-couchbase $cbranch
    repo_by_gomod go.mod gomemcached $cbranch
    repo_by_gomod go.mod cbauth $cbranch
    repo_by_gomod go.mod godbc $cbranch
    repo_by_gomod go.mod goutils $cbranch
    repo_by_gomod go.mod go_json $cbranch
    repo_by_gomod go.mod gometa $cbranch
    repo_by_gomod go.mod eventing-ee $cbranch
    repo_by_gomod go.mod n1fty $cbranch
    repo_by_gomod go.mod cbgt $cbranch
    repo_by_gomod go.mod cbft $cbranch
    repo_by_gomod ../n1fty/go.mod bleve $defbranch
    repo_by_gomod ../n1fty/go.mod bleve $defbranch "v2"
    repo_by_gomod ../cbft/go.mod zapx $defbranch "v11"
    repo_by_gomod ../cbft/go.mod zapx $defbranch "v12"
    repo_by_gomod ../cbft/go.mod zapx $defbranch "v13"
    repo_by_gomod ../cbft/go.mod zapx $defbranch "v14"
    repo_by_gomod ../cbft/go.mod zapx $defbranch "v15"
    repo_by_gomod go.mod gocbcore-transactions $defbranch
    repo_by_gomod go.mod gocbcore $defbranch "v10"
    repo_by_gomod ../cbgt/go.mod gocbcore $defbranch "v9"
}


DevStandaloneSetup() {
    # curl fix match manifest
       (cd ../../couchbasedeps/go-curl; git checkout 20161221-couchbase)
    # indexer generated files
       if [[ (! -f ../indexing/secondary/protobuf/query/query.pb.go) ]]; then
           if [[ -f ~/devbld/query.pb.go ]]; then
               cp ~/devbld/query.pb.go ../indexing/secondary/protobuf/query/query.pb.go
           fi
           if [[ -f ~/devbld/$cbranch.query.pb.go ]]; then
               cp ~/devbld/$cbranch.query.pb.go ../indexing/secondary/protobuf/query/query.pb.go
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
       if [[ ! -f ../eventing-ee/evaluator/impl/gen/parser/global_config_schema.go ]]; then
           (cd ../eventing-ee/evaluator/impl/gen/convertschema; go run generate.go  ../../parser/global_config_schema.json GlobalConfigSchema ../parser/global_config_schema.go)
       fi

       repo_setup
       (cd $GOPATH/src/golang.org/x/net; git checkout `go version |  awk -F'[. ]' '{print "release-branch." $3 "." $4}'`)
}

# turn off go module for non repo sync build or standalone build
if [[ ( ! -d ../../../../../cbft && "$GOPATH" != "") || ( $sflag == 1) ]]; then
     export GO111MODULE=off
     export CGO_CFLAGS="-I$GOPATH/src/github.com/couchbase/eventing-ee/evaluator/worker/include $CGO_FLAGS"
     export CGO_LDFLAGS="-L$GOPATH/lib $CGO_LDFLAGS"
     export LD_LIBRARY_PATH=$GOPATH/lib:${LD_LIBRARY_PATH}
     echo go get $* $uflag -d -v ./...
     go get $* $uflag -d -v ./...
     if [[ $sflag == 1 ]]; then
         DevStandaloneSetup
         # go get again because repro branches might changed
         go get $* $uflag -d -v ./...
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
