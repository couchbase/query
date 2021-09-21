//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package algebra

import (
	"encoding/json"
	"strings"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
)

type HintType int32

const (
	HINT_INVALID = HintType(iota)
	HINT_INDEX
	HINT_FTS_INDEX
	HINT_NL
	HINT_HASH
	HINT_ORDERED
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
	INVALID_HINT            = "Invalid hint name: "
	MISSING_ARG             = "Missing argument for "
	EXTRA_ARG               = "Argument not expected: "
	INVALID_SLASH           = "Invalid '/' found in "
	EXTRA_SLASH             = "Extra '/' found in "
	INVALID_HASH_OPTION     = "Invalid hash option (BUILD or PROBE only):  "
	INVALID_KEYSPACE        = "Invalid keyspace specified: "
	DUPLICATED_JOIN_HINT    = "Duplciated join hint specified for keyspace: "
	DUPLICATED_INDEX_HINT   = "Duplicated index or index_fts hint specified for keyspace: "
	NON_KEYSPACE_INDEX_HINT = "Index or index_fts hint specified on non-keyspace: "
)

type OptimHint interface {
	Type() HintType
	FormatHint(jsonStyle bool) string
	State() HintState
	SetState(state HintState)
	Error() string
	SetError(err string)
	Copy() OptimHint
	Derived() bool
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
	case *HintNL:
		name = "use_nl"
		obj = hint.formatJSON()
	case *HintHash:
		name = "use_hash"
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

func NewOptimHint(hint_name string, hint_args []string) []OptimHint {
	var hints []OptimHint
	invalid := false
	var err string
	lowerName := strings.ToLower(hint_name)
	switch lowerName {
	case "index", "index_fts":
		fts := (lowerName == "index_fts")
		if len(hint_args) == 0 {
			invalid = true
			err = MISSING_ARG + hint_name
			break
		}
		// first arg is keyspace (alias)
		indexes := make(IndexRefs, 0, len(hint_args)-1)
		for i := 1; i < len(hint_args); i++ {
			if strings.Contains(hint_args[i], "/") {
				invalid = true
				err = INVALID_SLASH + hint_args[i]
				break
			}
			idxType := datastore.DEFAULT
			if fts {
				idxType = datastore.FTS
			}
			index := NewIndexRef(hint_args[i], idxType)
			indexes = append(indexes, index)
		}
		if !invalid {
			if fts {
				hints = []OptimHint{NewFTSIndexHint(hint_args[0], indexes)}
			} else {
				hints = []OptimHint{NewIndexHint(hint_args[0], indexes)}
			}
		}
	case "use_nl":
		// USE_NL hint must include at least 1 keyspsace
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
			hint := NewNLHint(arg)
			hints = append(hints, hint)
		}
	case "use_hash":
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
			logging.Infof("===== len(parts) %d =====", len(parts))
			if len(parts) > 2 {
				invalid = true
				err = EXTRA_SLASH + arg
			} else if len(parts) == 2 {
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
			} else if len(parts) == 1 {
				keyspace = parts[0]
				option = HASH_OPTION_NONE
			}

			if invalid {
				break
			}

			hint := NewHashHint(keyspace, option)
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
		err = INVALID_HINT + hint_name
	}

	if invalid || len(hints) == 0 {
		logging.Infof("===== return invalid, invalid %t len(hints) %d =====", invalid, len(hints))
		return invalidHint(hint_name, hint_args, err)
	}
	return hints
}

type HintIndex struct {
	keyspace string
	indexes  IndexRefs
	derived  bool
	state    HintState
	err      string
}

func NewIndexHint(keyspace string, indexes IndexRefs) *HintIndex {
	return &HintIndex{
		keyspace: keyspace,
		indexes:  indexes,
	}
}

// derived from USE INDEX
func NewDerivedIndexHint(keyspace string, indexes IndexRefs) *HintIndex {
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
		rv.indexes = make(IndexRefs, 0, len(this.indexes))
		for _, idx := range this.indexes {
			rv.indexes = append(rv.indexes, idx)
		}
	}
	return rv
}

func (this *HintIndex) Keyspace() string {
	return this.keyspace
}

func (this *HintIndex) Indexes() IndexRefs {
	return this.indexes
}

func (this *HintIndex) Derived() bool {
	return this.derived
}

func (this *HintIndex) State() HintState {
	return this.state
}

func (this *HintIndex) SetState(state HintState) {
	this.state = state
}

func (this *HintIndex) Error() string {
	return this.err
}

func (this *HintIndex) SetError(err string) {
	this.err = err
	this.state = HINT_STATE_ERROR
}

