//  Copyright (c) 2018 Couchbase, Inc.
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

func TestSeaHash(t *testing.T) {
	s := SeaHashSum64([]byte("to be or not to be"))

	if s != 0x1b993a826f4ae575 {
		t.Errorf("Expected 0x1b993a826f4ae575, got %x", s)
	}

	s = SeaHashSum64([]byte(""))

	if s != 0xc920ca43256fdcb9 {
		t.Errorf("Expected 0xc920ca43256fdcb9, got %x", s)
	}

	s = SeaHashSum64([]byte("couchbase"))

	if s != 0x4e5d5535342df6ef {
		t.Errorf("Expected 0x, got %x4e5d5535342df6ef", s)
	}

	s = SeaHashSum64([]byte("12345678"))

	if s != 0x79476d25d4c6dfc4 {
		t.Errorf("Expected 0x79476d25d4c6dfc4, got %x", s)
	}

	s = SeaHashSum64([]byte("Couchbase N1QL"))

	if s != 0x682b2cc0145769e3 {
		t.Errorf("Expected 0x682b2cc0145769e3, got %x", s)
	}
}
