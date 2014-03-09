//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

import (
	"encoding/base64"
	"fmt"

	"github.com/couchbaselabs/query/value"
)

type Base64Value struct {
	nAryBase
}

func NewBase64Value(args Expressions) Function {
	return &Base64Value{
		nAryBase{
			operands: args,
		},
	}
}

func (this *Base64Value) Formalize(forbidden, allowed value.Value,
	bucket string) (Expression, error) {
	if len(this.operands) > 0 {
		var e error
		this.operands[0], e = this.operands[0].Formalize(forbidden, allowed, bucket)
		if e != nil {
			return nil, e
		}
		return this, nil
	}

	if bucket == "" {
		return nil, fmt.Errorf("No default bucket for BASE64_VALUE().")
	}

	this.operands = Expressions{NewIdentifier(bucket)}
	return this, nil
}

func (this *Base64Value) evaluate(args value.Values) (value.Value, error) {
	av := args[0]
	if av.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	str := base64.StdEncoding.EncodeToString(av.Bytes())
	return value.NewValue(str), nil
}

func (this *Base64Value) MinArgs() int { return 0 }

func (this *Base64Value) MaxArgs() int { return 1 }

func (this *Base64Value) Constructor() FunctionConstructor { return NewBase64Value }

type Meta struct {
	nAryBase
}

func NewMeta(args Expressions) Function {
	return &Meta{
		nAryBase{
			operands: args,
		},
	}
}

func (this *Meta) Formalize(forbidden, allowed value.Value,
	bucket string) (Expression, error) {
	if len(this.operands) > 0 {
		var e error
		this.operands[0], e = this.operands[0].Formalize(forbidden, allowed, bucket)
		if e != nil {
			return nil, e
		}
		return this, nil
	}

	if bucket == "" {
		return nil, fmt.Errorf("No default bucket for META().")
	}

	this.operands = Expressions{NewIdentifier(bucket)}
	return this, nil
}

func (this *Meta) evaluate(args value.Values) (value.Value, error) {
	av := args[0]
	if av.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	switch av := av.(type) {
	case value.AnnotatedValue:
		return value.NewValue(av.GetAttachment("meta")), nil
	default:
		return value.NULL_VALUE, nil
	}
}

func (this *Meta) MinArgs() int { return 0 }

func (this *Meta) MaxArgs() int { return 1 }

func (this *Meta) Constructor() FunctionConstructor { return NewMeta }

type Value struct {
	nAryBase
}

func NewValue(args Expressions) Function {
	return &Value{
		nAryBase{
			operands: args,
		},
	}
}

func (this *Value) Formalize(forbidden, allowed value.Value,
	bucket string) (Expression, error) {
	if len(this.operands) > 0 {
		var e error
		this.operands[0], e = this.operands[0].Formalize(forbidden, allowed, bucket)
		if e != nil {
			return nil, e
		}
		return this, nil
	}

	if bucket == "" {
		return nil, fmt.Errorf("No default bucket for VALUE().")
	}

	this.operands = Expressions{NewIdentifier(bucket)}
	return this, nil
}

func (this *Value) evaluate(args value.Values) (value.Value, error) {
	return args[0], nil
}

func (this *Value) MinArgs() int { return 0 }

func (this *Value) MaxArgs() int { return 1 }

func (this *Value) Constructor() FunctionConstructor { return NewValue }
