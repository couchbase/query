// Copyright 2026-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in
// that file, in accordance with the Business Source License, use of this
// software will be governed by the Apache License, Version 2.0, included in
// the file licenses/APL2.txt.

package expression

import (
	"testing"

	"github.com/couchbase/query/value"
)

func TestMultiKeyObjects_bareConstantHasNoAliasYieldsMissing(t *testing.T) {

	/* MULTIKEY_OBJECTS(doc), where doc is a Constant: a Constant has no
	   implicit alias, so there is no field name to build the implied
	   {"<alias>": doc} object construction with -- the result is
	   missing, even though doc itself is a present, non-missing value */
	doc := map[string]interface{}{"id": 1, "tags": []interface{}{"a", "b"}}
	rv, err := NewMultiKeyObjects(NewConstant(doc)).Evaluate(nil, nil)
	if err != nil {
		t.Fatalf("received error %v", err)
	}
	if rv.Type() != value.MISSING {
		t.Errorf("expected missing, got %v", rv.Actual())
	}
}

func TestMultiKeyObjects_missing(t *testing.T) {

	/* missing input yields missing */
	rv, err := NewMultiKeyObjects(NewConstant(value.MISSING_VALUE)).Evaluate(nil, nil)
	if err != nil {
		t.Fatalf("received error %v", err)
	}
	if rv.Type() != value.MISSING {
		t.Errorf("expected missing, got %v", rv.Actual())
	}
}

func TestMultiKeyObjects_nonObjectConstantHasNoAliasYieldsMissing(t *testing.T) {

	/* MULTIKEY_OBJECTS(5): a Constant has no implicit alias, so there is
	   no field name to build {"<alias>": 5} with -- the result is
	   missing */
	rv, err := NewMultiKeyObjects(NewConstant(5)).Evaluate(nil, nil)
	if err != nil {
		t.Fatalf("received error %v", err)
	}
	if rv.Type() != value.MISSING {
		t.Errorf("expected missing, got %v", rv.Actual())
	}
}

func TestMultiKeyObjects_barePathNotPresentOmitsKeyLikeObjectConstruct(t *testing.T) {

	/* MULTIKEY_OBJECTS(x.y), where x.y is not present: this is run through
	   the exact same pipeline as MULTIKEY_OBJECTS({"y": x.y}), so a
	   missing path just omits the "y" key -- and since that leaves the
	   (sole) result object with no fields set at all, it is dropped
	   entirely, rather than surfaced as a spurious empty object; the
	   overall result is an empty array, not missing */
	item := value.NewValue(map[string]interface{}{"x": map[string]interface{}{}})
	rv, err := NewMultiKeyObjects(
		NewField(NewIdentifier("x"), NewFieldName("y", false)),
	).Evaluate(item, nil)
	if err != nil {
		t.Fatalf("received error %v", err)
	}
	expected := value.NewValue([]interface{}{})
	if expected.Collate(rv) != 0 {
		t.Errorf("expected %v, got %v", expected.Actual(), rv.Actual())
	}
}

func TestMultiKeyObjects_bareIdentifierArrayExpandsLikeObjectConstruct(t *testing.T) {

	/* MULTIKEY_OBJECTS(tags) and MULTIKEY_OBJECTS({tags}) (i.e.
	   MULTIKEY_OBJECTS({"tags": tags})) must be the same: tags has an
	   implicit alias ("tags"), so the array is walked and denormalized
	   exactly as it would be as an object-construct field */
	item := value.NewValue(map[string]interface{}{"id": 1, "tags": []interface{}{"a", "b"}})

	bare, err := NewMultiKeyObjects(NewIdentifier("tags")).Evaluate(item, nil)
	if err != nil {
		t.Fatalf("received error %v", err)
	}

	braced, err := NewMultiKeyObjects(NewObjectConstruct(map[Expression]Expression{
		NewConstant("tags"): NewIdentifier("tags"),
	})).Evaluate(item, nil)
	if err != nil {
		t.Fatalf("received error %v", err)
	}

	if bare.Collate(braced) != 0 {
		t.Errorf("expected MULTIKEY_OBJECTS(tags) == MULTIKEY_OBJECTS({tags}), got %v vs %v", bare.Actual(), braced.Actual())
	}

	expected := []interface{}{
		map[string]interface{}{"tags": "a"},
		map[string]interface{}{"tags": "b"},
	}
	testMultiKeyObjectsPath_eval(map[Expression]Expression{NewConstant("tags"): NewIdentifier("tags")}, item, expected, t)
}

