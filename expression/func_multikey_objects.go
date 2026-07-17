//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
// MultiKeyObjects
//
///////////////////////////////////////////////////

/*
This represents the array function MULTIKEY_OBJECTS(expr). Denormalizing
expansion only happens when expr is written as an object construction
expression, e.g. MULTIKEY_OBJECTS({"a": a.b.c, "d": d.e}) -- the argument
must spell out the fields of interest. Each field's dotted path is
walked through arrays automatically -- a.b.c behaves as a[].b.c would
if b (or c) turned out to be an array -- without requiring the []
notation to be written explicitly.

Fields whose paths share a common prefix walk that shared prefix in
lock-step rather than independently: MULTIKEY_OBJECTS({"sku": items.sku,
"tag": items.tags}), where items is an array of {sku, tags} objects,
pairs each sku with the tags of *its own* item (multiplying out only
over that item's own tags array), rather than cross-producting every
sku against every tag as if they were unrelated. This mirrors how a
MongoDB compound multikey index refuses to index two array fields
independently (a "parallel arrays" error) -- here, instead of
rejecting it, same-array fields are simply correlated by position.
Fields whose paths do NOT share a common prefix (e.g. two genuinely
unrelated arrays) are still combined via an ordinary cartesian product,
so multiple, unrelated array-valued fields multiply out. A field (or
shared prefix) that evaluates to an empty array contributes no value
at all and is simply omitted from every result object -- it does not
suppress or affect any unrelated field. See multiKeyObjectsNode.

Any other argument -- e.g. a bare document reference, or a path such as
MULTIKEY_OBJECTS(items.sku) -- is NOT written as an object construction
expression, but is treated exactly as if it had been: MULTIKEY_OBJECTS(d)
is evaluated identically to MULTIKEY_OBJECTS({d}), i.e. MULTIKEY_OBJECTS({
"<alias of d>": d}), using d's own implicit alias (e.g. items.sku's
alias is "sku") as the sole result field's name, and run through the
very same field-grouping/denormalizing machinery
(groupMultiKeyObjectsFields / multiKeyObjectsFinalize) as a real
object-construct field -- there is no separate behaviour to keep
consistent with it. If d has no implicit alias at all (e.g. a literal,
or any other expression with no natural name), there is no field name
to construct with, so MULTIKEY_OBJECTS(d) is missing.
*/
type MultiKeyObjects struct {
	UnaryFunctionBase
}

func NewMultiKeyObjects(operand Expression) Function {
	rv := &MultiKeyObjects{}
	rv.Init("multikey_objects", operand)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *MultiKeyObjects) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *MultiKeyObjects) Type() value.Type { return value.ARRAY }

/*
Expansion only happens when the argument is written as an object
construction expression (e.g. { "a": a.b.c, "d": d.e }): its fields are
grouped by the common base each one's dotted path is built on (see
groupMultiKeyObjectsFields), and each group's shared path is walked once,
denormalizing the result (see evalMultiKeyObjectsNode / multiKeyObjectsFinalize).

Any other argument is treated as if it had been written as the
single-field object construction {"<alias>": expr} -- built as exactly
that one-entry mapping and run through the same
groupMultiKeyObjectsFields / multiKeyObjectsFinalize pipeline as an
object-construct argument, no separate logic -- provided expr has an
implicit alias to name that field with. Otherwise (no alias, e.g. a
literal) there is no field name to construct with, so the result is
missing.
*/
func (this *MultiKeyObjects) Evaluate(item value.Value, context Context) (value.Value, error) {
	mapping := multiKeyObjectsOperandMapping(this.operands[0])
	if mapping == nil {
		return value.MISSING_VALUE, nil
	}

	groups, isNull, err := groupMultiKeyObjectsFields(mapping, item, context)
	if err != nil {
		return nil, err
	} else if isNull {
		return value.NULL_VALUE, nil
	}

	rv, err := multiKeyObjectsFinalize(groups, item, context)
	if err != nil {
		return nil, err
	}
	return value.NewValue(rv), nil
}

/*
Returns the field-name-to-value-expression mapping that MULTIKEY_OBJECTS'
argument denormalizes: an object construction argument's own mapping
unchanged, or, for any other argument, the single-entry mapping {alias:
operand} built from operand's own implicit alias -- i.e. exactly the
mapping MULTIKEY_OBJECTS({operand}) would use. Returns nil if operand is
neither an object construction nor has an implicit alias to build that
single entry with.
*/
func multiKeyObjectsOperandMapping(operand Expression) map[Expression]Expression {
	if oc, ok := operand.(*ObjectConstruct); ok {
		return oc.Mapping()
	}

	alias := operand.Alias()
	if alias == "" {
		return nil
	}

	return map[Expression]Expression{NewConstant(alias): operand}
}

/*
Factory method pattern.
*/
func (this *MultiKeyObjects) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewMultiKeyObjects(operands[0])
	}
}

