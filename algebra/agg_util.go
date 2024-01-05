//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"fmt"
	"sort"

	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
Capacity for object sets.
*/
var _OBJECT_CAP = 64

/*
Capacity for List initialization
*/
var _INITIAL_LIST_SIZE = 16

/*
Constant subarray length for function medianofMedian()
*/
var _MEDIAN_SUB_LENGTH = 5

/*
Add input item to the cumulative set. Get the set. If
no errors enountered add the item to the set and return
it. If set has not been initialized yet, create a new set
with capacity _OBJECT_CAP and add the item. Return the
set value.
*/
func setAdd(item, cumulative value.Value, numeric bool) value.AnnotatedValue {
	av, ok := cumulative.(value.AnnotatedValue)
	if !ok {
		av = value.NewAnnotatedValue(cumulative)
	}

	set, e := getSet(av)
	if e == nil {
		set.Add(item)
		return av
	}

	set = value.NewSet(_OBJECT_CAP, true, numeric)
	set.Add(item)
	av.SetAttachment(value.ATT_SET, set)
	return av
}

/*
Aggregate distinct intermediate results and return them.
*/
func cumulateSets(part, cumulative value.Value) (value.AnnotatedValue, error) {
	pset, e := getSet(part)
	if e != nil {
		return nil, e
	}

	cset, e := getSet(cumulative)
	if e != nil {
		return nil, e
	}

	// Add smaller set to bigger
	var smaller, bigger *value.Set
	if pset.Len() <= cset.Len() {
		smaller, bigger = pset, cset
	} else {
		smaller, bigger = cset, pset
	}

	for _, v := range smaller.Values() {
		bigger.Add(v)
	}

	av, ok := cumulative.(value.AnnotatedValue)
	if !ok {
		return nil, fmt.Errorf("Invalid cumulative value, not an AnnotatedValue: %v", cumulative)
	}

	av.SetAttachment(value.ATT_SET, bigger)
	return av, nil
}

/*
Retrieve the set for annotated values. If the attachment type
is not a set, then throw an invalid distinct set error and
return.
*/
func getSet(item value.Value) (*value.Set, error) {
	switch item := item.(type) {
	case value.AnnotatedValue:
		ps := item.GetAttachment(value.ATT_SET)
		switch ps := ps.(type) {
		case *value.Set:
			return ps, nil
		default:
			return nil, fmt.Errorf("Invalid DISTINCT set %v of type %T.", ps, ps)
		}
	default:
		return nil, fmt.Errorf("Invalid DISTINCT %v of type %T.", item, item)
	}
}

/*
Add input item to the cumulative list. Get the list. If
no errors encountered, add the item to the list and return
it. If list has not been initialized yet, create a new list
with capacity _INITIAL_LIST_SIZE and add the item. Return the
list value.
*/
func listAdd(item, comulative value.Value) value.AnnotatedValue {
	av, ok := comulative.(value.AnnotatedValue)

	if !ok {
		av = value.NewAnnotatedValue(comulative)
	}
	list, e := getList(av)
	if e == nil {
		list.Add(item)
		return av
	}
	list = value.NewList(_INITIAL_LIST_SIZE)
	list.Add(item)
	av.SetAttachment(value.ATT_LIST, list)
	return av
}

/*
Retrieve the list for annotated values. If the attachment type
is not a list, then throw an invalid list error and
return.
*/
func getList(item value.Value) (*value.List, error) {
	switch item := item.(type) {
	case value.AnnotatedValue:
		ps := item.GetAttachment(value.ATT_LIST)
		switch ps := ps.(type) {
		case *value.List:
			return ps, nil
		default:
			return nil, fmt.Errorf("Invalid list %v of type %T.", ps, ps)
		}
	default:
		return nil, fmt.Errorf("Invalid %v of type %T.", item, item)
	}
}

/*
Aggregate intermediate results and return them.
*/
func cumulateLists(part, cumulative value.Value) (value.AnnotatedValue, error) {
	pList, e := getList(part)
	if e != nil {
		return nil, e
	}

	cList, e := getList(cumulative)
	if e != nil {
		return nil, e
	}
	cList.Union(pList)
	av, _ := cumulative.(value.AnnotatedValue)
	return av, nil
}