func TestMultiKeyObjects_bareIdentifierNonArrayIsWrappedUnderAlias(t *testing.T) {

	/* MULTIKEY_OBJECTS(doc), where doc is an Identifier (so it DOES have
	   an implicit alias, unlike a Constant): it is wrapped under that
	   alias, same as MULTIKEY_OBJECTS({"doc": doc}) would be -- unlike
	   TestMultiKeyObjects_bareConstantHasNoAliasYieldsMissing's Constant
	   case */
	item := value.NewValue(map[string]interface{}{
		"doc": map[string]interface{}{"id": 1, "tags": []interface{}{"a", "b"}},
	})
	rv, err := NewMultiKeyObjects(NewIdentifier("doc")).Evaluate(item, nil)
	if err != nil {
		t.Fatalf("received error %v", err)
	}
	expected := value.NewValue([]interface{}{
		map[string]interface{}{"doc": map[string]interface{}{"id": 1, "tags": []interface{}{"a", "b"}}},
	})
	if expected.Collate(rv) != 0 {
		t.Errorf("expected %v, got %v", expected.Actual(), rv.Actual())
	}
}

func TestMultiKeyObjects_bareIdentifierEmptyArrayMatchesObjectConstruct(t *testing.T) {

	/* MULTIKEY_OBJECTS(tags), where tags is present but an empty array:
	   run through the same pipeline as MULTIKEY_OBJECTS({"tags": tags}),
	   which -- since tags is the sole field -- contributes no value at
	   all, leaving nothing set; that result object is dropped, so the
	   overall result is an empty array */
	item := value.NewValue(map[string]interface{}{"tags": []interface{}{}})
	rv, err := NewMultiKeyObjects(NewIdentifier("tags")).Evaluate(item, nil)
	if err != nil {
		t.Fatalf("received error %v", err)
	}
	expected := value.NewValue([]interface{}{})
	if expected.Collate(rv) != 0 {
		t.Errorf("expected %v, got %v", expected.Actual(), rv.Actual())
	}
}

func TestMultiKeyObjects_barePathPartiallyMissingAcrossArrayOmitsKeyPerElement(t *testing.T) {

	/* MULTIKEY_OBJECTS(items.sku), where some items lack "sku": run
	   through the same pipeline as MULTIKEY_OBJECTS({"sku": items.sku}),
	   which omits the "sku" key for elements that lack it -- and since
	   "sku" is the sole field, that leaves those elements' result
	   objects with nothing set at all, so they're dropped entirely;
	   elements that do have "sku" still expand normally */
	item := value.NewValue(map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"sku": "A1"},
			map[string]interface{}{"other": "x"},
			map[string]interface{}{"sku": "B2"},
		},
	})
	rv, err := NewMultiKeyObjects(
		NewField(NewIdentifier("items"), NewFieldName("sku", false)),
	).Evaluate(item, nil)
	if err != nil {
		t.Fatalf("received error %v", err)
	}
	expected := value.NewValue([]interface{}{
		map[string]interface{}{"sku": "A1"},
		map[string]interface{}{"sku": "B2"},
	})
	if expected.Collate(rv) != 0 {
		t.Errorf("expected %v, got %v", expected.Actual(), rv.Actual())
	}
}

