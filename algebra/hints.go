//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

type HintType int32

const (
	HINT_INVALID = HintType(iota)
	HINT_INDEX
	HINT_INDEX_FTS
	HINT_NL
	HINT_HASH
	HINT_ORDERED
	HINT_NO_INDEX
	HINT_NO_INDEX_FTS
	HINT_NO_HASH
	HINT_NO_NL
	HINT_JOIN_FILTER
	HINT_NO_JOIN_FILTER
	HINT_INDEX_ALL
)

type HintState int32

const (
	HINT_STATE_UNKNOWN = HintState(iota)
	HINT_STATE_FOLLOWED
	HINT_STATE_NOT_FOLLOWED
	HINT_STATE_ERROR
	HINT_STATE_INVALID
)

const (
	INVALID_HINT                   = "Invalid hint name"
	MISSING_ARG                    = "Missing argument for "
	EXTRA_ARG                      = "Argument not expected: "
	INVALID_SLASH                  = "Invalid '/' found in "
	EXTRA_SLASH                    = "Extra '/' found in "
	INVALID_HASH_OPTION            = "Invalid hash option (BUILD or PROBE only):  "
	INVALID_NEG_HASH_OPTION        = "Hash option (BUILD or PROBE) not allowed in NO_HASH/AVOID_HASH hints"
	INVALID_KEYSPACE               = "Invalid keyspace specified: "
	INVALID_FIELD                  = "Invalid field specified: "
	INVALID_INDEX                  = "Index name(s) must be string in: "
	DUPLICATED_KEYSPACE            = "Duplicated keyspace/alias specified in: "
	DUPLICATED_JOIN_HINT           = "Duplicated join hint specified for keyspace: "
	DUPLICATED_JOIN_FLTR_HINT      = "Duplicated join filter hint specified for keyspace: "
	DUPLICATED_INDEX_HINT          = "Duplicated index hint specified for keyspace: "
	DUPLICATED_INDEX_FTS_HINT      = "Duplicated FTS index hint specified for keyspace: "
	DUPLICATED_INDEX_ALL_HINT      = "Duplicated INDEX_ALL hint specified for keyspace: "
	NON_KEYSPACE_INDEX_HINT        = "Index hint specified on non-keyspace: "
	NON_KEYSPACE_INDEX_FTS_HINT    = "FTS index hint specified on non-keyspace: "
	NON_KEYSPACE_INDEX_ALL_HINT    = "INDEX_ALL hint specified on non-keyspace: "
	HASH_JOIN_NOT_AVAILABLE        = "Hash Join/Nest is not supported"
	HASH_JOIN_BUILD_JOIN_FILTER    = "USE_HASH with BUILD option conflicts with JOIN_FILTER"
	NL_JOIN_JOIN_FILTER            = "USE_NL conflicts with JOIN_FILTER"
	ALL_TERM_JOIN_FILTER           = "JOIN_FILTER hint cannot be specified on all terms in a query"
	GSI_INDEXER_NOT_AVAIL          = "GSI Indexer not available"
	FTS_INDEXER_NOT_AVAIL          = "FTS Indexer not available"
	INVALID_GSI_INDEX              = "Invalid indexes specified: "
	INVALID_FTS_INDEX              = "Invalid FTS indexes specified: "
	JOIN_HINT_FIRST_TERM           = "Join hint (USE HASH or USE NL) cannot be specified on the first from term: "
	INDEX_HINT_NOT_FOLLOWED        = "INDEX hint cannot be followed"
	INDEX_FTS_HINT_NOT_FOLLOWED    = "INDEX_FTS hint cannot be followed"
	INDEX_ALL_HINT_NOT_FOLLOWED    = "INDEX_ALL hint cannot be followed"
	USE_NL_HINT_NOT_FOLLOWED       = "USE_NL hint cannot be followed"
	USE_HASH_HINT_NOT_FOLLOWED     = "USE_HASH hint cannot be followed"
	ORDERED_HINT_NOT_FOLLOWED      = "ORDERED hint cannot be followed"
	NO_INDEX_HINT_NOT_FOLLOWED     = "NO_INDEX hint cannot be followed"
	NO_INDEX_FTS_HINT_NOT_FOLLOWED = "NO_INDEX_FTS hint cannot be followed"
	NO_HASH_HINT_NOT_FOLLOWED      = "NO_HASH hint cannot be followed"
	NO_NL_HINT_NOT_FOLLOWED        = "NO_NL hint cannot be followed"
	AVD_IDX_HINT_NOT_FOLLOWED      = "AVOID_INDEX hint cannot be followed"
	AVD_IDX_FTS_HINT_NOT_FOLLOWED  = "AVOID_INDEX_FTS hint cannot be followed"
	AVD_HASH_HINT_NOT_FOLLOWED     = "AVOID_HASH hint cannot be followed"
	AVD_NL_HINT_NOT_FOLLOWED       = "AVOID_NL hint cannot be followed"
	JOIN_FILTER_HINT_NOT_FOLLOWED  = "JOIN_FILTER hint cannot be followed"
	NO_JN_FLTR_HINT_NOT_FOLLOWED   = "NO_JOIN_FILTER hint cannot be followed"
	AVD_JN_FLTR_HINT_NOT_FOLLOWED  = "AVOID_JOIN_FILTER hint cannot be followed"
	MERGE_ONKEY_JOIN_HINT_ERR      = "Join hint not supported in a MERGE statement with ON KEY clause"
	MERGE_ONKEY_INDEX_HINT_ERR     = "Index hint not supported for target keyspace in a MERGE statement with ON KEY clause"
	UPD_DEL_JOIN_HINT_ERR          = "Join hint not supported in an UPDATE or DELETE statement"
	DML_ORDERED_HINT_ERR           = "Ordered hint not supported in DML statements"
	MIXED_INDEX_ALL_WITH_INDEX     = "INDEX_ALL hint cannot be mixed with other index or FTS index hints for keyspace: "
	MIXED_INDEX_WITH_INDEX_ALL     = "Index hint cannot be mixed with INDEX_ALL hint for keyspace: "
	MIXED_INDEX_FTS_WITH_INDEX_ALL = "FTS index hint cannot be mixed with INDEX_ALL hint for keyspace: "
	INDEX_ALL_SINGLE_INDEX         = "INDEX_ALL hint must have more than one index specified"
	INDEX_ALL_LEGACY_JOIN          = "INDEX_ALL hint is not supported for keyspace under lookup join or index join: "
)

type OptimHint interface {
	Type() HintType
	FormatHint(jsonStyle bool) string
	State() HintState
	SetFollowed()
	SetNotFollowed()
	Error() string
	SetError(err string)
	Copy() OptimHint
	Derived() bool
	sortString() string
}

type OptimHints struct {
	hints         []OptimHint
	jsonStyle     bool              // JSON style hints
	subqTermHints []*SubqOptimHints // optimizer hints from SubqueryTerms
}

func NewOptimHints(hints []OptimHint, jsonStyle bool) *OptimHints {
	return &OptimHints{
		hints:     hints,
		jsonStyle: jsonStyle,
	}
}

func (this *OptimHints) Hints() []OptimHint {
	return this.hints
}

func (this *OptimHints) JSONStyle() bool {
	return this.jsonStyle
}

func (this *OptimHints) AddHints(hints []OptimHint) {
	this.hints = append(this.hints, hints...)
}

func (this *OptimHints) SubqTermHints() []*SubqOptimHints {
	return this.subqTermHints
}

func (this *OptimHints) AddSubqTermHints(subqTermHints []*SubqOptimHints) {
	this.subqTermHints = append(this.subqTermHints, subqTermHints...)
}

func (this *OptimHints) String() string {
	var s string
	var r map[string]interface{}
	found := false
	for _, hint := range this.hints {
		if hint.Derived() {
			continue
		}
		if found {
			if !this.jsonStyle {
				s += " "
			}
		} else {
			found = true
		}
		if this.jsonStyle {
			addJSONHint(r, hint)
		} else {
			s += hint.FormatHint(this.jsonStyle)
		}
	}

	if !found {
		return ""
	}

	if this.jsonStyle {
		bytes, _ := json.Marshal(r)
		return "/*+ " + string(bytes) + " */"
	}
	return "/*+ " + s + " */"
}