/*
Decomposes a field-access expression (a dotted path) into an opaque
"base" expression and the ordered sequence of literal field names
applied on top of it -- e.g. items.sku decomposes to (Identifier
"items", ["sku"]). Two fields whose paths decompose to an
EquivalentTo base can then be recognized as walking the same
underlying value and merged into one path trie by
groupMultiKeyObjectsFields, instead of being treated as unrelated.
Anything that is not itself a *Field with a statically known string
field name is left as an opaque base with no names -- it is evaluated
as a whole, exactly as MULTIKEY_OBJECTS' non-object-construct argument is.
*/
func decomposeMultiKeyObjectsPath(expr Expression) (Expression, []string) {
	var names []string
	cur := expr

	for {
		f, ok := cur.(*Field)
		if !ok {
			break
		}

		nameVal := f.Second().Value()
		if nameVal == nil || nameVal.Type() != value.STRING {
			break
		}

		names = append(names, nameVal.ToString())
		cur = f.First()
	}

	// names was built walking from the outermost field access down to base, i.e. in
	// reverse of the written path (e.g. items.sku.tags yields ["tags", "sku"] above);
	// reverse it here so it reads left-to-right, base-to-leaf, as ["sku", "tags"].
	for i, j := 0, len(names)-1; i < j; i, j = i+1, j-1 {
		names[i], names[j] = names[j], names[i]
	}

	return cur, names
}

/*
A node in the path trie built from a MULTIKEY_OBJECTS() object
construction argument's field expressions (see
groupMultiKeyObjectsFields). fieldNames holds the output field name(s)
whose path ends exactly at this node; children holds the subtrees
reached by a further field-name step. Two output fields whose paths
share a prefix -- e.g. items.sku and items.tags -- share the nodes for
that prefix, so the underlying array (items) is walked once, in
lock-step across both fields, rather than being walked independently
per field and combined via an unrelated cartesian product.
*/
type multiKeyObjectsNode struct {
	fieldNames []string
	children   map[string]*multiKeyObjectsNode
}

func newMultiKeyObjectsNode() *multiKeyObjectsNode {
	return &multiKeyObjectsNode{children: make(map[string]*multiKeyObjectsNode)}
}

func (this *multiKeyObjectsNode) insert(names []string, fieldName string) {
	node := this
	for _, n := range names {
		child, ok := node.children[n]
		if !ok {
			child = newMultiKeyObjectsNode()
			node.children[n] = child
		}
		node = child
	}
	node.fieldNames = append(node.fieldNames, fieldName)
}

/*
A group of MULTIKEY_OBJECTS() object-construct fields whose value
expressions share the same base expression (by EquivalentTo), together
with the path trie built from their remaining, per-field name
sequences.
*/
type multiKeyObjectsGroup struct {
	base Expression
	root *multiKeyObjectsNode
}

/*
Evaluates each of the object construct's field-name expressions and,
skipping missing/null names and bailing out (isNull) on a non-string
name exactly as plain object construction does, decomposes every
field's value expression via decomposeMultiKeyObjectsPath and groups them
by base expression, inserting each one's name sequence into that
base's shared path trie.
*/
func groupMultiKeyObjectsFields(mapping map[Expression]Expression, item value.Value, context Context) (
	groups []*multiKeyObjectsGroup, isNull bool, err error) {
	for nameExpr, valExpr := range mapping {
		n, err := nameExpr.Evaluate(item, context)
		if err != nil {
			return nil, false, err
		}

		if n.Type() == value.MISSING || n.Type() == value.NULL {
			continue
		} else if n.Type() != value.STRING {
			return nil, true, nil
		}

		base, names := decomposeMultiKeyObjectsPath(valExpr)

		var group *multiKeyObjectsGroup
		for _, g := range groups {
			if g.base.EquivalentTo(base) {
				group = g
				break
			}
		}
		if group == nil {
			group = &multiKeyObjectsGroup{base: base, root: newMultiKeyObjectsNode()}
			groups = append(groups, group)
		}

		group.root.insert(names, n.ToString())
	}

	return groups, false, nil
}

/*
Evaluates each group's base expression once and denormalizes it via
evalMultiKeyObjectsNode, then combines the groups' results into the final
list of result objects via an ordinary cartesian product across
groups -- since, by construction, different groups do NOT share an
underlying array, so there is nothing to correlate between them. A
group with no possibilities at all (e.g. its shared array is empty) is
simply skipped, omitting its field(s) from every result object rather
than suppressing unrelated groups.

A result object that, after all that, ends up with none of its fields
set at all -- e.g. its one field's path wasn't present for that
occurrence -- is dropped from the final array entirely, rather than
surfaced as a spurious empty object; a result object with at least one
field set is kept as-is, missing fields and all.
*/
func multiKeyObjectsFinalize(groups []*multiKeyObjectsGroup, item value.Value, context Context) ([]interface{}, error) {
	combos := []map[string]interface{}{make(map[string]interface{})}

	for _, g := range groups {
		baseVal, err := g.base.Evaluate(item, context)
		if err != nil {
			return nil, err
		}

		combos = multiKeyObjectsCombine(combos, evalMultiKeyObjectsNode(g.root, baseVal))
	}

	rv := make([]interface{}, 0, len(combos))
	for _, c := range combos {
		if len(c) == 0 {
			continue
		}
		rv = append(rv, c)
	}
	return rv, nil
}