/*
Evaluates MULTIKEY_OBJECTS(objectConstructExpr) against the given
top-level item, using the same order-insensitive comparison as
testMultiKeyObjectsPath_eval's callers rely on.
*/
func testMultiKeyObjectsPath_eval(mapping map[Expression]Expression, item value.Value, expected []interface{}, t *testing.T) {
	rv, err := NewMultiKeyObjects(NewObjectConstruct(mapping)).Evaluate(item, nil)
	if err != nil {
		t.Fatalf("received error %v", err)
	}

	actual, ok := rv.Actual().([]interface{})
	if !ok {
		t.Fatalf("expected array result, got %T: %v", rv.Actual(), rv.Actual())
	}
	if len(actual) != len(expected) {
		t.Fatalf("expected %d results, got %d: %v", len(expected), len(actual), actual)
	}

	used := make([]bool, len(actual))
	for _, ev := range expected {
		found := false
		e := value.NewValue(ev)
		for i, av := range actual {
			if used[i] {
				continue
			}
			if e.Collate(value.NewValue(av)) == 0 {
				used[i] = true
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected result %v not found in actual %v", ev, actual)
		}
	}
}

func TestMultiKeyObjects_singleArray(t *testing.T) {

	/* MULTIKEY_OBJECTS({"id": id, "tags": tags}): one array field is
	   expanded, one object per element */
	item := value.NewValue(map[string]interface{}{"id": 1, "tags": []interface{}{"a", "b"}})
	mapping := map[Expression]Expression{
		NewConstant("id"):   NewIdentifier("id"),
		NewConstant("tags"): NewIdentifier("tags"),
	}
	expected := []interface{}{
		map[string]interface{}{"id": 1, "tags": "a"},
		map[string]interface{}{"id": 1, "tags": "b"},
	}
	testMultiKeyObjectsPath_eval(mapping, item, expected, t)
}

func TestMultiKeyObjects_multipleArrays_cartesianProduct(t *testing.T) {

	/* MULTIKEY_OBJECTS({"id": id, "a": a, "b": b}): two array fields
	   multiply out instead of returning null */
	item := value.NewValue(map[string]interface{}{
		"id": 1, "a": []interface{}{1, 2}, "b": []interface{}{"x", "y"},
	})
	mapping := map[Expression]Expression{
		NewConstant("id"): NewIdentifier("id"),
		NewConstant("a"):  NewIdentifier("a"),
		NewConstant("b"):  NewIdentifier("b"),
	}
	expected := []interface{}{
		map[string]interface{}{"id": 1, "a": 1, "b": "x"},
		map[string]interface{}{"id": 1, "a": 1, "b": "y"},
		map[string]interface{}{"id": 1, "a": 2, "b": "x"},
		map[string]interface{}{"id": 1, "a": 2, "b": "y"},
	}
	testMultiKeyObjectsPath_eval(mapping, item, expected, t)
}

func TestMultiKeyObjects_arrayOfObjectsIsNotExpandedFurther(t *testing.T) {

	/* MULTIKEY_OBJECTS({"id": id, "d": d}): a top-level array field is
	   expanded one level -- its elements (here, objects with their own
	   nested arrays) are used as-is, without any further recursion */
	item := value.NewValue(map[string]interface{}{
		"id": 1,
		"d": []interface{}{
			map[string]interface{}{"e": []interface{}{1, 2}},
			map[string]interface{}{"e": []interface{}{3}},
		},
	})
	mapping := map[Expression]Expression{
		NewConstant("id"): NewIdentifier("id"),
		NewConstant("d"):  NewIdentifier("d"),
	}
	expected := []interface{}{
		map[string]interface{}{"id": 1, "d": map[string]interface{}{"e": []interface{}{1, 2}}},
		map[string]interface{}{"id": 1, "d": map[string]interface{}{"e": []interface{}{3}}},
	}
	testMultiKeyObjectsPath_eval(mapping, item, expected, t)
}

func TestMultiKeyObjects_correlatedFieldsFromSameArray(t *testing.T) {

	/* MULTIKEY_OBJECTS({"id": id, "sku": items.sku, "tag": items.tags}):
	   sku and tag are both walked from the same items array, so they
	   are correlated (zipped) per item rather than cross-produced --
	   sku "A1" only ever appears with tags belonging to its own item */
	field := func(base Expression, name string) Expression {
		return NewField(base, NewFieldName(name, false))
	}
	item := value.NewValue(map[string]interface{}{
		"id": 1,
		"items": []interface{}{
			map[string]interface{}{"sku": "A1", "tags": []interface{}{"x", "y"}},
			map[string]interface{}{"sku": "B2", "tags": []interface{}{"z"}},
		},
	})
	mapping := map[Expression]Expression{
		NewConstant("id"):  NewIdentifier("id"),
		NewConstant("sku"): field(NewIdentifier("items"), "sku"),
		NewConstant("tag"): field(NewIdentifier("items"), "tags"),
	}
	expected := []interface{}{
		map[string]interface{}{"id": 1, "sku": "A1", "tag": "x"},
		map[string]interface{}{"id": 1, "sku": "A1", "tag": "y"},
		map[string]interface{}{"id": 1, "sku": "B2", "tag": "z"},
	}
	testMultiKeyObjectsPath_eval(mapping, item, expected, t)
}

func TestMultiKeyObjects_unrelatedArraysStillCartesianProduct(t *testing.T) {

	/* MULTIKEY_OBJECTS({"id": id, "sku": items.sku, "city": addr.city}):
	   sku and city are walked from two different, unrelated arrays, so
	   they still combine via an ordinary cartesian product */
	field := func(base Expression, name string) Expression {
		return NewField(base, NewFieldName(name, false))
	}
	item := value.NewValue(map[string]interface{}{
		"id":    1,
		"items": []interface{}{map[string]interface{}{"sku": "A1"}, map[string]interface{}{"sku": "B2"}},
		"addr":  []interface{}{map[string]interface{}{"city": "NYC"}, map[string]interface{}{"city": "LA"}},
	})
	mapping := map[Expression]Expression{
		NewConstant("id"):   NewIdentifier("id"),
		NewConstant("sku"):  field(NewIdentifier("items"), "sku"),
		NewConstant("city"): field(NewIdentifier("addr"), "city"),
	}
	expected := []interface{}{
		map[string]interface{}{"id": 1, "sku": "A1", "city": "NYC"},
		map[string]interface{}{"id": 1, "sku": "A1", "city": "LA"},
		map[string]interface{}{"id": 1, "sku": "B2", "city": "NYC"},
		map[string]interface{}{"id": 1, "sku": "B2", "city": "LA"},
	}
	testMultiKeyObjectsPath_eval(mapping, item, expected, t)
}

func TestMultiKeyObjects_emptyArrayFieldOmitted(t *testing.T) {

	/* MULTIKEY_OBJECTS({"id": id, "a": a, "tags": tags}): an empty array
	   field is omitted from the result objects; a sibling array field
	   still expands normally */
	item := value.NewValue(map[string]interface{}{
		"id": 1, "a": []interface{}{1, 2}, "tags": []interface{}{},
	})
	mapping := map[Expression]Expression{
		NewConstant("id"):   NewIdentifier("id"),
		NewConstant("a"):    NewIdentifier("a"),
		NewConstant("tags"): NewIdentifier("tags"),
	}
	expected := []interface{}{
		map[string]interface{}{"id": 1, "a": 1},
		map[string]interface{}{"id": 1, "a": 2},
	}
	testMultiKeyObjectsPath_eval(mapping, item, expected, t)
}

func TestMultiKeyObjects_noArrayField(t *testing.T) {

	/* MULTIKEY_OBJECTS({"id": id, "name": name}): no array fields, so the
	   object is returned unchanged, as a single-element array */
	item := value.NewValue(map[string]interface{}{"id": 1, "name": "foo"})
	mapping := map[Expression]Expression{
		NewConstant("id"):   NewIdentifier("id"),
		NewConstant("name"): NewIdentifier("name"),
	}
	expected := []interface{}{
		map[string]interface{}{"id": 1, "name": "foo"},
	}
	testMultiKeyObjectsPath_eval(mapping, item, expected, t)
}

func TestMultiKeyObjects_pathWalksThroughArrayWithoutExplicitBrackets(t *testing.T) {

	/* a.b, where a is an array of objects, walks each element for "b"
	   as if written a[].b, without needing [] to be written explicitly */
	item := value.NewValue(map[string]interface{}{
		"a": []interface{}{
			map[string]interface{}{"b": 1},
			map[string]interface{}{"b": 2},
		},
	})
	mapping := map[Expression]Expression{
		NewConstant("x"): NewField(NewIdentifier("a"), NewFieldName("b", false)),
	}
	expected := []interface{}{
		map[string]interface{}{"x": 1},
		map[string]interface{}{"x": 2},
	}
	testMultiKeyObjectsPath_eval(mapping, item, expected, t)
}

func TestMultiKeyObjects_pathPastScalarProducesNull(t *testing.T) {

	/* a.b.c, where a.b is a scalar, cannot be navigated any further:
	   it produces null rather than missing */
	item := value.NewValue(map[string]interface{}{
		"a": map[string]interface{}{"b": 5},
	})
	mapping := map[Expression]Expression{
		NewConstant("x"): NewField(
			NewField(NewIdentifier("a"), NewFieldName("b", false)),
			NewFieldName("c", false),
		),
	}
	expected := []interface{}{
		map[string]interface{}{"x": nil},
	}
	testMultiKeyObjectsPath_eval(mapping, item, expected, t)
}

func TestMultiKeyObjects_bareDIdentifierWrapsWholeDocument(t *testing.T) {

	/* MULTIKEY_OBJECTS(d), where d is the whole current document (an
	   Identifier, so it has an implicit alias): wrapped under that
	   alias, as a single-element array, exactly like
	   MULTIKEY_OBJECTS({d}) == MULTIKEY_OBJECTS({"d": d}) would be */
	doc := map[string]interface{}{"id": 1, "tags": []interface{}{"a", "b"}}
	item := value.NewValue(map[string]interface{}{"d": doc})

	rv, err := NewMultiKeyObjects(NewIdentifier("d")).Evaluate(item, nil)
	if err != nil {
		t.Fatalf("received error %v", err)
	}
	expected := value.NewValue([]interface{}{
		map[string]interface{}{"d": doc},
	})
	if expected.Collate(rv) != 0 {
		t.Errorf("expected %v, got %v", expected.Actual(), rv.Actual())
	}
}

func TestMultiKeyObjects_siblingScalarAndArrayShareBaseCartesianProduct(t *testing.T) {

	/* {d.sku, d.tags}, where sku and tags are sibling fields sharing
	   the (non-array) base d: they combine via an ordinary cartesian
	   product, evaluated independently per row */
	field := func(name string) Expression {
		return NewField(NewIdentifier("d"), NewFieldName(name, false))
	}
	mapping := map[Expression]Expression{
		NewConstant("sku"):  field("sku"),
		NewConstant("tags"): field("tags"),
	}

	row1 := value.NewValue(map[string]interface{}{
		"d": map[string]interface{}{"sku": "A1", "tags": []interface{}{"x", "y"}},
	})
	testMultiKeyObjectsPath_eval(mapping, row1, []interface{}{
		map[string]interface{}{"sku": "A1", "tags": "x"},
		map[string]interface{}{"sku": "A1", "tags": "y"},
	}, t)

	row2 := value.NewValue(map[string]interface{}{
		"d": map[string]interface{}{"sku": "B2", "tags": []interface{}{"z"}},
	})
	testMultiKeyObjectsPath_eval(mapping, row2, []interface{}{
		map[string]interface{}{"sku": "B2", "tags": "z"},
	}, t)
}

func TestMultiKeyObjects_nestedBaseCorrelatedSiblingFields(t *testing.T) {

	/* {d.items.sku, d.items.tags}: sku and tags are correlated by
	   sharing the array base d.items (zipped per item), same as when
	   items is a top-level identifier rather than nested under d */
	field := func(base Expression, name string) Expression {
		return NewField(base, NewFieldName(name, false))
	}
	mapping := map[Expression]Expression{
		NewConstant("sku"):  field(field(NewIdentifier("d"), "items"), "sku"),
		NewConstant("tags"): field(field(NewIdentifier("d"), "items"), "tags"),
	}
	item := value.NewValue(map[string]interface{}{
		"d": map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"sku": "A1", "tags": []interface{}{"x", "y"}},
				map[string]interface{}{"sku": "B2", "tags": []interface{}{"z"}},
			},
		},
	})
	expected := []interface{}{
		map[string]interface{}{"sku": "A1", "tags": "x"},
		map[string]interface{}{"sku": "A1", "tags": "y"},
		map[string]interface{}{"sku": "B2", "tags": "z"},
	}
	testMultiKeyObjectsPath_eval(mapping, item, expected, t)
}

