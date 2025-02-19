//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/couchbase/query/logging"
)

// these are intentionally uni-directional
type Join struct {
	rightName string
	right     *Keyspace
	on        string

	rightschema *Schema

	ridx string // only for joins loaded from config; the field name for the index in the right keyspace
}

func NewJoin(i interface{}) (*Join, error) {
	logging.Tracef("%v", i)
	join, ok := i.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Join definition is not an object.")
	}

	jn := &Join{}
	v, ok := join["keyspace"]
	if !ok {
		return nil, fmt.Errorf("Join definition missing \"keyspace\" field.")
	}
	jn.rightName, ok = v.(string)
	if !ok {
		return nil, fmt.Errorf("Join \"keyspace\" field is not a string.")
	}
	jn.on, _ = join["on"].(string)
	lf, _ := join["from"].(string)
	rf, _ := join["to"].(string)

	if jn.on == "" && (lf == "" || rf == "") {
		return nil, fmt.Errorf("Invalid join definition.")
	} else if jn.on != "" && (lf != "" || rf != "") {
		return nil, fmt.Errorf("Invalid join definition.")
	}
	if jn.on == "" {
		jn.on = fmt.Sprintf("${left}.%s = ${right}.%s", lf, rf)
	}
	jn.ridx = rf

	return jn, nil
}

// it is intentional that there is no checking that the join fields are "good" types for joining, just that theoretically at least
// they can join
func NewJoinKeyspaces(left *Keyspace, right *Keyspace) *Join {
	leftSchema := left.schemas[0]
	if len(left.schemas) > 0 {
		leftSchema = left.schemas[rand.Intn(len(left.schemas))]
	}
	rightSchema := right.schemas[0]
	if len(right.schemas) > 0 {
		rightSchema = right.schemas[rand.Intn(len(right.schemas))]
	}
	n := 1
	if rand.Intn(10) == 3 {
		n += rand.Intn(3)
	}
	var idxKeys []string
	var lfields []Field
	var rfields []Field
	var sb strings.Builder
	if n > 1 {
		sb.WriteRune('(')
	}
	for i := 0; i < n; i++ {
		if i > 0 {
			sb.WriteString(" AND ")
		}
		var lf Field
		if rand.Intn(10) == 9 { // use an existing field in the left schema by duplicating it in the right
			lf = leftSchema.fields[rand.Intn(len(leftSchema.fields))]
		} else {
			lf = NewRandomJoinField(fmt.Sprintf("jf%d", nextSerial()))
			lfields = append(lfields, lf)
		}
		rf := lf.Copy()
		rfields = append(rfields, rf)

		sb.WriteString("${left}.")
		sb.WriteString(lf.Name())
		sb.WriteString(" = ${right}.")
		sb.WriteString(rf.Name())
		idxKeys = append(idxKeys, rf.Name())
	}
	if n > 1 {
		sb.WriteRune(')')
	}
	if len(lfields) > 0 {
		leftSchema.fields = append(leftSchema.fields, lfields...)
	}
	if len(rfields) > 0 {
		rightSchema.fields = append(rightSchema.fields, rfields...)
	}

	j := &Join{rightName: right.name, right: right, rightschema: rightSchema}
	j.on = sb.String()
	if rand.Intn(30) == 17 {
		right.addIndex("")
	} else {
		right.addIndex(strings.Join(idxKeys, ","))
	}
	return j
}

// ---------------------------------------------------------------------------------------------------------------------------------

func (this *Join) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"right": this.rightName,
		"on":    this.on,
	}
	return json.Marshal(m)
}

func (this *Join) onFilter(lalias string, ralias string) string {
	on := strings.ReplaceAll(this.on, "${left}", lalias)
	return strings.ReplaceAll(on, "${right}", ralias)
}

func (this *Join) clause(lalias string, ralias string) string {
	return fmt.Sprintf(" JOIN %s AS %s ON %s", this.right.name, ralias, this.onFilter(lalias, ralias))
}
