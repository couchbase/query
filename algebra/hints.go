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
	INVALID_KEYSPACE               = "Invalid keyspace specified: "
	DUPLICATED_JOIN_HINT           = "Duplciated join hint specified for keyspace: "
	DUPLICATED_INDEX_HINT          = "Duplicated INDEX/NO_INDEX hint specified for keyspace: "
	DUPLICATED_INDEX_FTS_HINT      = "Duplicated INDEX_FTS/NO_INDEX_FTS hint specified for keyspace: "
	NON_KEYSPACE_INDEX_HINT        = "INDEX/NO_INDEX hint specified on non-keyspace: "
	NON_KEYSPACE_INDEX_FTS_HINT    = "INDEX_FTS/NO_INDEX_FTS hint specified on non-keyspace: "
	HASH_JOIN_NOT_AVAILABLE        = "Hash Join/Nest is not supported"
	INDEX_HINT_NOT_FOLLOWED        = "INDEX hint cannot be followed"
	INDEX_FTS_HINT_NOT_FOLLOWED    = "INDEX_FTS hint cannot be followed"
	USE_NL_HINT_NOT_FOLLOWED       = "USE_NL hint cannot be followed"
	USE_HASH_HINT_NOT_FOLLOWED     = "USE_HASH hint cannot be followed"
	ORDERED_HINT_NOT_FOLLOWED      = "ORDERED hint cannot be followed"
	NO_INDEX_HINT_NOT_FOLLOWED     = "NO_INDEX hint cannot be followed"
	NO_INDEX_FTS_HINT_NOT_FOLLOWED = "NO_INDEX_FTS hint cannot be followed"
	NO_HASH_HINT_NOT_FOLLOWED      = "NO_HASH hint cannot be followed"
	NO_NL_HINT_NOT_FOLLOWED        = "NO_NL hint cannot be followed"
	MERGE_ONKEY_JOIN_HINT_ERR      = "Join hint not supported in a MERGE statement with ON KEY clause"
	MERGE_ONKEY_INDEX_HINT_ERR     = "Index hint not supported for target keyspace in a MERGE statement with ON KEY clause"
	UPD_DEL_JOIN_HINT_ERR          = "Join hint not supported in an UPDATE or DELETE statement"
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
	hints     []OptimHint
	jsonStyle bool // JSON style hints
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
		obj = hint.formatJSON()
	case *HintNoFTSIndex:
		name = "no_index_fts"
		obj = hint.formatJSON()
	case *HintNL:
		name = "use_nl"
		obj = hint.formatJSON()
	case *HintHash:
		name = "use_hash"
		obj = hint.formatJSON()
	case *HintNoNL:
		name = "no_use_nl"
		obj = hint.formatJSON()
	case *HintNoHash:
		name = "no_use_hash"
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
	case "index", "index_fts", "no_index", "no_index_fts":
		fts := (lowerName == "index_fts") || (lowerName == "no_index_fts")
		negative := (lowerName == "no_index") || (lowerName == "no_index_fts")
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
		if !invalid {
			if fts {
				if negative {
					hints = []OptimHint{NewNoFTSIndexHint(hint_args[0], indexes)}
				} else {
					hints = []OptimHint{NewFTSIndexHint(hint_args[0], indexes)}
				}
			} else {
				if negative {
					hints = []OptimHint{NewNoIndexHint(hint_args[0], indexes)}
				} else {
					hints = []OptimHint{NewIndexHint(hint_args[0], indexes)}
				}
			}
		}
	case "use_nl", "no_use_nl":
		negative := (lowerName == "no_use_nl")
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
				hint = NewNoNLHint(arg)
			} else {
				hint = NewNLHint(arg)
			}
			hints = append(hints, hint)
		}
	case "use_hash", "no_use_hash":
		negative := (lowerName == "no_use_hash")
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
					err = EXTRA_SLASH + arg
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
				hint = NewNoHashHint(keyspace)
			} else {
				hint = NewHashHint(keyspace, option)
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
}

