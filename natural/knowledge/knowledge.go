//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// Package knowledge implements storage for CREATE KNOWLEDGE <name> FOR <keyspace> AS <value>.
//
// Each keyspace (bucket.scope.collection) that has knowledge entries gets a single durable
// document in that bucket's system collection, holding all of that keyspace's entries as a
// name-to-value object (not an array): this makes a duplicate-name check a single map lookup
// rather than a full scan, and an update-in-place a single key assignment:
//
//	{
//	    "namespace": "default", "bucket": "b", "scope": "s", "collection": "c",
//	    "scopeUid": "...", "collectionUid": "...",
//	    "knowledge": { "icao_hint": "...", "named_param_hint": "..." }
//	}
//
// The document's storage key embeds the scope's and collection's current UIDs ahead of the
// human-readable scope.collection name ("know::<scopeUid>::<collectionUid>::<scope>.<collection>",
// mirroring aus's key scheme exactly) rather than being purely name-based, so a scope/collection
// dropped and later recreated under the same name gets a distinct key instead of colliding with
// (and appearing to extend) the old, now-orphaned one. namespace/bucket are omitted from the key
// (unlike the external composite key below) since the document already lives in that bucket's own
// system collection, so they're always implicit. scopeUid/collectionUid are also carried in the
// document body so Scan's cache-first pass can recompute a cached doc's exact storage key without
// an extra live UID lookup.
//
// Individual entries are addressed externally (e.g. for system:knowledge) via a purely name-based
// composite key of the form "<namespace>:<bucket>.<scope>.<collection>::<name>" - the UIDs are an
// internal storage-layer detail and never appear in user-facing syntax or output.
package knowledge

