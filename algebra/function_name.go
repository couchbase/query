//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

type FunctionName interface {
	Name() string
	Key() string
	SetComponents(components []string)
	Components() []string
}

// A global storage function name
type globalName struct {
	namespace string `json:"namespace"`
	name      string `json:"name"`
}

func NewGlobalFunctionName(namespace, name string) FunctionName {
	return &globalName{
		namespace: namespace,
		name:      name,
	}
}

func (name *globalName) Name() string {
	return name.name
}

func (name *globalName) Key() string {
	return name.namespace + ":" + name.name
}

func (name *globalName) SetComponents(components []string) {
	if name.namespace == "" {
		name.namespace = components[0]
	}
}

func (name *globalName) Components() []string {
	return []string{name.namespace, name.name}
}