func NewNoIndexHint(keyspace string, indexes []string) *HintNoIndex {
	return &HintNoIndex{
		keyspace: keyspace,
		indexes:  indexes,
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
		this.err = NO_INDEX_HINT_NOT_FOLLOWED
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
		hint := map[string]interface{}{
			"no_index": this.formatJSON(),
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
	return formatHint("NO_INDEX", args)
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
}

func NewNoFTSIndexHint(keyspace string, indexes []string) *HintNoFTSIndex {
	return &HintNoFTSIndex{
		keyspace: keyspace,
		indexes:  indexes,
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
		this.err = NO_INDEX_FTS_HINT_NOT_FOLLOWED
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
		hint := map[string]interface{}{
			"no_index_fts": this.formatJSON(),
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
	return formatHint("NO_INDEX_FTS", args)
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
}

func NewNoNLHint(keyspace string) *HintNoNL {
	return &HintNoNL{
		keyspace: keyspace,
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
		this.err = NO_NL_HINT_NOT_FOLLOWED
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
		hint := map[string]interface{}{
			"no_use_nl": this.formatJSON(),
		}
		bytes, _ := json.Marshal(hint)
		return string(bytes)
	}
	return formatHint("NO_USE_NL", []string{this.keyspace})
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
}

func NewNoHashHint(keyspace string) *HintNoHash {
	return &HintNoHash{
		keyspace: keyspace,
	}
}

func (this *HintNoHash) Type() HintType {
	return HINT_NO_HASH
}

func (this *HintNoHash) Copy() OptimHint {
	return &HintHash{
		keyspace: this.keyspace,
		state:    this.state,
		err:      this.err,
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
		this.err = NO_HASH_HINT_NOT_FOLLOWED
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
		hint := map[string]interface{}{
			"no_use_hash": this.formatJSON(),
		}
		bytes, _ := json.Marshal(hint)
		return string(bytes)
	}
	return formatHint("NO_USE_HASH", []string{this.keyspace})
}

func (this *HintNoHash) formatJSON() map[string]interface{} {
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
		invalid := false

		vval := value.NewValue(v)
		lowerKey := strings.ToLower(k)
		switch lowerKey {
		case "index":
			hints, invalid = newIndexHints(vval)
		case "index_fts":
			hints, invalid = newFTSIndexHints(vval)
		case "no_index":
			hints, invalid = newNoIndexHints(vval)
		case "no_index_fts":
			hints, invalid = newNoFTSIndexHints(vval)
		case "use_nl":
			hints, invalid = newNLHints(vval)
		case "use_hash":
			hints, invalid = newHashHints(vval)
		case "no_use_nl":
			hints, invalid = newNoNLHints(vval)
		case "no_use_hash":
			hints, invalid = newNoHashHints(vval)
		case "ordered":
			hints, invalid = newOrderedHint(vval)
		default:
			invalid = true
		}

		if invalid {
			r := map[string]interface{}{
				k: v,
			}
			hints = genInvalidJSONHint(r, INVALID_HINT)
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

func newIndexHints(val value.Value) ([]OptimHint, bool) {
	return newHints(val, procIndexHints, false)
}

func newNoIndexHints(val value.Value) ([]OptimHint, bool) {
	return newHints(val, procIndexHints, true)
}

func procIndexHints(fields map[string]interface{}, negative bool) (OptimHint, bool) {
	invalid := false
	var keyspace string
	var indexes []string
	for k, v := range fields {
		key := strings.ToLower(k)
		if key == "keyspace" || key == "alias" {
			keyspace = value.NewValue(v).ToString()
			if keyspace == "" {
				invalid = true
			}
		} else if key == "indexes" {
			idxs := value.NewValue(v).Actual()
			switch idxs := idxs.(type) {
			case []interface{}:
				for _, idx := range idxs {
					name := value.NewValue(idx).ToString()
					if name == "" {
						return nil, true
					}
					indexes = append(indexes, name)
				}
			case nil:
				// if NULL is specified, ignore (no-op)
			default:
				name := value.NewValue(idxs).ToString()
				if name == "" {
					return nil, true
				}
				indexes = append(indexes, name)
			}
		} else {
			invalid = true
			break
		}
	}
	if invalid || keyspace == "" {
		return nil, true
	}

	if negative {
		return NewNoIndexHint(keyspace, indexes), false
	}
	return NewIndexHint(keyspace, indexes), false
}

func newFTSIndexHints(val value.Value) ([]OptimHint, bool) {
	return newHints(val, procFTSIndexHints, false)
}

func newNoFTSIndexHints(val value.Value) ([]OptimHint, bool) {
	return newHints(val, procFTSIndexHints, true)
}

func procFTSIndexHints(fields map[string]interface{}, negative bool) (OptimHint, bool) {
	invalid := false
	var keyspace string
	var indexes []string
	for k, v := range fields {
		key := strings.ToLower(k)
		if key == "keyspace" || key == "alias" {
			keyspace = value.NewValue(v).ToString()
			if keyspace == "" {
				invalid = true
			}
		} else if key == "indexes" {
			idxs := value.NewValue(v).Actual()
			switch idxs := idxs.(type) {
			case []interface{}:
				for _, idx := range idxs {
					name := value.NewValue(idx).ToString()
					if name == "" {
						return nil, true
					}
					indexes = append(indexes, name)
				}
			case nil:
				// if NULL is specified, ignore (no-op)
			default:
				name := value.NewValue(idxs).ToString()
				if name == "" {
					return nil, true
				}
				indexes = append(indexes, name)
			}
		} else {
			invalid = true
			break
		}
	}
	if invalid || keyspace == "" {
		return nil, true
	}

	if negative {
		return NewNoFTSIndexHint(keyspace, indexes), false
	}
	return NewFTSIndexHint(keyspace, indexes), false
}

func newNLHints(val value.Value) ([]OptimHint, bool) {
	return newHints(val, procNLHints, false)
}

func newNoNLHints(val value.Value) ([]OptimHint, bool) {
	return newHints(val, procNLHints, true)
}

func procNLHints(fields map[string]interface{}, negative bool) (OptimHint, bool) {
	invalid := false
	var keyspace string
	for k, v := range fields {
		key := strings.ToLower(k)
		if key == "keyspace" || key == "alias" {
			keyspace = value.NewValue(v).ToString()
			if keyspace == "" {
				invalid = true
			}
		} else {
			invalid = true
			break
		}
	}
	if invalid || keyspace == "" {
		return nil, true
	}

	if negative {
		return NewNoNLHint(keyspace), false
	}
	return NewNLHint(keyspace), false
}

func newHashHints(val value.Value) ([]OptimHint, bool) {
	return newHints(val, procHashHints, false)
}

func newNoHashHints(val value.Value) ([]OptimHint, bool) {
	return newHints(val, procHashHints, true)
}

func procHashHints(fields map[string]interface{}, negative bool) (OptimHint, bool) {
	invalid := false
	var keyspace string
	var option HashOption = HASH_OPTION_NONE
	for k, v := range fields {
		key := strings.ToLower(k)
		if key == "keyspace" || key == "alias" {
			keyspace = value.NewValue(v).ToString()
			if keyspace == "" {
				invalid = true
			}
		} else if key == "option" {
			if negative {
				invalid = true
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
					invalid = true
				}
			}
		} else {
			invalid = true
			break
		}
	}
	if invalid || keyspace == "" {
		return nil, true
	}

	if negative {
		return NewNoHashHint(keyspace), false
	}
	return NewHashHint(keyspace, option), false
}

func newHints(val value.Value, procFunc func(map[string]interface{}, bool) (OptimHint, bool), negative bool) ([]OptimHint, bool) {

	hints := make([]OptimHint, 0, 1)
	actual := val.Actual()
	switch actual := actual.(type) {
	case []interface{}:
		for _, a := range actual {
			ahints, invalid := newHints(value.NewValue(a), procFunc, negative)
			if invalid {
				return nil, true
			}
			if len(ahints) > 0 {
				hints = append(hints, ahints...)
			}
		}
	case map[string]interface{}:
		hint, invalid := procFunc(actual, negative)
		if invalid {
			return nil, true
		}
		if hint != nil {
			hints = append(hints, hint)
		}
	}

	return hints, false
}

func newOrderedHint(val value.Value) ([]OptimHint, bool) {
	if val != nil && val.Truth() {
		return []OptimHint{NewOrderedHint()}, false
	}
	return nil, true
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

	r := make(map[string]interface{}, 5)
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
