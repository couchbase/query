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
	"strings"
	"text/template"
	"time"

	"github.com/couchbase/query/logging"
)

type RandomRange struct {
	min uint
	max uint
}

func (this *RandomRange) isZero() bool {
	return this.max == 0
}

func (this *RandomRange) get() uint {
	if this.max == 0 {
		return 0
	}
	if this.min == this.max {
		return this.min
	}
	return uint(rand.Intn(int(this.max-this.min))) + this.min
}

func NewRandomRange(m map[string]interface{}, def uint) *RandomRange {
	var minOK bool
	var maxOK bool
	var f float64
	rr := &RandomRange{}
	if f, minOK = m["min"].(float64); minOK && f > 0 {
		rr.min = uint(f)
	}
	if f, maxOK = m["max"].(float64); maxOK && f > 0 {
		rr.max = uint(f)
	} else {
		if minOK {
			rr.max = rr.min
		} else {
			rr.min = def
			rr.max = def
		}
	}
	// normalise
	if rr.min > rr.max {
		rr.min, rr.max = rr.max, rr.min
	}
	return rr
}

func NewFixedRange(val uint) *RandomRange {
	return &RandomRange{min: val, max: val}
}

// ---------------------------------------------------------------------------------------------------------------------------------

type Database struct {
	purge           bool
	forceCreate     bool
	keyspaces       []*Keyspace
	bucketConfig    map[string]map[string]interface{}
	queryConfig     map[string]interface{}
	updateStats     bool
	randomKeyspaces *RandomRange
	rkSchemaDocs    *RandomRange
	rkSchemas       *RandomRange
	rkSchemaFields  *RandomRange
	awrConfig       map[string]interface{} // Settings to be updated in system:awr.
	awrKeyspace     *Keyspace              // The keyspace where AWR data will be stored.
	testStartTime   time.Time              // The time that the iteration started executing the test queries.
}

func NewDatabase(i interface{}) (*Database, error) {
	database, ok := i.(map[string]interface{})
	if !ok {
		logging.Fatalf("Database element is not an object")
		return nil, os.ErrNotExist
	}
	db := &Database{bucketConfig: make(map[string]map[string]interface{})}

	v, ok := database["purge"]
	if ok {
		if db.purge, ok = v.(bool); !ok {
			logging.Fatalf("\"purge\" is not a boolean.")
			return nil, os.ErrNotExist
		}
	}

	v, ok = database["force_create"]
	if ok {
		if db.forceCreate, ok = v.(bool); !ok {
			logging.Fatalf("\"force_create\" is not a boolean.")
			return nil, os.ErrNotExist
		}
	}

	v, ok = database["random_keyspaces"]
	if ok {
		if m, ok := v.(map[string]interface{}); !ok {
			logging.Fatalf("\"random_keyspaces\" is not an object.")
			return nil, os.ErrNotExist
		} else {
			db.randomKeyspaces = NewRandomRange(m, 0)
			if m, ok := m["size"].(map[string]interface{}); ok {
				db.rkSchemaDocs = NewRandomRange(m, 100)
			} else {
				db.rkSchemaDocs = NewFixedRange(100)
			}
			if m, ok := m["schemas"].(map[string]interface{}); ok {
				db.rkSchemas = NewRandomRange(m, 1)
			} else {
				db.rkSchemas = NewFixedRange(1)
			}
			if m, ok := m["fields"].(map[string]interface{}); ok {
				db.rkSchemaFields = NewRandomRange(m, 10)
			} else {
				db.rkSchemaFields = NewFixedRange(10)
			}
			if db.randomKeyspaces.isZero() || db.rkSchemaDocs.isZero() || db.rkSchemas.isZero() || db.rkSchemaFields.isZero() {
				db.randomKeyspaces = nil
				logging.Debugf("No random keyspaces will be generated.")
			}
		}
	}

	v, ok = database["config"]
	if ok {
		if m, ok := v.(map[string]interface{}); ok {
			db.queryConfig = make(map[string]interface{}, len(m))
			for k, v := range m {
				if k[0] != '#' {
					db.queryConfig[k] = v
				}
			}
		} else {
			logging.Warnf("Database \"config\" is not an object.")
		}
	}

	v, ok = database["update_statistics"]
	if ok {
		if db.updateStats, ok = v.(bool); !ok {
			logging.Fatalf("\"update_statistics\" is not a boolean.")
			return nil, os.ErrNotExist
		}
	}

	v, ok = database["awr"]
	if ok {
		if m, ok := v.(map[string]interface{}); ok {
			db.awrConfig = m

			// Configure the keyspace location for AWR data
			if path, ok := m["location"].(string); ok {
				awrKs, err := newKeyspace(path)
				if err != nil {
					logging.Errorf("AWR: Failed to create keyspace with error: %v", err)
					// Do not return an error. Do not fail the test for now.
				} else {
					db.awrKeyspace = awrKs
					logging.Infof("AWR: Keyspace location set to: %s", path)
				}
			} else {
				logging.Errorf("AWR: `location` setting is not a string.")
				// Do not return an error. Do not fail the test for now.
			}

			logging.Debugf("AWR: Settings configuration: %v", db.awrConfig)
		} else {
			logging.Errorf("AWR: Configuration specified is not an object.")
		}
	} else {
		logging.Warnf("AWR: Not configured for the iteration.")
	}

	keyspaces, ok := database["keyspaces"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("No keyspaces found in the configuration.")
	}

	for i := range keyspaces {
		keyspace, ok := keyspaces[i].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("Keyspaces element %d is not an object (%T).", i, keyspaces[i])
		}

		ks, bc, err := NewKeyspace(keyspace)
		if err != nil {
			return nil, fmt.Errorf("\"keyspaces\"[%d]: %v", i, err)
		}
		if !db.addKeyspace(ks) {
			return nil, fmt.Errorf("Duplicate keyspace: %s", ks.name)
		}
		db.addBucketConfig(ks.bucket(), bc)
	}
	return db, nil
}

