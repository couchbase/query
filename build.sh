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
# Add -fix to run go fix (applies modernizing rewrites to the source)
#   All analyzers run by default. Set GOFIXFLAGS to select some, e.g.
#     GOFIXFLAGS="-any -rangeint" ./build.sh -fix -tags enterprise    (run only these)
#     GOFIXFLAGS="-inline=false" ./build.sh -fix -tags enterprise     (run all but these)
#     GOFIXFLAGS="-diff" ./build.sh -fix -tags enterprise             (print diff, don't rewrite)
#   Analyzers ('go tool fix help' for the current list):
#     any              replace interface{} with any
#     buildtag         check //go:build and // +build directives
#     fmtappendf       replace []byte(fmt.Sprintf) with fmt.Appendf
#     forvar           remove redundant re-declaration of loop variables
#     hostport         check format of addresses passed to net.Dial
#     inline           apply fixes based on 'go:fix inline' comment directives
#     mapsloop         replace explicit loops over maps with calls to maps package
#     minmax           replace if/else statements with calls to min or max
#     newexpr          simplify code by using go1.26's new(expr)
#     omitzero         suggest replacing omitempty with omitzero for struct fields
#     plusbuild        remove obsolete //+build comments
#     rangeint         replace 3-clause for loops with for-range over integers
#     reflecttypefor   replace reflect.TypeOf(x) with TypeFor[T]()
#     slicescontains   replace loops with slices.Contains or slices.ContainsFunc
#     slicessort       replace sort.Slice with slices.Sort for basic types
#     stditerators     use iterators instead of Len/At-style APIs
#     stringsbuilder   replace += with strings.Builder
#     stringscut       replace strings.Index etc. with strings.Cut
#     stringscutprefix replace HasPrefix/TrimPrefix with CutPrefix
#     stringsseq       replace ranging over Split/Fields with SplitSeq/FieldsSeq
#     testingcontext   replace context.WithCancel with t.Context in tests
#     waitgroup        replace wg.Add(1)/go/wg.Done() with wg.Go
# Add -cover to build with Go code coverage (go_coverage build tag + go build -cover):
#   1. Build: './build.sh -cover -tags enterprise'
#      needs github.com/couchbase/gocoverage at ../../../../../gocoverage (-s clones and links it)
#   2. Run: export GOCOVERDIR=/tmp/qcov before starting cbq-engine (or cluster_run, so the
#      cbq-engine it spawns inherits it). Counters are flushed every 2 minutes and on
#      SIGTERM/SIGINT; anything since the last flush is lost on SIGKILL.
#   3. Report:
#        go tool covdata percent -i=$GOCOVERDIR [-pkg=github.com/couchbase/query/planner]
#        go tool covdata textfmt -i=$GOCOVERDIR -o /tmp/qcov.out
#        grep -vE '/yaccpar:|\.y:' /tmp/qcov.out > /tmp/qcov1.out   # drop goyacc line directives
#        go tool cover -func=/tmp/qcov1.out
#        go tool cover -html=/tmp/qcov1.out -o /tmp/qcov.html
#   4. Rebuild without -cover ('./build.sh -tags enterprise') to go back to normal binaries
# Add -vuln to run govulncheck on cbq-engine and the cbq shell (requires govulncheck in PATH)
#   go install golang.org/x/vuln/cmd/govulncheck@latest
# Add -s to fix standalone build issues. Keep indexer generated files in ~/devbld
# Note standalone build requires libraries from installed server, make sure installed server is
# compatible with source that is being built
#   -s also fast-forwards query's branch and brings the sibling repos that go.mod replaces to the
#   revisions github.com/couchbase/manifest names for query's branch (see sync_siblings). The
#   manifest is cached in $CB_MANIFEST_DIR (default ~/.cache/couchbase-manifest), which -s
#   resets to origin's master on every run: don't point it at a checkout you work in.

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
fixflag=0
vflag=0
vulnflag=0
coverflag=0
tags=
while [ $# -gt 0 ]; do
  case $1 in
    -tags)
      shift
      [[ "$1" == "enterprise" ]] && enterprise=1
      tags="$1,udf_query"
      ;;
    -u) uflag=-u ;;
    -s) sflag=1 ;;
    -S) sflag=2 ;;
    -nofmt) fflag=0 ;;
    -fix) fixflag=1 ;;
    -vet) vflag=1 ;;
    -vuln) vulnflag=1 ;;
    -cover) coverflag=1 ;;
    *) args="$args $1" ;;
  esac
  shift