/*
Evaluates a single path-trie node against a value, returning the list
of partial result objects (as string-keyed maps) contributed by this
node and everything below it.

A pure leaf (no children) contributes one result per element if v is
an array (its own, one-level expansion -- elements are used as-is, not
expanded any further) or, otherwise, a single result setting every
field name assigned to it to v (omitted if v is missing).

A node with children walks v's named field accesses (see
multiKeyObjectsAccessChildren); if v is an array, this happens once per
element, in lock-step across every child AND any field names assigned
directly to this node -- so a field colocated with a prefix of a
sibling field's path (e.g. "item": items alongside "sku": items.sku)
takes each element's own value, without being expanded any further
itself.
*/
func evalMultiKeyObjectsNode(node *multiKeyObjectsNode, v value.Value) []map[string]interface{} {
	if len(node.children) == 0 {
		return multiKeyObjectsLeafRows(node.fieldNames, v)
	}

	if v.Type() == value.ARRAY {
		elems := v.Actual().([]interface{})
		rv := make([]map[string]interface{}, 0, len(elems))
		for _, e := range elems {
			for _, row := range multiKeyObjectsAccessChildren(node, value.NewValue(e)) {
				for _, fn := range node.fieldNames {
					row[fn] = e
				}
				rv = append(rv, row)
			}
		}
		return rv
	}

	rows := multiKeyObjectsAccessChildren(node, v)
	for _, row := range rows {
		for _, fn := range node.fieldNames {
			if v.Type() != value.MISSING {
				row[fn] = v.Actual()
			}
		}
	}
	return rows
}

/*
Builds the leaf-node result rows for a multiKeyObjectsNode: if v is an
array, one row per element (one level of expansion, elements used
as-is); otherwise a single row, with every field name omitted if v is
missing.
*/
func multiKeyObjectsLeafRows(fieldNames []string, v value.Value) []map[string]interface{} {
	if v.Type() == value.ARRAY {
		elems := v.Actual().([]interface{})
		rv := make([]map[string]interface{}, 0, len(elems))
		for _, e := range elems {
			row := make(map[string]interface{}, len(fieldNames))
			for _, fn := range fieldNames {
				row[fn] = e
			}
			rv = append(rv, row)
		}
		return rv
	}

	row := make(map[string]interface{}, len(fieldNames))
	for _, fn := range fieldNames {
		if v.Type() != value.MISSING {
			row[fn] = v.Actual()
		}
	}
	return []map[string]interface{}{row}
}

/*
Applies, and recursively evaluates, every one of a node's named
children against a single instance of v (an object, or a scalar/
missing/null value that a field access cannot navigate any further
into -- see evalMultiKeyObjectsNode's *Field / NULL-vs-MISSING handling
elsewhere in this file), then combines the children's own result rows
via an ordinary cartesian product -- since, at a single node, different
named children are, by definition, different fields and have nothing
further to correlate against each other.
*/
func multiKeyObjectsAccessChildren(node *multiKeyObjectsNode, v value.Value) []map[string]interface{} {
	combos := []map[string]interface{}{make(map[string]interface{})}

	for name, child := range node.children {
		var cv value.Value
		switch v.Type() {
		case value.OBJECT:
			cv, _ = v.Field(name)
		case value.MISSING:
			cv = value.MISSING_VALUE
		default:
			cv = value.NULL_VALUE
		}

		combos = multiKeyObjectsCombine(combos, evalMultiKeyObjectsNode(child, cv))
	}

	return combos
}

/*
Combines two lists of partial result-object rows via a cartesian
product -- every combination of one row from each side, merged by
field name. An empty subRows list (no possibilities at all) makes the
whole combination contribute nothing further from that side, without
otherwise affecting combos -- i.e. its field(s) are simply omitted.
*/
func multiKeyObjectsCombine(combos []map[string]interface{}, subRows []map[string]interface{}) []map[string]interface{} {
	if len(subRows) == 0 {
		return combos
	}

	next := make([]map[string]interface{}, 0, len(combos)*len(subRows))
	for _, c := range combos {
		for _, s := range subRows {
			merged := make(map[string]interface{}, len(c)+len(s))
			for k, v := range c {
				merged[k] = v
			}
			for k, v := range s {
				merged[k] = v
			}
			next = append(next, merged)
		}
	}
	return next
}