// ---------------------------------------------------------------------------------------------------------------------------------

func (this *Database) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"purge":     this.purge,
		"force":     this.forceCreate,
		"keyspaces": this.keyspaces,
	}
	return json.Marshal(m)
}

func (this *Database) addRandomKeyspaces() error {
	if this.randomKeyspaces == nil {
		return nil
	}
	num := this.randomKeyspaces.get()
	for rks := uint(0); rks < num; rks++ {
		var name string
		if rand.Intn(7) == 3 || len(this.keyspaces) == 0 {
			name = fmt.Sprintf("rks%d", rks)
		} else {
			bn := rand.Intn(len(this.keyspaces))
			b := this.keyspaces[bn].bucket()
			var scope string
			switch rand.Intn(10) {
			case 0:
				scope = "scope0"
			case 3:
				scope = "a_moderately_long_scope_name"
			case 5:
				scope = "s5"
			default:
				scope = "_default"
			}
			name = fmt.Sprintf("%s.%s.rks%d", b, scope, rks)
		}
		ks, err := NewRandomKeyspace(name, this.rkSchemaDocs, this.rkSchemas, this.rkSchemaFields)
		if err != nil {
			return err
		}
		if !this.addKeyspace(ks) {
			logging.Debugf("'%s' is a duplicate name", ks.name)
			ks.name = fmt.Sprintf("%s_%d", ks.name, nextSerial())
			if !this.addKeyspace(ks) {
				logging.Errorf("Failed to add keyspace '%s': duplicate name", ks.name)
			}
		}
	}
	this.randomKeyspaces = nil
	return nil
}

func (this *Database) keyspaceByName(name string) *Keyspace {
	for i := range this.keyspaces {
		if this.keyspaces[i].Is(name) {
			return this.keyspaces[i]
		}
	}
	return nil
}

func (this *Database) addKeyspace(ks *Keyspace) bool {
	if this.keyspaceByName(ks.name) != nil {
		return false
	}
	this.keyspaces = append(this.keyspaces, ks)
	return true
}

func (this *Database) addJoins() error {
	// update all existing joins to point to the corresponding keyspace
	rfn := func(name string) *Keyspace {
		for i := range this.keyspaces {
			if this.keyspaces[i].name == name {
				return this.keyspaces[i]
			}
		}
		return nil
	}
	for i := range this.keyspaces {
		if err := this.keyspaces[i].resolveJoins(rfn); err != nil {
			return err
		}
	}

	// for every keyspace, pick a random number of others to join to
	for i := range this.keyspaces {
		others := getOthers(i, len(this.keyspaces))
		rand.Shuffle(len(others), func(i int, j int) {
			others[i], others[j] = others[j], others[i]
		})
		others = others[:rand.Intn(len(others))]
		for _, o := range others {
			// if already joined this is a no-op
			this.keyspaces[i].join(this.keyspaces[o])
		}
	}

	return nil
}

// generates an array of numbers from 0 to "num" omitting "skip"
func getOthers(skip int, num int) []int {
	res := make([]int, 0, num-1)
	for i := 0; i < num; i++ {
		if i != skip {
			res = append(res, i)
		}
	}
	return res
}

func (this *Database) addBucketConfig(bucket string, config map[string]interface{}) {
	m, ok := this.bucketConfig[bucket]
	if !ok {
		this.bucketConfig[bucket] = config
	} else {
		for k, v := range config {
			m[k] = v
		}
	}
}