done

if [[ $coverflag != 0 ]]; then
  tags="${tags:+$tags,}go_coverage"
  # via GOFLAGS so that go vet (which rejects -cover) ignores it
  export GOFLAGS="$GOFLAGS -cover"
fi
[[ -n $tags ]] && args="$args -tags $tags"

set -- $args

GIT=`which git`

cbranch=`$GIT rev-parse --abbrev-ref HEAD`
# prefer the tracked upstream; otherwise only follow first parents so that an unpushed
# forward merge doesn't pick up the merged-in branch (e.g. origin/morpheus on totoro)
rbranch=`$GIT rev-parse --abbrev-ref --symbolic-full-name @{u} 2>/dev/null`
rbranch_label="upstream"
if [[ -z $rbranch ]]
then
  rbranch=`$GIT log -n 25 --first-parent --pretty=format:"%D"|\
    awk '/->/&&NF>=4{p=$4;exit}!/->/&&NF>0{p=$1;exit}END{if (length(p)>0) { gsub(",","",p); print p} }'`
  rbranch_label="based on"
fi
defbranch="master"
defbranch2="main"

# Source shared functions
source "$(dirname "$0")/build_util.sh"

# ff_pull: fast-forward the current branch from its upstream. --no-rebase --ff-only neither
# rebases nor creates a merge commit, whatever pull.rebase/pull.ff say (before git 2.34,
# --ff-only alone still rebased under pull.rebase=true). Prints one line naming ${D} when the
# branch moved, git's error when it couldn't, and nothing when it was current or has no
# upstream to pull.
function ff_pull {
  local before after res
  $GIT rev-parse --abbrev-ref @{u} >/dev/null 2>&1 || return
  before=$($GIT rev-parse -q --verify HEAD)
  if ! res=$($GIT -c advice.diverging=false pull -q --no-rebase --ff-only 2>&1)
  then
    echo -e "${D} -> git pull --ff-only:\n${res}\n"
    return
  fi
  after=$($GIT rev-parse -q --verify HEAD)
  if [[ $before != "$after" ]]
  then
    echo "${D} -> $($GIT symbolic-ref -q --short HEAD) ${before:0:7}..${after:0:7}"
  fi
}

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
    if [[ $branch == $current ]]
    then
      ff_pull
      return
    fi
    if [[ $branch == $commit ]]
    then
      return
    fi
    cmd1="$GIT checkout $branch"
    res=`$cmd1 2>&1`
    if [[ $res =~ "did not match any file" ]]
    then
      # try refreshing the repo (a fetch: a pull would also merge into the current branch)
      ($GIT fetch -q 2>/dev/null 1>/dev/null)  # no need to report status
      res=`$cmd1 2>&1`
    fi
    if [[ $res =~ "Switched to branch" || $res =~ "Switched to a new branch" ]]
    then
      # a checkout doesn't fetch: bring the branch up to date with its upstream
      ff_pull
      return
    fi
    if [[ ! $res =~ "is now at" ]]
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
    # quoted, so that an empty $subpath stays an argument and failures are reported. The
    # branches to try are $@: $mcommit is either the first of them or a go.mod version's
    # hash, which the old unquoted call took as report_errors and never tried.
    checkout_if_necessary "$subpath" $@
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