import (
	"fmt"
	"strings"
	"time"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

const _PREFIX = "know::"
const _FIELD = "knowledge"
const _BATCH_SIZE = 16

// writeKnowledgeEntries gives up after this many lost CAS races rather than retrying forever
// against sustained concurrent writers, backing off a little longer between each attempt.
const _MAX_WRITE_RETRIES = 10
const _WRITE_RETRY_BACKOFF = 5 * time.Millisecond

// MaxHintsPerKeyspace caps how many named entries a single keyspace's knowledge document may hold,
// and MaxHintValueSize caps the size (in bytes, UTF-8 encoded) of any one hint's value. Both exist
// to keep a keyspace's knowledge document well clear of the underlying KV document size limit, and
// to keep the node-local read cache's per-entry size (see cache.go's _CACHE_LIMIT) bounded and
// predictable.
const (
	MaxHintsPerKeyspace = 100
	MaxHintValueSize    = 10 * util.KiB
)

// _UUID_LENGTH bounds how long a name segment can be before capKeySegment replaces it with a
// hash of that length instead.
const _UUID_LENGTH = 36

// capKeySegment bounds a single key segment to at most a UUID's length, replacing it with a
// deterministic hash (so the same over-long name always maps to the same key) when it's longer -
// mirroring query-ee's CBO stats key generation (genKey, in dictionary/system_keyspace.go), which
// solves the identical problem: scope/collection names can be up to 251 bytes on their own,
// enough to push a document key past the 251-byte KV key limit once combined with the rest of the
// key. space salts the hash so the same over-long name used in two different contexts doesn't
// produce the same replacement.
func capKeySegment(space, input string) string {
	if len(input) <= _UUID_LENGTH {
		return input
	}
	key, err := util.UUIDV5(space, input)
	if err != nil {
		return input
	}
	return key
}

// getStorageKey embeds the scope's and collection's current UIDs ahead of the human-readable
// scope.collection name, mirroring aus's key format exactly (aus_setting::<scopeUid>::
// <collectionUid>::<scopeName>.<collectionName>). This is what lets a scope/collection that's
// dropped and later recreated under the same name get a distinct storage key rather than colliding
// with (and appearing to extend) the old, now-orphaned one - the UIDs change on recreation even
// though the names don't. Deliberately omits namespace/bucket (unlike path.FullName()): the
// document lives in that bucket's own system collection, so they're always implicit, and Couchbase
// scope/collection names (unlike bucket names) can never contain a ".", so this plain join is
// unambiguous to split back apart - no back-tick escaping needed the way FullName() requires. The
// scope/collection names themselves go through capKeySegment first, since either can be long
// enough on its own to breach the 251-byte KV key limit; every caller always rebuilds the key from
// the UIDs and path rather than parsing it back apart, so this substitution is transparent.
//
// With that cap in place, the key's total length has a fixed, provable ceiling regardless of how
// long the actual scope/collection names are: "know::" (6) + scopeUid (8, always "%08x"-padded) +
// "::" (2) + collectionUid (≤16, an unpadded hex uint64 - in practice ≤8, since collection IDs are
// 32-bit) + "::" (2) + capped scope segment (≤36) + "." (1) + capped collection segment (≤36) =
// 107 bytes worst case, comfortably under the 251-byte limit.
func getStorageKey(scopeUid, collectionUid string, path *algebra.Path) string {
	space := scopeUid + "::" + collectionUid
	return _PREFIX + scopeUid + "::" + collectionUid + "::" +
		capKeySegment(space, path.Scope()) + "." + capKeySegment(space, path.Keyspace())
}

// MaybeCappedSegment reports whether s could be a capKeySegment hash rather than a literal
// scope/collection name - i.e. it has the exact length and dash placement of a UUIDv5 string.
// Mirrors query-ee's CBO stats key handling of the identical ambiguity (isUUIDv5Format, in
// dictionary/migration.go): a real name that happens to be exactly 36 characters in this exact
// shape is possible but vanishingly unlikely. Exported for external reconciliation sweeps
// (datastore/couchbase.CleanupSystemCollection) that parse a know:: key's scope.collection segment
// and need to know whether to trust it, or resolve the real names from the document body instead
// (see ResolveStoredNames).
func MaybeCappedSegment(s string) bool {
	if len(s) != _UUID_LENGTH || strings.Count(s, "-") != 4 {
		return false
	}
	return s[8] == '-' && s[13] == '-' && s[18] == '-' && s[23] == '-'
}

// ResolveStoredNames returns the scope and collection names stored in the knowledge document at
// key, for external reconciliation sweeps that can't trust the key's own scope.collection segment
// once MaybeCappedSegment says it might be a hash rather than the literal name. found is false if
// the document doesn't exist (or is missing the fields, which shouldn't happen for a document this
// package wrote itself).
func ResolveStoredNames(b datastore.Keyspace, key string) (scope, collection string, found bool, err errors.Error) {
	av, exists, lerr := loadDoc(b, key)
	if lerr != nil {
		return "", "", false, lerr
	}
	if !exists {
		return "", "", false, nil
	}
	sv, ok := av.Field("scope")
	if !ok {
		return "", "", false, nil
	}
	cv, ok := av.Field("collection")
	if !ok {
		return "", "", false, nil
	}
	return sv.ToString(), cv.ToString(), true, nil
}

// resolveUids resolves path's scope and collection to their current live UIDs, for embedding in
// the storage key (see getStorageKey).
func resolveUids(path *algebra.Path) (scopeUid, collectionUid string, err errors.Error) {
	scope, err := datastore.GetScope(path.Namespace(), path.Bucket(), path.Scope())
	if err != nil {
		return "", "", err
	}
	coll, err := scope.KeyspaceByName(path.Keyspace())
	if err != nil {
		return "", "", err
	}
	return scope.Uid(), coll.Uid(), nil
}

// rejectSystemScope rejects path if it targets the bucket-local system scope (_system, which backs
// every bucket's own system collection) - that's a reserved, internal location, not a regular user
// keyspace. Mirrors sequences.CreateSequence's check (b.ScopeId() == path.Scope()): b is the
// caller's already-fetched system collection, whose own ScopeId() is always "_system", so this is
// just comparing scope names, no separate lookup of the target collection needed. The earlier
// system-namespace check in validateAndResolvePath only catches the separate, global system:
// namespace, not this per-bucket one.
func rejectSystemScope(b datastore.Keyspace, path *algebra.Path) errors.Error {
	if b.ScopeId() == path.Scope() {
		return errors.NewDatastoreInvalidPathError("system scope not permitted", "knowledge keyspace")
	}
	return nil
}

// getSystemCollection returns (nil, nil) if the bucket has no system collection (i.e. knowledge
// is not enabled for it) - callers decide for themselves whether that's an error or a no-op.
func getSystemCollection(bucket string) (datastore.Keyspace, errors.Error) {
	store := datastore.GetDatastore()
	if store == nil {
		return nil, errors.NewNoDatastoreError()
	}
	return store.GetSystemCollection(bucket)
}

// writeKnowledgeEntries applies build to the currently-stored knowledge entries for path (nil if
// the document doesn't exist yet) and writes the result back, retrying if a concurrent writer wins
// the race. build returns an error to abort without writing (e.g. a duplicate name) - that error
// is returned as-is, already a full domain error; any other failure (loading or writing the
// document) is passed through wrap to become one.
func writeKnowledgeEntries(b datastore.Keyspace, scopeUid, collectionUid string, path *algebra.Path,
	wrap func(error) errors.Error, build func(existing map[string]interface{}) (map[string]interface{}, errors.Error)) errors.Error {

	key := getStorageKey(scopeUid, collectionUid, path)

	for retry := 0; ; retry++ {
		if retry == _MAX_WRITE_RETRIES {
			return wrap(fmt.Errorf("failed after %d retries", retry))
		} else if retry > 0 {
			time.Sleep(time.Duration(retry) * _WRITE_RETRY_BACKOFF)
		}

		av, exists, lerr := loadDoc(b, key)
		if lerr != nil {
			return wrap(lerr)
		}

		var existing map[string]interface{}
		if exists {
			if obj, ok := av.Field(_FIELD); ok && obj.Type() == value.OBJECT {
				existing, _ = obj.Actual().(map[string]interface{})
			}
		}

		newEntries, berr := build(existing)
		if berr != nil {
			return berr
		}

		pairs := make([]value.Pair, 1)
		pairs[0].Name = key

		// build the new document independently of av rather than mutate it in place: av may be
		// the cached copy, and mutating it ahead of the write committing would let a concurrent
		// reader see not-yet-durable data
		var newDoc value.AnnotatedValue
		var errs2 errors.Errors
		if exists {
			newDoc = av.CopyForUpdate().(value.AnnotatedValue)
			newDoc.SetField(_FIELD, value.NewValue(newEntries))
			pairs[0].Value = newDoc
			_, _, errs2 = b.Update(pairs, datastore.GetDurableQueryContextFor(b), true)
		} else {
			m := map[string]interface{}{
				"namespace":  path.Namespace(),
				"bucket":     path.Bucket(),
				"scope":      path.Scope(),
				"collection": path.Keyspace(),
				// carried in the document body (not just the key) so Scan's cache-first pass can
				// recompute the exact storage key for a cached doc without a live UID lookup
				"scopeUid":      scopeUid,
				"collectionUid": collectionUid,
				_FIELD:          newEntries,
			}
			newDoc = value.NewAnnotatedValue(value.NewValue(m))
			pairs[0].Value = newDoc
			_, _, errs2 = b.Insert(pairs, datastore.GetDurableQueryContextFor(b), true)
		}
		if errs2 != nil && len(errs2) > 0 {
			// E_CAS_MISMATCH is Update losing a race against a concurrent writer; E_DUPLICATE_KEY is
			// Insert losing one (a concurrent writer created the document between our loadDoc seeing
			// it not exist and this Insert) - errors.NewDuplicateKeyError's message ("Duplicate Key:
			// ...") doesn't match IsExistsError's "already exist" pattern, so it needs its own check.
			if errs2[0].HasCause(errors.E_CAS_MISMATCH) || errs2[0].HasCause(errors.E_DUPLICATE_KEY) ||
				errors.IsExistsError("", errs2[0]) {
				invalidate(key) // our cached copy (if any) is out of date; refetch on retry
				continue        // lost a race with a concurrent writer; retry
			}
			return wrap(errs2[0])
		}
		setChange()
		refreshDoc(b, key)
		return nil
	}
}

// validateAndResolvePath checks that path is eligible for knowledge and resolves it to its current
// live scope/collection UIDs (see getStorageKey/resolveUids).
func validateAndResolvePath(path *algebra.Path) (b datastore.Keyspace, scopeUid, collectionUid string, err errors.Error) {
	if path.Scope() == "" || !path.IsCollection() {
		return nil, "", "", errors.NewKnowledgeError(errors.E_KNOWLEDGE_INVALID_PATH, path.SimpleString())
	}
	if path.Namespace() == datastore.SYSTEM_NAMESPACE {
		return nil, "", "", errors.NewKnowledgeError(errors.E_KNOWLEDGE_CREATE, path.SimpleString(),
			errors.NewDatastoreInvalidPathError("system namespace not permitted", "knowledge keyspace"))
	}
	if datastore.GetDatastore() == nil {
		return nil, "", "", errors.NewKnowledgeError(errors.E_KNOWLEDGE_CREATE, path.SimpleString(), errors.NewNoDatastoreError())
	}

	scopeUid, collectionUid, uerr := resolveUids(path)
	if uerr != nil {
		return nil, "", "", errors.NewKnowledgeError(errors.E_KNOWLEDGE_CREATE, path.SimpleString(), uerr)
	}

	b, err = getSystemCollection(path.Bucket())
	if err != nil {
		return nil, "", "", errors.NewKnowledgeError(errors.E_KNOWLEDGE_CREATE, path.SimpleString(), err)
	}
	if b == nil {
		return nil, "", "", errors.NewKnowledgeError(errors.E_KNOWLEDGE_CREATE, path.SimpleString(),
			"system collection not available for bucket "+path.Bucket())
	}
	if serr := rejectSystemScope(b, path); serr != nil {
		return nil, "", "", errors.NewKnowledgeError(errors.E_KNOWLEDGE_CREATE, path.SimpleString(), serr)
	}
	return b, scopeUid, collectionUid, nil
}

// CreateKnowledge adds a new named entry to the knowledge kept for the keyspace identified by
// path. Unless replace is set, it fails if an entry with the same name already exists for that
// keyspace; with replace, an existing entry's value is overwritten instead.
func CreateKnowledge(path *algebra.Path, name string, val string, replace bool) errors.Error {
	b, scopeUid, collectionUid, err := validateAndResolvePath(path)
	if err != nil {
		return err
	}

	wrap := func(cause error) errors.Error {
		return errors.NewKnowledgeError(errors.E_KNOWLEDGE_CREATE, name, cause)
	}
	return writeKnowledgeEntries(b, scopeUid, collectionUid, path, wrap, func(existing map[string]interface{}) (map[string]interface{}, errors.Error) {
		if _, dup := existing[name]; dup && !replace {
			return nil, errors.NewKnowledgeError(errors.E_KNOWLEDGE_ALREADY_EXISTS, name)
		}
		entries := make(map[string]interface{}, len(existing)+1)
		for k, v := range existing {
			entries[k] = v
		}
		entries[name] = val
		if len(entries) > MaxHintsPerKeyspace {
			return nil, errors.NewKnowledgeError(errors.E_KNOWLEDGE_LIMIT_EXCEEDED,
				fmt.Sprintf("keyspace already has the maximum of %d knowledge entries", MaxHintsPerKeyspace))
		}
		return entries, nil
	})
}

// CreateKnowledgeBulk adds multiple named entries to the keyspace identified by path in a single
// write (CREATE KNOWLEDGE FOR ... FROM <object>). Unless replace is set, it fails, without writing
// anything, if any of the given names already exists for that keyspace; with replace, existing
// entries among them are overwritten instead.
func CreateKnowledgeBulk(path *algebra.Path, entries map[string]string, replace bool) errors.Error {
	b, scopeUid, collectionUid, err := validateAndResolvePath(path)
	if err != nil {
		return err
	}

	wrap := func(cause error) errors.Error {
		return errors.NewKnowledgeError(errors.E_KNOWLEDGE_CREATE, path.SimpleString(), cause)
	}
	return writeKnowledgeEntries(b, scopeUid, collectionUid, path, wrap, func(existing map[string]interface{}) (map[string]interface{}, errors.Error) {
		if !replace {
			for name := range entries {
				if _, dup := existing[name]; dup {
					return nil, errors.NewKnowledgeError(errors.E_KNOWLEDGE_ALREADY_EXISTS, name)
				}
			}
		}
		merged := make(map[string]interface{}, len(existing)+len(entries))
		for k, v := range existing {
			merged[k] = v
		}
		for name, val := range entries {
			merged[name] = val
		}
		if len(merged) > MaxHintsPerKeyspace {
			return nil, errors.NewKnowledgeError(errors.E_KNOWLEDGE_LIMIT_EXCEEDED,
				fmt.Sprintf("keyspace would exceed the maximum of %d knowledge entries", MaxHintsPerKeyspace))
		}
		return merged, nil
	})
}

// splitExtKey splits a composite external key ("<ns>:<bucket>.<scope>.<collection>::<name>")
// into its keyspace-path and name components. Uses the first "::", not the last: the path portion
// (algebra.Path.FullName()) only ever joins elements with single ":"/"." separators and so can
// never itself contain "::", but name is a user-supplied (possibly back-tick-quoted) identifier
// that can - so the first occurrence is always the real delimiter, even if the name has one too.
func splitExtKey(key string) (string, string, bool) {
	i := strings.Index(key, "::")
	if i < 0 {
		return "", "", false
	}
	return key[:i], key[i+2:], true
}

func rowFor(path *algebra.Path, name string, val value.Value) value.AnnotatedValue {
	m := map[string]interface{}{
		"namespace":  path.Namespace(),
		"bucket":     path.Bucket(),
		"scope":      path.Scope(),
		"collection": path.Keyspace(),
		"keyspace":   path.FullName(),
		"name":       name,
	}
	if val != nil {
		m["value"] = val
	}
	return value.NewAnnotatedValue(value.NewValue(m))
}

// parseExtKey splits a composite external key into the keyspace path and name it addresses.
func parseExtKey(extKey string) (*algebra.Path, string, errors.Error) {
	ksPath, name, ok := splitExtKey(extKey)
	if !ok {
		return nil, "", errors.NewKnowledgeError(errors.E_KNOWLEDGE_INVALID_PATH, extKey)
	}
	elements := algebra.ParsePath(ksPath)
	if len(elements) != 4 {
		return nil, "", errors.NewKnowledgeError(errors.E_KNOWLEDGE_INVALID_PATH, extKey)
	}
	return algebra.NewPathFromElements(elements), name, nil
}

// BucketFromExtKey returns the bucket a composite external key addresses, without fetching
// anything. Used by the system:natural_knowledge catalog keyspace's Fetch (USE KEYS support) to
// check the requesting user's access to that bucket before calling FetchEntry - unlike Scan/Count,
// USE KEYS never limits itself to datastore.GetUserBuckets(), so it must check per-key instead.
func BucketFromExtKey(extKey string) (string, errors.Error) {
	path, _, err := parseExtKey(extKey)
	if err != nil {
		return "", err
	}
	return path.Bucket(), nil
}

// FetchEntry fetches a single knowledge entry addressed by its composite external key, as used by
// the system:knowledge catalog keyspace. Returns (nil, nil) if the entry does not exist.
func FetchEntry(extKey string) (value.AnnotatedValue, errors.Error) {
	path, name, err := parseExtKey(extKey)
	if err != nil {
		return nil, err
	}

	b, resolved, scopeUid, collectionUid, rerr := resolveForRead(path)
	if rerr != nil {
		return nil, rerr
	}
	if b == nil {
		return nil, nil
	}

	av, exists, lerr := loadDoc(b, getStorageKey(scopeUid, collectionUid, resolved))
	if lerr != nil {
		return nil, lerr
	}
	if !exists {
		return nil, nil
	}
	obj, ok := av.Field(_FIELD)
	if !ok || obj.Type() != value.OBJECT {
		return nil, nil
	}
	vv, ok := obj.Field(name)
	if !ok {
		return nil, nil
	}
	return rowFor(resolved, name, vv), nil
}

// resolveForRead resolves path the same way validateAndResolvePath does for writes, but tolerates
// (returns nil, "", "", nil) rather than erroring on any "there's nothing here to read" condition -
// keyspace/scope not found, no system collection for the bucket, or a path targeting the reserved
// system scope - since callers reading knowledge legitimately expect "no knowledge for this
// keyspace" to look identical to "keyspace not there", not to be a hard error.
func resolveForRead(path *algebra.Path) (b datastore.Keyspace, resolved *algebra.Path, scopeUid, collectionUid string, err errors.Error) {
	// a bare bucket name (no scope/collection given) implicitly means bucket._default._default,
	// mirroring algebra.newCreateKnowledge's normalization on the write side - without this, a
	// USING AI AND KNOWLEDGE request naming just a bucket (the common case) would never find
	// knowledge created against that same bucket. The normalized path is returned so callers use
	// it (not their original, possibly-short path) to compute the matching storage key.
	if path.Scope() == "" {
		path = algebra.NewPathLong(path.Namespace(), path.Bucket(), "_default", "_default")
	}
	if !path.IsCollection() || path.Namespace() == datastore.SYSTEM_NAMESPACE {
		return nil, nil, "", "", nil
	}

	scopeUid, collectionUid, uerr := resolveUids(path)
	if uerr != nil {
		if errors.IsNotFoundError("", uerr) {
			return nil, nil, "", "", nil
		}
		return nil, nil, "", "", uerr
	}

	b, err = getSystemCollection(path.Bucket())
	if err != nil {
		if errors.IsNotFoundError("", err) {
			return nil, nil, "", "", nil
		}
		return nil, nil, "", "", err
	}
	if b == nil || rejectSystemScope(b, path) != nil {
		return nil, nil, "", "", nil
	}
	return b, path, scopeUid, collectionUid, nil
}

// FetchAll returns every knowledge entry stored for the keyspace identified by path, as a
// name-to-value map, or (nil, nil) if the keyspace has no knowledge entries (including if it has
// no system collection, or targets the reserved system scope). Used by natural.KnowledgeInjector
// to gather knowledge for a USING AI AND KNOWLEDGE prompt - the caller is expected to have already
// authorized the keyspace itself before calling this.
func FetchAll(path *algebra.Path) (map[string]string, errors.Error) {
	b, resolved, scopeUid, collectionUid, rerr := resolveForRead(path)
	if rerr != nil {
		return nil, rerr
	}
	if b == nil {
		return nil, nil
	}

	av, exists, lerr := loadDoc(b, getStorageKey(scopeUid, collectionUid, resolved))
	if lerr != nil {
		return nil, lerr
	}
	if !exists {
		return nil, nil
	}
	obj, ok := av.Field(_FIELD)
	if !ok || obj.Type() != value.OBJECT {
		return nil, nil
	}
	fields := obj.Fields()
	if len(fields) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(fields))
	for name, v := range fields {
		out[name] = value.NewValue(v).ToString()
	}
	return out, nil
}