func addJSONHint(r map[string]interface{}, hint OptimHint) {
	var name string
	var obj map[string]interface{}
	switch hint := hint.(type) {
	case *HintIndex:
		name = "index"
		obj = hint.formatJSON()
	case *HintFTSIndex:
		name = "index_fts"
		obj = hint.formatJSON()
	case *HintNoIndex:
		name = "no_index"
		if hint.avoid {
			name = "avoid_index"
		}
		obj = hint.formatJSON()
	case *HintNoFTSIndex:
		name = "no_index_fts"
		if hint.avoid {
			name = "avoid_index_fts"
		}
		obj = hint.formatJSON()
	case *HintIndexAll:
		name = "index_all"
		obj = hint.formatJSON()
	case *HintNL:
		name = "use_nl"
		obj = hint.formatJSON()
	case *HintHash:
		name = "use_hash"
		obj = hint.formatJSON()
	case *HintNoNL:
		name = "no_use_nl"
		if hint.avoid {
			name = "avoid_nl"
		}
		obj = hint.formatJSON()
	case *HintNoHash:
		name = "no_use_hash"
		if hint.avoid {
			name = "avoid_hash"
		}
		obj = hint.formatJSON()
	case *HintJoinFilter:
		name = "join_filter"
		obj = hint.formatJSON()
	case *HintNoJoinFilter:
		name = "no_join_filter"
		if hint.avoid {
			name = "avoid_join_filter"
		}
		obj = hint.formatJSON()
	case *HintOrdered:
		name = "ordered"
		obj = hint.formatJSON()
	case *HintInvalid:
		name = "invalid_hints"
		obj = hint.formatJSON()
	}

	if name != "" {
		curr, ok := r[name]
		if ok {
			// already has a hint of same type
			var newHints []interface{}
			switch curr := curr.(type) {
			case []interface{}:
				newHints = append(curr, obj)
			default:
				newHints = []interface{}{curr, obj}
			}
			r[name] = newHints
		} else {
			r[name] = obj
		}
	}
}

/*
hint_args == nil: hint is just an identifier with no paren, e.g. ORDERED
hint_args == []string{}: hint has paren, but nothing inside the paren, e.g. ORDERED()
*/
func NewOptimHint(hint_name string, hint_args []string) []OptimHint {
	var hints []OptimHint
	invalid := false
	var err string
	lowerName := strings.ToLower(hint_name)
	switch lowerName {
	case "index", "index_fts", "no_index", "no_index_fts", "avoid_index", "avoid_index_fts", "index_all", "index_combine":
		fts := (lowerName == "index_fts") || (lowerName == "no_index_fts") || (lowerName == "avoid_index_fts")
		avoid := (lowerName == "avoid_index") || (lowerName == "avoid_index_fts")
		negative := (lowerName == "no_index") || (lowerName == "no_index_fts") || avoid
		indexAll := (lowerName == "index_all") || (lowerName == "index_combine")
		if len(hint_args) == 0 {
			invalid = true
			err = MISSING_ARG + hint_name
			break
		}
		// first arg is keyspace (alias)
		indexes := make([]string, 0, len(hint_args)-1)
		for i := 1; i < len(hint_args); i++ {
			if strings.Contains(hint_args[i], "/") {
				invalid = true
				err = INVALID_SLASH + hint_args[i]
				break
			}
			indexes = append(indexes, hint_args[i])
		}
		if indexAll && len(indexes) < 2 {
			invalid = true
			err = INDEX_ALL_SINGLE_INDEX
		}
		if !invalid {
			if indexAll {
				hints = []OptimHint{NewIndexAllHint(hint_args[0], indexes)}
			} else if fts {
				if negative {
					hints = []OptimHint{NewNoFTSIndexHint(hint_args[0], indexes, avoid)}
				} else {
					hints = []OptimHint{NewFTSIndexHint(hint_args[0], indexes)}
				}
			} else {
				if negative {
					hints = []OptimHint{NewNoIndexHint(hint_args[0], indexes, avoid)}
				} else {
					hints = []OptimHint{NewIndexHint(hint_args[0], indexes)}
				}
			}
		}
	case "use_nl", "no_use_nl", "avoid_nl":
		avoid := (lowerName == "avoid_nl")
		negative := (lowerName == "no_use_nl") || avoid
		// USE_NL/NO_USE_NL hint must include at least 1 keyspsace
		if len(hint_args) == 0 {
			invalid = true
			err = MISSING_ARG + hint_name
			break
		}
		hints = make([]OptimHint, 0, len(hint_args))
		for _, arg := range hint_args {
			if strings.Contains(arg, "/") {
				invalid = true
				err = INVALID_SLASH + arg
				break
			}
			var hint OptimHint
			if negative {
				hint = NewNoNLHint(arg, avoid)
			} else {
				hint = NewNLHint(arg)
			}
			hints = append(hints, hint)
		}
	case "use_hash", "no_use_hash", "avoid_hash":
		avoid := (lowerName == "avoid_hash")
		negative := (lowerName == "no_use_hash") || avoid
		// USE_HASH must include at least 1 keyspace
		if len(hint_args) == 0 {
			invalid = true
			err = MISSING_ARG
			break
		}
		hints = make([]OptimHint, 0, len(hint_args))
		for _, arg := range hint_args {
			var keyspace string
			var option HashOption

			// check whether /BUILD or /PROBE is present
			parts := strings.Split(arg, "/")
			if len(parts) > 2 {
				invalid = true
				err = EXTRA_SLASH + arg
			} else if len(parts) == 2 {
				if negative {
					invalid = true
					err = INVALID_NEG_HASH_OPTION
				} else {
					keyspace = parts[0]
					switch strings.ToLower(parts[1]) {
					case "build":
						option = HASH_OPTION_BUILD
					case "probe":
						option = HASH_OPTION_PROBE
					default:
						invalid = true
						err = INVALID_HASH_OPTION + parts[1]
					}
				}
			} else if len(parts) == 1 {
				keyspace = parts[0]
				if !negative {
					option = HASH_OPTION_NONE
				}
			}

			if invalid {
				break
			}

			var hint OptimHint
			if negative {
				hint = NewNoHashHint(keyspace, avoid)
			} else {
				hint = NewHashHint(keyspace, option)
			}
			hints = append(hints, hint)
		}
	case "px_join_filter", "no_px_join_filter":
		// allow PX_JOIN_FILTER/NO_PX_JOIN_FILTER
		lowerName = strings.Replace(lowerName, "px_", "", 1)
		fallthrough
	case "join_filter", "no_join_filter", "avoid_join_filter":
		avoid := (lowerName == "avoid_join_filter")
		negative := (lowerName == "no_join_filter") || avoid
		// JOIN_FILTER/NO_JOIN_FILTER hint must include at least 1 keyspsace
		if len(hint_args) == 0 {
			invalid = true
			err = MISSING_ARG + hint_name
			break
		}
		hints = make([]OptimHint, 0, len(hint_args))
		for _, arg := range hint_args {
			if strings.Contains(arg, "/") {
				invalid = true
				err = INVALID_SLASH + arg
				break
			}
			var hint OptimHint
			if negative {
				hint = NewNoJoinFilterHint(arg, avoid)
			} else {
				hint = NewJoinFilterHint(arg)
			}
			hints = append(hints, hint)
		}
	case "ordered":
		if hint_args != nil {
			invalid = true
			err = EXTRA_ARG + "(" + strings.Join(hint_args, " ") + ")"
			break
		}
		hints = []OptimHint{NewOrderedHint()}
	default:
		invalid = true
		err = INVALID_HINT
	}

	if invalid || len(hints) == 0 {
		return invalidHint(hint_name, hint_args, err)
	}
	return hints
}

type HintIndex struct {
	keyspace string
	indexes  []string
	derived  bool
	state    HintState
	err      string
}

func NewIndexHint(keyspace string, indexes []string) *HintIndex {
	return &HintIndex{
		keyspace: keyspace,
		indexes:  indexes,
	}
}

// derived from USE INDEX
func NewDerivedIndexHint(keyspace string, indexes []string) *HintIndex {
	return &HintIndex{
		keyspace: keyspace,
		indexes:  indexes,
		derived:  true,
	}
}

