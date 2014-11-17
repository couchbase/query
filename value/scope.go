//  Copieright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package value

/*
ScopeValue provides alias scoping for subqueries, ranging, LETs,
projections, etc. It is a type struct that inherits Value and 
has a parent Value.
*/
type ScopeValue struct {
	Value
	parent Value
}

/*
Return a pointer to a new ScopeValue populated using the input
arguments value and parent. 
*/
func NewScopeValue(val interface{}, parent Value) *ScopeValue {
	return &ScopeValue{
		Value:  NewValue(val),
		parent: parent,
	}
}

/*
Call the Values MarshalJSON implementation.
*/
func (this *ScopeValue) MarshalJSON() ([]byte, error) {
	return this.Value.MarshalJSON()
}

/*
Return a pointer to the ScopeValue, where the Value field
is the receivers value Copy and the parent is the receivers
parent.
*/
func (this *ScopeValue) Copy() Value {
	return &ScopeValue{
		Value:  this.Value.Copy(),
		parent: this.parent,
	}
}

/*
Return a pointer to the ScopeValue, where the Value field
calls the CopyForUpdate the receivers value, and the parent 
is assigned to the parent field in the receiver.
*/
func (this *ScopeValue) CopyForUpdate() Value {
	return &ScopeValue{
		Value:  this.Value.CopyForUpdate(),
		parent: this.parent,
	}
}

/*
Implements scoping. Checks field of the value in the receiver 
into result, and if valid returns the result.  If the parent 
is not nil call Field on the parent and return that. Else a 
missingField is returned. It searches itself and then the 
parent for the input parameter field.
*/
func (this *ScopeValue) Field(field string) (Value, bool) {
	result, ok := this.Value.Field(field)
	if ok {
		return result, true
	}

	if this.parent != nil {
		return this.parent.Field(field)
	}

	return missingField(field), false
}

/*
Flattens out the fields from the parent and value into a map
and returns it. If the parent in scopeValue is nil then, 
return the fields in the value. If not, create a map that 
contains both the parents’ fields and the values fields and return that map.
*/
func (this *ScopeValue) Fields() map[string]interface{} {
	if this.parent == nil {
		return this.Value.Fields()
	}

	rv := make(map[string]interface{})

	p := this.parent.Fields()
	for pf, pv := range p {
		rv[pf] = pv
	}

	v := this.Value.Fields()
	for vf, vv := range v {
		rv[vf] = vv
	}

	return rv
}

/*
Return the value field in the scopeValue struct.
*/
func (this *ScopeValue) GetValue() Value {
	return this.Value
}

/*
Return the value field in the scopeValue struct.
*/
func (this *ScopeValue) Parent() Value {
	return this.parent
}
