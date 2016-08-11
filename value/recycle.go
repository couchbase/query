package value

import json "github.com/couchbase/go_json"

func recycle(o interface{}) {
	if o == nil {
		return
	}

	_, ok := o.(copiedObjectValue)
	if ok {
		// The copied object value is a map that does not own its elements.
		// Recycling can therefore stop right here. It does not need to go deeper.
		return
	}

	// Do we need to get at the base type?
	act, ok := o.(Value)
	if ok {
		o = act.Actual()
	}

	// It's a JSON object, a map.
	m, ok := o.(map[string]interface{})
	if ok {
		for _, v := range m {
			recycle(v)
		}
		json.RecycleMap(m)
		return
	}

	// It's a JSON array.
	a, ok := o.([]interface{})
	if ok {
		for _, v := range a {
			recycle(v)
		}
		json.RecycleArray(a)
		return
	}

	// Don't care about the other possibilities.
}