func (this *Database) getBucketConfig(bucket string) (map[string]interface{}, bool) {
	m, ok := this.bucketConfig[bucket]
	if !ok {
		return make(map[string]interface{}, 2), false
	}
	res := make(map[string]interface{}, 2+len(m))
	for k, v := range m {
		res[k] = v
	}
	return res, true
}

// create the buckets, scopes & collections
func (this *Database) create() error {
	if !checkWait("http://localhost:8091/pools/default", "Waiting for instance prior to collections configuration...") {
		return fmt.Errorf("Unable to create/configure keyspaces.")
	}

	// configure the Query node at this point since we know it is up and running by this point
	if len(this.queryConfig) > 0 {
		_, _, err := doQueryPost("/admin/settings", this.queryConfig, false)
		if err != nil {
			logging.Warnf("Failed to configure query node: %v", err)
		}
	}

	if this.purge {
		if err := purgeKeyspaces(); err != nil {
			logging.Warnf("Failed to purge keyspaces: %v", err)
		}
	}

	logging.Infof("Attempting to create specified keyspaces.")

	n := 0
	created := make(map[string]bool)

	allKeyspaces := make([]*Keyspace, 0, len(this.keyspaces)+1)
	allKeyspaces = append(allKeyspaces, this.keyspaces...)
	if this.awrKeyspace != nil {
		allKeyspaces = append(allKeyspaces, this.awrKeyspace)
	}

	for _, ks := range allKeyspaces {
		config, customConfig := this.getBucketConfig(ks.bucket())
		if _, ok := config["ramQuota"]; !ok {
			if v, ok := config["storageBackend"]; ok && v == "magma" {
				config["ramQuota"] = 1024
			} else {
				config["ramQuota"] = 100
			}
		}
		for {
			err := createBucket(ks.bucket(), config)
			if err == nil {
				logging.Infof("Created bucket `%s`.", ks.bucket())
				created[ks.bucket()] = true
				break
			} else if err == os.ErrExist {
				if _, ok := created[ks.bucket()]; ok {
					break
				}
				if DB.forceCreate {
					logging.Infof("Dropping bucket `%s`.", ks.bucket())
					err := dropBucket(ks.bucket())
					if err != nil {
						if err == os.ErrNotExist {
							break
						}
						logging.Fatalf("Failed to drop bucket `%s`: %v", ks.bucket(), err)
						return err
					}
					continue
				}
				logging.Infof("Bucket `%s` already exists.", ks.bucket())
				if customConfig {
					logging.Debugf("Altering bucket '%s'.", ks.bucket())
					if err = alterBucket(ks.bucket(), config); err != nil {
						logging.Warnf("Failed to apply custom configuration to the existing bucket `%s`: %v", ks.bucket(), err)
					}
				}
				break
			} else {
				logging.Fatalf("Failed to create bucket `%s`: %v", ks.bucket(), err)
				return err
			}
		}
		if scope, ok := ks.scope(); ok {
			err := createScope(ks.bucket(), scope)
			if err == nil {
				logging.Infof("Created scope `%s` in bucket `%s`.", scope, ks.bucket())
			} else if err == os.ErrExist {
				logging.Infof("Scope `%s` already exists in bucket `%s`.", scope, ks.bucket())
			} else {
				logging.Fatalf("Failed to create scope `%s` in bucket `%s`: %v", scope, ks.bucket(), err)
				return err
			}
			collection, _ := ks.collection()
			err = createCollection(ks.bucket(), scope, collection)
			if err == nil {
				logging.Infof("Created collection `%s`.`%s` in bucket `%s`.", scope, collection, ks.bucket())
			} else if err == os.ErrExist {
				logging.Infof("Collection `%s`.`%s` already exists in bucket `%s`.", scope, collection, ks.bucket())
			} else {
				logging.Fatalf("Failed to create collection `%s`.`%s` in bucket `%s`: %v", scope, collection, ks.bucket(), err)
				return err
			}
		}
		if err := cleanupKeyspace(ks.name); err != nil {
			if !strings.Contains(err.Error(), "not found in CB datastore") {
				logging.Warnf("Failed to cleanup '%s': %v", ks.name, err)
			}
		}
		for _, idx := range ks.indexes {
			var stmt string
			iname := fmt.Sprintf("idx%d", nextSerial())
			if len(idx.defn) > 2 {
				stmt = fmt.Sprintf("CREATE INDEX %s ON %s%s", iname, ks.name, idx.defn)
			} else {
				stmt = fmt.Sprintf("CREATE PRIMARY INDEX %s ON %s", iname, ks.name)
			}
			if err := executeSQLWithoutResults(stmt, nil, false); err != nil {
				logging.Errorf("Failed to create index: %v", err)
				return err
			}
			ks.indexNames = append(ks.indexNames, iname)
		}
		n++
	}
	if n == 0 {
		logging.Fatalf("No valid keyspaces in the configuration.")
		return os.ErrNotExist
	} else {
		logging.Infof("%d keyspace(s) found or created.", n)
	}
	return nil
}