func TestMultiKeyObjects_nestedBaseCorrelatedSiblingFieldsOfObjects(t *testing.T) {

	/* {d.items.sku, d.items.tags}, where each item's tags array holds
	   objects rather than scalars: tags is a leaf relative to items
	   (not decomposed further), so its elements -- whole objects -- are
	   used as-is and cartesian-combined against that item's own sku */
	field := func(base Expression, name string) Expression {
		return NewField(base, NewFieldName(name, false))
	}
	mapping := map[Expression]Expression{
		NewConstant("sku"):  field(field(NewIdentifier("d"), "items"), "sku"),
		NewConstant("tags"): field(field(NewIdentifier("d"), "items"), "tags"),
	}
	item := value.NewValue(map[string]interface{}{
		"d": map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{
					"sku": "A1",
					"tags": []interface{}{
						map[string]interface{}{"x": "x"},
						map[string]interface{}{"y": []interface{}{"y1", "y2"}},
					},
				},
				map[string]interface{}{
					"sku": "B1",
					"tags": []interface{}{
						map[string]interface{}{"x": "x1"},
						map[string]interface{}{"y": []interface{}{"y11", "y12"}},
					},
				},
			},
		},
	})
	expected := []interface{}{
		map[string]interface{}{"sku": "A1", "tags": map[string]interface{}{"x": "x"}},
		map[string]interface{}{"sku": "A1", "tags": map[string]interface{}{"y": []interface{}{"y1", "y2"}}},
		map[string]interface{}{"sku": "B1", "tags": map[string]interface{}{"x": "x1"}},
		map[string]interface{}{"sku": "B1", "tags": map[string]interface{}{"y": []interface{}{"y11", "y12"}}},
	}
	testMultiKeyObjectsPath_eval(mapping, item, expected, t)
}