// DropCollection removes the knowledge document (if any) holding the entries for the
// collection. Mirrors aus.DropCollection (same signature shape, taking both name and UID): invoked
// whenever a collection disappears from a bucket's manifest - whether dropped via DROP COLLECTION,
// as part of a DROP SCOPE/bucket drop, or externally - so that knowledge doesn't outlive the
// collection it describes. A bucket with no system collection (and therefore no knowledge to begin
// with) is not an error.
func DropCollection(namespace, bucket, scope, scopeUid, collection, collectionUid string) errors.Error {
	key := getStorageKey(scopeUid, collectionUid, algebra.NewPathLong(namespace, bucket, scope, collection))

	// invalidate the local cache unconditionally, up front, as a hygiene measure: the live cleanup
	// below can fail with "not found" if the bucket itself is already gone (e.g. this collection
	// disappeared because its whole bucket was dropped), in which case this wouldn't otherwise run.
	// Since the storage key is UID-specific, a collection recreated under the same name computes a
	// different key anyway, so this isn't load-bearing for correctness the way it was with the
	// old, purely name-based key - it just frees the slot promptly instead of waiting for LRU
	// eviction.
	invalidate(key)

	b, err := getSystemCollection(bucket)
	if err != nil {
		if errors.IsNotFoundError("", err) {
			return nil
		}
		return err
	}
	if b == nil {
		return nil
	}

	pairs := make([]value.Pair, 1)
	pairs[0].Name = key
	_, _, errs := b.Delete(pairs, datastore.GetDurableQueryContextFor(b), true)
	if errs != nil && len(errs) > 0 && !errors.IsNotFoundError("", errs[0]) {
		return errors.NewKnowledgeError(errors.E_KNOWLEDGE_DROP, key, errs[0])
	}
	setChange()
	return nil
}