func (this *HintIndex) Type() HintType {
	return HINT_INDEX
}

func (this *HintIndex) Copy() OptimHint {
	rv := &HintIndex{
		keyspace: this.keyspace,
		derived:  this.derived,
		state:    this.state,
		err:      this.err,
	}
	if len(this.indexes) > 0 {
		rv.indexes = make([]string, 0, len(this.indexes))
		rv.indexes = append(rv.indexes, this.indexes...)
	}
	return rv
}

func (this *HintIndex) Keyspace() string {
	return this.keyspace
}

func (this *HintIndex) Indexes() []string {
	return this.indexes
}

func (this *HintIndex) Derived() bool {
	return this.derived
}

func (this *HintIndex) State() HintState {
	return this.state
}

func (this *HintIndex) SetFollowed() {
	if this.state == HINT_STATE_UNKNOWN {
		this.state = HINT_STATE_FOLLOWED
	}
}

func (this *HintIndex) SetNotFollowed() {
	if this.state == HINT_STATE_UNKNOWN {
		this.state = HINT_STATE_NOT_FOLLOWED
		this.err = INDEX_HINT_NOT_FOLLOWED
	}
}

func (this *HintIndex) Error() string {
	return this.err
}

func (this *HintIndex) SetError(err string) {
	this.err = err
	this.state = HINT_STATE_ERROR
}

func (this *HintIndex) sortString() string {
	return fmt.Sprintf("%d%d%t%s%d%s", this.Type(), this.state, this.derived, this.keyspace, len(this.indexes), this.err)
}

func (this *HintIndex) FormatHint(jsonStyle bool) string {
	if jsonStyle {
		hint := map[string]interface{}{
			"index": this.formatJSON(),
		}
		bytes, _ := json.Marshal(hint)
		return string(bytes)
	}
	args := make([]string, 0, len(this.indexes)+1)
	args = append(args, this.keyspace)
	args = append(args, this.indexes...)
	return formatHint("INDEX", args)
}

func (this *HintIndex) formatJSON() map[string]interface{} {
	r := make(map[string]interface{}, 2)
	r["keyspace"] = this.keyspace
	if len(this.indexes) > 0 {
		indexes := make([]interface{}, 0, len(this.indexes))
		for _, idx := range this.indexes {
			indexes = append(indexes, idx)
		}
		r["indexes"] = indexes
	}
	return r
}

type HintFTSIndex struct {
	keyspace string
	indexes  []string
	derived  bool
	state    HintState
	err      string
}

func NewFTSIndexHint(keyspace string, indexes []string) *HintFTSIndex {
	return &HintFTSIndex{
		keyspace: keyspace,
		indexes:  indexes,
	}
}

func NewDerivedFTSIndexHint(keyspace string, indexes []string) *HintFTSIndex {
	return &HintFTSIndex{
		keyspace: keyspace,
		indexes:  indexes,
		derived:  true,
	}
}

func (this *HintFTSIndex) Type() HintType {
	return HINT_INDEX_FTS
}

func (this *HintFTSIndex) Copy() OptimHint {
	rv := &HintFTSIndex{
		keyspace: this.keyspace,
		derived:  this.derived,
		state:    this.state,
		err:      this.err,
	}
	if len(this.indexes) > 0 {
		rv.indexes = make([]string, 0, len(this.indexes))
		rv.indexes = append(rv.indexes, this.indexes...)
	}
	return rv
}

func (this *HintFTSIndex) Keyspace() string {
	return this.keyspace
}

func (this *HintFTSIndex) Indexes() []string {
	return this.indexes
}

func (this *HintFTSIndex) Derived() bool {
	return this.derived
}

func (this *HintFTSIndex) State() HintState {
	return this.state
}

func (this *HintFTSIndex) SetFollowed() {
	if this.state == HINT_STATE_UNKNOWN {
		this.state = HINT_STATE_FOLLOWED
	}
}

func (this *HintFTSIndex) SetNotFollowed() {
	if this.state == HINT_STATE_UNKNOWN {
		this.state = HINT_STATE_NOT_FOLLOWED
		this.err = INDEX_FTS_HINT_NOT_FOLLOWED
	}
}

func (this *HintFTSIndex) Error() string {
	return this.err
}

func (this *HintFTSIndex) SetError(err string) {
	this.err = err
	this.state = HINT_STATE_ERROR
}

func (this *HintFTSIndex) sortString() string {
	return fmt.Sprintf("%d%d%t%s%d%s", this.Type(), this.state, this.derived, this.keyspace, len(this.indexes), this.err)
}

func (this *HintFTSIndex) FormatHint(jsonStyle bool) string {
	if jsonStyle {
		hint := map[string]interface{}{
			"index_fts": this.formatJSON(),
		}
		bytes, _ := json.Marshal(hint)
		return string(bytes)
	}
	args := make([]string, 0, len(this.indexes)+1)
	args = append(args, this.keyspace)
	args = append(args, this.indexes...)
	return formatHint("INDEX_FTS", args)
}

func (this *HintFTSIndex) formatJSON() map[string]interface{} {
	r := make(map[string]interface{}, 2)
	r["keyspace"] = this.keyspace
	if len(this.indexes) > 0 {
		indexes := make([]interface{}, 0, len(this.indexes))
		for _, idx := range this.indexes {
			indexes = append(indexes, idx)
		}
		r["indexes"] = indexes
	}
	return r
}

type HintNoIndex struct {
	keyspace   string
	indexes    []string
	oriIndexes []string
	state      HintState
	err        string
	avoid      bool
}

func NewNoIndexHint(keyspace string, indexes []string, avoid bool) *HintNoIndex {
	return &HintNoIndex{
		keyspace: keyspace,
		indexes:  indexes,
		avoid:    avoid,
	}
}

func (this *HintNoIndex) Type() HintType {
	return HINT_NO_INDEX
}

func (this *HintNoIndex) Copy() OptimHint {
	rv := &HintNoIndex{
		keyspace: this.keyspace,
		state:    this.state,
		err:      this.err,
		avoid:    this.avoid,
	}
	if len(this.indexes) > 0 {
		rv.indexes = make([]string, 0, len(this.indexes))
		rv.indexes = append(rv.indexes, this.indexes...)
	}
	if len(this.oriIndexes) > 0 {
		rv.oriIndexes = make([]string, 0, len(this.oriIndexes))
		rv.oriIndexes = append(rv.oriIndexes, this.oriIndexes...)
	}
	return rv
}

func (this *HintNoIndex) Keyspace() string {
	return this.keyspace
}

func (this *HintNoIndex) Indexes() []string {
	return this.indexes
}

func (this *HintNoIndex) SetIndexes(indexes []string) {
	if this.oriIndexes == nil {
		this.oriIndexes = this.indexes
	}
	this.indexes = indexes
}

func (this *HintNoIndex) Derived() bool {
	return false
}

func (this *HintNoIndex) State() HintState {
	return this.state
}

func (this *HintNoIndex) SetFollowed() {
	if this.state == HINT_STATE_UNKNOWN {
		this.state = HINT_STATE_FOLLOWED
	}
}

func (this *HintNoIndex) SetNotFollowed() {
	if this.state == HINT_STATE_UNKNOWN {
		this.state = HINT_STATE_NOT_FOLLOWED
		if this.avoid {
			this.err = AVD_IDX_HINT_NOT_FOLLOWED
		} else {
			this.err = NO_INDEX_HINT_NOT_FOLLOWED
		}
	}
}

func (this *HintNoIndex) Error() string {
	return this.err
}

func (this *HintNoIndex) SetError(err string) {
	this.err = err
	this.state = HINT_STATE_ERROR
}

func (this *HintNoIndex) sortString() string {
	return fmt.Sprintf("%d%d%t%s%d%s", this.Type(), this.state, false, this.keyspace, len(this.indexes), this.err)
}

