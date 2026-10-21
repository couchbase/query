package util

import (
	"encoding/json"
	"strconv"
	"testing"
)

// Mirrors execution.groupKey(): pull a map from the pool, populate it with the
// evaluated GROUP BY key values, marshal it to a string key, return it to the
// pool. The only difference between Old and New is the clear strategy in Put().
//
// groupKey() runs once per input document in each of the 3 GROUP BY phases
// (initial, intermediate, final), so per-call cost is multiplied by
// 3 x document count.

const _NKEYS = 3 // GROUP BY city, category, tier

var benchVals = []any{"San Francisco", "electronics", "premium"}

var newPool = NewMapPool[string, any](16)
var oldPool = NewMapPool[string, any](16)

// oldPut reproduces the pre-change Put(): zero each value, then delete it.
func oldPut(p *MapPool[string, any], s map[string]any) {
	if s == nil || len(s) > p.Size() {
		return
	}
	for k, _ := range s {
		s[k] = nil
		delete(s, k)
	}
	p.pool.Put(s)
}

func groupKeyOld() string {
	kvs := oldPool.GetCapped(_NKEYS)
	defer oldPut(oldPool, kvs)
	for i := 0; i < _NKEYS; i++ {
		kvs[strconv.Itoa(i)] = benchVals[i]
	}
	b, _ := json.Marshal(kvs)
	return string(b)
}

func groupKeyNew() string {
	kvs := newPool.GetCapped(_NKEYS)
	defer newPool.Put(kvs)
	for i := 0; i < _NKEYS; i++ {
		kvs[strconv.Itoa(i)] = benchVals[i]
	}
	b, _ := json.Marshal(kvs)
	return string(b)
}

func BenchmarkGroupKeyOld(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if groupKeyOld() == "" {
			b.Fatal("empty key")
		}
	}
}

func BenchmarkGroupKeyNew(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if groupKeyNew() == "" {
			b.Fatal("empty key")
		}
	}
}
