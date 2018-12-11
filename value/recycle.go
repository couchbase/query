package value

func recycle(o interface{}) {
	if o == nil {
		return
	}

	// copiedObjectValue and copiedSliceValue do not own their elements.
	// Recycling can therefore stop here.
	_, ok := o.(copiedObjectValue)
	if ok {
		return
	}
	_, ok = o.(*copiedObjectValue)
	if ok {
		return
	}
	_, ok = o.(copiedSliceValue)
	if ok {
		return
	}
	_, ok = o.(*copiedSliceValue)
	if ok {
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
		return
	}

	// It's a JSON array.
	a, ok := o.([]interface{})
	if ok {
		for _, v := range a {
			recycle(v)
		}
		return
	}

	// Don't care about the other possibilities.
}
