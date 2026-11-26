//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package http

import (
	"fmt"
	"strings"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	dictionary "github.com/couchbase/query/datastore/couchbase"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
)

// MB-75645: cbbackupmgr sends the include/exclude filters and the remap of a restore with full keyspace paths only
// on the global restore call (/api/v1/backup), which comes after the QUERY_METADATA restore; the QUERY_METADATA
// restore gets none of them. Saved plans are therefore restored in full, and keep using the keyspaces named in
// their statements: a saved plan can not be redirected to a mapped keyspace (re-preparing starts from the statement
// text). After the global restore, a warning is logged for each saved plan referencing a keyspace that the restore
// maps elsewhere or leaves out, so that the statement can be prepared again if needed. Saved plans are not changed.
// The global restore can not tell whether this restore included QUERY_METADATA (separate requests): the warnings
// cover all saved plans on the target, including those that were there before the restore.
//
// Parameter format, as sent by cbbackupmgr 8.5.0-1500 on the global restore call (captured in query.log, see
// MB-75645): include/exclude are comma separated bucket[.scope[.collection]] paths, e.g.
// include=QUERY_METADATA,b1._default._default; remap is comma separated from:to entries of such paths, e.g.
// remap=b1._default._default:b1.s2.c1 or remap=b1:b2. The bucket level restore call gets scope relative paths
// instead (see newFilter(), newRemapper()).

// at most this many saved plans are listed, the rest are counted
const _MAX_SAVED_PLAN_WARNINGS = 100

// a keyspace remap: bucket[.scope[.collection]] to bucket[.scope[.collection]]
type keyspaceRemap struct {
	from []string
	to   []string
}

// a comma separated list of bucket[.scope[.collection]] paths (a backslash escapes a dot)
func parseKeyspacePaths(p string) [][]string {
	if p == "" {
		return nil
	}
	var rv [][]string
	for _, entry := range strings.Split(p, ",") {
		path := parsePath(strings.TrimSpace(entry))
		if len(path) == 0 || len(path) > 3 || path[0] == "" {
			continue
		}
		rv = append(rv, path)
	}
	return rv
}

// a comma separated list of from:to entries, each a bucket[.scope[.collection]] path
func parseKeyspaceRemaps(p string) []keyspaceRemap {
	if p == "" {
		return nil
	}
	var rv []keyspaceRemap
	for _, entry := range strings.Split(p, ",") {
		fromTo := strings.Split(strings.TrimSpace(entry), ":")
		if len(fromTo) != 2 {
			continue
		}
		from := parsePath(fromTo[0])
		to := parsePath(fromTo[1])
		if len(from) == 0 || len(from) > 3 || from[0] == "" || len(to) == 0 || len(to) > 3 || to[0] == "" {
			continue
		}
		rv = append(rv, keyspaceRemap{from: from, to: to})
	}
	return rv
}

// the bucket, scope and collection of a saved plan's keyspace reference (namespace:bucket[.scope.collection]);
// a bucket stands for its default collection
func keyspaceRefParts(ref string) []string {
	parts := algebra.ParsePath(ref)
	switch len(parts) {
	case 2:
		return []string{parts[1], "_default", "_default"}
	case 4:
		return parts[1:]
	}
	return nil
}

// MB-75645: the path of a saved plan's keyspace reference for the include/exclude filter of a direct call to the
// QUERY_METADATA backup/restore (see filterEval()), matched down to the collection: namespace, bucket, scope,
// collection; a bucket stands for its default collection
func savedPlanFilterPath(ref string) []string {
	parts := algebra.ParsePath(ref)
	if len(parts) == 2 {
		return []string{parts[0], parts[1], "_default", "_default"}
	}
	return parts
}

func hasPathPrefix(parts, prefix []string) bool {
	if len(prefix) > len(parts) {
		return false
	}
	for i := range prefix {
		if parts[i] != prefix[i] {
			return false
		}
	}
	return true
}

