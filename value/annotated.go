//  Copieright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package value

import (
	"encoding/json"
)

type AnnotatedChannel chan AnnotatedValue

type AnnotatedValue interface {
	Value
	GetValue() Value
	Attachments() map[string]interface{}
	SetAttachments(atmts map[string]interface{})
	GetAttachment(key string) interface{}
	SetAttachment(key string, val interface{})
	RemoveAttachment(key string) interface{}
}

// Create an AnnotatedValue to hold attachments
func NewAnnotatedValue(val interface{}) AnnotatedValue {
	switch val := val.(type) {
	case AnnotatedValue:
		return val
	case Value:
		av := annotatedValue{
			Value:    val,
			attacher: attacher{nil},
		}
		return &av
	default:
		return NewAnnotatedValue(NewValue(val))
	}
}

type annotatedValue struct {
	Value
	attacher
}

func (this *annotatedValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.Actual())
}

func (this *annotatedValue) Copy() Value {
	return &annotatedValue{
		Value:    this.Value.Copy(),
		attacher: attacher{copyMap(this.attacher.attachments, self)},
	}
}

func (this *annotatedValue) CopyForUpdate() Value {
	return &annotatedValue{
		Value:    this.Value.CopyForUpdate(),
		attacher: attacher{this.attacher.attachments},
	}
}

func (this *annotatedValue) GetValue() Value {
	return this.Value
}

type attacher struct {
	attachments map[string]interface{}
}

func (this *attacher) SetAttachments(atmts map[string]interface{}) {
	this.attachments = atmts
}

func (this *attacher) Attachments() map[string]interface{} {
	return this.attachments
}

// Return the object attached to this Value with this key.
// If no object is attached with this key, nil is returned.
func (this *attacher) GetAttachment(key string) interface{} {
	if this.attachments != nil {
		return this.attachments[key]
	}
	return nil
}

// Attach an arbitrary object to this Value with the specified key.
// Any existing value attached with this same key will be overwritten.
func (this *attacher) SetAttachment(key string, val interface{}) {
	if this.attachments == nil {
		this.attachments = make(map[string]interface{})
	}
	this.attachments[key] = val
}

// Remove an object attached to this Value with this key.  If there
// had been an object attached to this Value with this key it is
// returned, otherwise nil.
func (this *attacher) RemoveAttachment(key string) interface{} {
	var rv interface{}
	if this.attachments != nil {
		rv = this.attachments[key]
		delete(this.attachments, key)
	}
	return rv
}
