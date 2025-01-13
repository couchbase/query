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
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/couchbase/query/logging"
)

type Query struct {
	sync.Mutex

	sql string

	// used when building
	projection    []string
	aggs          []string
	where         []string
	from          []string
	group         []string
	order         []string
	offset        string
	limit         string
	unnest        []string
	followedJoins map[*Join]bool
	aliasNum      int
	nextAliasNum  int

	// runtime stats
	executions  uint64
	failed      uint64 // failed to execute, not executed with errors
	lastFailure error
	elapsedMs   uint64
	results     uint64
	lastErrors  []int // keep the last 10 reported error numbers
}

func LoadQuery(file string) (*Query, error) {
	logging.Debugf("%s", file)
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	f.Close()
	return &Query{sql: strings.TrimSpace(string(b))}, nil
}

// loads all .sql files found under the directory and recurses into any sub-directories
// the content of each .sql file is expected to be a single SQL statement; no parsing/processing of the file content is undertaken
// and they will be submitted as-is
func LoadQueries(dir string) ([]*Query, error) {
	logging.Debugf("%s", dir)
	d, err := os.Open(dir)
	if err != nil {
		return nil, fmt.Errorf("Failed to open directory: %s - %v", dir, err)
	}
	var res []*Query
	for {
		ents, err := d.ReadDir(10)
		if err == nil {
			for i := range ents {
				if ents[i].IsDir() {
					if ents[i].Name() != "." && ents[i].Name() != ".." {
						if qrys, err := LoadQueries(path.Join(dir, ents[i].Name())); err != nil {
							return nil, err
						} else {
							res = append(res, qrys...)
						}
					}
				} else if strings.HasSuffix(ents[i].Name(), ".sql") {
					q, err := LoadQuery(path.Join(dir, ents[i].Name()))
					if err != nil {
						return nil, fmt.Errorf("Failed to load query from %s: %v", path.Join(dir, ents[i].Name()), err)
					} else {
						res = append(res, q)
					}
				}
			}
		}
		if err != nil || len(ents) < 10 {
			break
		}
	}
	return res, nil
}

func NewQuery(keyspace *Keyspace, joins int) *Query {
	qry := &Query{}
	qry.followedJoins = make(map[*Join]bool)
	qry.from = append(qry.from, fmt.Sprintf("%s AS %s", keyspace.name, qry.alias()))
	buildQuery(qry, keyspace, joins)
	qry.complete()
	return qry
}

func buildQuery(qry *Query, keyspace *Keyspace, joins int) int {
	qry.add(keyspace)
	if joins > 0 && len(keyspace.joins) > 0 {
		rem := getOthers(-1, len(keyspace.joins))
		rand.Shuffle(len(rem), func(i int, j int) {
			rem[i], rem[j] = rem[j], rem[i]
		})
		var join *Join
		for len(rem) > 0 {
			join = keyspace.joins[rem[len(rem)-1]]
			if _, ok := qry.followedJoins[join]; ok {
				rem = rem[:len(rem)-1]
			} else {
				qry.followedJoins[join] = true
				qry.addJoin(keyspace, join)
				keep := qry.aliasNum
				joins = buildQuery(qry, join.right, joins-1)
				if joins == 0 {
					break
				}
				qry.aliasNum = keep
			}
		}
	}
	return joins
}

// ---------------------------------------------------------------------------------------------------------------------------------

func (this *Query) add(keyspace *Keyspace) {
	alias := this.alias()
	schema := keyspace.schemas[rand.Intn(len(keyspace.schemas))]
	this.projection = append(this.projection, schema.randomProjection(alias)...)
	this.aggs = append(this.aggs, schema.randomAggs(alias)...)
	this.where = append(this.where, schema.randomFilter(alias)...)
	this.order = append(this.order, schema.randomOrder(alias)...)
	this.unnest = append(this.unnest, schema.randomUnnest(alias)...)
}

func (this *Query) addJoin(keyspace *Keyspace, join *Join) {
	lalias := this.alias()
	this.nextAlias()
	ralias := this.alias()
	this.from = append(this.from, join.clause(lalias, ralias))
}

func (this *Query) nextAlias() {
	this.nextAliasNum++
	this.aliasNum = this.nextAliasNum
}

func (this *Query) alias() string {
	var res []rune
	res = append(res, []rune("${alias:")...)
	n := this.aliasNum
	for {
		c := n % 26
		res = append(res, rune('a'+c))
		if n < 26 {
			res = append(res, rune('}'))
			return string(res)
		}
		n /= 26
	}
}

func (this *Query) complete() {
	if len(this.aggs) == 0 && len(this.projection) == 0 {
		this.projection = append(this.projection, "*")
	}
	if len(this.aggs) > 0 {
		this.group = make([]string, len(this.order), len(this.order)+len(this.projection))
		if len(this.order) > 0 {
			copy(this.group, this.order)
		}
		for i := range this.projection {
			p := strings.TrimSuffix(this.projection[i], ".*")
			found := false
			for j := range this.group {
				if this.group[j] == p {
					found = true
					break
				}
			}
			if !found {
				this.group = append(this.group, p)
			}
		}
	} else if rand.Intn(25) == 17 && len(Queries) > 1 {
		switch rand.Intn(5) {
		case 1:
			// add a random query as a join (constant ON)
			// TODO: improve this (may need to keep a record of projection Field objects in the query object)
			this.nextAlias()
			this.from = append(this.from, fmt.Sprintf(" JOIN %s AS %s ON true",
				Queries[rand.Intn(len(Queries))].AsSubQuery(), this.alias()))
		case 3:
			// add a random query as a filter (random element, operation & value)
			// TODO: improve this (may need to keep a record of projection Field objects in the query object)
			//       Add correlation ?
			this.where = append(this.where, fmt.Sprintf(" AND %s[%d] %s",
				Queries[rand.Intn(len(Queries))].AsSubQuery(),
				rand.Intn(10),
				NewRandomJoinField("").GenerateFilter("")))
		default:
			// add a random query as a sub-query projection
			this.projection = append(this.projection, Queries[rand.Intn(len(Queries))].AsSubQuery())
		}
	}
	switch rand.Intn(20) {
	case 5:
		this.limit = fmt.Sprintf(" LIMIT %d", rand.Intn(100)+1)
	case 10:
		this.offset = fmt.Sprintf(" OFFSET %d", rand.Intn(100)+1)
	case 15:
		this.limit = fmt.Sprintf(" LIMIT %d", rand.Intn(100)+1)
		this.offset = fmt.Sprintf(" OFFSET %d", rand.Intn(100)+1)
	}
	this.followedJoins = nil
}

