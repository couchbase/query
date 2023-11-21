//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type Prepared struct {
	Operator
	signature       value.Value
	name            string
	encoded_plan    string
	text            string
	reqType         string
	indexApiVersion int
	featureControls uint64
	namespace       string
	queryContext    string
	tenant          string
	useFts          bool
	useCBO          bool
	preparedTime    time.Time // time the plan was generated
	optimHints      *algebra.OptimHints

	indexScanKeyspaces map[string]bool
	indexers           []idxVersion // for reprepare checking
	keyspaces          []ksVersion
	subqueryPlans      *algebra.SubqueryPlans
	txPrepareds        map[string]*Prepared
	sync.RWMutex
}

type idxVersion struct {
	indexer datastore.Indexer
	version uint64
}

type ksVersion struct {
	ksMeta  datastore.KeyspaceMetadata
	version uint64
}

func NewPrepared(operator Operator, signature value.Value, indexScanKeyspaces map[string]bool,
	optimHints *algebra.OptimHints) *Prepared {

	return &Prepared{
		Operator:           operator,
		signature:          signature,
		optimHints:         optimHints,
		indexScanKeyspaces: indexScanKeyspaces,
	}
}

func NewPreparedFromEncodedPlan(prepared_stmt string) (*Prepared, []byte, int, errors.Error) {
	r := 0
	prepared := NewPrepared(nil, nil, nil, nil)
	decoded, err := base64.StdEncoding.DecodeString(prepared_stmt)
	if err != nil {
		return prepared, nil, r, errors.NewPreparedDecodingError(err)
	}
	var buf bytes.Buffer
	buf.Write(decoded)
	reader, err := gzip.NewReader(&buf)
	if err != nil {
		return prepared, nil, r, errors.NewPreparedDecodingError(err)
	}
	prepared_bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return prepared, nil, r, errors.NewPreparedDecodingError(err)
	}
	version, err := prepared.unmarshalInternal(prepared_bytes)
	if err != nil {
		return prepared, prepared_bytes, r, errors.NewUnrecognizedPreparedError(err)
	}

	if version < util.PLAN_VERSION {
		r = -1
	} else if version > util.PLAN_VERSION {
		r = 1
	}
	return prepared, nil, r, nil
}

func (this *Prepared) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Prepared) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := make(map[string]interface{}, 5)
	this.marshalInternal(r)
	if f != nil {
		f(r)
	}
	return r
}

func (this *Prepared) marshalInternal(r map[string]interface{}) {
	r["operator"] = this.Operator
	r["signature"] = this.signature
	r["name"] = this.name
	r["encoded_plan"] = this.encoded_plan
	r["text"] = this.text
	r["indexApiVersion"] = this.indexApiVersion
	r["featureControls"] = this.featureControls
	r["namespace"] = this.namespace
	r["queryContext"] = this.queryContext
	r["reqType"] = this.reqType
	if this.useFts {
		r["useFts"] = this.useFts
	}
	if this.useCBO {
		r["useCBO"] = this.useCBO
	}
	if len(this.indexScanKeyspaces) > 0 {
		r["indexScanKeyspaces"] = this.IndexScanKeyspaces()
	}
	if this.optimHints != nil {
		r["optimizer_hints"] = this.optimHints
	}
}

func (this *Prepared) UnmarshalJSON(body []byte) error {
	_, err := this.unmarshalInternal(body)
	return err
}