func TestMultiKeyObjects_deeperPathIntoArrayOfMixedSchemaObjects(t *testing.T) {

	/* {d.items.sku, d.items.tags.y}: tags.y decomposes one level deeper
	   than the previous test, walking into each tags-array element for
	   "y" -- present on some elements, missing on others. A missing "y"
	   just omits that key (sku is a sibling field, so the row survives
	   without it), it does not drop the whole row */
	field := func(base Expression, name string) Expression {
		return NewField(base, NewFieldName(name, false))
	}
	dItems := func() Expression { return field(NewIdentifier("d"), "items") }
	mapping := map[Expression]Expression{
		NewConstant("sku"): field(dItems(), "sku"),
		NewConstant("y"):   field(field(dItems(), "tags"), "y"),
	}
	item := value.NewValue(map[string]interface{}{
		"d": map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{
					"sku": "A1",
					"tags": []interface{}{
						map[string]interface{}{"x": "x"},
						map[string]interface{}{"y": []interface{}{"y1", "y2"}},
					},
				},
				map[string]interface{}{
					"sku": "B1",
					"tags": []interface{}{
						map[string]interface{}{"x": "x1"},
						map[string]interface{}{"y": []interface{}{"y11", "y12"}},
					},
				},
			},
		},
	})
	expected := []interface{}{
		map[string]interface{}{"sku": "A1"},
		map[string]interface{}{"sku": "A1", "y": "y1"},
		map[string]interface{}{"sku": "A1", "y": "y2"},
		map[string]interface{}{"sku": "B1"},
		map[string]interface{}{"sku": "B1", "y": "y11"},
		map[string]interface{}{"sku": "B1", "y": "y12"},
	}
	testMultiKeyObjectsPath_eval(mapping, item, expected, t)
}

