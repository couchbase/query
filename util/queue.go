//  Copyright (c) 2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package util

////////////////////////////////////////////////////////////
//
// Copied from https://gist.github.com/moraes/2141121
//
// with gratitude.
//
////////////////////////////////////////////////////////////

// Queue is a basic FIFO queue based on a circular list that resizes as needed.
type Queue struct {
	nodes []interface{}
	head  int
	tail  int
	count int
}

func NewQueue(size int) *Queue {
	if size < 1 {
		size = 1
	}

	rv := &Queue{
		nodes: make([]interface{}, size),
	}

	return rv
}

// Add a node to the queue.
func (q *Queue) Add(n interface{}) {
	if q.head == q.tail && q.count > 0 {
		nodes := make([]interface{}, 2*len(q.nodes))
		copy(nodes, q.nodes[q.head:])
		copy(nodes[len(q.nodes)-q.head:], q.nodes[:q.head])
		q.head = 0
		q.tail = len(q.nodes)
		q.nodes = nodes
	}

	q.nodes[q.tail] = n
	q.tail = (q.tail + 1) % len(q.nodes)
	q.count++
}

// Remove and return a node from the queue in FIFO order.
func (q *Queue) Remove() interface{} {
	if q.count == 0 {
		return nil
	}

	node := q.nodes[q.head]
	q.nodes[q.head] = nil
	q.head = (q.head + 1) % len(q.nodes)
	q.count--
	return node
}

// Remove and return a node from the queue in FIFO order.
func (q *Queue) Peek() interface{} {
	if q.count == 0 {
		return nil
	}

	return q.nodes[q.head]
}

func (q *Queue) Capacity() int {
	return len(q.nodes)
}

func (q *Queue) Size() int {
	return q.count
}

func (q *Queue) Clear() {
	n := len(q.nodes)

	if q.tail < q.head {
		q.tail += n
	}

	for i := q.head; i <= q.tail; i++ {
		q.nodes[i%n] = nil
	}
}