func (this *Prepared) unmarshalInternal(body []byte) (int, error) {
	var _unmarshalled struct {
		Operator           json.RawMessage        `json:"operator"`
		Signature          json.RawMessage        `json:"signature"`
		Name               string                 `json:"name"`
		EncodedPlan        string                 `json:"encoded_plan"`
		Text               string                 `json:"text"`
		ReqType            string                 `json:"reqType"`
		ApiVersion         int                    `json:"indexApiVersion"`
		FeatureControls    uint64                 `json:"featureControls"`
		Namespace          string                 `json:"namespace"`
		QueryContext       string                 `json:"queryContext"`
		UseFts             bool                   `json:"useFts"`
		UseCBO             bool                   `json:"useCBO"`
		IndexScanKeyspaces map[string]interface{} `json:"indexScanKeyspaces"`
		Version            int                    `json:"planVersion"`
		OptimHints         json.RawMessage        `json:"optimizer_hints"`
	}

	var op_type struct {
		Operator string `json:"#operator"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return 0, err
	}

	err = json.Unmarshal(_unmarshalled.Operator, &op_type)
	if err != nil {
		return 0, err
	}

	if _unmarshalled.ApiVersion < datastore.INDEX_API_MIN {
		_unmarshalled.ApiVersion = datastore.INDEX_API_MIN
	} else if _unmarshalled.ApiVersion > datastore.INDEX_API_MAX {
		_unmarshalled.ApiVersion = datastore.INDEX_API_MAX
	}
	this.signature = value.NewValue(_unmarshalled.Signature)
	this.name = _unmarshalled.Name
	this.encoded_plan = _unmarshalled.EncodedPlan
	this.text = _unmarshalled.Text
	this.reqType = _unmarshalled.ReqType
	this.indexApiVersion = _unmarshalled.ApiVersion
	this.featureControls = _unmarshalled.FeatureControls
	this.namespace = _unmarshalled.Namespace
	this.queryContext = _unmarshalled.QueryContext
	this.useFts = _unmarshalled.UseFts
	this.useCBO = _unmarshalled.UseCBO
	if len(_unmarshalled.IndexScanKeyspaces) > 0 {
		this.indexScanKeyspaces = make(map[string]bool, len(_unmarshalled.IndexScanKeyspaces))
		for ks, v := range _unmarshalled.IndexScanKeyspaces {
			this.indexScanKeyspaces[ks] = v.(bool)
		}
	}
	if len(_unmarshalled.OptimHints) > 0 {
		this.optimHints, err = unmarshalOptimHints(_unmarshalled.OptimHints)
		if err != nil {
			return 0, err
		}
	}
	this.Operator, err = MakeOperator(op_type.Operator, _unmarshalled.Operator)

	return _unmarshalled.Version, err
}

func (this *Prepared) Signature() value.Value {
	return this.signature
}

func (this *Prepared) Name() string {
	return this.name
}

func (this *Prepared) SetName(name string) {
	this.name = name
}

func (this *Prepared) Text() string {
	return this.text
}

func (this *Prepared) SetText(text string) {
	this.text = text
}

func (this *Prepared) Type() string {
	return this.reqType
}

func (this *Prepared) SetType(reqType string) {
	this.reqType = reqType
}

func (this *Prepared) IndexApiVersion() int {
	return this.indexApiVersion
}

func (this *Prepared) SetIndexApiVersion(indexApiVersion int) {
	this.indexApiVersion = indexApiVersion
}

func (this *Prepared) FeatureControls() uint64 {
	return this.featureControls
}

func (this *Prepared) SetFeatureControls(featureControls uint64) {
	this.featureControls = featureControls
}

func (this *Prepared) Namespace() string {
	return this.namespace
}

func (this *Prepared) SetNamespace(namespace string) {
	this.namespace = namespace
}

func (this *Prepared) QueryContext() string {
	return this.queryContext
}

func (this *Prepared) SetQueryContext(queryContext string) {
	this.queryContext = queryContext
	if queryContext != "" {
		path := algebra.ParseQueryContext(queryContext)
		if len(path) > 1 {
			this.tenant = path[1]
		}
	}
}

func (this *Prepared) Tenant() string {
	return this.tenant
}

func (this *Prepared) UseFts() bool {
	return this.useFts
}

func (this *Prepared) SetUseFts(useFts bool) {
	this.useFts = useFts
}

func (this *Prepared) UseCBO() bool {
	return this.useCBO
}

func (this *Prepared) SetUseCBO(useCBO bool) {
	this.useCBO = useCBO
}

func (this *Prepared) EncodedPlan() string {
	return this.encoded_plan
}

func (this *Prepared) SetEncodedPlan(encoded_plan string) {
	this.encoded_plan = encoded_plan
}

func (this *Prepared) BuildEncodedPlan() (string, error) {
	var b bytes.Buffer

	r := make(map[string]interface{}, 5)
	r["planVersion"] = util.PLAN_VERSION
	this.marshalInternal(r)
	json_bytes, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	w := gzip.NewWriter(&b)
	w.Write(json_bytes)
	w.Close()
	str := base64.StdEncoding.EncodeToString(b.Bytes())
	this.encoded_plan = str
	return str, nil
}

func (this *Prepared) MismatchingEncodedPlan(encoded_plan string) bool {
	return this.encoded_plan != encoded_plan
}

func (this *Prepared) OptimHints() *algebra.OptimHints {
	return this.optimHints
}

func (this *Prepared) SetIndexScanKeyspaces(ik map[string]bool) {
	this.indexScanKeyspaces = ik
}

func (this *Prepared) IndexScanKeyspaces() (rv map[string]interface{}) {
	if len(this.indexScanKeyspaces) > 0 {
		rv = make(map[string]interface{}, len(this.indexScanKeyspaces))
		for ks, v := range this.indexScanKeyspaces {
			rv[ks] = v
		}
	}
	return rv
}

// Locking is handled by the top level caller!
func (this *Prepared) addIndexer(indexer datastore.Indexer) {
	indexer.Refresh()
	version := indexer.MetadataVersion()
	for i, idx := range this.indexers {
		if idx.indexer.Name() == indexer.Name() &&
			datastore.IndexerQualifiedKeyspacePath(idx.indexer) == datastore.IndexerQualifiedKeyspacePath(indexer) {
			this.indexers[i].indexer = indexer
			this.indexers[i].version = version
			return
		}
	}
	this.indexers = append(this.indexers, idxVersion{indexer, version})
}

// Locking is handled by the top level caller!
func (this *Prepared) addKeyspaceMetadata(ksMeta datastore.KeyspaceMetadata) {
	version := ksMeta.MetadataVersion()
	for i, ks := range this.keyspaces {
		if ks.ksMeta.MetadataId() == ksMeta.MetadataId() {
			this.keyspaces[i].ksMeta = ksMeta
			this.keyspaces[i].version = version
			return
		}
	}
	this.keyspaces = append(this.keyspaces, ksVersion{ksMeta, version})
}

func (this *Prepared) MetadataCheck() bool {

	// check that metadata is the same for the indexers involved
	for _, idx := range this.indexers {
		idx.indexer.Refresh()
		if idx.indexer.MetadataVersion() != idx.version {
			return false
		}
	}

	// now check that metadata is good for the namespaces involved
	// if the bucket has been deleted, the version is expected to be different
	for _, ks := range this.keyspaces {
		if ks.ksMeta.MetadataVersion() != ks.version {
			return false
		}
	}
	return true
}

// verify prepared+subquery plans
func (this *Prepared) Verify() bool {
	good := this.Operator.verify(this)
	if good {
		subqueryPlans := this.GetSubqueryPlans(false)
		if subqueryPlans != nil {
			// Verify subquery plans
			verifyF := func(key *algebra.Select, options uint32, plan, isk interface{}) (bool, bool) {
				var good, local bool
				if qp, ok := plan.(*QueryPlan); ok {
					good = qp.PlanOp().verify(this)
				}
				return good, local
			}
			good, _ = subqueryPlans.ForEach(nil, uint32(0), true, verifyF)
		}
	}
	return good
}

// Subquery plans of prepared statement
func (this *Prepared) GetSubqueryPlans(init bool) *algebra.SubqueryPlans {
	this.RLock()
	subqueryPlans := this.subqueryPlans
	this.RUnlock()

	if subqueryPlans == nil && init {
		this.Lock()
		subqueryPlans = this.subqueryPlans
		if subqueryPlans == nil {
			subqueryPlans = algebra.NewSubqueryPlans()
			this.subqueryPlans = subqueryPlans
		}
		this.Unlock()
	}
	return subqueryPlans
}

const (
	_TX_KEYSPACES = 2
)

func (this *Prepared) SetTxPrepared(txPrepared *Prepared, hashCode string) {
	if hashCode == "" || len(this.indexScanKeyspaces) > _TX_KEYSPACES {
		return
	}
	this.Lock()
	defer this.Unlock()
	if this.txPrepareds == nil {
		this.txPrepareds = make(map[string]*Prepared, (1 << _TX_KEYSPACES))
	}
	this.txPrepareds[hashCode] = txPrepared
}

func (this *Prepared) GetTxPrepared(deltaKeyspaces map[string]bool) (*Prepared, string) {
	good := this.Type() != "DELETE"
	if len(deltaKeyspaces) > 0 {
		if len(this.indexScanKeyspaces) == 0 {
			if good {
				return this, ""
			} else {
				deltaKeyspaces = nil
			}
		} else if len(this.indexScanKeyspaces) > _TX_KEYSPACES {
			return nil, ""
		}
		if good {
			for ks, _ := range this.indexScanKeyspaces {
				if _, ok := deltaKeyspaces[ks]; ok {
					good = false
					break
				}
			}
			if good {
				return this, ""
			}
		}

	}

	hashCode := this.txHashCode(deltaKeyspaces)
	this.RLock()
	defer this.RUnlock()
	prepared, _ := this.txPrepareds[hashCode]
	return prepared, hashCode
}

func (this *Prepared) TxPrepared() (rv map[string]interface{}, rvp map[string]interface{}) {
	if len(this.txPrepareds) == 0 {
		return
	}
	this.RLock()
	defer this.RUnlock()
	rv = make(map[string]interface{}, len(this.txPrepareds))
	rvp = make(map[string]interface{}, len(this.txPrepareds))
	i := 1
	for _, p := range this.txPrepareds {
		plan := p.EncodedPlan()
		isks := p.IndexScanKeyspaces()
		if plan != "" && len(isks) > 0 {
			key := "p" + strconv.Itoa(i)
			i++
			rv[key] = map[string]interface{}{
				"plan":               plan,
				"indexScanKeyspaces": isks,
			}
			b, _ := json.Marshal(p.Operator)
			rvp[key] = map[string]interface{}{
				"plan":               b,
				"indexScanKeyspaces": isks,
			}
		}
	}
	return
}

func (this *Prepared) txHashCode(deltaKeyspaces map[string]bool) (hashCode string) {
	if len(deltaKeyspaces) == 0 {
		return "(delete)"
	}

	var i int
	var nameBuf [_TX_KEYSPACES]string
	names := nameBuf[0:len(this.indexScanKeyspaces)]

	for ks, _ := range this.indexScanKeyspaces {
		names[i] = ks
		i++
	}

	sort.Strings(names)

	for i, ks := range names {
		if i != 0 {
			hashCode += ","
		}
		if ok, _ := deltaKeyspaces[ks]; ok {
			hashCode += "(" + ks + "):true"
		} else {
			hashCode += "(" + ks + "):false"
		}
	}

	return hashCode
}

func (this *Prepared) SetPreparedTime(time time.Time) {
	this.preparedTime = time
}

func (this *Prepared) PreparedTime() time.Time {
	return this.preparedTime
}

// Generates subquery plan information for all subquery plans stored in the Prepared object
// for its system:prepareds entry
func (this *Prepared) GetSubqueryPlansEntry() map[string]interface{} {
	subqueryPlans := this.GetSubqueryPlans(false)

	if subqueryPlans != nil {
		index := 1
		rv := make(map[string]interface{}, 0)

		// Iterate through the subquery plans and create the entry
		verifyF := func(key *algebra.Select, options uint32, splan, isk interface{}) (bool, bool) {
			var sqOperator Operator

			if qp, ok := splan.(*QueryPlan); ok {
				sqOperator = qp.PlanOp()
			}

			entryKey := "sq" + strconv.Itoa(index)
			entry := map[string]interface{}{
				"plan":     value.NewMarshalledValue(sqOperator),
				"subquery": key.String(),
			}

			// process index scan keyspaces since value.Value creation not supported for map[string]bool type
			if i, ok := isk.(map[string]bool); ok {
				if len(i) > 0 {
					isksEntry := make(map[string]interface{}, len(i))

					for ks, v := range i {
						isksEntry[ks] = v
					}
					entry["indexScanKeyspaces"] = isksEntry
				}
			}

			rv[entryKey] = entry
			index++

			return true, false
		}

		subqueryPlans.ForEach(nil, uint32(0), true, verifyF)

		return rv
	}

	return nil
}