func TestMultiKeyObjects_twoUnrelatedArraysUnderSharedNonArrayBase(t *testing.T) {

	/* {d.item.sku, d.item.tags, d.item.names}: tags and names are two
	   unrelated array fields sharing the base d.item, which is itself
	   NOT an array -- so, unlike the items.sku/items.tags correlation
	   above, there's no per-position zip to do; sku, tags, and names
	   all combine via an ordinary cartesian product */
	field := func(base Expression, name string) Expression {
		return NewField(base, NewFieldName(name, false))
	}
	dItem := func() Expression { return field(NewIdentifier("d"), "item") }
	mapping := map[Expression]Expression{
		NewConstant("sku"):   field(dItem(), "sku"),
		NewConstant("tags"):  field(dItem(), "tags"),
		NewConstant("names"): field(dItem(), "names"),
	}
	item := value.NewValue(map[string]interface{}{
		"d": map[string]interface{}{
			"item": map[string]interface{}{
				"sku":   "A1",
				"tags":  []interface{}{"x", "y"},
				"names": []interface{}{"n1", "n2"},
			},
		},
	})
	expected := []interface{}{
		map[string]interface{}{"sku": "A1", "tags": "x", "names": "n1"},
		map[string]interface{}{"sku": "A1", "tags": "x", "names": "n2"},
		map[string]interface{}{"sku": "A1", "tags": "y", "names": "n1"},
		map[string]interface{}{"sku": "A1", "tags": "y", "names": "n2"},
	}
	testMultiKeyObjectsPath_eval(mapping, item, expected, t)
}

