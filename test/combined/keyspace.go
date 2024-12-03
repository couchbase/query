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
	"os"

	"github.com/couchbase/query/logging"
)

type Keyspace struct {
	name    string
	schemas []*Schema
	indexes []*Index
	joins   []*Join

	parts      []string // parsed name parts
	indexNames []string // generated index names
}

func newKeyspace(name string) (*Keyspace, error) {
	ks := &Keyspace{name: name}
	ks.parts = parsePath(ks.name)
	if len(ks.parts) != 2 && len(ks.parts) != 4 {
		return nil, fmt.Errorf("Keyspaces name is not valid: %v.", name)
	} else if ks.parts[0] != "" && ks.parts[0] != "default" {
		logging.Warnf("\"%s\" specifies an invalid namespace. Ignoring the namespace.", name)
		ks.parts[0] = ""
	}
	return ks, nil
}

func NewKeyspace(i interface{}) (*Keyspace, map[string]interface{}, error) {
	logging.Tracef("%v", i)
	keyspace, ok := i.(map[string]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("Keyspace is not an object.")
	}

	name, ok := keyspace["keyspace"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("Keyspace lacks valid \"name\" field.")
	}
	ks, err := newKeyspace(name)
	if err != nil {
		return nil, nil, err
	}

	schemas, ok := keyspace["schemas"].([]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("Keyspace %s lacks valid \"schemas\" field.", ks.name)
	}

	for j := range schemas {
		s, err := NewSchema(schemas[j])
		if err != nil {
			return nil, nil, fmt.Errorf("%s \"schemas\"[%d]: %v", ks.name, j, err)
		}
		ks.schemas = append(ks.schemas, s)
	}

	indexes, ok := keyspace["indexes"].([]interface{})
	if ok {
		for j := range indexes {
			idx, err := NewIndex(indexes[j])
			if err != nil {
				return nil, nil, fmt.Errorf("Keyspace %s \"indexes\" element %d is not valid: %v", ks.name, j, err)
			}
			ks.indexes = append(ks.indexes, idx)
		}
	}

	joins, ok := keyspace["joins"].([]interface{})
	if ok {
		for j := range joins {
			join, err := NewJoin(joins[j])
			if err != nil {
				return nil, nil, fmt.Errorf("Keyspace %s \"joins\" element %d is not valid: %v", ks.name, j, err)
			}
			ks.joins = append(ks.joins, join)
		}
	}

	var bucketConfig map[string]interface{}
	if c, ok := keyspace["bucket-config"]; ok {
		if bucketConfig, ok = c.(map[string]interface{}); !ok {
			logging.Warnf("Invalid \"bucket-config\" element for keyspace %s.", ks.name)
		}
	}

	return ks, bucketConfig, nil
}

func NewRandomKeyspace(name string, docs *RandomRange, schemas *RandomRange, fields *RandomRange) (*Keyspace, error) {
	ks, err := newKeyspace(name)
	if err != nil {
		return nil, err
	}
	for n := schemas.get(); n > 0; n-- {
		schema, err := NewRandomSchema(fmt.Sprintf("typ_%d", n), docs, fields)
		if err != nil {
			return nil, fmt.Errorf("Error generating \"%s\": %v", name, err)
		}
		ks.schemas = append(ks.schemas, schema)
	}
	// indexes will be generated when joins to these keyspaces are
	return ks, nil
}

// ---------------------------------------------------------------------------------------------------------------------------------

func (this *Keyspace) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"name":    this.name,
		"schemas": this.schemas,
		"indexes": this.indexes,
		"joins":   this.joins,
	}
	if len(this.parts) > 0 {
		m["parts"] = this.parts
	}
	return json.Marshal(m)
}

func (this *Keyspace) Is(name string) bool {
	parts := parsePath(name)
	if parts[0] != "" && parts[0] != "default" {
		parts[0] = ""
	}
	if len(this.parts) != len(parts) {
		return false
	}
	if parts[0] == "" && this.parts[0] == "default" {
		parts[0] = "default"
	} else if parts[0] == "default" && this.parts[0] == "" {
		parts[0] = ""
	}
	for i := range this.parts {
		if this.parts[i] != parts[i] {
			return false
		}
	}
	return true
}

func (this *Keyspace) resolveJoins(fn func(string) *Keyspace) error {
	for i := range this.joins {
		if this.joins[i].right == nil {
			this.joins[i].right = fn(this.joins[i].rightName)
		}
		if this.joins[i].right == nil {
			return fmt.Errorf("Keyspace \"%s\" not found.", this.joins[i].rightName)
		}
		if this.joins[i].ridx != "" {
			this.joins[i].right.addIndex(this.joins[i].ridx)
		}
	}
	return nil
}

func (this *Keyspace) join(other *Keyspace) *Join {
	for i := range this.joins {
		if this.joins[i].right == other {
			return this.joins[i]
		}
	}
	join := NewJoinKeyspaces(this, other)
	this.joins = append(this.joins, join)
	return join
}

func (this *Keyspace) addIndex(key string) {
	for i := range this.indexes {
		if this.indexes[i].sameAs(key) {
			return
		}
	}
	this.indexes = append(this.indexes, NewIndexFromKey(key))
}

func (this *Keyspace) populate() error {
	f, err := os.CreateTemp(os.TempDir(), fmt.Sprintf("import_data_%s_*", this.name))
	if err != nil {
		return err
	}
	f.Chmod(0666)
	for i := range this.schemas {
		if err := this.schemas[i].generateData(f); err != nil {
			return fmt.Errorf("Error generating data for %s, schema %d: %v", this.name, i, err)
		}
	}
	f.Close()
	DataFiles = append(DataFiles, f.Name())
	err = importData(this.name, f.Name())
	return err
}

func (this *Keyspace) bucket() string {
	return this.parts[1]
}

func (this *Keyspace) scope() (string, bool) {
	if len(this.parts) != 4 {
		return "", false
	}
	return this.parts[2], true
}

func (this *Keyspace) collection() (string, bool) {
	if len(this.parts) != 4 {
		return "", false
	}
	return this.parts[3], true
}

func (this *Keyspace) GetJoin(alias string, other *Keyspace, oalias string) string {
	join := this.join(other)
	return join.onFilter(alias, oalias)
}

func (this *Keyspace) RandomFieldNames(num int) []string {
	sn := rand.Intn(len(this.schemas))
	return this.schemas[sn].randomFields("", num, false)
}

func (this *Keyspace) RandomFilter() string {
	schema := this.schemas[rand.Intn(len(this.schemas))]
	field := schema.fields[rand.Intn(len(schema.fields))]
	return field.GenerateFilter(field.Name())
}

func (this *Keyspace) FieldValue(name string) string {
	for i := range this.schemas {
		for j := range this.schemas[i].fields {
			if this.schemas[i].fields[j].Name() == name {
				if m := this.schemas[i].fields[j].GenerateValue(); m != nil {
					for _, v := range m {
						if v != nil {
							b, _ := json.Marshal(v)
							return string(b)
						}
						return ""
					}
				}
			}
		}
	}
	return ""
}