func (this *HintNoIndex) FormatHint(jsonStyle bool) string {
	if jsonStyle {
		key := "no_index"
		if this.avoid {
			key = "avoid_index"
		}
		hint := map[string]interface{}{
			key: this.formatJSON(),
		}
		bytes, _ := json.Marshal(hint)
		return string(bytes)
	}
	idxs := this.indexes
	if this.oriIndexes != nil {
		idxs = this.oriIndexes
	}
	args := make([]string, 0, len(idxs)+1)
	args = append(args, this.keyspace)
	args = append(args, idxs...)
	name := "NO_INDEX"
	if this.avoid {
		name = "AVOID_INDEX"
	}
	return formatHint(name, args)
}

func (this *HintNoIndex) formatJSON() map[string]interface{} {
	r := make(map[string]interface{}, 2)
	r["keyspace"] = this.keyspace
	idxs := this.indexes
	if this.oriIndexes != nil {
		idxs = this.oriIndexes
	}
	if len(idxs) > 0 {
		indexes := make([]interface{}, 0, len(idxs))
		for _, idx := range idxs {
			indexes = append(indexes, idx)
		}
		r["indexes"] = indexes
	}
	return r
}

type HintNoFTSIndex struct {
	keyspace   string
	indexes    []string
	oriIndexes []string
	state      HintState
	err        string
	avoid      bool
}

func NewNoFTSIndexHint(keyspace string, indexes []string, avoid bool) *HintNoFTSIndex {
	return &HintNoFTSIndex{
		keyspace: keyspace,
		indexes:  indexes,
		avoid:    avoid,
	}
}

func (this *HintNoFTSIndex) Type() HintType {
	return HINT_NO_INDEX_FTS
}

func (this *HintNoFTSIndex) Copy() OptimHint {
	rv := &HintNoFTSIndex{
		keyspace: this.keyspace,
		state:    this.state,
		err:      this.err,
		avoid:    this.avoid,
	}
	if len(this.indexes) > 0 {
		rv.indexes = make([]string, 0, len(this.indexes))
		rv.indexes = append(rv.indexes, this.indexes...)
	}
	if len(this.oriIndexes) > 0 {
		rv.oriIndexes = make([]string, 0, len(this.oriIndexes))
		rv.oriIndexes = append(rv.oriIndexes, this.oriIndexes...)
	}
	return rv
}

func (this *HintNoFTSIndex) Keyspace() string {
	return this.keyspace
}

func (this *HintNoFTSIndex) Indexes() []string {
	return this.indexes
}

func (this *HintNoFTSIndex) SetIndexes(indexes []string) {
	if this.oriIndexes == nil {
		this.oriIndexes = this.indexes
	}
	this.indexes = indexes
}

func (this *HintNoFTSIndex) Derived() bool {
	return false
}

func (this *HintNoFTSIndex) State() HintState {
	return this.state
}

func (this *HintNoFTSIndex) SetFollowed() {
	if this.state == HINT_STATE_UNKNOWN {
		this.state = HINT_STATE_FOLLOWED
	}
}

func (this *HintNoFTSIndex) SetNotFollowed() {
	if this.state == HINT_STATE_UNKNOWN {
		this.state = HINT_STATE_NOT_FOLLOWED
		if this.avoid {
			this.err = AVD_IDX_FTS_HINT_NOT_FOLLOWED
		} else {
			this.err = NO_INDEX_FTS_HINT_NOT_FOLLOWED
		}
	}
}

func (this *HintNoFTSIndex) Error() string {
	return this.err
}

func (this *HintNoFTSIndex) SetError(err string) {
	this.err = err
	this.state = HINT_STATE_ERROR
}

func (this *HintNoFTSIndex) sortString() string {
	return fmt.Sprintf("%d%d%t%s%d%s", this.Type(), this.state, false, this.keyspace, len(this.indexes), this.err)
}

func (this *HintNoFTSIndex) FormatHint(jsonStyle bool) string {
	if jsonStyle {
		key := "no_index_fts"
		if this.avoid {
			key = "avoid_index_fts"
		}
		hint := map[string]interface{}{
			key: this.formatJSON(),
		}
		bytes, _ := json.Marshal(hint)
		return string(bytes)
	}
	idxs := this.indexes
	if this.oriIndexes != nil {
		idxs = this.oriIndexes
	}
	args := make([]string, 0, len(idxs)+1)
	args = append(args, this.keyspace)
	args = append(args, idxs...)
	name := "NO_INDEX_FTS"
	if this.avoid {
		name = "AVOID_INDEX_FTS"
	}
	return formatHint(name, args)
}

func (this *HintNoFTSIndex) formatJSON() map[string]interface{} {
	r := make(map[string]interface{}, 2)
	r["keyspace"] = this.keyspace
	idxs := this.indexes
	if this.oriIndexes != nil {
		idxs = this.oriIndexes
	}
	if len(idxs) > 0 {
		indexes := make([]interface{}, 0, len(idxs))
		for _, idx := range idxs {
			indexes = append(indexes, idx)
		}
		r["indexes"] = indexes
	}
	return r
}

type HintIndexAll struct {
	keyspace string
	indexes  []string
	state    HintState
	err      string
}

func NewIndexAllHint(keyspace string, indexes []string) *HintIndexAll {
	return &HintIndexAll{
		keyspace: keyspace,
		indexes:  indexes,
	}
}

func (this *HintIndexAll) Type() HintType {
	return HINT_INDEX_ALL
}

func (this *HintIndexAll) Copy() OptimHint {
	rv := &HintIndexAll{
		keyspace: this.keyspace,
		state:    this.state,
		err:      this.err,
	}
	if len(this.indexes) > 0 {
		rv.indexes = make([]string, 0, len(this.indexes))
		rv.indexes = append(rv.indexes, this.indexes...)
	}
	return rv
}

func (this *HintIndexAll) Keyspace() string {
	return this.keyspace
}

func (this *HintIndexAll) Indexes() []string {
	return this.indexes
}

func (this *HintIndexAll) Derived() bool {
	return false
}

func (this *HintIndexAll) State() HintState {
	return this.state
}

func (this *HintIndexAll) SetFollowed() {
	if this.state == HINT_STATE_UNKNOWN {
		this.state = HINT_STATE_FOLLOWED
	}
}

func (this *HintIndexAll) SetNotFollowed() {
	if this.state == HINT_STATE_UNKNOWN {
		this.state = HINT_STATE_NOT_FOLLOWED
		this.err = INDEX_ALL_HINT_NOT_FOLLOWED
	}
}

func (this *HintIndexAll) Error() string {
	return this.err
}

func (this *HintIndexAll) SetError(err string) {
	this.err = err
	this.state = HINT_STATE_ERROR
}

func (this *HintIndexAll) sortString() string {
	return fmt.Sprintf("%d%d%t%s%d%s", this.Type(), this.state, false, this.keyspace, len(this.indexes), this.err)
}

func (this *HintIndexAll) FormatHint(jsonStyle bool) string {
	if jsonStyle {
		hint := map[string]interface{}{
			"index_all": this.formatJSON(),
		}
		bytes, _ := json.Marshal(hint)
		return string(bytes)
	}
	args := make([]string, 0, len(this.indexes)+1)
	args = append(args, this.keyspace)
	args = append(args, this.indexes...)
	return formatHint("INDEX_ALL", args)
}

func (this *HintIndexAll) formatJSON() map[string]interface{} {
	r := make(map[string]interface{}, 2)
	r["keyspace"] = this.keyspace
	if len(this.indexes) > 0 {
		indexes := make([]interface{}, 0, len(this.indexes))
		for _, idx := range this.indexes {
			indexes = append(indexes, idx)
		}
		r["indexes"] = indexes
	}
	return r
}

type HintNL struct {
	keyspace string
	derived  bool
	state    HintState
	err      string
}

func NewNLHint(keyspace string) *HintNL {
	return &HintNL{
		keyspace: keyspace,
	}
}

func NewDerivedNLHint(keyspace string) *HintNL {
	return &HintNL{
		keyspace: keyspace,
		derived:  true,
	}
}

func (this *HintNL) Type() HintType {
	return HINT_NL
}

func (this *HintNL) Copy() OptimHint {
	return &HintNL{
		keyspace: this.keyspace,
		derived:  this.derived,
		state:    this.state,
		err:      this.err,
	}
}

func (this *HintNL) Keyspace() string {
	return this.keyspace
}

func (this *HintNL) Derived() bool {
	return this.derived
}

func (this *HintNL) State() HintState {
	return this.state
}