/*
Linear time algorithm to compute the kth smallest value in an unsorted list.
It devides the list into sublists of size 5 and finds the approximate median in each of the sublists.
Put these medians in a list and find the median of it as the pivot.
Use this pivot to partition the original unsorted list recursively to get the kth smallest value.
Time complexity is O(n): https://www.ics.uci.edu/~eppstein/161/960130.html
*/
func medianOfMedian(data []value.Value, k int, even bool) value.Value {

	n := len(data)

	if n <= 2*_MEDIAN_SUB_LENGTH {
		dataCopy := value.NewValue(data)
		sort.Sort(value.NewSorter(dataCopy))
		array := dataCopy.Actual()
		switch array := array.(type) {
		case []interface{}:
			if even {
				f := (array[k].(value.NumberValue).Float64() + array[k-1].(value.NumberValue).Float64()) / 2.0
				return value.NewValue(f)
			}
			return value.NewValue(array[k-1])

		}
	}

	m := n / _MEDIAN_SUB_LENGTH
	medians := make([]value.Value, m)
	var arr []value.Value

	for i := 0; i < m; i++ {

		j := (i * _MEDIAN_SUB_LENGTH) + _MEDIAN_SUB_LENGTH
		arr = nil

		if j >= n {
			arr = make([]value.Value, len(data[(i*_MEDIAN_SUB_LENGTH):]))
			copy(arr, data[(i*_MEDIAN_SUB_LENGTH):])
		} else {
			arr = make([]value.Value, _MEDIAN_SUB_LENGTH)
			copy(arr, data[(i*_MEDIAN_SUB_LENGTH):j])
		}

		v := medianOfMedian(arr, (len(arr)+1)/2, false)
		medians[i] = v
	}

	pivot := medianOfMedian(medians, (m+1)/2, false)
	var left, right []value.Value
	left = make([]value.Value, 0, n/2)
	right = make([]value.Value, 0, n/2)

	for i := range data {
		if pivot.(value.NumberValue).Float64() < data[i].(value.NumberValue).Float64() {
			right = append(right, data[i])
		} else if pivot.(value.NumberValue).Float64() > data[i].(value.NumberValue).Float64() {
			left = append(left, data[i])
		}
	}

	t := n - len(left) - len(right)
	switch {
	case k > len(left) && k <= n-len(right):
		if !even || (even && k+1 <= n-len(right)) {
			return pivot
		} else {
			second := medianOfMedian(right, 1, false)
			avg := (pivot.(value.NumberValue).Float64() + second.(value.NumberValue).Float64()) / 2.0
			return value.NewValue(avg)
		}

	case k <= len(left):
		if t > 0 {
			left = append(left, pivot)
		}
		return medianOfMedian(left, k, even)
	default:
		if t > 0 {
			right = append(right, pivot)
			t--
		}
		return medianOfMedian(right, k-len(left)-t, even)
	}
}

/*
Aggregate initial results for standard deviation.
Flag distinct help to specify stddev(...) and stddev(DISTINCT...)
*/
func addStddevVariance(item, cumulative value.Value, distinct bool) (value.Value, error) {
	var cSum float64
	cv, ok := cumulative.(value.AnnotatedValue)

	if !ok {
		cv = value.NewAnnotatedValue(cumulative)
		if distinct {
			cv = setAdd(item, cv, true)
		} else {
			cv = listAdd(item, cv)
		}
	} else {
		cSum = cv.GetAttachment(value.ATT_SUM).(value.NumberValue).Float64()
		if distinct {
			set, e := getSet(cv)
			if e != nil {
				return nil, e
			}
			if set.Has(item) {
				return cumulative, nil
			}
			set.Add(item)
		} else {
			listAdd(item, cumulative)
		}
	}

	cSum += item.(value.NumberValue).Float64()
	cv.SetAttachment(value.ATT_SUM, value.NewValue(cSum))
	return cv, nil
}

/*
Aggregate intermediate results for standard deviation.
Flag distinct help to specify stddev(...) and stddev(DISTINCT ...).
*/
func cumulateStddevVariance(part, cumulative value.Value, distinct bool) (value.Value, error) {
	if part == value.NULL_VALUE {
		return cumulative, nil
	} else if cumulative == value.NULL_VALUE {
		return part, nil
	}

	cv, _ := cumulative.(value.AnnotatedValue)
	cSum := cv.GetAttachment(value.ATT_SUM).(value.NumberValue).Float64()

	if distinct {
		pSet, e := getSet(part)
		if e != nil {
			return nil, e
		}

		cSet, e := getSet(cumulative)
		if e != nil {
			return nil, e
		}
		for _, v := range pSet.Values() {
			if !cSet.Has(v) {
				cSet.Add(v)
				cSum += v.(value.NumberValue).Float64()
			}
		}
	} else {
		pList, e := getList(part)
		if e != nil {
			return nil, e
		}

		cList, e := getList(cumulative)
		if e != nil {
			return nil, e
		}
		cSum += part.(value.AnnotatedValue).GetAttachment(value.ATT_SUM).(value.NumberValue).Float64()
		cList.Union(pList)
	}

	cv.SetAttachment(value.ATT_SUM, value.NewValue(cSum))
	return cumulative, nil
}