// DropScope removes every knowledge document stored against a collection within the given scope.
// Mirrors aus.DropScope (same signature shape, taking both name and UID): invoked whenever a scope
// disappears from a bucket's manifest - whether dropped via DROP SCOPE, as part of a bucket drop,
// or externally.
func DropScope(namespace, bucket, scope, scopeUid string) errors.Error {
	// targeted prefix scan by scope UID, not by name: every collection-level key under this scope
	// shares the scopeUid segment (see getStorageKey), so this one scan catches all of them without
	// a full bucket scan - mirrors aus.DropScope's key scheme exactly.
	prefix := _PREFIX + scopeUid + "::"

	// purge the local cache unconditionally, up front, as a hygiene measure - see the comment in
	// DropCollection: no longer load-bearing for correctness now that keys are UID-specific, but it
	// frees cache slots for this scope promptly rather than waiting for LRU eviction.
	purgeScope(bucket, scope)

	var context datastore.QueryContext
	pairs := make(value.Pairs, 0, _BATCH_SIZE)
	dropped := false

	flush := func(systemCollection datastore.Keyspace) {
		// another node's own manifest-diff cleanup for this same scope can be racing this one, so a
		// delete failing here (e.g. NotFound, because that node already removed it) isn't a real
		// error - mirrors aus.DropScope, which likewise treats this delete as fire-and-forget.
		systemCollection.Delete(pairs, context, false)
		for _, p := range pairs {
			invalidate(p.Name)
		}
		dropped = true
		pairs = pairs[:0]
	}

	serr := datastore.ScanSystemCollection(bucket, prefix,
		func(systemCollection datastore.Keyspace) errors.Error {
			context = datastore.GetDurableQueryContextFor(systemCollection)
			return nil
		},
		func(key string, systemCollection datastore.Keyspace) errors.Error {
			pairs = append(pairs, value.Pair{Name: key})
			if len(pairs) >= _BATCH_SIZE {
				flush(systemCollection)
			}
			return nil
		},
		func(systemCollection datastore.Keyspace) errors.Error {
			if len(pairs) > 0 {
				flush(systemCollection)
			}
			return nil
		})
	if serr != nil && serr.Code() != errors.E_SYSTEM_COLLECTION {
		return errors.NewKnowledgeError(errors.E_KNOWLEDGE_DROP, prefix, serr)
	}
	if dropped {
		setChange()
	}
	return nil
}