func (this *HintIndex) FormatHint(jsonStyle bool) string {
	if this.derived {
		return ""
	} else if jsonStyle {
		hint := map[string]interface{}{
			"index": this.formatJSON(),
		}
		bytes, _ := json.Marshal(hint)
		return string(bytes)
	}
	args := make([]string, 0, len(this.indexes)+1)
	args = append(args, this.keyspace)
	for _, idx := range this.indexes {
		args = append(args, idx.Name())
	}
	return formatHint("index", args)
}

func (this *HintIndex) formatJSON() map[string]interface{} {
	r := make(map[string]interface{}, 2)
	r["keyspace"] = this.keyspace
	if len(this.indexes) > 0 {
		indexes := make([]interface{}, 0, len(this.indexes))
		for _, idx := range this.indexes {
			indexes = append(indexes, idx.Name())
		}
		r["indexes"] = indexes
	}
	return r
}

type HintFTSIndex struct {
	keyspace string
	indexes  IndexRefs
	derived  bool
	state    HintState
	err      string
}

func NewFTSIndexHint(keyspace string, indexes IndexRefs) *HintFTSIndex {
	return &HintFTSIndex{
		keyspace: keyspace,
		indexes:  indexes,
	}
}

func NewDerivedFTSIndexHint(keyspace string, indexes IndexRefs) *HintFTSIndex {
	return &HintFTSIndex{
		keyspace: keyspace,
		indexes:  indexes,
		derived:  true,
	}
}

func (this *HintFTSIndex) Type() HintType {
	return HINT_FTS_INDEX
}

func (this *HintFTSIndex) Copy() OptimHint {
	rv := &HintFTSIndex{
		keyspace: this.keyspace,
		derived:  this.derived,
		state:    this.state,
		err:      this.err,
	}
	if len(this.indexes) > 0 {
		rv.indexes = make(IndexRefs, 0, len(this.indexes))
		for _, idx := range this.indexes {
			rv.indexes = append(rv.indexes, idx)
		}
	}
	return rv
}

func (this *HintFTSIndex) Keyspace() string {
	return this.keyspace
}

func (this *HintFTSIndex) Indexes() IndexRefs {
	return this.indexes
}

func (this *HintFTSIndex) Derived() bool {
	return this.derived
}

func (this *HintFTSIndex) State() HintState {
	return this.state
}

func (this *HintFTSIndex) SetState(state HintState) {
	this.state = state
}

func (this *HintFTSIndex) Error() string {
	return this.err
}

func (this *HintFTSIndex) SetError(err string) {
	this.err = err
	this.state = HINT_STATE_ERROR
}

func (this *HintFTSIndex) FormatHint(jsonStyle bool) string {
	if this.derived {
		return ""
	} else if jsonStyle {
		hint := map[string]interface{}{
			"index_fts": this.formatJSON(),
		}
		bytes, _ := json.Marshal(hint)
		return string(bytes)
	}
	args := make([]string, 0, len(this.indexes)+1)
	args = append(args, this.keyspace)
	for _, idx := range this.indexes {
		args = append(args, idx.Name())
	}
	return formatHint("index_fts", args)
}