func (this *HintNL) SetFollowed() {
	if this.state == HINT_STATE_UNKNOWN {
		this.state = HINT_STATE_FOLLOWED
	}
}

func (this *HintNL) SetNotFollowed() {
	if this.state == HINT_STATE_UNKNOWN {
		this.state = HINT_STATE_NOT_FOLLOWED
		this.err = USE_NL_HINT_NOT_FOLLOWED
	}
}

func (this *HintNL) Error() string {
	return this.err
}

func (this *HintNL) SetError(err string) {
	this.err = err
	this.state = HINT_STATE_ERROR
}

func (this *HintNL) sortString() string {
	return fmt.Sprintf("%d%d%t%s%s", this.Type(), this.state, this.derived, this.keyspace, this.err)
}

func (this *HintNL) FormatHint(jsonStyle bool) string {
	if jsonStyle {
		hint := map[string]interface{}{
			"use_nl": this.formatJSON(),
		}
		bytes, _ := json.Marshal(hint)
		return string(bytes)
	}
	return formatHint("USE_NL", []string{this.keyspace})
}

func (this *HintNL) formatJSON() map[string]interface{} {
	r := make(map[string]interface{}, 1)
	r["keyspace"] = this.keyspace
	return r
}

type HashOption int32

const (
	HASH_OPTION_NONE = HashOption(iota)
	HASH_OPTION_BUILD
	HASH_OPTION_PROBE
)

type HintHash struct {
	keyspace string
	option   HashOption
	derived  bool
	state    HintState
	err      string
}

func NewHashHint(keyspace string, option HashOption) *HintHash {
	return &HintHash{
		keyspace: keyspace,
		option:   option,
	}
}

func NewDerivedHashHint(keyspace string, option HashOption) *HintHash {
	return &HintHash{
		keyspace: keyspace,
		option:   option,
		derived:  true,
	}
}

func (this *HintHash) Type() HintType {
	return HINT_HASH
}

func (this *HintHash) Copy() OptimHint {
	return &HintHash{
		keyspace: this.keyspace,
		option:   this.option,
		derived:  this.derived,
		state:    this.state,
		err:      this.err,
	}
}

func (this *HintHash) Keyspace() string {
	return this.keyspace
}

func (this *HintHash) Option() HashOption {
	return this.option
}

func (this *HintHash) Derived() bool {
	return this.derived
}

func (this *HintHash) State() HintState {
	return this.state
}

func (this *HintHash) SetFollowed() {
	if this.state == HINT_STATE_UNKNOWN {
		this.state = HINT_STATE_FOLLOWED
	}
}

func (this *HintHash) SetNotFollowed() {
	if this.state == HINT_STATE_UNKNOWN {
		this.state = HINT_STATE_NOT_FOLLOWED
		this.err = USE_HASH_HINT_NOT_FOLLOWED
	}
}

func (this *HintHash) Error() string {
	return this.err
}

func (this *HintHash) SetError(err string) {
	this.err = err
	this.state = HINT_STATE_ERROR
}

func (this *HintHash) sortString() string {
	return fmt.Sprintf("%d%d%t%s%s", this.Type(), this.state, this.derived, this.keyspace, this.err)
}

func (this *HintHash) FormatHint(jsonStyle bool) string {
	if jsonStyle {
		hint := map[string]interface{}{
			"use_hash": this.formatJSON(),
		}
		bytes, _ := json.Marshal(hint)
		return string(bytes)
	}
	s := this.keyspace
	switch this.option {
	case HASH_OPTION_BUILD:
		s += "/BUILD"
	case HASH_OPTION_PROBE:
		s += "/PROBE"
	}
	return formatHint("USE_HASH", []string{s})
}

func (this *HintHash) formatJSON() map[string]interface{} {
	r := make(map[string]interface{}, 2)
	r["keyspace"] = this.keyspace
	switch this.option {
	case HASH_OPTION_BUILD:
		r["option"] = "BUILD"
	case HASH_OPTION_PROBE:
		r["option"] = "PROBE"
	}
	return r
}

type HintNoNL struct {
	keyspace string
	state    HintState
	err      string
	avoid    bool
}

func NewNoNLHint(keyspace string, avoid bool) *HintNoNL {
	return &HintNoNL{
		keyspace: keyspace,
		avoid:    avoid,
	}
}

func (this *HintNoNL) Type() HintType {
	return HINT_NO_NL
}

func (this *HintNoNL) Copy() OptimHint {
	return &HintNoNL{
		keyspace: this.keyspace,
		state:    this.state,
		err:      this.err,
		avoid:    this.avoid,
	}
}

func (this *HintNoNL) Keyspace() string {
	return this.keyspace
}

func (this *HintNoNL) Derived() bool {
	return false
}

func (this *HintNoNL) State() HintState {
	return this.state
}

func (this *HintNoNL) SetFollowed() {
	if this.state == HINT_STATE_UNKNOWN {
		this.state = HINT_STATE_FOLLOWED
	}
}

func (this *HintNoNL) SetNotFollowed() {
	if this.state == HINT_STATE_UNKNOWN {
		this.state = HINT_STATE_NOT_FOLLOWED
		if this.avoid {
			this.err = AVD_NL_HINT_NOT_FOLLOWED
		} else {
			this.err = NO_NL_HINT_NOT_FOLLOWED
		}
	}
}

func (this *HintNoNL) Error() string {
	return this.err
}

func (this *HintNoNL) SetError(err string) {
	this.err = err
	this.state = HINT_STATE_ERROR
}

func (this *HintNoNL) sortString() string {
	return fmt.Sprintf("%d%d%t%s%s", this.Type(), this.state, false, this.keyspace, this.err)
}

func (this *HintNoNL) FormatHint(jsonStyle bool) string {
	if jsonStyle {
		key := "no_use_nl"
		if this.avoid {
			key = "avoid_nl"
		}
		hint := map[string]interface{}{
			key: this.formatJSON(),
		}
		bytes, _ := json.Marshal(hint)
		return string(bytes)
	}
	name := "NO_USE_NL"
	if this.avoid {
		name = "AVOID_NL"
	}
	return formatHint(name, []string{this.keyspace})
}

func (this *HintNoNL) formatJSON() map[string]interface{} {
	r := make(map[string]interface{}, 1)
	r["keyspace"] = this.keyspace
	return r
}

type HintNoHash struct {
	keyspace string
	state    HintState
	err      string
	avoid    bool
}

func NewNoHashHint(keyspace string, avoid bool) *HintNoHash {
	return &HintNoHash{
		keyspace: keyspace,
		avoid:    avoid,
	}
}

func (this *HintNoHash) Type() HintType {
	return HINT_NO_HASH
}

func (this *HintNoHash) Copy() OptimHint {
	return &HintNoHash{
		keyspace: this.keyspace,
		state:    this.state,
		err:      this.err,
		avoid:    this.avoid,
	}
}

func (this *HintNoHash) Keyspace() string {
	return this.keyspace
}

func (this *HintNoHash) Derived() bool {
	return false
}

func (this *HintNoHash) State() HintState {
	return this.state
}

func (this *HintNoHash) SetFollowed() {
	if this.state == HINT_STATE_UNKNOWN {
		this.state = HINT_STATE_FOLLOWED
	}
}

func (this *HintNoHash) SetNotFollowed() {
	if this.state == HINT_STATE_UNKNOWN {
		this.state = HINT_STATE_NOT_FOLLOWED
		if this.avoid {
			this.err = AVD_HASH_HINT_NOT_FOLLOWED
		} else {
			this.err = NO_HASH_HINT_NOT_FOLLOWED
		}
	}
}

func (this *HintNoHash) Error() string {
	return this.err
}

func (this *HintNoHash) SetError(err string) {
	this.err = err
	this.state = HINT_STATE_ERROR
}

func (this *HintNoHash) sortString() string {
	return fmt.Sprintf("%d%d%t%s%s", this.Type(), this.state, false, this.keyspace, this.err)
}