func TestMultiKeyObjects_ordinaryFieldDropsEmptyOccurrencesAcrossArray(t *testing.T) {

	/* {d.items.tags.y}: "tags" is walked implicitly (array), and "y"
	   (the terminal step) is itself expanded when it's an array,
	   matching how a single-step terminal array field (e.g. items.tags
	   alone) already expands. tags elements lacking "y" contribute
	   empty result objects, which are dropped from the final array
	   rather than surfacing as spurious {} entries */
	field := func(base Expression, name string) Expression {
		return NewField(base, NewFieldName(name, false))
	}
	tagsPath := func() Expression { return field(field(NewIdentifier("d"), "items"), "tags") }
	mapping := map[Expression]Expression{
		NewConstant("y"): field(tagsPath(), "y"),
	}
	item := value.NewValue(map[string]interface{}{
		"d": map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{
					"sku": "A1",
					"tags": []interface{}{
						map[string]interface{}{"x": "x"},
						map[string]interface{}{"y": []interface{}{"y1", "y2"}},
					},
				},
				map[string]interface{}{
					"sku": "B1",
					"tags": []interface{}{
						map[string]interface{}{"x": "x1"},
						map[string]interface{}{"y": []interface{}{"y11", "y12"}},
					},
				},
			},
		},
	})
	expected := []interface{}{
		map[string]interface{}{"y": "y1"},
		map[string]interface{}{"y": "y2"},
		map[string]interface{}{"y": "y11"},
		map[string]interface{}{"y": "y12"},
	}
	testMultiKeyObjectsPath_eval(mapping, item, expected, t)
}