func (this *Query) SQL(baseAlias string) string {
	return this.doSQL(baseAlias, true)
}

func (this *Query) doSQL(baseAlias string, lock bool) string {
	if this.sql != "" {
		return this.aliasedSQL(baseAlias)
	}
	if lock {
		this.Lock()
	}
	if this.sql != "" {
		if lock {
			this.Unlock()
		}
		return this.aliasedSQL(baseAlias)
	}

	var sb strings.Builder

	sb.WriteString("SELECT ")
	first := true
	for i := range this.projection {
		if !first {
			sb.WriteRune(',')
		}
		sb.WriteString(this.projection[i])
		if this.projection[i] != "*" && !strings.HasSuffix(this.projection[i], ".*") {
			sb.WriteString(fmt.Sprintf(" AS p%d", i))
		}
		first = false
	}
	this.projection = nil
	for i := range this.aggs {
		if !first {
			sb.WriteRune(',')
		}
		sb.WriteString(this.aggs[i])
		first = false
	}
	this.aggs = nil

	sb.WriteString(" FROM ")
	for i := range this.from {
		sb.WriteString(this.from[i])
	}
	this.from = nil

	for i := range this.unnest {
		sb.WriteString(" UNNEST ")
		sb.WriteString(this.unnest[i])
	}
	this.unnest = nil

	if len(this.where) > 0 {
		sb.WriteString(" WHERE 1 = 1 ")
		for i := range this.where {
			sb.WriteString(this.where[i])
		}
	}
	this.where = nil

	if len(this.group) > 0 {
		sb.WriteString(" GROUP BY ")
		for i := range this.group {
			if i > 0 {
				sb.WriteRune(',')
			}
			sb.WriteString(this.group[i])
		}
	}
	this.group = nil

	if len(this.order) > 0 {
		sb.WriteString(" ORDER BY ")
		for i := range this.order {
			if i > 0 {
				sb.WriteRune(',')
			}
			sb.WriteString(this.order[i])
		}
	}
	this.order = nil

	sb.WriteString(this.offset)
	sb.WriteString(this.limit)

	this.sql = sb.String()
	if lock {
		this.Unlock()
	}
	return this.aliasedSQL(baseAlias)
}

var matchAliases = regexp.MustCompile("\\$\\{alias:([a-z]*)\\}")

func (this *Query) aliasedSQL(baseAlias string) string {
	return matchAliases.ReplaceAllStringFunc(this.sql, func(p string) string {
		p = p[8 : len(p)-1]
		return baseAlias + p
	})
}

func (this *Query) AsSubQuery() string {
	// adjust the aliases so there is no conflict with the parent query
	sql := this.SQL(fmt.Sprintf("sq%d_", nextSerial()))

	// Remove suffixing semi-colons if present
	sql = strings.TrimSpace(sql)

	if strings.LastIndex(sql, ";") == len(sql)-1 {
		sql = strings.TrimRight(sql, ";")
	}

	return "(" + sql + ")"
}

func (this *Query) Execute(requestParams map[string]interface{}) error {
	atomic.AddUint64(&this.executions, 1)
	results, elapsed, errs, err := executeSQLProcessingResults(this.SQL(""), requestParams)
	if err != nil {
		//logging.Debugf("%v", err)
		//logging.Errorf("Executing query %s has failed with error: %v", this.SQL(""), err)	// Uncomment for verbose error logging
		this.Lock()
		this.failed++
		this.lastFailure = err
		this.Unlock()
		return err
	}
	atomic.AddUint64(&this.results, uint64(results))
	atomic.AddUint64(&this.elapsedMs, uint64(elapsed.Milliseconds()))
	if len(errs) > 0 {
		this.Lock()
		this.lastErrors = append(errs, this.lastErrors...)
		if len(this.lastErrors) > 10 {
			this.lastErrors = this.lastErrors[:10]
		}
		this.Unlock()
	}
	return nil
}

func (this *Query) MarshalJSON() ([]byte, error) {
	this.Lock()
	m := map[string]interface{}{
		"statement":   this.doSQL("", false),
		"executions":  this.executions,
		"failed":      this.failed,
		"lastFailure": this.lastFailure,
		"lastErrors":  this.lastErrors,
	}
	if this.executions > 0 {
		m["avgElapsed"] = fmt.Sprintf("%v", time.Duration(this.elapsedMs/this.executions)*time.Millisecond)
		m["avgResults"] = this.results / this.executions
	}
	this.Unlock()
	return json.Marshal(m)
}

func (this *Query) reportAsFailed() bool {
	if this.failed > 0 {
		return true
	}
	for _, errCode := range this.lastErrors {
		// check for any unexpected/not-tolerated SQL errors
		if _, ok := IgnoredErrors[errCode]; !ok {
			return true
		}
	}
	return false
}