func (this *HintNoHash) FormatHint(jsonStyle bool) string {
	if jsonStyle {
		key := "no_use_hash"
		if this.avoid {
			key = "avoid_hash"
		}
		hint := map[string]interface{}{
			key: this.formatJSON(),
		}
		bytes, _ := json.Marshal(hint)
		return string(bytes)
	}
	name := "NO_USE_HASH"
	if this.avoid {
		name = "AVOID_HASH"
	}
	return formatHint(name, []string{this.keyspace})
}

func (this *HintNoHash) formatJSON() map[string]interface{} {
	r := make(map[string]interface{}, 1)
	r["keyspace"] = this.keyspace
	return r
}

type HintJoinFilter struct {
	keyspace string
	state    HintState
	err      string
}

func NewJoinFilterHint(keyspace string) *HintJoinFilter {
	return &HintJoinFilter{
		keyspace: keyspace,
	}
}

func (this *HintJoinFilter) Type() HintType {
	return HINT_JOIN_FILTER
}

func (this *HintJoinFilter) Copy() OptimHint {
	return &HintJoinFilter{
		keyspace: this.keyspace,
		state:    this.state,
		err:      this.err,
	}
}

func (this *HintJoinFilter) Keyspace() string {
	return this.keyspace
}

func (this *HintJoinFilter) Derived() bool {
	return false
}

func (this *HintJoinFilter) State() HintState {
	return this.state
}

func (this *HintJoinFilter) SetFollowed() {
	if this.state == HINT_STATE_UNKNOWN {
		this.state = HINT_STATE_FOLLOWED
	}
}

func (this *HintJoinFilter) SetNotFollowed() {
	if this.state == HINT_STATE_UNKNOWN {
		this.state = HINT_STATE_NOT_FOLLOWED
		this.err = JOIN_FILTER_HINT_NOT_FOLLOWED
	}
}

func (this *HintJoinFilter) Error() string {
	return this.err
}

func (this *HintJoinFilter) SetError(err string) {
	this.err = err
	this.state = HINT_STATE_ERROR
}

func (this *HintJoinFilter) sortString() string {
	return fmt.Sprintf("%d%d%t%s%s", this.Type(), this.state, false, this.keyspace, this.err)
}

func (this *HintJoinFilter) FormatHint(jsonStyle bool) string {
	if jsonStyle {
		hint := map[string]interface{}{
			"join_filter": this.formatJSON(),
		}
		bytes, _ := json.Marshal(hint)
		return string(bytes)
	}
	return formatHint("JOIN_FILTER", []string{this.keyspace})
}

func (this *HintJoinFilter) formatJSON() map[string]interface{} {
	r := make(map[string]interface{}, 1)
	r["keyspace"] = this.keyspace
	return r
}

type HintNoJoinFilter struct {
	keyspace string
	state    HintState
	err      string
	avoid    bool
}

func NewNoJoinFilterHint(keyspace string, avoid bool) *HintNoJoinFilter {
	return &HintNoJoinFilter{
		keyspace: keyspace,
		avoid:    avoid,
	}
}

func (this *HintNoJoinFilter) Type() HintType {
	return HINT_NO_JOIN_FILTER
}

func (this *HintNoJoinFilter) Copy() OptimHint {
	return &HintNoJoinFilter{
		keyspace: this.keyspace,
		state:    this.state,
		err:      this.err,
		avoid:    this.avoid,
	}
}

func (this *HintNoJoinFilter) Keyspace() string {
	return this.keyspace
}

func (this *HintNoJoinFilter) Derived() bool {
	return false
}

func (this *HintNoJoinFilter) State() HintState {
	return this.state
}

func (this *HintNoJoinFilter) SetFollowed() {
	if this.state == HINT_STATE_UNKNOWN {
		this.state = HINT_STATE_FOLLOWED
	}
}

func (this *HintNoJoinFilter) SetNotFollowed() {
	if this.state == HINT_STATE_UNKNOWN {
		this.state = HINT_STATE_NOT_FOLLOWED
		if this.avoid {
			this.err = AVD_JN_FLTR_HINT_NOT_FOLLOWED
		} else {
			this.err = NO_JN_FLTR_HINT_NOT_FOLLOWED
		}
	}
}

func (this *HintNoJoinFilter) Error() string {
	return this.err
}

func (this *HintNoJoinFilter) SetError(err string) {
	this.err = err
	this.state = HINT_STATE_ERROR
}

func (this *HintNoJoinFilter) sortString() string {
	return fmt.Sprintf("%d%d%t%s%s", this.Type(), this.state, false, this.keyspace, this.err)
}

func (this *HintNoJoinFilter) FormatHint(jsonStyle bool) string {
	if jsonStyle {
		key := "no_join_filter"
		if this.avoid {
			key = "avoid_join_filter"
		}
		hint := map[string]interface{}{
			key: this.formatJSON(),
		}
		bytes, _ := json.Marshal(hint)
		return string(bytes)
	}
	name := "NO_JOIN_FILTER"
	if this.avoid {
		name = "AVOID_JOIN_FILTER"
	}
	return formatHint(name, []string{this.keyspace})
}

func (this *HintNoJoinFilter) formatJSON() map[string]interface{} {
	r := make(map[string]interface{}, 1)
	r["keyspace"] = this.keyspace
	return r
}

type HintOrdered struct {
	state HintState
	err   string
}

func NewOrderedHint() *HintOrdered {
	return &HintOrdered{}
}

func (this *HintOrdered) Type() HintType {
	return HINT_ORDERED
}

func (this *HintOrdered) Copy() OptimHint {
	return &HintOrdered{
		state: this.state,
		err:   this.err,
	}
}

func (this *HintOrdered) Derived() bool {
	return false
}

func (this *HintOrdered) State() HintState {
	return this.state
}

func (this *HintOrdered) SetFollowed() {
	if this.state == HINT_STATE_UNKNOWN {
		this.state = HINT_STATE_FOLLOWED
	}
}

func (this *HintOrdered) SetNotFollowed() {
	if this.state == HINT_STATE_UNKNOWN {
		this.state = HINT_STATE_NOT_FOLLOWED
		this.err = ORDERED_HINT_NOT_FOLLOWED
	}
}

func (this *HintOrdered) Error() string {
	return this.err
}

func (this *HintOrdered) SetError(err string) {
	this.err = err
	this.state = HINT_STATE_ERROR
}

func (this *HintOrdered) sortString() string {
	return fmt.Sprintf("%d%d%s", this.Type(), this.state, this.err)
}

func (this *HintOrdered) FormatHint(jsonStyle bool) string {
	if jsonStyle {
		bytes, _ := json.Marshal(this.formatJSON())
		return string(bytes)
	}
	return formatHint("ORDERED", nil)
}

func (this *HintOrdered) formatJSON() map[string]interface{} {
	r := make(map[string]interface{}, 1)
	r["ordered"] = true
	return r
}

type HintInvalid struct {
	input    string
	inputObj map[string]interface{}
	err      string
}

func NewInvalidHint(input string) *HintInvalid {
	return &HintInvalid{
		input: input,
	}
}

func NewInvalidJSONHint(r map[string]interface{}) *HintInvalid {
	return &HintInvalid{
		inputObj: r,
	}
}

func (this *HintInvalid) Type() HintType {
	return HINT_INVALID
}

func (this *HintInvalid) Copy() OptimHint {
	rv := &HintInvalid{
		input: this.input,
		err:   this.err,
	}
	if len(this.inputObj) > 0 {
		inputObj := make(map[string]interface{}, len(this.inputObj))
		for k, v := range this.inputObj {
			inputObj[k] = v
		}
		rv.inputObj = inputObj
	}
	return rv
}

func (this *HintInvalid) Input() string {
	return this.input
}

func (this *HintInvalid) InputObj() map[string]interface{} {
	return this.inputObj
}

func (this *HintInvalid) Derived() bool {
	return false
}

// invalid hint only has HINT_STATE_INVALID
func (this *HintInvalid) State() HintState {
	return HINT_STATE_INVALID
}

func (this *HintInvalid) SetFollowed() {
	// no-op
}

func (this *HintInvalid) SetNotFollowed() {
	// no-op
}

func (this *HintInvalid) Error() string {
	return this.err
}

func (this *HintInvalid) SetError(err string) {
	this.err = err
}

func (this *HintInvalid) sortString() string {
	return fmt.Sprintf("%d%d%s%s", this.Type(), this.State(), this.input, this.err)
}

