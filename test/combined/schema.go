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
	"io"
	"math/rand"
	"strings"

	"github.com/couchbase/query/logging"
)

type Schema struct {
	typ    string
	count  uint
	fields []Field
}

func NewSchema(i interface{}) (*Schema, error) {
	logging.Tracef("%v", i)
	elem, ok := i.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Schema is not an object.")
	}

	schema := &Schema{}
	v, ok := elem["count"]
	if !ok {
		return nil, fmt.Errorf("Schema missing \"count\" field.")
	}
	if m, ok := v.(map[string]interface{}); !ok {
		return nil, fmt.Errorf("Schema \"count\" field is not valid.")
	} else {
		schema.count = NewRandomRange(m, 0).get()
	}

	schema.typ, ok = elem["type"].(string)
	if !ok {
		schema.typ = fmt.Sprintf("type_%d", nextSerial())
	}

	fields, ok := elem["fields"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("Schema \"fields\" field is missing or not valid.")
	}

	for k := range fields {
		f, err := NewField(fields[k])
		if err != nil {
			return nil, fmt.Errorf("\"fields\"[%d]: %v", k, err)
		}
		schema.fields = append(schema.fields, f)
	}
	return schema, nil
}

func NewRandomSchema(typ string, docs *RandomRange, fields *RandomRange) (*Schema, error) {
	schema := &Schema{typ: typ}
	schema.count = docs.get()
	for n := fields.get(); n > 0; n-- {
		schema.fields = append(schema.fields, NewRandomField(fmt.Sprintf("gf%d", nextSerial())))
	}
	return schema, nil
}

// ---------------------------------------------------------------------------------------------------------------------------------

func (this *Schema) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"type":   this.typ,
		"count":  this.count,
		"fields": this.fields,
	}
	return json.Marshal(m)
}

func (this *Schema) generateData(w io.Writer) error {
	doc := make(map[string]interface{}, len(this.fields)+1)
	logging.Debugf("Generating %v documents.", this.count)
	for i := uint(0); i < this.count; i++ {
		clear(doc)
		doc["type"] = this.typ
		for j := range this.fields {
			if m := this.fields[j].GenerateValue(); m != nil {
				for k, v := range m {
					doc[k] = v
				}
			}
		}
		b, err := json.Marshal(doc)
		if err != nil {
			return err
		}
		if len(b) < _DOC_SIZE_LIMIT {
			if n, err := w.Write(b); err != nil {
				return err
			} else if n != len(b) {
				logging.Debugf("Wrote only %d of %d bytes.", n, len(b))
			}
			if n, err := w.Write([]byte{'\n'}); err != nil {
				return err
			} else if n != 1 {
				logging.Debugf("Failed to write newline.")
			}
		} else {
			logging.Debugf("Discarding %d byte generated document.", len(b))
		}
	}
	return nil
}

func (this *Schema) randomProjection(alias string) []string {
	return this.randomFields(alias, len(this.fields), true)
}

func (this *Schema) randomFields(alias string, numFields int, allowWildcards bool) []string {
	n := rand.Intn(numFields)
	if n == 0 {
		return nil
	}
	res := make([]string, 0, n)
	for ; n > 0; n-- {
		fn := rand.Intn(len(this.fields)) // can project the same field multiple times
		if len(alias) > 0 {
			res = append(res, fmt.Sprintf("%s.%s", alias, this.fields[fn].GenerateProjection(allowWildcards)))
		} else {
			res = append(res, this.fields[fn].GenerateProjection(allowWildcards))
		}
	}
	return res
}

func (this *Schema) randomAggs(alias string) []string {
	if rand.Intn(5) != 4 {
		return nil
	}
	var res []string
	for n := rand.Intn(5); n > 0; n-- {
		var agg string
		switch rand.Intn(5) {
		case 0:
			res = append(res, "COUNT(1)")
			continue
		case 1:
			agg = "AVG"
		case 2:
			agg = "SUM"
		case 3:
			agg = "MIN"
		case 4:
			agg = "MAX"
		}
		fn := rand.Intn(len(this.fields))
		res = append(res, fmt.Sprintf("%s(%s.%s)", agg, alias, this.fields[fn].GenerateProjection(false)))
	}
	return res
}

func (this *Schema) randomFilter(alias string) []string {
	n := rand.Intn(len(this.fields)) / 5
	if n == 0 {
		return nil
	}
	res := make([]string, 0, n)
	for ; n > 0; n-- {
		var filter string
		if rand.Intn(10) == 7 {
			for c := rand.Intn(3) + 2; c > 0; c-- {
				filter += strings.Join(this.randomFilter(alias), "")
			}
			if len(filter) > 5 {
				filter = " AND ( " + strings.ReplaceAll(filter[5:], " AND ", " OR ") + ")"
			} else {
				continue
			}
		} else {
			fn := rand.Intn(len(this.fields))
			filter = " AND " + this.fields[fn].GenerateFilter(fmt.Sprintf("%s.%s", alias, this.fields[fn].Name()))
		}
		res = append(res, filter)
	}
	return res
}

func (this *Schema) randomOrder(alias string) []string {
	if rand.Intn(3) != 2 {
		return nil
	}
	n := len(this.fields)
	if n > 3 {
		n = 3
	}
	return this.randomFields(alias, n, false)
}

func (this *Schema) randomUnnest(alias string) []string {
	list := make([]string, 0, len(this.fields))
	for i := range this.fields {
		if this.fields[i].Type() == _FT_ARRAY {
			list = append(list, fmt.Sprintf("%s.%s AS un_%d", alias, this.fields[i].Name(), nextSerial()))
		}
	}
	if len(list) == 0 {
		return nil
	}
	n := rand.Intn(len(list))
	if n == 0 {
		return nil
	}
	rand.Shuffle(len(list), func(i int, j int) {
		list[i], list[j] = list[j], list[i]
	})
	return list[:n]
}
