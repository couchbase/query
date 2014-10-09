//  Copyright (c) 2013 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package util

import (
	"testing"
)

func TestStack(t *testing.T) {
	s := Stack{}

	s.Push(1)
	s.Push(2)

	v2 := s.Pop()
	if v2 != 2 {
		t.Errorf("Expected 2, got %v", v2)
	}

	v1 := s.Pop()
	if v1 != 1 {
		t.Errorf("Expected 1, got %v", v1)
	}

	v0 := s.Pop()
	if v0 != nil {
		t.Errorf("Expected nil, got %v", v0)
	}
}
