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

/*
Type AnnotatedChannel is a channel of AnnotatedValue.
*/
type AnnotatedChannel chan AnnotatedValue

/*
Type AnnotatedValues is a slice of AnnotatedValue.
*/
type AnnotatedValues []AnnotatedValue

/*
It is used to handle any extra information about the value.
The interface inherits from Value and extends it. It has
additional methods pertaining to the attachments.
*/
type AnnotatedValue interface {
	Value
	GetValue() Value
	Attachments() map[string]interface{}
	SetAttachments(atmts map[string]interface{})
	GetAttachment(key string) interface{}
	SetAttachment(key string, val interface{})
	RemoveAttachment(key string) interface{}
}

/*
Create an AnnotatedValue to hold attachments.
If the type of the value interface is AnnotatedValue,
then, return the value itself. If the type is Value,
set the Value to the value variable for struct
annotatedValue and attacher as a nil. A pointer to
the structure is returned. For the default behavior,
have it call itself again by creating a value from
the input interface and passing it into the function.
*/
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

/*
AnnotatedValue is defined as a structure with Value
and an attacher. It is used to represent JSON object
with additional metadata.
*/
type annotatedValue struct {
	Value
	attacher
}

/*
Call json.Marshal to encode the input value.
*/
func (this *annotatedValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.Value)
}

/*
Return a pointer to the structure annotatedValue with the
value attribute being the return value of the Copy call.
For the second variable defined in the struct, namely
attacher call the copyMap function with the receivers
attachments and self, to copy the attachments.
*/
func (this *annotatedValue) Copy() Value {
	return &annotatedValue{
		Value:    this.Value.Copy(),
		attacher: attacher{copyMap(this.attacher.attachments, self)},
	}
}

/*
Return a pointer to the structure annotatedValue with the
value attribute being the return value of the CopyForUpdate
call. The attachments field is populated by the receivers
attachments.
*/
func (this *annotatedValue) CopyForUpdate() Value {
	return &annotatedValue{
		Value:    this.Value.CopyForUpdate(),
		attacher: attacher{this.attacher.attachments},
	}
}

/*
Return the value component of the receiver
annotatedValue.
*/
func (this *annotatedValue) GetValue() Value {
	return this.Value
}

/*
It represents the metadata. It is a struct that contains
a map.
*/
type attacher struct {
	attachments map[string]interface{}
}

/*
Populate the receivers attachment field by setting
it to the input argument map.
*/
func (this *attacher) SetAttachments(atmts map[string]interface{}) {
	this.attachments = atmts
}

/*
Used to access the attachments, by returning the attachments
from the receiver this.
*/
func (this *attacher) Attachments() map[string]interface{} {
	return this.attachments
}

/*
Return the object attached to this Value with this key.
If no object is attached with this key, nil is returned.
*/
func (this *attacher) GetAttachment(key string) interface{} {
	if this.attachments != nil {
		return this.attachments[key]
	}
	return nil
}

/*
Attach an arbitrary object to this Value with the specified key.
Any existing value attached with this same key will be overwritten.
*/
func (this *attacher) SetAttachment(key string, val interface{}) {
	if this.attachments == nil {
		this.attachments = make(map[string]interface{})
	}
	this.attachments[key] = val
}

/*
Remove an object attached to this Value with this key.  If there
had been an object attached to this Value with this key it is
returned, otherwise nil.
*/
func (this *attacher) RemoveAttachment(key string) interface{} {
	var rv interface{}
	if this.attachments != nil {
		rv = this.attachments[key]
		delete(this.attachments, key)
	}
	return rv
}