# Sibling repos for -s. The official build takes each repo's revision from
# github.com/couchbase/manifest. manifest_plan picks the manifest for query's branch and
# reads the branch or commit it names for each github.com/couchbase module that go.mod
# replaces with a sibling directory, plus sigar, whose headers the build uses. Each sibling
# is then brought to that revision:
#   branch -> fetch, then fast-forward the local branch (created to track origin if missing)
#   commit -> fetch if missing, then check it out detached
# A sibling is left as it is, with a message naming it, when it is on a branch that no
# manifest uses (a developer's own branch), when it has local changes and would have to
# switch, when its local branch has commits that are not on origin, or when it is detached
# at a commit that origin doesn't have (a Gerrit change being tested). A sibling that is
# already current prints nothing. A module that go.mod replaces but doesn't require isn't
# read by the build, so it is updated if present but never cloned.
MANIFEST_DIR=${CB_MANIFEST_DIR:-${XDG_CACHE_HOME:-$HOME/.cache}/couchbase-manifest}
MANIFEST_URL=https://github.com/couchbase/manifest.git
FETCH_JOBS=8

# clone or update the cached manifest checkout; fails only when there is no usable copy.
# $MANIFEST_DIR is a cache only: each update detaches it at origin's master.
function manifest_update {
  local res
  if [[ -d $MANIFEST_DIR/.git ]]
  then
    res=$($GIT -C "$MANIFEST_DIR" fetch -q --depth 1 origin master 2>&1 &&
          $GIT -C "$MANIFEST_DIR" checkout -q --detach FETCH_HEAD 2>&1)
    if [[ $? -ne 0 ]]
    then
      echo -e "manifest -> could not update ${MANIFEST_DIR}, using the cached copy:\n${res}\n"
    fi
  else
    res=$(mkdir -p "$(dirname "$MANIFEST_DIR")" 2>&1 &&
          $GIT clone -q --depth 1 $MANIFEST_URL "$MANIFEST_DIR" 2>&1)
    if [[ $? -ne 0 ]]
    then
      echo -e "manifest -> could not clone ${MANIFEST_URL} into ${MANIFEST_DIR}:\n${res}\n"
      return 1
    fi
  fi
  [[ -f $MANIFEST_DIR/branch-master.xml ]]
}

# fetch_rev <revision>: in the current repo, fetch a branch into origin/<branch>, or fetch
# a commit that isn't present yet
function fetch_rev {
  if [[ $1 =~ ^[0-9a-f]{40}$ ]]
  then
    $GIT cat-file -e "$1^{commit}" 2>/dev/null || $GIT fetch -q origin "$1"
  else
    $GIT fetch -q origin "+refs/heads/$1:refs/remotes/origin/$1"
  fi
}

# whether the current repo has changes to tracked files (a switch would carry them along)
function is_dirty {
  [[ -n $($GIT status --porcelain --untracked-files=no) ]]
}