// emitDocEntries invokes cb with the composite external key ("<namespace>:<bucket>.<scope>.
// <collection>::<name>") of every named entry in doc.
func emitDocEntries(av value.AnnotatedValue, cb func(string) error) error {
	obj, ok := av.Field(_FIELD)
	if !ok || obj.Type() != value.OBJECT {
		return nil
	}
	ns, _ := av.Field("namespace")
	bk, _ := av.Field("bucket")
	sc, _ := av.Field("scope")
	co, _ := av.Field("collection")
	// escape via FullName, matching how FetchEntry's splitExtKey/ParsePath expect the keyspace
	// portion of the composite key to be formatted
	ksPath := algebra.NewPathLong(ns.ToString(), bk.ToString(), sc.ToString(), co.ToString()).FullName()

	for name := range obj.Fields() {
		if err := cb(ksPath + "::" + name); err != nil {
			return err
		}
	}
	return nil
}

// Scan walks the knowledge entries stored for the given bucket, invoking cb with the composite
// external key of each entry found.
func Scan(bucket string, cb func(string) error) errors.Error {
	// serve whatever this node already has cached for the bucket first - closes the read-after-
	// write gap for entries this node itself created/refreshed, without waiting on the GSI index
	// behind ScanSystemCollection to catch up. cachedKeys tracks which storage keys were already
	// served this way, so the live scan below doesn't emit them a second time.
	cachedKeys := make(map[string]bool)
	var cerr error
	forEachCachedDoc(func(doc value.AnnotatedValue) {
		if cerr != nil {
			return
		}
		bk, ok := doc.Field("bucket")
		if !ok || bk.ToString() != bucket {
			return
		}
		ns, _ := doc.Field("namespace")
		sc, _ := doc.Field("scope")
		co, _ := doc.Field("collection")
		scopeUid, _ := doc.Field("scopeUid")
		collectionUid, _ := doc.Field("collectionUid")
		path := algebra.NewPathLong(ns.ToString(), bucket, sc.ToString(), co.ToString())
		cachedKeys[getStorageKey(scopeUid.ToString(), collectionUid.ToString(), path)] = true
		if err := emitDocEntries(doc, cb); err != nil {
			cerr = err
		}
	})
	if cerr != nil {
		return errors.NewKnowledgeError(errors.E_KNOWLEDGE, cerr)
	}

	// batch the live-scanned keys rather than fetching one at a time, mirroring DropScope's
	// flush pattern
	keys := make([]string, 0, _BATCH_SIZE)
	var ferr errors.Error

	flush := func(systemCollection datastore.Keyspace) {
		if len(keys) == 0 {
			return
		}
		res := make(map[string]value.AnnotatedValue, len(keys))
		errs := systemCollection.Fetch(keys, res, datastore.NULL_QUERY_CONTEXT, nil, nil, false)
		if len(errs) > 0 && !errors.IsNotFoundError("", errs[0]) && !errs[0].HasCause(errors.E_CB_BULK_GET) {
			ferr = errs[0]
		}
		for _, key := range keys {
			if av, ok := res[key]; ok {
				if err := emitDocEntries(av, cb); err != nil {
					ferr = errors.NewKnowledgeError(errors.E_KNOWLEDGE, err)
				}
			}
		}
		keys = keys[:0]
	}

	return datastore.ScanSystemCollection(bucket, _PREFIX, nil,
		func(key string, systemCollection datastore.Keyspace) errors.Error {
			if cachedKeys[key] {
				return nil
			}
			keys = append(keys, key)
			if len(keys) >= _BATCH_SIZE {
				flush(systemCollection)
				if ferr != nil {
					return ferr
				}
			}
			return nil
		},
		func(systemCollection datastore.Keyspace) errors.Error {
			flush(systemCollection)
			return ferr
		})
}