func (this *HintInvalid) FormatHint(jsonStyle bool) string {
	if jsonStyle && len(this.inputObj) != 0 {
		bytes, _ := json.Marshal(this.inputObj)
		return string(bytes)
	}
	return this.input
}

func (this *HintInvalid) formatJSON() map[string]interface{} {
	return this.inputObj
}

func formatHint(hint_name string, hint_args []string) string {
	s := hint_name
	if hint_args != nil {
		s += "(" + strings.Join(hint_args, " ") + ")"
	}
	return s
}

func invalidHint(hint_name string, hint_args []string, err string) []OptimHint {
	return genInvalidHint(formatHint(hint_name, hint_args), err)
}

func genInvalidHint(input, err string) []OptimHint {
	hint := NewInvalidHint(input)
	hint.SetError(err)
	return []OptimHint{hint}
}

func genInvalidJSONHint(r map[string]interface{}, err string) []OptimHint {
	hint := NewInvalidJSONHint(r)
	hint.SetError(err)
	return []OptimHint{hint}
}

func InvalidOptimHints(input, err string) *OptimHints {
	return &OptimHints{
		hints:     genInvalidHint(input, err),
		jsonStyle: false,
	}
}

// JSON style hints

func ParseObjectHints(object expression.Expression) []OptimHint {
	if object == nil {
		return nil
	}

	val := object.Value()
	if val == nil || val.Type() != value.OBJECT {
		return nil
	}

	fields := val.Fields()
	optimHints := make([]OptimHint, 0, len(fields))
	for k, v := range fields {
		var hints []OptimHint
		var invalid bool
		var err string

		vval := value.NewValue(v)
		lowerKey := strings.ToLower(k)
		switch lowerKey {
		case "index":
			hints, invalid, err = newIndexHints(vval)
		case "index_fts":
			hints, invalid, err = newFTSIndexHints(vval)
		case "no_index":
			hints, invalid, err = newNoIndexHints(vval, false)
		case "avoid_index":
			hints, invalid, err = newNoIndexHints(vval, true)
		case "no_index_fts":
			hints, invalid, err = newNoFTSIndexHints(vval, false)
		case "avoid_index_fts":
			hints, invalid, err = newNoFTSIndexHints(vval, true)
		case "index_all", "index_combine":
			hints, invalid, err = newIndexAllHints(vval)
		case "use_nl":
			hints, invalid, err = newNLHints(vval)
		case "use_hash":
			hints, invalid, err = newHashHints(vval)
		case "no_use_nl":
			hints, invalid, err = newNoNLHints(vval, false)
		case "avoid_nl":
			hints, invalid, err = newNoNLHints(vval, true)
		case "no_use_hash":
			hints, invalid, err = newNoHashHints(vval, false)
		case "avoid_hash":
			hints, invalid, err = newNoHashHints(vval, true)
		case "join_filter", "px_join_filter":
			hints, invalid, err = newJoinFilterHints(vval)
		case "no_join_filter", "no_px_join_filter":
			hints, invalid, err = newNoJoinFilterHints(vval, false)
		case "avoid_join_filter":
			hints, invalid, err = newNoJoinFilterHints(vval, true)
		case "ordered":
			hints, invalid, err = newOrderedHint(vval)
		default:
			invalid = true
		}

		if invalid {
			r := map[string]interface{}{
				k: v,
			}
			if err == "" {
				err = INVALID_HINT
			}
			hints = genInvalidJSONHint(r, err)
		}

		if len(hints) > 0 {
			optimHints = append(optimHints, hints...)
		}
	}

	if len(optimHints) == 0 {
		return nil
	}

	// JSON-style hints do not have order for multiple hints, sort the hints
	// for explain purpose
	SortOptimHints(optimHints)
	return optimHints
}

func getIndexHintInfo(hint_name string, fields map[string]interface{}) (keyspace string, indexes []string, err string) {
	for k, v := range fields {
		key := strings.ToLower(k)
		if key == "keyspace" || key == "alias" {
			if keyspace != "" {
				err = DUPLICATED_KEYSPACE + hint_name
				return
			}
			keyspace = value.NewValue(v).ToString()
			if keyspace == "" {
				err = INVALID_KEYSPACE + hint_name
				return
			}
		} else if key == "indexes" {
			idxs := value.NewValue(v).Actual()
			switch idxs := idxs.(type) {
			case []interface{}:
				for _, idx := range idxs {
					name := value.NewValue(idx).ToString()
					if name == "" {
						err = INVALID_INDEX + hint_name
						return
					} else if strings.Contains(name, "/") {
						err = INVALID_SLASH + name
						return
					}
					indexes = append(indexes, name)
				}
			case nil:
				// if NULL is specified, ignore (no-op)
			default:
				name := value.NewValue(idxs).ToString()
				if name == "" {
					err = INVALID_INDEX + hint_name
					return
				} else if strings.Contains(name, "/") {
					err = INVALID_SLASH + name
					return
				}
				indexes = append(indexes, name)
			}
		} else {
			err = INVALID_FIELD + key
			return
		}
	}

	return
}

func newIndexHints(val value.Value) ([]OptimHint, bool, string) {
	return newHints(val, procIndexHints, false, false)
}

func newNoIndexHints(val value.Value, avoid bool) ([]OptimHint, bool, string) {
	return newHints(val, procIndexHints, true, avoid)
}

func procIndexHints(fields map[string]interface{}, negative, avoid bool) (OptimHint, bool, string) {
	hint_name := "index"
	if negative {
		if avoid {
			hint_name = "avoid_index"
		} else {
			hint_name = "no_index"
		}
	}

	var invalid bool
	keyspace, indexes, err := getIndexHintInfo(hint_name, fields)
	if err != "" {
		invalid = true
	} else if keyspace == "" {
		err = MISSING_ARG + hint_name
		invalid = true
	}
	if invalid {
		return nil, true, err
	}

	if negative {
		return NewNoIndexHint(keyspace, indexes, avoid), false, ""
	}
	return NewIndexHint(keyspace, indexes), false, ""
}

func newFTSIndexHints(val value.Value) ([]OptimHint, bool, string) {
	return newHints(val, procFTSIndexHints, false, false)
}

func newNoFTSIndexHints(val value.Value, avoid bool) ([]OptimHint, bool, string) {
	return newHints(val, procFTSIndexHints, true, avoid)
}

func procFTSIndexHints(fields map[string]interface{}, negative, avoid bool) (OptimHint, bool, string) {
	hint_name := "index_fts"
	if negative {
		if avoid {
			hint_name = "avoid_index_fts"
		} else {
			hint_name = "no_index_fts"
		}
	}

	var invalid bool
	keyspace, indexes, err := getIndexHintInfo(hint_name, fields)
	if err != "" {
		invalid = true
	} else if keyspace == "" {
		err = MISSING_ARG + hint_name
		invalid = true
	}
	if invalid {
		return nil, true, err
	}

	if negative {
		return NewNoFTSIndexHint(keyspace, indexes, avoid), false, ""
	}
	return NewFTSIndexHint(keyspace, indexes), false, ""
}

func newIndexAllHints(val value.Value) ([]OptimHint, bool, string) {
	return newHints(val, procIndexAllHints, false, false)
}

func procIndexAllHints(fields map[string]interface{}, negative, avoid bool) (OptimHint, bool, string) {
	hint_name := "index_all"

	var invalid bool
	keyspace, indexes, err := getIndexHintInfo(hint_name, fields)
	if err != "" {
		invalid = true
	} else if keyspace == "" {
		err = MISSING_ARG + hint_name
		invalid = true
	} else if len(indexes) < 2 {
		err = INDEX_ALL_SINGLE_INDEX
		invalid = true
	}
	if invalid {
		return nil, true, err
	}

	return NewIndexAllHint(keyspace, indexes), false, ""
}

func newNLHints(val value.Value) ([]OptimHint, bool, string) {
	return newHints(val, procNLHints, false, false)
}

func newNoNLHints(val value.Value, avoid bool) ([]OptimHint, bool, string) {
	return newHints(val, procNLHints, true, avoid)
}