# sync_repo <name> <revision>: in the current repo, move to the (already fetched) manifest
# revision. Prints a line only when it changes something or leaves the repo as it is.
# $release and $pinned are the branch names and the commits the manifests use, set by
# manifest_plan.
function sync_repo {
  local name=$1 rev=$2
  local current head at target branch res rc
  current=$($GIT symbolic-ref -q --short HEAD)
  head=$($GIT rev-parse -q --verify HEAD)
  at=${current:-detached HEAD}

  if [[ -n $current && $release != *" $current "* ]]
  then
    echo "${name} -> on ${current}, which no manifest uses: left as is"
    return
  fi
  # detached at a commit that no origin branch contains and no manifest pins (a pinned
  # commit fetched by its SHA isn't under refs/remotes): the developer's own commit
  if [[ -z $current && $pinned != *" $head "* &&
        -z $($GIT for-each-ref --count=1 --contains HEAD refs/remotes/origin) ]]
  then
    echo "${name} -> detached at ${head:0:12}, which is not on origin: left as is"
    return
  fi

  if [[ $rev =~ ^[0-9a-f]{40}$ ]]
  then
    [[ $head == "$rev" ]] && return
    if is_dirty
    then
      echo "${name} -> has local changes: left on ${at} (manifest: ${rev:0:12})"
    elif res=$($GIT checkout -q --detach "$rev" 2>&1)
    then
      echo "${name} -> detached at ${rev:0:12}"
    else
      echo -e "${name} -> git checkout --detach ${rev} failed:\n${res}\n"
    fi
    return
  fi

  target=$($GIT rev-parse -q --verify "refs/remotes/origin/$rev")
  if [[ $current == "$rev" ]]
  then
    [[ $head == "$target" ]] && return
    if ! $GIT merge-base --is-ancestor HEAD "$target"
    then
      echo "${name} -> ${rev} has commits that are not on origin/${rev}: left as is" \
           "(reset it to origin/${rev} if Gerrit has merged them)"
    elif res=$($GIT merge -q --ff-only "$target" 2>&1)
    then
      echo "${name} -> ${rev} ${head:0:7}..${target:0:7}"
    else
      echo -e "${name} -> git merge --ff-only origin/${rev} failed:\n${res}\n"
    fi
    return
  fi

  if is_dirty
  then
    echo "${name} -> has local changes: left on ${at} (manifest: ${rev})"
    return
  fi
  branch=$($GIT rev-parse -q --verify "refs/heads/$rev")
  if [[ -z $branch ]]
  then
    res=$($GIT checkout -q -b "$rev" --track "origin/$rev" 2>&1)
    rc=$?
  elif $GIT merge-base --is-ancestor "$branch" "$target"
  then
    res=$($GIT checkout -q "$rev" -- 2>&1 && $GIT merge -q --ff-only "$target" 2>&1)
    rc=$?
  else
    echo "${name} -> local ${rev} has commits that are not on origin/${rev}: left on ${at}" \
         "(reset it to origin/${rev} if Gerrit has merged them)"
    return
  fi
  if [[ $rc -eq 0 ]]
  then
    echo "${name} -> switched to ${rev} at ${target:0:7}"
  else
    echo -e "${name} -> git checkout ${rev} failed:\n${res}\n"
  fi
}

# The manifests are read with grep/sed, not an XML parser. That relies on each <project>
# element being on one line and ignores <include>, <extend-project> and <remove-project>,
# which the couchbase-server manifests for query's branches don't use.

