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


function get_repo {
     local path=$1
     local mcommit=$2
     local subpath=$3
     local scommit=$4
     local rbranch=$5

     #echo "$path" "$mcommit" "$subpath" "$scommit" "$rbranch"

     cd $GOPATH/src/$path
     abranch=`$GIT branch | awk '{print $2}'`
     if [[ $abranch != $mcommit ]]; then
         checkout_if_necessary $mcommit $rbranch
     fi
     if [[ $subpath != "" ]]
     then
         if [[ ! -d $subpath ]]
         then
            url="https://"$path
            $GIT clone -b $mcommit $url $subpath
         fi
         (cd $subpath; checkout_if_necessary $scommit $rbranch)
     fi
     cd - >> /dev/null
}

function get_path_subpath_commit {
    dirs=`echo $4 | tr "\/" "\n"`
    declare -i l=0
    for d in $dirs
    do
       subpath=$d
       l+=1
    done

    if [[ $subpath == $1 ]]; then
         subpath=""
         path=$4
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

    if [[ $5 == "" ]]; then
         commit=$2
    else
          versions=`echo $5 | tr "-" "\n"`
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

    get_repo "$path" "$mcommit" "$subpath" "$commit" "$3"

}


function repo_by_gomod {
     local file=$1
     local repo=$2
     local branch=$3
     local rootbranch=$4
     local subbranch=$5

     C=`grep replace $file | grep "../${repo}" | grep -v "module" | grep -v "${repo}-"`
     if [[ $C != "" ]]; then
           opath=`echo $C| awk '{print $2}'`
            get_path_subpath_commit "$repo" "$branch" "$rootbranch" "$opath"  ""
     else
           grepo=$repo
           if [[ $subbranch != "" ]]; then
               grepo="${repo}/${subbranch}"
           fi
           C=`grep "${grepo}" $file | grep -v module | grep -v replace | grep -v indirect| grep -v "${repo}-"`
           if [[ $C == "" ]]; then
               return
           fi
           gpath=`echo $C| awk '{print $1}'`
           vers=`echo $C| awk '{print $2}'`
           get_path_subpath_commit "$repo" "$branch" "$rootbranch" "$gpath"  "$vers"
     fi
}


function repo_setup {
    repo_by_gomod go.mod query $cbranch $rbranch
    repo_by_gomod go.mod query-ee $cbranch $rbranch
    repo_by_gomod go.mod indexing $cbranch $rbranch
    repo_by_gomod go.mod go-couchbase $cbranch $rbranch
    repo_by_gomod go.mod gomemcached $cbranch $rbranch
    repo_by_gomod go.mod cbauth $cbranch $rbranch
    repo_by_gomod go.mod godbc $cbranch $rbranch
    repo_by_gomod go.mod goutils $cbranch $rbranch
    repo_by_gomod go.mod go_json $cbranch $rbranch
    repo_by_gomod go.mod gometa $cbranch $rbranch
    repo_by_gomod go.mod eventing-ee $cbranch $rbranch
    repo_by_gomod go.mod n1fty $cbranch $rbranch
    repo_by_gomod go.mod cbgt $cbranch $rbranch
    repo_by_gomod go.mod cbft $cbranch $rbranch
    repo_by_gomod ../n1fty/go.mod bleve $defbranch $defbranch
    repo_by_gomod ../n1fty/go.mod bleve $defbranch $defbranch "v2"
    repo_by_gomod ../cbft/go.mod zapx $defbranch $defbranch "v11"
    repo_by_gomod ../cbft/go.mod zapx $defbranch $defbranch "v12"
    repo_by_gomod ../cbft/go.mod zapx $defbranch $defbranch "v13"
    repo_by_gomod ../cbft/go.mod zapx $defbranch $defbranch "v14"
    repo_by_gomod ../cbft/go.mod zapx $defbranch $defbranch "v15"
    repo_by_gomod go.mod gocbcore-transactions $defbranch $defbranch
    repo_by_gomod go.mod gocbcore $defbranch $defbranch "v10"
    repo_by_gomod ../cbgt/go.mod gocbcore $defbranch $defbranch "v9"
}

function checkout_if_necessary {
  local branch=$1
  local current=`$GIT rev-parse --abbrev-ref HEAD 2>/dev/null`
  local commit=`$GIT log -n 1 --pretty=format:"%h"`
  local rbranch=$2

  if [[ $branch == $current ]]
  then
    return
  elif [[ $branch == $commit ]]
  then
    return
  fi
  res=`$GIT checkout "${branch}000" 2>&1`
  if [[ $res =~ "did not match any file" ]]
  then
    ($GIT pull 2>/dev/null 1>/dev/null)  # no need to report status
    res=`$GIT checkout $branch 2>&1`
    if [[ $res =~ "did not match any file" ]]
    then
      if [[ $rbranch != $current ]]
      then
        $GIT checkout $rbranch
      fi
    elif [[ ! $res =~ "is now at" ]]
    then
      echo "$res"
    fi
  elif [[ ! $res =~ "is now at" ]]
  then
    echo "$res"
  fi
}

function DevStandaloneSetup {
    # curl fix match manifest
       (cd ../../couchbasedeps/go-curl; checkout_if_necessary 20161221-couchbase)
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
       (cd $GOPATH/src/golang.org/x/net; checkout_if_necessary `go version |  awk -F'[. ]' '{print "release-branch." $3 "." $4}'`)
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