// why a saved plan referencing the keyspace (bucket, scope, collection) may have to be prepared again, "" if not;
// the most specific remap applies (e.g. b.s over b), whatever the order of the entries
func savedPlanKeyspaceWarning(parts []string, remaps []keyspaceRemap, include, exclude [][]string) string {
	keyspace := strings.Join(parts, ".")
	var remap *keyspaceRemap
	for i := range remaps {
		if hasPathPrefix(parts, remaps[i].from) && (remap == nil || len(remaps[i].from) > len(remap.from)) {
			remap = &remaps[i]
		}
	}
	if remap != nil {
		target := strings.Join(append(append([]string{}, remap.to...), parts[len(remap.from):]...), ".")
		return fmt.Sprintf("mapped to %s by the remap of this restore; the plan still uses %s and has to be prepared "+
			"again to use %s", target, keyspace, target)
	}
	if len(include) > 0 {
		included := false
		for _, in := range include {
			if hasPathPrefix(parts, in) {
				included = true
				break
			}
		}
		if !included {
			return fmt.Sprintf("not included by the filter of this restore; the plan still uses %s, whose data "+
				"this restore does not restore", keyspace)
		}
	}
	for _, ex := range exclude {
		if hasPathPrefix(parts, ex) {
			return fmt.Sprintf("excluded by the filter of this restore; the plan still uses %s, whose data this "+
				"restore does not restore", keyspace)
		}
	}
	return ""
}

// the warning for a saved plan (name, keyspaceReferences), "" if none
func savedPlanWarning(name string, keyspaceRefs []interface{}, remaps []keyspaceRemap, include,
	exclude [][]string) string {

	for _, ref := range keyspaceRefs {
		s, ok := ref.(string)
		if !ok {
			continue
		}
		parts := keyspaceRefParts(s)
		if parts == nil {
			continue
		}
		if reason := savedPlanKeyspaceWarning(parts, remaps, include, exclude); reason != "" {
			return fmt.Sprintf("Plan Stability restore: saved plan %s references keyspace %s, %s", name,
				strings.Join(parts, "."), reason)
		}
	}
	return ""
}

// log a warning for each saved plan referencing a keyspace that the restore (global restore call) maps elsewhere
// or leaves out; nothing fails the restore
func warnSavedPlanKeyspaces(remapParam, includeParam, excludeParam string) {
	remaps := parseKeyspaceRemaps(remapParam)
	include := parseKeyspacePaths(includeParam)
	exclude := parseKeyspacePaths(excludeParam)
	if len(remaps) == 0 && len(include) == 0 && len(exclude) == 0 {
		return
	}

	warned := 0
	keys := make([]string, 1)
	res := make(map[string]value.AnnotatedValue, 1)
	err := datastore.ScanSystemCollection(dictionary.QUERY_METADATA_BUCKET, "ppn::", nil,
		func(key string, systemCollection datastore.Keyspace) errors.Error {
			keys[0] = key
			errs := systemCollection.Fetch(keys, res, datastore.NULL_QUERY_CONTEXT, nil, nil, false)
			if len(errs) > 0 {
				logging.Infof("Plan Stability restore: unable to read saved plan %s: %v", key, errs[0])
				return nil
			}
			av, ok := res[key]
			delete(res, key)
			if !ok {
				return nil
			}
			name := key
			if v, ok := av.Field("name"); ok {
				if s, ok := v.Actual().(string); ok && s != "" {
					name = s
				}
			}
			refsVal, ok := av.Field("keyspaceReferences")
			if !ok {
				return nil
			}
			refs, _ := refsVal.Actual().([]interface{})
			if msg := savedPlanWarning(name, refs, remaps, include, exclude); msg != "" {
				warned++
				if warned <= _MAX_SAVED_PLAN_WARNINGS {
					logging.Warnf("%s", msg)
				}
			}
			return nil
		}, nil)
	if err != nil {
		logging.Infof("Plan Stability restore: unable to check the saved plans against the keyspaces of this "+
			"restore: %v", err)
	}
	if warned > _MAX_SAVED_PLAN_WARNINGS {
		logging.Warnf("Plan Stability restore: %d more saved plans reference keyspaces that this restore maps "+
			"elsewhere or leaves out (not listed)", warned-_MAX_SAVED_PLAN_WARNINGS)
	}
}