# manifest_files <branch>: the manifests for a query branch, most specific first
#   master     -> branch-master.xml
#   <codename> -> couchbase-server/<codename>.xml
#   <x.y.z>    -> the point manifest couchbase-server/<codename>/<x.y.z>.xml, then
#                 couchbase-server/<codename>.xml for the repos it doesn't list. Before the
#                 point manifest exists: the codename manifest whose VERSION has the same x.y.
function manifest_files {
  local b=$1 point codename xy
  case $b in
    ""|HEAD|.*|*/*) return ;;
    master) echo branch-master.xml; return ;;
  esac
  if [[ -f $MANIFEST_DIR/couchbase-server/$b.xml ]]
  then
    echo couchbase-server/$b.xml
    return
  fi
  [[ $b =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || return
  point=$(cd "$MANIFEST_DIR" && ls couchbase-server/*/$b.xml 2>/dev/null | head -1)
  if [[ -n $point ]]
  then
    echo $point
    codename=${point%/*}.xml
    [[ -f $MANIFEST_DIR/$codename ]] && echo $codename
    return
  fi
  xy=${b%.*}
  for codename in $(cd "$MANIFEST_DIR" &&
                    grep -lE "name=\"VERSION\" value=\"${xy//./\\.}\.[0-9]+\"" couchbase-server/*.xml)
  do
    if grep -qE '<project [^>]*name="query"' "$MANIFEST_DIR/$codename"
    then
      echo $codename
      return
    fi
  done
}

# manifest_rev <manifest> <project>: the project's revision attribute, else the manifest's
# default revision; nothing when the manifest doesn't list the project
function manifest_rev {
  local re='revision="([^"]+)"' line
  line=$(grep -E "<project [^>]*name=\"$2\"" "$1" | head -1)
  [[ -n $line ]] || return
  [[ $line =~ $re ]] || line=$(grep -E '<default ' "$1" | head -1)
  [[ $line =~ $re ]] && echo "${BASH_REMATCH[1]}"
}

# manifest_plan <branch>...: for the first branch that has a manifest, set the caller's
#   files    the manifests used, most specific first
#   names    the github.com/couchbase modules that go.mod replaces with a sibling directory,
#            plus sigar
#   revs     the revision of each, from the first manifest that lists it, else "-"
#   reqs     1 if go.mod requires the module (so the build reads it) or it is sigar, else 0
#   n        the number of names
#   release  every branch name the manifests use for these repos, space separated; -s may
#            move a sibling off these, but not off any other local branch
#   pinned   every commit the manifests use for these repos, space separated; -s may move
#            a detached sibling off these even when no origin branch it has fetched has them
# Fails when no branch has a manifest.
function manifest_plan {
  local b f name rev pattern revisions
  for b in "$@"
  do
    files=$(manifest_files "$b")
    [[ -n $files ]] && break
  done
  [[ -n $files ]] || return 1

  n=0
  for name in $( (grep -E '^replace[[:space:]]+github.com/couchbase/[^[:space:]/]+[[:space:]]+=>[[:space:]]+\.\./' go.mod |
                  awk '{print $2}' | sed 's,.*/,,'; echo sigar) | sort -u)
  do
    rev=
    for f in $files
    do
      rev=$(manifest_rev "$MANIFEST_DIR/$f" $name)
      [[ -n $rev ]] && break
    done
    names[$n]=$name
    revs[$n]=${rev:--}
    if [[ $name == sigar ]] || grep -qE "^(require)?[[:space:]]+github.com/couchbase/$name[[:space:]]+v" go.mod
    then
      reqs[$n]=1
    else
      reqs[$n]=0
    fi
    n=$((n+1))
  done

  pattern=$(echo ${names[*]} | tr ' ' '|')
  revisions=$(cd "$MANIFEST_DIR" &&
              cat branch-master.xml couchbase-server/*.xml couchbase-server/*/*.xml 2>/dev/null |
              grep -E "<default |<project [^>]*name=\"($pattern)\"" | grep -oE 'revision="[^"]+"' |
              sed 's/^revision="//; s/"$//' | sort -u)
  release=" $(echo "$revisions" | grep -vE '^[0-9a-f]{40}$' | tr '\n' ' ')"
  pinned=" $(echo "$revisions" | grep -E '^[0-9a-f]{40}$' | tr '\n' ' ')"
}