func procNLHints(fields map[string]interface{}, negative, avoid bool) (OptimHint, bool, string) {
	hint_name := "use_nl"
	if negative {
		if avoid {
			hint_name = "avoid_nl"
		} else {
			hint_name = "no_use_nl"
		}
	}

	var keyspace, err string
	for k, v := range fields {
		key := strings.ToLower(k)
		if key == "keyspace" || key == "alias" {
			if keyspace != "" {
				err = DUPLICATED_KEYSPACE + hint_name
				break
			}
			keyspace = value.NewValue(v).ToString()
			if keyspace == "" {
				err = INVALID_KEYSPACE + hint_name
				break
			} else if strings.Contains(keyspace, "/") {
				err = INVALID_SLASH + keyspace
				break
			}
		} else {
			err = INVALID_FIELD + key
			break
		}
	}

	var invalid bool
	if err != "" {
		invalid = true
	} else if keyspace == "" {
		err = MISSING_ARG + hint_name
		invalid = true
	}
	if invalid {
		return nil, true, err
	}

	if negative {
		return NewNoNLHint(keyspace, avoid), false, ""
	}
	return NewNLHint(keyspace), false, ""
}

func newHashHints(val value.Value) ([]OptimHint, bool, string) {
	return newHints(val, procHashHints, false, false)
}

func newNoHashHints(val value.Value, avoid bool) ([]OptimHint, bool, string) {
	return newHints(val, procHashHints, true, avoid)
}

func procHashHints(fields map[string]interface{}, negative, avoid bool) (OptimHint, bool, string) {
	hint_name := "use_hash"
	if negative {
		if avoid {
			hint_name = "avoid_hash"
		} else {
			hint_name = "no_use_hash"
		}
	}

	var keyspace, err string
	var option HashOption = HASH_OPTION_NONE
	for k, v := range fields {
		key := strings.ToLower(k)
		if key == "keyspace" || key == "alias" {
			if keyspace != "" {
				err = DUPLICATED_KEYSPACE + hint_name
				break
			}
			keyspace = value.NewValue(v).ToString()
			if keyspace == "" {
				err = INVALID_KEYSPACE + hint_name
				break
			} else if strings.Contains(keyspace, "/") {
				err = INVALID_SLASH + keyspace
				break
			}
		} else if key == "option" {
			if negative {
				err = INVALID_NEG_HASH_OPTION
				break
			} else {
				op := strings.ToLower(value.NewValue(v).ToString())
				switch op {
				case "build":
					option = HASH_OPTION_BUILD
				case "probe":
					option = HASH_OPTION_PROBE
				case "null":
				// if null is specified, ignore (no-op)
				default:
					err = INVALID_HASH_OPTION + op
				}
				if err != "" {
					break
				}
			}
		} else {
			err = INVALID_FIELD + hint_name
			break
		}
	}

	var invalid bool
	if err != "" {
		invalid = true
	} else if keyspace == "" {
		err = MISSING_ARG + hint_name
		invalid = true
	}
	if invalid {
		return nil, true, err
	}

	if negative {
		return NewNoHashHint(keyspace, avoid), false, ""
	}
	return NewHashHint(keyspace, option), false, ""
}

func newJoinFilterHints(val value.Value) ([]OptimHint, bool, string) {
	return newHints(val, procJoinFilterHints, false, false)
}

func newNoJoinFilterHints(val value.Value, avoid bool) ([]OptimHint, bool, string) {
	return newHints(val, procJoinFilterHints, true, avoid)
}

func procJoinFilterHints(fields map[string]interface{}, negative, avoid bool) (OptimHint, bool, string) {
	hint_name := "join_filter"
	if negative {
		if avoid {
			hint_name = "avoid_join_filter"
		} else {
			hint_name = "no_join_filter"
		}
	}

	var keyspace, err string
	for k, v := range fields {
		key := strings.ToLower(k)
		if key == "keyspace" || key == "alias" {
			if keyspace != "" {
				err = DUPLICATED_KEYSPACE + hint_name
				break
			}
			keyspace = value.NewValue(v).ToString()
			if keyspace == "" {
				err = INVALID_KEYSPACE + hint_name
				break
			}
		} else {
			err = INVALID_FIELD + key
			break
		}
	}

	var invalid bool
	if err != "" {
		invalid = true
	} else if keyspace == "" {
		err = MISSING_ARG + hint_name
		invalid = true
	}
	if invalid {
		return nil, true, err
	}

	if negative {
		return NewNoJoinFilterHint(keyspace, avoid), false, ""
	}
	return NewJoinFilterHint(keyspace), false, ""
}

func newHints(val value.Value, procFunc func(map[string]interface{}, bool, bool) (OptimHint, bool, string), negative, avoid bool) ([]OptimHint, bool, string) {

	hints := make([]OptimHint, 0, 1)
	actual := val.Actual()
	switch actual := actual.(type) {
	case []interface{}:
		for _, a := range actual {
			ahints, invalid, aerr := newHints(value.NewValue(a), procFunc, negative, avoid)
			if invalid {
				return nil, true, aerr
			}
			if len(ahints) > 0 {
				hints = append(hints, ahints...)
			}
		}
	case map[string]interface{}:
		hint, invalid, err := procFunc(actual, negative, avoid)
		if invalid {
			return nil, true, err
		}
		if hint != nil {
			hints = append(hints, hint)
		}
	}

	return hints, false, ""
}

func newOrderedHint(val value.Value) ([]OptimHint, bool, string) {
	if val != nil && val.Truth() {
		return []OptimHint{NewOrderedHint()}, false, ""
	}
	return nil, true, ""
}

// when marshalling we put the optimizer hints in groups:
// hints_followed, hints_not_followed, invalid_hints
func (this *OptimHints) MarshalJSON() ([]byte, error) {
	var followed, not_followed, invalid, errored, unknown []interface{}
	for _, hint := range this.hints {
		obj := formatOptimHint(hint, this.jsonStyle)
		switch hint.State() {
		case HINT_STATE_FOLLOWED:
			followed = append(followed, obj)
		case HINT_STATE_NOT_FOLLOWED:
			not_followed = append(not_followed, obj)
		case HINT_STATE_ERROR:
			errored = append(errored, obj)
		case HINT_STATE_INVALID:
			invalid = append(invalid, obj)
		case HINT_STATE_UNKNOWN:
			unknown = append(unknown, obj)
		}
	}

	r := make(map[string]interface{}, 6)
	if len(followed) > 0 {
		r["hints_followed"] = followed
	}
	if len(not_followed) > 0 {
		r["hints_not_followed"] = not_followed
	}
	if len(errored) > 0 {
		r["hints_with_error"] = errored
	}
	if len(invalid) > 0 {
		r["invalid_hints"] = invalid
	}
	if len(unknown) > 0 {
		r["hints_status_unknown"] = unknown
	}

	if len(this.subqTermHints) > 0 {
		subqs := make([]interface{}, 0, len(this.subqTermHints))
		for _, subq := range this.subqTermHints {
			if subq != nil {
				subqs = append(subqs, subq)
			}
		}
		if len(subqs) > 0 {
			r["~from_clause_subqueries"] = subqs
		}
	}
	return json.Marshal(r)
}

func formatOptimHint(hint OptimHint, jsonStyle bool) interface{} {
	err := hint.Error()
	if jsonStyle {
		r := make(map[string]interface{}, 2)
		r["hint"] = hint.FormatHint(jsonStyle)
		if err != "" {
			r["error"] = err
		}
		return r
	}
	s := hint.FormatHint(false)
	if err != "" {
		s += ": " + err
	}
	return s
}

func SortOptimHints(hints []OptimHint) {
	sort.Slice(hints, func(i, j int) bool {
		return hints[i].sortString() < hints[j].sortString()
	})
}

type SubqOptimHints struct {
	alias string
	hints *OptimHints
}

func NewSubqOptimHints(alias string, hints *OptimHints) *SubqOptimHints {
	return &SubqOptimHints{
		alias: alias,
		hints: hints,
	}
}

func (this *SubqOptimHints) Alias() string {
	return this.alias
}

func (this *SubqOptimHints) MarshalJSON() ([]byte, error) {
	r := make(map[string]interface{}, 2)
	r["alias"] = this.alias
	r["optimizer_hints"] = this.hints
	return json.Marshal(r)
}