func (this *Database) populate() error {
	for i := range this.keyspaces {
		if err := this.keyspaces[i].populate(); err != nil {
			return err
		}
	}
	if this.updateStats {
		logging.Infof("Updating statistics.")
		for _, ks := range this.keyspaces {
			if len(ks.indexNames) == 0 {
				continue
			}
			var sb strings.Builder
			sb.WriteString("UPDATE STATISTICS FOR ")
			sb.WriteString(ks.name)
			if len(ks.parts) == 4 {
				sb.WriteString(" INDEX ALL")
			} else {
				sb.WriteString(" INDEX(")
				for i, iname := range ks.indexNames {
					if i > 0 {
						sb.WriteRune(',')
					}
					sb.WriteString(iname)
				}
				sb.WriteRune(')')
			}
			if err := executeSQLWithoutResults(sb.String(), nil, false); err != nil {
				logging.Errorf("Failed to update statistics for %s: %v", ks.name, err)
				return err
			}
		}
	}

	if this.awrKeyspace != nil {
		var sb strings.Builder
		sb.WriteString("UPDATE system:awr SET ")

		i := 0
		for k, v := range this.awrConfig {
			switch v.(type) {
			case string:
				v = fmt.Sprintf("'%s'", v)
			}

			if i > 0 {
				sb.WriteRune(',')
			}

			i++
			sb.WriteString(fmt.Sprintf("%s = %v", k, v))
		}

		if err := executeSQLWithoutResults(sb.String(), nil, false); err != nil {
			logging.Errorf("AWR: Failed to update system:awr with error: %v", err)
			// Do not return an error. Do not fail the test for now.
		}
	}
	return nil
}

func (this *Database) generateQueries(numQueries uint) {
	if numQueries == 0 {
		return
	}
	for i := uint(0); i < numQueries; i++ {
		ksn := rand.Intn(len(this.keyspaces))
		njn := rand.Intn(5)
		Queries = append(Queries, NewQuery(this.keyspaces[ksn], njn))
	}
}

func (this *Database) generateQueriesFromTemplates(templates []*Template) {
	var funcMap = template.FuncMap{
		"JoinStrings": strings.Join,
		"GetJoinOn": func(ks1 string, a1 string, ks2 string, a2 string) string {
			ks1p := this.keyspaceByName(ks1)
			ks2p := this.keyspaceByName(ks2)
			if ks1p == nil || ks2p == nil {
				return ""
			}
			return ks1p.GetJoin(a1, ks2p, a2)
		},
		"RandomFields": func(ks string, num int) []string {
			ksp := this.keyspaceByName(ks)
			if ksp == nil {
				return nil
			}
			return ksp.RandomFieldNames(num)
		},
		"RandomFilter": func(ks string) string {
			ksp := this.keyspaceByName(ks)
			if ksp == nil {
				return ""
			}
			return ksp.RandomFilter()
		},
		"GetValue": func(ks string, field string) string {
			ksp := this.keyspaceByName(ks)
			if ksp == nil {
				return ""
			}
			return ksp.FieldValue(field)
		},
	}
	n := 0
	data := &TemplateData{}
	data.Keyspaces = make([]string, len(this.keyspaces))
	for i := range this.keyspaces {
		data.Keyspaces[i] = this.keyspaces[i].name
	}
	data.RandomKeyspaces = make([]string, len(this.keyspaces))
	copy(data.RandomKeyspaces, data.Keyspaces)
	sb := &strings.Builder{}

	for _, tpl := range templates {
		for i := 0; i < tpl.iterations; i++ {
			data.Iteration = i
			rand.Shuffle(len(data.RandomKeyspaces), func(i int, j int) {
				data.RandomKeyspaces[i], data.RandomKeyspaces[j] = data.RandomKeyspaces[j], data.RandomKeyspaces[i]
			})
			sb.Reset()
			if err := tpl.tpl.Funcs(funcMap).Execute(sb, data); err != nil {
				logging.Errorf("Error generating query from (iteration %d): %v", i, err)
			} else {
				sql := strings.TrimSpace(sb.String())
				if sql != "" {
					Queries = append(Queries, &Query{sql: sql})
					n++
				}
			}
		}
	}
	logging.Infof("%d queries generated from templates.", n)
}