func (this *HintFTSIndex) formatJSON() map[string]interface{} {
	r := make(map[string]interface{}, 2)
	r["keyspace"] = this.keyspace
	if len(this.indexes) > 0 {
		indexes := make([]interface{}, 0, len(this.indexes))
		for _, idx := range this.indexes {
			indexes = append(indexes, idx.Name())
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

func (this *HintNL) SetState(state HintState) {
	this.state = state
}

func (this *HintNL) Error() string {
	return this.err
}

func (this *HintNL) SetError(err string) {
	this.err = err
	this.state = HINT_STATE_ERROR
}

func (this *HintNL) FormatHint(jsonStyle bool) string {
	if this.derived {
		return ""
	} else if jsonStyle {
		hint := map[string]interface{}{
			"use_nl": this.formatJSON(),
		}
		bytes, _ := json.Marshal(hint)
		return string(bytes)
	}
	return formatHint("use_nl", []string{this.keyspace})
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

func (this *HintHash) SetState(state HintState) {
	this.state = state
}

func (this *HintHash) Error() string {
	return this.err
}

func (this *HintHash) SetError(err string) {
	this.err = err
	this.state = HINT_STATE_ERROR
}

func (this *HintHash) FormatHint(jsonStyle bool) string {
	if this.derived {
		return ""
	} else if jsonStyle {
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
	return formatHint("use_hash", []string{s})
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

func (this *HintOrdered) SetState(state HintState) {
	this.state = state
}

func (this *HintOrdered) Error() string {
	return this.err
}

func (this *HintOrdered) SetError(err string) {
	this.err = err
	this.state = HINT_STATE_ERROR
}

func (this *HintOrdered) FormatHint(jsonStyle bool) string {
	if jsonStyle {
		bytes, _ := json.Marshal(this.formatJSON())
		return string(bytes)
	}
	return formatHint("ordered", nil)
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

func (this *HintInvalid) SetState(state HintState) {
	// no-op
}

func (this *HintInvalid) Error() string {
	return this.err
}

func (this *HintInvalid) SetError(err string) {
	this.err = err
}

func (this *HintInvalid) FormatHint(jsonStyle bool) string {
	if jsonStyle && len(this.inputObj) != 0 {
		bytes, _ := json.Marshal(this.inputObj)
		return string(bytes)
	}
	return "invalid hint: " + this.input
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
	logging.Infof("===== JSON hints: %v =====", object)
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
		logging.Infof("===== key %v %T value %v %T =====", k, k, v, v)
		lowerKey := strings.ToLower(k)
		switch lowerKey {
		case "index":
			hints, invalid = newIndexHints(vval)
		case "index_fts":
			hints, invalid = newFTSIndexHints(vval)
		case "use_nl":
			hints, invalid = newNLHints(vval)
		case "use_hash":
			hints, invalid = newHashHints(vval)
		case "ordered":
			hints, invalid = newOrderedHint(vval)
		default:
			invalid = true
		}

		if invalid {
			r := map[string]interface{}{
				k: v,
			}
			hints = genInvalidJSONHint(r, INVALID_HINT+k)
		}

		if len(hints) > 0 {
			optimHints = append(optimHints, hints...)
		}
	}

	if len(optimHints) == 0 {
		return nil
	}
	return optimHints
}

func newIndexHints(val value.Value) ([]OptimHint, bool) {
	return newHints(val, procIndexHints)
}

func procIndexHints(fields map[string]interface{}) (OptimHint, bool) {
	invalid := false
	var keyspace string
	var indexes IndexRefs
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
					indexes = append(indexes, NewIndexRef(name, datastore.DEFAULT))
				}
			default:
				name := value.NewValue(idxs).ToString()
				if name == "" {
					return nil, true
				}
				indexes = append(indexes, NewIndexRef(name, datastore.DEFAULT))
			}
		} else {
			invalid = true
			break
		}
	}
	if invalid || keyspace == "" || len(indexes) == 0 {
		return nil, true
	}

	return NewIndexHint(keyspace, indexes), false
}

func newFTSIndexHints(val value.Value) ([]OptimHint, bool) {
	return newHints(val, procFTSIndexHints)
}

func procFTSIndexHints(fields map[string]interface{}) (OptimHint, bool) {
	invalid := false
	var keyspace string
	var indexes IndexRefs
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
					indexes = append(indexes, NewIndexRef(name, datastore.DEFAULT))
				}
			default:
				name := value.NewValue(idxs).ToString()
				if name == "" {
					return nil, true
				}
				indexes = append(indexes, NewIndexRef(name, datastore.DEFAULT))
			}
		} else {
			invalid = true
			break
		}
	}
	if invalid || keyspace == "" || len(indexes) == 0 {
		return nil, true
	}

	return NewFTSIndexHint(keyspace, indexes), false
}

func newNLHints(val value.Value) ([]OptimHint, bool) {
	return newHints(val, procNLHints)
}

func procNLHints(fields map[string]interface{}) (OptimHint, bool) {
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

	return NewNLHint(keyspace), false
}

func newHashHints(val value.Value) ([]OptimHint, bool) {
	return newHints(val, procHashHints)
}

func procHashHints(fields map[string]interface{}) (OptimHint, bool) {
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
			op := strings.ToLower(value.NewValue(v).ToString())
			switch op {
			case "build":
				option = HASH_OPTION_BUILD
			case "probe":
				option = HASH_OPTION_PROBE
			default:
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

	return NewHashHint(keyspace, option), false
}

func newHints(val value.Value, procFunc func(fields map[string]interface{}) (OptimHint, bool)) ([]OptimHint, bool) {

	hints := make([]OptimHint, 0, 1)
	actual := val.Actual()
	switch actual := actual.(type) {
	case []interface{}:
		for _, a := range actual {
			ahints, invalid := newHints(value.NewValue(a), procFunc)
			if invalid {
				return nil, true
			}
			if len(ahints) > 0 {
				hints = append(hints, ahints...)
			}
		}
	case map[string]interface{}:
		hint, invalid := procFunc(actual)
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