# bring the sibling repos in line with the manifest for query's branch; returns 1, having
# changed nothing, when no manifest applies. Always prints one line naming query's branch
# and where the sibling revisions come from.
function sync_siblings {
  local files release pinned names=() reqs=() revs=() n=0
  local tmp mux i j name rev dir res running=0 njobs=$FETCH_JOBS errs=() failed=()
  local q="query -> ${cbranch}${rbranch:+ (${rbranch_label} ${rbranch})}"

  if ! manifest_update
  then
    echo "${q}: no copy of the manifest, guessing sibling branches"
    return 1
  fi
  if ! manifest_plan "$cbranch" "${rbranch#*/}"
  then
    echo "${q}: no manifest for this branch, guessing sibling branches"
    return 1
  fi
  echo "${q}: siblings from $(echo $files | sed 's/ /, then /g')"
  tmp=$(mktemp -d "${TMPDIR:-/tmp}/build_sh.XXXXXX") || return 1

  # Fetch in parallel, over one ssh connection to GitHub: GitHub resets new ssh connections
  # beyond about 20 a minute ("kex_exchange_identification: ... Connection reset by peer"),
  # and there is a fetch per sibling. With ssh's connection sharing each fetch is a session
  # on a master connection, which also asks for a key's passphrase only once ($FETCH_JOBS
  # sessions at a time stay under sshd's usual limit of 10). The socket goes in a private
  # directory with a short path, as a socket path is limited to about 100 bytes. It is named
  # by the host as typed (%n), so a host alias for github.com, which may pick another key,
  # doesn't share it. Not done when the developer sets the ssh command (which may pick the
  # key), nor possible when the master can't connect: then fetch one at a time, as a burst
  # of new connections is what GitHub resets, and each ssh could ask for the passphrase.
  if [[ -n $GIT_SSH_COMMAND$GIT_SSH$($GIT config --get core.sshCommand) ]]
  then
    res="GIT_SSH_COMMAND, GIT_SSH or core.sshCommand is set"
  elif mux=$(mktemp -d /tmp/build_sh.XXXXXX) &&
       ssh -o ControlMaster=yes -o ControlPath="$mux/%r@%n:%p" -o ControlPersist=60 -fN \
           git@github.com </dev/null >/dev/null 2>"$tmp/ssh.err"
  then
    export GIT_SSH_COMMAND="ssh -o ControlMaster=no -o ControlPath=$mux/%r@%n:%p"
    res=
  else
    res=$(tr -d '\r' <"$tmp/ssh.err" 2>/dev/null | grep -m1 .)
    res=${res:-could not start ssh}
  fi
  if [[ -n $res ]]
  then
    echo "github.com -> no shared ssh connection (${res}): fetching one at a time"
    njobs=1
  fi
  for ((i = 0; i < n; i++))
  do
    name=${names[$i]}
    dir=$GOPATH/src/github.com/couchbase/$name
    [[ ${revs[$i]} == "-" ]] && continue
    if [[ ! -d $dir ]]
    then
      [[ ${reqs[$i]} == 1 ]] || continue
      if ! res=$($GIT clone -q git@github.com:couchbase/$name.git "$dir" 2>&1)
      then
        echo -e "${name} -> git clone failed:\n${res}\n"
        continue
      fi
      echo "${name} -> cloned"
    fi
    (cd "$dir" && fetch_rev "${revs[$i]}" >"$tmp/$name.out" 2>&1; echo $? >"$tmp/$name.rc") &
    if (( ++running % njobs == 0 ))
    then
      wait
    fi
  done
  wait

  # retry each failed fetch once, one at a time, as a network blip passes. Not when every
  # fetch failed: offline, a retry only takes longer.
  if grep -qx 0 "$tmp"/*.rc 2>/dev/null && grep -qvx 0 "$tmp"/*.rc
  then
    for ((i = 0; i < n; i++))
    do
      name=${names[$i]}
      [[ -f $tmp/$name.rc && $(cat "$tmp/$name.rc") != 0 ]] || continue
      (cd "$GOPATH/src/github.com/couchbase/$name" &&
       fetch_rev "${revs[$i]}" >"$tmp/$name.out" 2>&1; echo $? >"$tmp/$name.rc")
    done
  fi

  for ((i = 0; i < n; i++))
  do
    name=${names[$i]}
    rev=${revs[$i]}
    dir=$GOPATH/src/github.com/couchbase/$name
    if [[ $rev == "-" ]]
    then
      # no manifest lists it: guess its branch as before, ending with the repo's default
      # branch, master or main
      if [[ -d $dir || ${reqs[$i]} == 1 ]]
      then
        repo_by_gomod go.mod $name "" $cbranch $rbranch $defbranch $defbranch2
      fi
      continue
    fi
    [[ -f $tmp/$name.rc ]] || continue
    if [[ $(cat "$tmp/$name.rc") != 0 ]]
    then
      # group the failed fetches by their error, printed once at the end: offline, every
      # fetch fails the same way
      res=$(cat "$tmp/$name.out")
      for ((j = 0; j < ${#errs[@]}; j++))
      do
        [[ ${errs[$j]} == "$res" ]] && break
      done
      errs[$j]=$res
      failed[$j]="${failed[$j]}${name} "
      continue
    fi
    res=$(cd "$dir" && sync_repo "$name" "$rev")
    if [[ -n $res ]]
    then
      echo -e "$res"
    fi
  done
  for ((j = 0; j < ${#errs[@]}; j++))
  do
    printf '%s-> git fetch failed: left as is\n%s\n\n' "${failed[$j]}" "${errs[$j]}"
  done
  if [[ -n $mux ]]
  then
    [[ $GIT_SSH_COMMAND == *"$mux"* ]] && unset GIT_SSH_COMMAND
    ssh -o ControlPath="$mux/%r@%n:%p" -O exit git@github.com 2>/dev/null
    rm -rf "$mux"
  fi
  rm -rf "$tmp"
  return 0
}

function repo_setup {
    # query itself: fast-forward its branch
    repo_by_gomod go.mod query "" $cbranch $rbranch $defbranch
    sync_siblings || repo_setup_legacy
}

# the sibling sync used when no manifest applies: try query's branch, its upstream, then
# the repo's default branch
function repo_setup_legacy {
    repo_by_gomod go.mod query-ee "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod indexing "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod bhive "" $cbranch $rbranch $defbranch2
    repo_by_gomod go.mod go-couchbase "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod gocbcrypto "" $cbranch $rbranch $defbranch2
    repo_by_gomod go.mod gocoverage "" $cbranch $rbranch $defbranch2
    repo_by_gomod go.mod logstats "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod gomemcached "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod cbauth "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod godbc "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod goutils "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod go_json "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod gometa "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod eventing-ee "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod n1fty "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod regulator "" $cbranch $rbranch $defbranch2
    repo_by_gomod go.mod sigar "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod cbgt "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod cbft "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod hebrew "" $cbranch $rbranch $defbranch
    repo_by_gomod go.mod cbftx "" $cbranch $rbranch $defbranch
}

function DevStandaloneSetup {

    repo_setup

    # go.mod replaces these with ../../../../../<repo>: link $GOPATH/../<repo> to the sibling,
    # when there is one (-s doesn't clone cbftx, which the build doesn't read). -n replaces
    # an existing link instead of following it and creating <repo>/<repo> inside the sibling,
    # as runs before -n did: such a link, recognised by its target, is removed.
    ( dir=`echo $cwd1 |awk -F/ '{print  $(NF-4) "/" $(NF-3) "/" $(NF-2) "/" $(NF-1)}'`;
      cd $GOPATH/..;
      for r in cbgt cbft cbftx hebrew gocoverage
      do
        if [[ -d $dir/$r ]]
        then
          ln -s -f -n $dir/$r $r
          if [[ -L $dir/$r/$r && $(readlink "$dir/$r/$r") == "$dir/$r" ]]
          then
            rm -f "$dir/$r/$r"
          fi
        fi
      done
      cd $cwd1)

    # OpenSSL setup
    OPENSSL_BUILD_DIR="$GOPATH/src/couchbasedeps/openssl"
    if [[ -d ~/devbld ]]; then
        OPENSSL_BUILD_DIR=~/devbld
    fi

    if [[ -d ~/devbld/$OPENSSL_VERSION ]]; then
        if [[ ! -L $GOPATH/src/couchbasedeps/openssl/$OPENSSL_VERSION ]]; then
            mkdir -p "$GOPATH/src/couchbasedeps/openssl"
            ln -s ~/devbld/$OPENSSL_VERSION "$GOPATH/src/couchbasedeps/openssl/$OPENSSL_VERSION"
        fi
    elif [[ ! -d "$GOPATH/src/couchbasedeps/openssl/$OPENSSL_VERSION" ]]; then
        echo -e "OpenSSL $OPENSSL_VERSION not found, downloading and building to $OPENSSL_BUILD_DIR..."
        mkdir -p "$OPENSSL_BUILD_DIR"
        cd "$OPENSSL_BUILD_DIR"
        curl -L https://github.com/openssl/openssl/releases/download/$OPENSSL_VERSION/$OPENSSL_VERSION.tar.gz | tar -xz
        cd $OPENSSL_VERSION
        ./Config --prefix=$PWD/openssl-local --openssldir=$PWD/openssl-local/ssl no-shared
        make build_libs
        cd $cwd1

        if [[ $OPENSSL_BUILD_DIR == ~/devbld ]]; then
            mkdir -p "$GOPATH/src/couchbasedeps/openssl"
            ln -s $OPENSSL_BUILD_DIR/$OPENSSL_VERSION "$GOPATH/src/couchbasedeps/openssl/$OPENSSL_VERSION"
        fi
    fi

    # indexer generated files
    if [[ -f ~/devbld/protoc-gen-go ]]
    then
            ln -sf ~/devbld/protoc-gen-go $GOPATH/bin
	    (cd ../indexing/secondary/protobuf/query; protoc -I. --plugin=protoc-gen-go=$GOPATH/bin/protoc-gen-go query.proto --go_out=. --go_opt=paths=source_relative)
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

OPENSSL_VERSION=$(get_openssl_version)

if [[ ( ( ! -d ../../../../../cbft || -h ../../../../../cbft ) && "$GOPATH" != "") || ( $sflag != 0) ]]; then
    if [[ $sflag == 1 ]]; then
        DevStandaloneSetup
    fi
    export CGO_CFLAGS="-I$GOPATH/src/github.com/couchbase/eventing-ee/evaluator/worker/include -I$GOPATH/src/github.com/couchbase/sigar/include -I$GOPATH/src/couchbasedeps/openssl/$OPENSSL_VERSION/include $CGO_FLAGS"
    export CGO_LDFLAGS="-L$GOPATH/lib $CGO_LDFLAGS"
    export LD_LIBRARY_PATH=$GOPATH/lib:${LD_LIBRARY_PATH}
fi

cd $cwd1
echo cd parser/n1ql
cd parser/n1ql
./build.sh $*
cd ../..

if [[ ($fixflag != 0) ]]
then
  echo go fix $* $GOFIXFLAGS ./...
  go fix $* $GOFIXFLAGS ./...
  if [ $? -ne 0 ]
  then
    exit 1
  fi
  if [[ $enterprise == 1 ]]; then
    (echo go fix $* $GOFIXFLAGS ../query-ee/...; cd ../query-ee; go fix $* $GOFIXFLAGS ./...)
    if [ $? -ne 0 ]
    then
      exit 1
    fi
  fi
fi

if [[ ($fflag != 0) ]]
then
  echo go fmt ./...
  go fmt ./...
  if [[ $enterprise == 1 ]]; then
    (echo go fmt ../query-ee/...; cd ../query-ee; go fmt ./...)
  fi
fi

if [[ ($vflag != 0) ]]
then
  echo go vet $* ./...
  go vet $* ./...
  if [ $? -ne 0 ]
  then
    exit 1
  fi
  if [[ $enterprise == 1 ]]; then
    (echo go vet $* ../query-ee/...; cd ../query-ee; go vet $* ./...)
    if [ $? -ne 0 ]
    then
      exit 1
    fi
  fi
fi

if [[ ($vulnflag != 0) ]]
then
  echo govulncheck $* ./server/cbq-engine/... ./shell/cbq/...
  govulncheck $* ./server/cbq-engine/... ./shell/cbq/...
  if [ $? -ne 0 ]
  then
    exit 1
  fi
fi

echo cd server/cbq-engine
cd server/cbq-engine
./build.sh $*
if [ $? -ne 0 ]
then
  exit 1
fi
if [[ "Darwin" = `uname` ]]; then
    lpath=`otool -l cbq-engine |grep LC_RPATH -A 2 | grep "/Applications/Couchbase Server.app/Contents/Resources/couchbase-core/lib"`
    if [[ -z "$lpath" ]]; then
       install_name_tool -add_rpath /Applications/Couchbase\ Server.app/Contents/Resources/couchbase-core/lib cbq-engine
    fi
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
