//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	_ "fmt"
	_ "github.com/couchbaselabs/query/value"
)

type UpdateNode struct {
	bucket    *BucketNode          `json:"bucket"`
	keys      Expression           `json:"keys"`
	set       SetNodeList          `json:"set"`
	unset     UnsetNodeList        `json:"unset"`
	where     Expression           `json:"where"`
	limit     Expression           `json:"limit"`
	returning ResultExpressionList `json:"returning"`
}

type SetNode struct {
	path    Path         `json:"path"`
	value   Expression   `json:"value"`
	pathFor *PathForNode `json:"path-for"`
}

type SetNodeList []SetNode

type UnsetNode struct {
	path    Path         `json:"path"`
	pathFor *PathForNode `json:"path-for"`
}

type UnsetNodeList []UnsetNode

type PathForBinding struct {
	variable string
	path     Path
}

type PathForNode struct {
	bindings []*PathForBinding
	when     Expression
}