/*
Function to compute variance, population and sample variance can be returned
by setting delta to 0.0 and 1.0 respectively.
If arithmatic overflow happens, return +Infinity.
*/
func computeVariance(cumulative value.Value, distinct, samp bool, delta float64) (value.Value, error) {
	var count float64
	var values value.Values

	if distinct {
		set, e := getSet(cumulative)
		if e != nil {
			return nil, e
		}
		count = float64(set.Len())
		values = set.Values()
	} else {
		list, e := getList(cumulative)
		if e != nil {
			return nil, e
		}
		count = float64(list.Len())
		values = list.Values()
	}

	if count == 0.0 {
		return value.NULL_VALUE, nil
	}
	if count == 1.0 {
		if samp {
			return value.NULL_VALUE, nil
		}
		return value.ZERO_NUMBER, nil
	}

	sum := cumulative.(value.AnnotatedValue).GetAttachment(value.ATT_SUM).(value.NumberValue).Float64()
	mean := float64(sum / count)
	var variance float64

	for _, v := range values {
		f := v.(value.NumberValue).Float64() - mean
		variance += f * f
	}

	return value.NewValue(variance / (count - delta)), nil
}

/*
Return Window attachment
*/
func getWindowAttachment(item value.Value, name string) (value.Value, error) {
	switch item := item.(type) {
	case value.AnnotatedValue:
		ps := item.GetAttachment(value.ATT_WINDOW_ATTACHMENT)
		switch ps := ps.(type) {
		case value.Value:
			return ps, nil
		default:
			return nil, fmt.Errorf("Invalid %s %v of type %T.", name, item, item)
		}
	default:
		return nil, fmt.Errorf("Invalid %s %v of type %T.", name, item, item)
	}
}

/*
Return list attachment and startpos attachments
*/

func getNthValues(aggname string, cumpart value.Value, valfunc bool) (*value.List, int, error) {
	list, e := getList(cumpart)
	if e != nil {
		return nil, 0, fmt.Errorf("Invalid %s %v of type %T.", aggname, cumpart.Actual(), cumpart.Actual())
	}

	if !valfunc {
		return list, 0, nil
	}

	switch item := cumpart.(type) {
	case value.AnnotatedValue:
		ps := item.GetAttachment(value.ATT_STARTPOS)
		switch ps := ps.(type) {
		case value.NumberValue:
			return list, int(ps.Int64()), nil
		default:
			return nil, 0, fmt.Errorf("Invalid %s %v of type %T.", aggname, cumpart.Actual(), cumpart.Actual())
		}
	default:
		return nil, 0, fmt.Errorf("Invalid %s %v of type %T.", aggname, cumpart.Actual(), cumpart.Actual())
	}
}

/*
 Place the item value in the list array at right place.
    Handles RESPECT|IGNORE NULLS
    list array size to maxium nitems
    It inserts item in the order from startpos (i.e start)
*/

func compute_nth_value(item, cumpart value.Value, expr expression.Expression, nitems, direction int, valfunc, ignoreNulls bool,
	aggname string, context Context) error {

	var part value.Value
	list, start, e := getNthValues(aggname, cumpart, valfunc)
	if !valfunc {
		start = nitems
	}

	if e != nil {
		return e
	}

	if list.Len() < nitems || start < nitems {
		part, e = expr.Evaluate(item, context)
		if e != nil {
			return e
		}

		if !ignoreNulls || part.Type() > value.NULL {
			inserted := false
			for c := start; c < list.Len() && c < nitems; c++ {
				cc := part.Collate(list.ItemAt(c))
				if cc*direction < 0 {
					list.Insert(c, nitems, part)
					inserted = true
					break
				}
			}

			if list.Len() < nitems && !inserted {
				list.Add(part)
			}
		}
	}
	return nil
}
