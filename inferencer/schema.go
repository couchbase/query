/*
Copyright 2019-Present Couchbase, Inc.

Use of this software is governed by the Business Source License included in
the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
file, in accordance with the Business Source License, use of this software will
be governed by the Apache License, Version 2.0, included in the file
licenses/APL2.txt.
*/

package inferencer

/*
 * schema.go contains structs and methods for creating schemas from a set of JSON documents.
 *
 * it relies on the query/value package to describe and compare different types of values.
 *
 * The types it provides are:
 *  FieldType - supports relaxed type equality, if one value is null
 *  Field - name, type, sample values, and how many docs have the field
 *  Schema - a set of fields, and count of docs that match the schema
 *  SchemaFlavor - for a set of 'similar' documents, record a union of all Fields and the frequency of
 *    each field across the document set.
 *  SchemaCollection - a set of Schemas indexed by their hash value for quick lookup
 *
 * How would you use this? Let's say you have a set of JSON documents, each of which is a string.
 * For each document, you would:
 *
 * - make a SchemaCollection. E.g.:

     collection := make(SchemaCollection)

 * - convert each JSON doc into a schema, and add to the collection. E.g.,

 		// make a schema out of the JSON document
		aSchema := NewSchemaFromValue(value.NewValue(resp.Body))

		// add it to the collection
		collection.AddSchema(aSchema, numSampleValues)

 * - after all the documents have been added, 'collection' holds the set of distinct schemas
 *   with a count of how many documents match each. You can merge these distinct schemas into
 *   'flavors', collections of similar (but not identical) schemas. E.g.

 	flavors := collection.GetFlavorsFromCollection(similarityMetric,numSampleValues)

 *   The similarity metric describes how similar two schemas must be to be merged into a single
 *   flavor. It is a value between 0 and 1, indicating the fraction of top level fields that are
 *   equal. E.g., a value of 0 would cause every schema to be merged into a single universal flavor.
 *   A value of 1.0 would create a different flavor for every distinct schema. I have found that 0.6
 *   to be about the right threshold to handle the Couchbase sample data.
 *
 *   The numSampleValues parameter indicates how many sample values should be kept for each field.
 *   Sample values can help a user understand the domain of a field, and they are also used in
 *   determining what fields only have a single value for a flavor, i.e., as a way of determining
 *   invariant fields such as type fields. The flavor name won't work if this value is set to less
 *   than 2.
 *
 * - For a JSON representation of a 'flavor', I use MarshalJSON.
*/
import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc64"
	"math"
	"sort"

	"github.com/couchbase/query/value"
)

//
// boolean used to indicate whether to send debugging info to stdout
//

var debug bool = false

//
// This struct is used to keep track of array contents, so we can describe
// the 'type' of an array, meaning a map indicating all the array element types
// seen, plus a record of the max & min number of elements seen
//

type ArrayType struct {
	typesSeen         map[string]FieldType // map from type string to type object, i.e. hash table for types
	minItems          int
	maxItems          int
	mergedObjectTypes SchemaFlavors        // similar types merged together into flavors
	nonObjectTypes    map[string]FieldType // nonobject types needed for schema
}

func NewArrayType(val value.Value) *ArrayType {
	// sanity check
	if val.Type() != value.ARRAY {
		fmt.Errorf("Tried to create array type with non array.")
		return nil
	}

	// get the underlying slice
	arr := val.Actual().([]interface{})
	result := new(ArrayType)
	result.minItems = len(arr)
	result.maxItems = len(arr)

	// we want to keep track of all the types in the array. Set up a map
	// from the string description to the type, so we can avoid dups

	//fmt.Printf("Creating array type size: %d...\n",len(arr))

	result.typesSeen = make(map[string]FieldType)
	for index := 0; index < len(arr); index++ { // get the type of each element in the array
		arrVal, found := val.Index(index)

		if found { // we got the value from the array
			newType := NewFieldType(arrVal) // make a type for the value
			newTypeKey := newType.StringNoValues(0)

			// if we have seen this type before, merge it in, otherwise add it to the map
			curVal := result.typesSeen[newTypeKey]

			// both are object types, merge the two object types
			if curVal.subtype != nil && newType.subtype != nil {
				curVal.subtype.MergeWith(newType.subtype, 0)
				result.typesSeen[newTypeKey] = curVal

				// both are array types, merge the two
			} else if curVal.arrtype != nil && newType.arrtype != nil {
				curVal.arrtype.MergeWith(newType.arrtype, 0)
				result.typesSeen[newTypeKey] = curVal

			} else { // either primitive or haven't seen it before
				result.typesSeen[newTypeKey] = newType
			}
		}
	}

	return result
}

func (at *ArrayType) Copy() *ArrayType {
	copyType := new(ArrayType)
	copyType.minItems = at.minItems
	copyType.maxItems = at.maxItems
	copyType.typesSeen = at.typesSeen
	copyType.mergedObjectTypes = at.mergedObjectTypes
	copyType.nonObjectTypes = at.nonObjectTypes

	return (copyType)
}

//
// when merging two fields with the same array type, need to combine the two
// ArrayTypes

func (at *ArrayType) MergeWith(other *ArrayType, numSampleValues int32) {
	if other.minItems < at.minItems {
		at.minItems = other.minItems
	}
	if other.maxItems > at.maxItems {
		at.maxItems = other.maxItems
	}
	// for each of the other array's types, merge them into our own
	for otherTypeDesc, otherType := range other.typesSeen {
		curVal := at.typesSeen[otherTypeDesc]

		if curVal.subtype != nil && otherType.subtype != nil {
			curVal.subtype.MergeWith(otherType.subtype, numSampleValues)
			at.typesSeen[otherTypeDesc] = curVal

		} else if curVal.arrtype != nil && otherType.arrtype != nil {
			curVal.arrtype.MergeWith(otherType.arrtype, numSampleValues)
			at.typesSeen[otherTypeDesc] = curVal

		} else { // haven't seen it before
			at.typesSeen[otherTypeDesc] = otherType
		}
	}

}

// need to be able to write out array type as JSON. We are letting
// the parent object deal with min/max items, we just need to output
// the array of types as either as array, or if there is only one as
// a single type

func (at *ArrayType) MarshalJSON() ([]byte, error) {
	totalNumTypes := len(at.mergedObjectTypes) + len(at.nonObjectTypes)
	if totalNumTypes == 0 {
		return []byte("{}"), nil

	} else if len(at.typesSeen) == 1 { // output a single type
		for _, theType := range at.typesSeen {
			return theType.MarshalJSON()
		}

		// single merged object type
	} else if totalNumTypes == 1 { // output a single type

		if len(at.mergedObjectTypes) == 1 {
			for _, theType := range at.mergedObjectTypes {
				return theType.MarshalJSON()
			}
		} else if len(at.nonObjectTypes) == 1 {
			for _, theType := range at.nonObjectTypes {
				return theType.MarshalJSON()
			}
		}

		// otherwise we have have a set of types, output them all as array
	} else {

		buf := bytes.NewBuffer(make([]byte, 0, 256))
		buf.WriteString("[")
		first := true

		// start with primitive types
		names := sortedNamesFT(at.nonObjectTypes)
		for _, name := range names {
			aType := at.nonObjectTypes[name]

			// now get the type
			typeDesc, err := aType.MarshalJSON()

			if err == nil {
				// need a comma before all but the first element
				if first {
					first = false
				} else {
					buf.WriteString(",")
				}

				buf.Write(typeDesc)
			} else {
				fmt.Errorf("Error getting array type JSON")
			}
		}

		// now do the object types
		for _, aType := range at.mergedObjectTypes {
			// now get the type
			typeDesc, err := aType.MarshalJSON()

			if err == nil {
				// need a comma before all but the first element
				if first {
					first = false
				} else {
					buf.WriteString(",")
				}

				buf.Write(typeDesc)
			} else {
				fmt.Errorf("Error getting array type JSON")
			}
		}

		buf.WriteString("]") // close the array
		return buf.Bytes(), nil
	}
	// shouldn't get here
	return nil, errors.New("Error creating JSON for array type")
}

//
// This struct holds the type for a field. If the type is an object, we
// need a pointer to the subschema of that object
//

type FieldType struct {
	value.Type
	subtype *Schema    // OBJECT type has a schema indicating its structure
	arrtype *ArrayType // ARRAY types has a struct describing the array contents
}

// infer the type from a sample value

func NewFieldType(val value.Value) FieldType {
	fieldType := new(FieldType)
	fieldType.Type = val.Type()

	// for object types, the subtype is a schema for the object
	if val.Type() == value.OBJECT {
		fieldType.subtype = NewSchemaFromValue(val)
	}

	// for array types, fill in arrtype
	if val.Type() == value.ARRAY {
		fieldType.arrtype = NewArrayType(val)
	}

	return (*fieldType)
}

func (ft *FieldType) Copy() FieldType {
	copyType := new(FieldType)
	copyType.Type = ft.Type
	if ft.subtype != nil {
		copyType.subtype = ft.subtype.Copy()
	}
	if ft.arrtype != nil {
		copyType.arrtype = ft.arrtype.Copy()
	}

	return (*copyType)
}

func (ft *FieldType) String(indent int) string {
	result := ft.Type.String()
	if ft.Type == value.OBJECT {
		result = result + "\n" + ft.subtype.StringIndent(indent+2)
	}
	if ft.Type == value.ARRAY {
		var buffer bytes.Buffer
		for i := 0; i < indent+2; i++ {
			buffer.WriteString(" ")
		}
		var indentStr = buffer.String()

		result = result + "\n"
		for _, elementType := range ft.arrtype.typesSeen {
			result = result + indentStr + elementType.String(indent+2) + "\n"
		}
	}

	return (result)
}

func (ft *FieldType) StringNoValues(indent int) string {
	result := ft.Type.String()
	if ft.Type == value.OBJECT {
		result = result + "\n" + ft.subtype.StringIndentNoValues(indent+2)
	}
	if ft.Type == value.ARRAY {
		//fmt.Printf("   going through %d typesSeen\n",len(ft.arrtype.typesSeen))
		var buffer bytes.Buffer
		for i := 0; i < indent+2; i++ {
			buffer.WriteString(" ")
		}
		var indentStr = buffer.String()

		result = result + "\n"

		names := sortedNamesFT(ft.arrtype.typesSeen)
		for _, name := range names {
			elementType := ft.arrtype.typesSeen[name]
			//fmt.Printf("   got array type: %s\n",elementType.String(0))
			result = result + indentStr + elementType.StringNoValues(indent+2) + "\n"
		}
	}

	return (result)
}

func (ft *FieldType) MarshalJSON() ([]byte, error) {
	if ft.subtype != nil {
		return ft.subtype.MarshalJSON()
		//r["subtype"] = ft.subtype
	}

	//	r := map[string]interface{}{"#schema": "FieldType"}
	r := map[string]interface{}{}
	r["type"] = ft.Type.String()

	if ft.arrtype != nil {
		r["minItems"] = ft.arrtype.minItems
		r["maxItems"] = ft.arrtype.maxItems
		r["items"] = ft.arrtype
	}
	return json.Marshal(r)
}

//
// are two types equal? Yes, if the types are the same, but they can also be
// equivalent if one of the types is "NULL" (meaning unknown). Also, if the
// type of both is "OBJECT", we must recursively check the subtypes
//

func (ft *FieldType) EqualTo(other *FieldType) bool {

	// if either type is NULL, the two are equivalent
	//	if ft.Type == value.NULL || other.Type == value.NULL {
	//	return (true)
	//}

	// if types are different, can't be the same
	if ft.Type != other.Type {
		return (false)
	}

	// since both types are the same, if both types are OBJECT, we must test
	// the equality of the subtype
	if ft.Type == value.OBJECT {
		return (ft.subtype.EqualTo(other.subtype))
	}

	// if we get this far, they must be the same
	return (true)
}

//
// This struct holds a schema definition for a field, including name, type,
// and a small set of sample values. In general a Field can only have one type,
// except null values are treated as a separate type. Also, when when merging
// similar schemas into flavors, we may end up with multiple fields with a single
// name and multiple types. Thus we will keep a pointer to 'namesake' fields that
// have the same name but a different type.
//

type Field struct {
	Name                string
	Kind                FieldType
	sampleValues        value.Values
	numMatchingDocs     *int64
	percentMatchingDocs *float32
	isDictionary        bool
	namesake            *Field
}

// make a Field given an AnnotatePair from a document

func NewField(name string, val value.Value) Field {
	field := new(Field)
	field.Name = name
	field.Kind = NewFieldType(val)
	field.sampleValues = make(value.Values, 1)
	field.sampleValues[0] = val
	field.isDictionary = false
	field.namesake = nil

	return (*field)
}

// make a Dictionary Field

func NewDictionaryField(ftype FieldType, sampleValues value.Values) Field {
	field := new(Field)
	field.isDictionary = true
	field.Name = "Dictionary(string-to-Object)"
	field.Kind = ftype
	field.sampleValues = sampleValues
	field.namesake = nil

	return (*field)
}

func (f *Field) Copy() *Field {
	copyField := new(Field)
	copyField.Name = f.Name
	copyField.Kind = f.Kind.Copy()
	copyField.sampleValues = make(value.Values, len(f.sampleValues))
	copy(copyField.sampleValues, f.sampleValues)
	copyField.numMatchingDocs = f.numMatchingDocs
	copyField.percentMatchingDocs = f.percentMatchingDocs
	copyField.isDictionary = f.isDictionary

	if f.namesake != nil {
		copyField.namesake = f.namesake.Copy()
	}

	return (copyField)
}

func (f Field) String(indent int) string {
	result := f.Name + " - " + f.Kind.String(indent)

	if f.Kind.Type != value.NULL && f.Kind.Type != value.OBJECT &&
		len(f.sampleValues) > 0 {
		result += " ["
		first := true
		for idx, _ := range f.sampleValues {
			if !first {
				result += ","
			} else {
				first = false
			}

			valBytes, _ := f.sampleValues[idx].MarshalJSON()
			if len(valBytes) > 20 {
				valBytes = valBytes[0:20]
			}
			result += string(valBytes)
		}
		result += "]"
	}

	if f.namesake != nil {
		result += "\n" + f.namesake.String(indent)
	}

	return (result)
}

//
// when converting the schema to JSON, we want to truncate sample values to make them
// easier to read
//

func (f Field) TruncatedSampleValues() value.Values {
	result := make(value.Values, 0)

	if f.Kind.Type != value.NULL && f.Kind.Type != value.OBJECT &&
		len(f.sampleValues) > 0 {
		for idx, _ := range f.sampleValues {
			val, ok := f.sampleValues[idx].Actual().(string)
			if ok {
				if len(val) > 30 {
					val = string(append([]byte(val[0:30]), '.', '.', '.'))
				}
				result = append(result, value.NewValue(val))
			}
		}
	}

	// put the samples in order, to make them easier to read
	sort.Slice(result, func(i, j int) bool { return result[i].Actual().(string) < result[j].Actual().(string) })

	// if the first entry in the array is an empty string, swap it with the last
	if len(result) > 1 && result[0].Actual().(string) == "" {
		temp := result[0]
		result[0] = result[len(result)-1]
		result[len(result)-1] = temp
	}
	return (result)
}

func (f *Field) StringNoType() string {
	result := f.Name
	if f.Kind.Type == value.OBJECT {
		result += " " + f.Kind.subtype.StringNoTypes() + " "
	}

	if f.namesake != nil {
		result += "\n" + f.namesake.StringNoType()
	}

	return (result)
}

func (f *Field) StringNoValues(indent int) string {
	result := f.Name + " - " + f.Kind.StringNoValues(indent)

	if f.namesake != nil {
		result += "\n" + f.namesake.StringNoValues(indent)
	}

	return (result)
}

func (f *Field) NameTypeOnly() string {
	return (f.Name + " - " + f.Kind.Type.String())
}

func (f *Field) GetJSONMap() map[string]interface{} {

	r := map[string]interface{}{}

	//
	// simple case:no namesake types, our type is a string
	//
	if f.namesake == nil {
		r["type"] = f.Kind.Type.String()

		if f.percentMatchingDocs != nil {
			percentDocs := math.Trunc(float64(*f.percentMatchingDocs*100.0)) / 100.0
			if !math.IsNaN(percentDocs) {
				r["%docs"] = percentDocs
			} else {
				r["%docs"] = 0.0
			}
		}
		if f.numMatchingDocs != nil {
			r["#docs"] = f.numMatchingDocs
		}

		if len(f.sampleValues) > 0 {
			if f.Kind.Type == value.STRING {
				r["samples"] = f.TruncatedSampleValues()
			} else {
				sort.Slice(f.sampleValues, func(i, j int) bool { return f.sampleValues[i].Collate(f.sampleValues[j]) < 0 })
				r["samples"] = f.sampleValues
			}
		}

		if f.Kind.subtype != nil {
			fieldObject := make(fieldMap)

			for _, field := range f.Kind.subtype.fields {
				fieldObject[field.Name] = field
			}

			r["properties"] = fieldObject
		}

		if f.Kind.arrtype != nil {
			r["minItems"] = f.Kind.arrtype.minItems
			r["maxItems"] = f.Kind.arrtype.maxItems
			r["items"] = f.Kind.arrtype
		}

		//
		// complex case: we have multiple types for the same field name, need to iterate over namesakes
		//

	} else {
		// we will keep an array of type names, for future sorting, and a map for the other values
		typeArray := make([]string, 0)
		percentMatchingMap := make(map[string]float32)
		numMatchingMap := make(map[string]int64)
		samplesMap := make(map[string]value.Values)

		for f != nil {
			// names of types
			name := f.Kind.Type.String()
			typeArray = append(typeArray, name)

			// sample values
			if f.Kind.Type == value.STRING {
				samplesMap[name] = f.TruncatedSampleValues()
			} else {
				sort.Slice(f.sampleValues, func(i, j int) bool { return f.sampleValues[i].Collate(f.sampleValues[j]) < 0 })
				samplesMap[name] = f.sampleValues
			}

			// percent matching
			if f.percentMatchingDocs != nil {
				percentDocs := math.Trunc(float64(*f.percentMatchingDocs*100.0)) / 100.0
				if math.IsNaN(percentDocs) {
					percentDocs = 0.0
				}
				percentMatchingMap[name] = float32(percentDocs)
			} else {
				percentMatchingMap[name] = 0.0
			}

			// num matching
			if f.numMatchingDocs != nil {
				//fmt.Printf("Getting map with #matching: %d\n", *f.numMatchingDocs)
				numMatchingMap[name] = *f.numMatchingDocs
			} else {
				//fmt.Printf("Getting map, #matching nil\n")
				numMatchingMap[name] = 0
			}

			// if array or object type, remember subtypes
			if f.Kind.subtype != nil {
				fieldObject := make(fieldMap)

				for _, field := range f.Kind.subtype.fields {
					fieldObject[field.Name] = field
				}

				r["properties"] = fieldObject
			}

			if f.Kind.arrtype != nil {
				r["minItems"] = f.Kind.arrtype.minItems
				r["maxItems"] = f.Kind.arrtype.maxItems
				r["items"] = f.Kind.arrtype
			}

			// move on to the next one
			f = f.namesake
		}

		// we want sorted arrays so that types are always in the same order
		// now use the arrays of types
		sort.Slice(typeArray, func(i, j int) bool { return typeArray[i] < typeArray[j] })

		percentMatchingArray := make([]float32, 0)
		numMatchingArray := make([]int64, 0)
		samplesArray := make([]value.Values, 0)

		// get the items from the maps in sorted order
		for _, name := range typeArray {
			percentMatchingArray = append(percentMatchingArray, percentMatchingMap[name])
			numMatchingArray = append(numMatchingArray, numMatchingMap[name])
			samplesArray = append(samplesArray, samplesMap[name])
		}

		r["type"] = typeArray
		r["%docs"] = percentMatchingArray
		r["#docs"] = numMatchingArray
		r["samples"] = samplesArray
	}
	return r
}

func (f *Field) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.GetJSONMap())
}

//
// when two fields, inferred from different documents, are equivalent, we need to be
// able to merge them. Merging means
// - if our type is null, use the type from other (since null type == any type)
// - if there is space, merge the list of sample values
//

func (f *Field) MergeWith(other *Field, numSampleValues int32) {
	thisType := f.Kind.Type
	otherType := other.Kind.Type

	switch {
	// if we are merging two objects, merge their schemas
	case thisType == value.OBJECT && otherType == value.OBJECT:
		f.Kind.subtype.MergeWith(other.Kind.subtype, numSampleValues)

	// if we are merging two arrays, merge the array type description
	case thisType == value.ARRAY && otherType == value.ARRAY:
		if debug {
			fmt.Printf("Merging array types %v and %v\n", f.Kind.arrtype, other.Kind.arrtype)
		}

		if f.Kind.arrtype == nil && other.Kind.arrtype != nil {
			f.Kind.arrtype = other.Kind.arrtype
		} else if f.Kind.arrtype != nil && other.Kind.arrtype != nil {
			f.Kind.arrtype.MergeWith(other.Kind.arrtype, numSampleValues)
		}

	}

	//
	// if we have space for more sample values, fold them in
	//

	if int32(len(f.sampleValues)) < numSampleValues {
		// make sure new values not already in the list
		for idx, _ := range other.sampleValues {
			found := false
			newVal := other.sampleValues[idx]

			for idx2, _ := range f.sampleValues {
				curVal := f.sampleValues[idx2]

				if (newVal.Equals(curVal) == value.TRUE_VALUE) ||
					(newVal == value.NULL_VALUE && curVal == value.NULL_VALUE) {
					found = true
					break
				}
			}

			if !found {
				f.sampleValues = append(f.sampleValues, newVal)
				//newValStr,_ := newVal.MarshalJSON();
				//fmt.Printf("Adding sample value for %s which is %s, length now %d\n",f.Name,newValStr,len(f.sampleValues))
			}
		}
	}

	if int32(len(f.sampleValues)) > numSampleValues {
		f.sampleValues = f.sampleValues[:numSampleValues]
	}

	//
	// if the other field is a Dictionary field, we become a dictionary as well
	//

	if other.isDictionary {
		f.isDictionary = true
		f.Name = other.Name
	}

}

//
// we need to be able to sort fields by name, so we must implement sort.interface
//

type Fields []Field

func (f Fields) Len() int           { return (len(f)) }
func (f Fields) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }
func (f Fields) Less(i, j int) bool { return (f[i].Name < f[j].Name) }

//
//
//
//

/*
fieldMap is a type of map from string to Field, used for having a set of
named fields that we can call MarshalJSON on to produce standart schema.
*/
type fieldMap map[string]Field

func (this fieldMap) MarshalJSON() ([]byte, error) {
	if this == nil {
		return []byte("null"), nil
	}

	buf := bytes.NewBuffer(make([]byte, 0, 256))
	buf.WriteString("{")

	names := sortedNames(this)
	for i, n := range names {
		if i > 0 {
			buf.WriteString(",")
		}

		b, err := json.Marshal(n)
		if err != nil {
			return nil, err
		}

		buf.Write(b)
		buf.WriteString(":")

		field := this[n]
		b, err = field.MarshalJSON()
		if err != nil {
			return nil, err
		}

		buf.Write(b)
	}

	buf.WriteString("}")
	return buf.Bytes(), nil
}

func sortedNames(obj map[string]Field) []string {
	names := make(sort.StringSlice, 0, len(obj))
	for name, _ := range obj {
		names = append(names, name)
	}

	names.Sort()
	return names
}

func sortedNamesFT(obj map[string]FieldType) []string {
	names := make(sort.StringSlice, 0, len(obj))
	for name, _ := range obj {
		names = append(names, name)
	}

	names.Sort()
	return names
}

//
// This struct holds a schema, which is a set of Fields. For the purposes
// of schema inference, we keep track of the number of matching documents
// we've seen.
//
// Schemas might describe JSON objects, which are a collection of name/value
// pairs, but they also might describe bare values, such as numbers, booleans,
// strings, or arrays. That is an either/or situation, either an object, or
// a bare value, never both.
//
// We need methods for:
// - creating a hash value, used for doing quick lookups
// - doing equality comparisons between two schemas
// - finding the intersection of two schemas (those fields common to both)
// - merging two equivalent schemas (merging the sample values for types,
//    and adding the number of matched documents)
//

type Schema struct {
	fields           Fields
	bareValue        *Field
	hashValue        uint64
	byteSize         uint64
	matchingDocCount int64
}

func (s *Schema) GetDocCount() int64 {
	return (s.matchingDocCount)
}

func (s *Schema) GetFieldCount() int {
	return (len(s.fields))
}

// make a schema out of a key-value collection representing a document

func NewSchema(doc []value.AnnotatedPair) *Schema {
	if len(doc) > 0 {
		return (NewSchemaFromValue(doc[0].Value))
	} else {
		fmt.Println("Error, zero size doc.")
		return (nil)
	}
}

// make a schema out of an object-typed Value

func NewSchemaFromValue(val value.Value) *Schema {
	schema := new(Schema)
	schema.matchingDocCount = 1
	// the normal case is objects comprised of fields
	if val.Type() == value.OBJECT {
		schema.bareValue = nil
		v1 := val.Actual()
		elements := v1.(map[string]interface{})
		schema.fields = make([]Field, 0, len(elements))
		for name, v2 := range elements {
			//fmt.Printf("  Got field2: %s, value: %s\n",name, value.NewValue(v2))
			schema.fields = append(schema.fields, NewField(name, value.NewValue(v2)))
		}

		if av, ok := val.(value.AnnotatedValue); ok {
			if m := av.GetMeta(); m != nil {
				if x := m["xattrs"]; x != nil {
					// low probability of a conflict with an existing document field
					// nevertheless generate a unique name if necessary
					name := "xattr"
					for i := 0; ; i++ {
						if _, ok := elements[name]; !ok {
							break
						}
						name = fmt.Sprintf("xattrs:%d", i)
					}
					schema.fields = append(schema.fields, NewField(name, value.NewValue(x)))
				}
			}
		}

		sort.Sort(schema.fields)

		// it is also possible to have a bare string, boolean, number, or array

	} else {
		bareField := NewField("bare", val)
		schema.bareValue = &bareField
	}

	// make a hashValue for the schema
	crcTable := crc64.MakeTable(crc64.ECMA)
	schema.hashValue = crc64.Checksum([]byte(schema.StringNoTypes()), crcTable)
	schema.byteSize = uint64(len(schema.String()))

	if debug {
		fmt.Printf("Created schema hashValue: %d for schema %s\n", schema.hashValue, schema.StringNoTypes())
	}
	return (schema)
}

// make a copy of a schema

func (s *Schema) Copy() *Schema {
	schemaCopy := new(Schema)
	schemaCopy.fields = make(Fields, len(s.fields))
	for idx, _ := range s.fields {
		schemaCopy.fields[idx] = *s.fields[idx].Copy()
	}
	schemaCopy.hashValue = s.hashValue
	schemaCopy.matchingDocCount = s.matchingDocCount
	schemaCopy.bareValue = s.bareValue

	return (schemaCopy)
}

//
// one potential pattern in schemas is the 'dictionary', where
// a schema has a set of fields where the field name is a data
// value, and all the field types have the same signiture. In
// this case we want to collapse all the fields into a single
// 'dictionary' type. It may not be possible to definitively
// distinguish a set of fields with the same type vs. a dictionary,
// so we will insist on a minimum number of fields with identical
// types.
//
// we'll do this recursively, depth first
//
// side effect warning: changing our fields can change our signature, which
// in turn changes our hashValue. If the schema is part of a SchemaCollection
// hashtable, it should be removed before calling this function, and re-inserted
// afterward.
//
//

func (s *Schema) CollapseDictionaryFields(numSampleValues int32, dictionary_threshold int32) {

	// keep track of our fields' schemas
	schemasFound := make(SchemaCollection)

	objectFieldCount := 0

	// depth first traversal
	for _, field := range s.fields {
		if field.Kind.Type == value.OBJECT {
			objectFieldCount++
			subtypeCopy := field.Kind.subtype.Copy()
			subtypeCopy.matchingDocCount = 1
			schemasFound.AddSchema(subtypeCopy, 0)
			field.Kind.subtype.CollapseDictionaryFields(numSampleValues, dictionary_threshold)
		}
	}

	//
	// was every field an object? And did they all have the same schema?
	// if so, we replace every field with a single field indicating a
	// dictionary
	//

	if schemasFound.Size() == 1 && objectFieldCount == len(s.fields) &&
		objectFieldCount > int(dictionary_threshold) {

		// make a list of sample keys and sample values for the dictionary
		sampleValues := make(value.Values, 0)

		for _, field := range s.fields {
			if len(sampleValues) < int(numSampleValues) {
				numSamplesNeeded := int(numSampleValues) - len(sampleValues)
				numToCopy := int(math.Min(float64(numSamplesNeeded),
					float64(len(field.sampleValues))))
				sampleValues = append(sampleValues, field.sampleValues[0:numToCopy]...)
			}
		}

		//fmt.Printf("Collapsing %d fields, found %d schemas\n", len(s.fields), schemasFound.Size())

		// create a new field to replace all the others
		dictType := s.fields[0].Kind
		numDocs := s.fields[0].numMatchingDocs
		s.fields = make(Fields, 1)
		s.fields[0] = NewDictionaryField(dictType, sampleValues)
		s.fields[0].numMatchingDocs = numDocs

		// since our fields have changed, so also has our hashvalue
		crcTable := crc64.MakeTable(crc64.ECMA)
		s.hashValue = crc64.Checksum([]byte(s.StringNoTypes()), crcTable)
	}
}

//
// Array types can contain anything - arrays, objects, primitives. When looking at array values,
// we keep track of each distinct type. At the end of the process, we want to merge the "similar"
// types, based on the similarity metric.
//

func (s *Schema) RemoveExtraSamples(numSampleValues int32, similarityMetric float32) {

	// collapse any arrays within sub-objects, or nested in arrays, depth first

	for _, field := range s.fields {
		// work recursively
		if field.Kind.Type == value.OBJECT {
			field.Kind.subtype.RemoveExtraSamples(numSampleValues, similarityMetric)
		}

		if field.Kind.Type == value.ARRAY {
			field.Kind.arrtype.RemoveExtraSamples(numSampleValues, similarityMetric)
		}

		// objects inside arrays sometimes end up with too many sample values
		//fmt.Printf("Checking field: %s ns %d nsv %d\n", field.Name, len(field.sampleValues), numSampleValues)
		if int32(len(field.sampleValues)) > numSampleValues {
			field.sampleValues = field.sampleValues[:numSampleValues]
			//fmt.Printf("  truncd field: %s ns %d nsv %d\n", field.Name, len(field.sampleValues), numSampleValues)
		}

	}
}

func (at *ArrayType) RemoveExtraSamples(numSampleValues int32, similarityMetric float32) {

	// step one: go through types seen, recursively handle Object and Array types
	for _, typeSeen := range at.typesSeen {
		if typeSeen.Type == value.OBJECT {
			typeSeen.subtype.RemoveExtraSamples(numSampleValues, similarityMetric)

		} else if typeSeen.Type == value.ARRAY {
			typeSeen.arrtype.RemoveExtraSamples(numSampleValues, similarityMetric)

		} else {
		}
	}
}

//
// Array types can contain anything - arrays, objects, primitives. When looking at array values,
// we keep track of each distinct type. At the end of the process, we want to merge the "similar"
// types, based on the similarity metric.
//

func (s *Schema) CollapseArrayTypes(numSampleValues int32, similarityMetric float32) {

	// collapse any arrays within sub-objects, or nested in arrays, depth first

	for _, field := range s.fields {
		// work recursively
		if field.Kind.Type == value.OBJECT {
			field.Kind.subtype.CollapseArrayTypes(numSampleValues, similarityMetric)
		}

		if field.Kind.Type == value.ARRAY {
			field.Kind.arrtype.CollapseArrayTypes(numSampleValues, similarityMetric)
		}
	}

	if s.bareValue != nil && s.bareValue.Kind.Type == value.ARRAY {
		s.bareValue.Kind.arrtype.CollapseArrayTypes(numSampleValues, similarityMetric)
	}
}

// since arrays can be arbitrarily nested within arrays, need to work with them recursively
// to get all the way to the bottom.

func (at *ArrayType) CollapseArrayTypes(numSampleValues int32, similarityMetric float32) {
	at.nonObjectTypes = make(map[string]FieldType)
	objectSchemas := make([]FieldType, 0) // remember objects separately for later merge

	// step one: go through types seen, recursively handle Object and Array types
	for signature, typeSeen := range at.typesSeen {
		if typeSeen.Type == value.OBJECT {
			typeSeen.subtype.CollapseArrayTypes(numSampleValues, similarityMetric)
			objectSchemas = append(objectSchemas, typeSeen)

		} else if typeSeen.Type == value.ARRAY {
			typeSeen.arrtype.CollapseArrayTypes(numSampleValues, similarityMetric)
			at.nonObjectTypes[signature] = typeSeen

		} else {
			// keep track of primitive types seen in the array
			at.nonObjectTypes[signature] = typeSeen
		}
	}

	// step two: compare object types, if similar, merge them together
	// 2a: create an array of SchemaCollections where each collection will
	//   contain schemas similar to eachother
	similarCollections := make([]SchemaCollection, 0)

	// 2b: loop through array of types, compare each to similar collections, merge if similar
	for _, otype := range objectSchemas {
		schema := otype.subtype
		//fmt.Printf("     checking schema %s.\n", schema.StringNoValues())

		foundMatch := false

		for idx, _ := range similarCollections {
			collSchema := similarCollections[idx].Get()
			if collSchema != nil && schema.OverlapsWith(collSchema) > similarityMetric {
				//fmt.Printf("    found match.\n")
				foundMatch = true
				similarCollections[idx].AddSchema(schema, numSampleValues)
				break
			}
		}

		// didn't find a close match, create a new collection
		if !foundMatch {
			//fmt.Printf("    No match.")

			newCollection := make(SchemaCollection)
			newCollection.AddSchema(schema, numSampleValues)
			similarCollections = append(similarCollections, newCollection)
		}
	}

	// Step 3: Now we have an array of similar Collections, let's merge each.
	at.mergedObjectTypes = make(SchemaFlavors, len(similarCollections))
	for idx, c := range similarCollections {
		at.mergedObjectTypes[idx].schema, at.mergedObjectTypes[idx].fieldFreq = c.Union("", nil, numSampleValues)
	}

	// for consistency, sort the lists of objects and non-objects
	sort.Slice(at.mergedObjectTypes,
		func(i, j int) bool {
			return at.mergedObjectTypes[i].schema.StringNoValues() < at.mergedObjectTypes[j].schema.StringNoValues()
		})
}

//
// create a JSON schema version of the schema data structure
//

func (s *Schema) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.getSchemaMap())
}

// convenience function to get a string -> interface map for schemas,
// used by both schema and flavor MarshalJSON

func (s *Schema) getSchemaMap() map[string]interface{} {
	// need a map for schema type, num docs, subfields, any bare values, etc.
	r := map[string]interface{}{}

	r["$schema"] = "http://json-schema.org/draft-06/schema"
	r["#docs"] = s.matchingDocCount

	fieldObject := make(fieldMap)
	for _, field := range s.fields {
		fieldObject[field.Name] = field
	}

	if len(fieldObject) > 0 {
		r["properties"] = fieldObject
		r["type"] = "object"
	} else if s.bareValue != nil {
		for key, value := range s.bareValue.GetJSONMap() {
			r[key] = value
		}
	}

	return (r)
}

//
// are two schemas equal? If they have the same field names (which is summed up in
// the hashValue), and those fields have equivalent types. For our purposes, two
// types are equivalent if one of them is "NULL", since that just means a document
// (from which the schema was inferred) had a null value for that field
//

func (s *Schema) EqualTo(other *Schema) bool {

	// if hashValue is different, they have different fields
	if s.hashValue != other.hashValue || len(s.fields) != len(other.fields) {
		return (false)
	}

	// since hashValues are the same, the field names are the same,
	// so now we must check type equivalence

	for idx, _ := range s.fields {
		if !s.fields[idx].Kind.EqualTo(&other.fields[idx].Kind) {
			return (false)
		}
	}

	// if one has a bare type and not the other, they aren't equal
	if (s.bareValue == nil && other.bareValue != nil) || (s.bareValue != nil && other.bareValue == nil) {
		return false
	} else if s.bareValue != nil && other.bareValue != nil && !s.bareValue.Kind.EqualTo(&other.bareValue.Kind) {
		return false
	}

	// if we get this far, they must be the same

	return (true)
}

//
// Make a human-readable version of a schema
//

func (s *Schema) String() string {
	header := fmt.Sprintf("Schema has %d fields, %d docs\n", len(s.fields), s.matchingDocCount)
	return (header + s.StringIndent(0))
}

func (s *Schema) StringIndent(indent int) string {
	var buffer bytes.Buffer

	// object fields, if any
	for idx, _ := range s.fields {
		for i := 0; i < indent+2; i++ {
			buffer.WriteString(" ")
		}
		buffer.WriteString(s.fields[idx].String(indent + 2))
		if s.fields[idx].Kind.Type != value.OBJECT && s.fields[idx].Kind.Type != value.ARRAY {
			buffer.WriteString("\n")
		}
	}

	// bare value, if any
	if s.bareValue != nil {
		for i := 0; i < indent+2; i++ {
			buffer.WriteString(" ")
		}
		buffer.WriteString(s.bareValue.String(indent + 2))
		if s.bareValue.Kind.Type != value.OBJECT && s.bareValue.Kind.Type != value.ARRAY {
			buffer.WriteString("\n")
		}

	}

	return (buffer.String())
}

//
// for debugging, here is a version that dosen't show sample values
//

func (s *Schema) StringNoValues() string {
	header := fmt.Sprintf("Schema has %d fields, %d docs, hash: %d\n", len(s.fields), s.matchingDocCount, s.hashValue)
	return (header + s.StringIndentNoValues(0))
}

func (s Schema) StringIndentNoValues(indent int) string {
	var buffer bytes.Buffer

	// for comparison sake, we want the fields in alphabetical order
	sort.Sort(s.fields)

	for idx, _ := range s.fields {
		for i := 0; i < indent+2; i++ {
			buffer.WriteString(" ")
		}
		buffer.WriteString(s.fields[idx].StringNoValues(indent + 2))
		if s.fields[idx].Kind.Type != value.OBJECT && s.fields[idx].Kind.Type != value.ARRAY {
			buffer.WriteString("\n")
		}
	}

	// if we have a bare value, add that as well
	if s.bareValue != nil {
		buffer.WriteString(s.bareValue.StringNoValues(indent + 2))
	}

	return (buffer.String())
}

//
// when describing a group of documents, a schema can also be printed along with
// the frequency of each field
//

func (s *Schema) StringWithFrequency(indent int, freqMap map[string]int64, parentName string,
	matchingDocCount int64) string {
	var buffer bytes.Buffer

	if parentName == "" {
		for i := 0; i < indent; i++ { // indentation
			buffer.WriteString(" ")
		}
		buffer.WriteString(fmt.Sprintf("Schema has %d fields, %d docs, hash: %d\n", len(s.fields), matchingDocCount, s.hashValue))
	}

	// for comparison sake, we want the fields in alphabetical order
	sort.Sort(s.fields)

	// iterate over the schema's fields
	for _, field := range s.fields {
		fieldFreq := freqMap[parentName+field.NameTypeOnly()]
		//fmt.Printf("Managing field: (%d) %s\n", fieldFreq, parentName+field.NameTypeOnly())

		for i := 0; i < indent+2; i++ { // indentation
			buffer.WriteString(" ")
		}

		buffer.WriteString(fmt.Sprintf("- %6.2f: %s - %s",
			100.0*float32(fieldFreq)/float32(matchingDocCount),
			field.Name, field.Kind.Type.String()))
		//buffer.WriteString(fmt.Sprintf("%3d/%d: %s - %s",
		//	fieldFreq, matchingDocCount,
		//	field.Name, field.Kind.Type.String()))

		// for objects, need subtype
		if field.Kind.Type == value.OBJECT {
			buffer.WriteString(fmt.Sprintf(" %d fields %d docs\n", len(field.Kind.subtype.fields),
				field.Kind.subtype.matchingDocCount) +
				field.Kind.subtype.StringWithFrequency(indent+2, freqMap, parentName+field.Name+".", fieldFreq))
		}

		// for arrays, we have a set of subtypes
		if field.Kind.Type == value.ARRAY {
			buffer.WriteString(fmt.Sprintf(" array with %d subtypes\n", len(field.Kind.arrtype.typesSeen)))

			for _, elementType := range field.Kind.arrtype.typesSeen {
				for i := 0; i < indent+4; i++ { // indentation
					buffer.WriteString(" ")
				}

				buffer.WriteString(elementType.String(indent + 4))

				if elementType.Type != value.OBJECT && elementType.Type != value.ARRAY {
					buffer.WriteString("\n")
				}

			}

		}

		// output sample values

		if field.Kind.Type != value.NULL &&
			field.Kind.Type != value.OBJECT &&
			len(field.sampleValues) > 0 {
			buffer.WriteString(" [")
			first := true
			for _, val := range field.sampleValues {
				if !first {
					buffer.WriteString(",")
				} else {
					first = false
				}

				valBytes, _ := val.MarshalJSON()
				if len(valBytes) > 20 {
					valBytes = valBytes[0:20]
				}
				buffer.WriteString(string(valBytes))
			}
			buffer.WriteString("]")
		}

		// close with a carriage return

		if field.Kind.Type != value.OBJECT {
			buffer.WriteString("\n")
		}
	}

	// if there is a bare value instead of fields, output that
	if s.bareValue != nil {
		fieldFreq := freqMap[parentName+s.bareValue.NameTypeOnly()]

		for i := 0; i < indent+2; i++ { // indentation
			buffer.WriteString(" ")
		}
		buffer.WriteString(fmt.Sprintf("- %6.2f: %s - %s",
			100.0*float32(fieldFreq)/float32(matchingDocCount),
			s.bareValue.Name, s.bareValue.Kind.Type.String()))

		// for arrays, we have a set of subtypes
		if s.bareValue.Kind.Type == value.ARRAY {
			buffer.WriteString(fmt.Sprintf(" array with %d subtypes\n", len(s.bareValue.Kind.arrtype.typesSeen)))

			for _, elementType := range s.bareValue.Kind.arrtype.typesSeen {
				for i := 0; i < indent+4; i++ { // indentation
					buffer.WriteString(" ")
				}

				buffer.WriteString(elementType.String(indent + 4))

				if elementType.Type != value.OBJECT && elementType.Type != value.ARRAY {
					buffer.WriteString("\n")
				}

			}

		}
	}

	return (buffer.String())
}

//
// for JSON purposes, we may update the fields themselves with their frequencies
//

func (s *Schema) UpdateFieldFrequencies(freqMap map[string]int64, parentName string, matchingDocCount int64) {

	// iterate over the schema's fields
	for idx, _ := range s.fields {

		//field := &s.fields[idx]
		UpdateSingleFieldFrequencies(&s.fields[idx], freqMap, parentName, matchingDocCount)
	}

	// if we have a bare value, do that as well
	if s.bareValue != nil {
		UpdateSingleFieldFrequencies(s.bareValue, freqMap, parentName, matchingDocCount)
	}

}

func UpdateSingleFieldFrequencies(field *Field, freqMap map[string]int64, parentName string, matchingDocCount int64) {

	for field != nil { // loop through any namesakes

		fieldFreq := freqMap[parentName+field.NameTypeOnly()]

		if debug {
			fmt.Printf("Updating field '%s' with freq %v matchingDocCount %d\n", parentName+field.NameTypeOnly(), fieldFreq,
				matchingDocCount)
		}

		field.numMatchingDocs = new(int64)
		*field.numMatchingDocs = fieldFreq
		field.percentMatchingDocs = new(float32)

		// sanity check, we shouldn't get here now that a subtle bug was fixed, but
		// leave it here just in case
		if matchingDocCount == 0 {
			fmt.Errorf("UpdateFieldFrequencies for parent %s count %d\n", parentName, matchingDocCount)
			fmt.Errorf("Got zero doc_count for field: %v\n", field)
			matchingDocCount = 1
		}
		*field.percentMatchingDocs = (100 * float32(fieldFreq) / float32(matchingDocCount))

		// handle any subtypes
		if field.Kind.Type == value.OBJECT { // for objects, need subtype
			field.Kind.subtype.UpdateFieldFrequencies(freqMap, parentName+field.Name+".", fieldFreq)
		}

		// handle any name sakes
		field = field.namesake
	}
}

//
// for the purposes of doing maps and fast equality comparisons, we have a hashValue
// that we will base on the list of field names, but not their types. This is because
// a field with no value has an unknown type, but that field is equivalent to a field
// with the same name, and a known type. This method generates a list of field names,
// with no types, that we can hash to get a hash code for the schema. If two schemas
// have different fields, they are definitely different.
//

func (s Schema) StringNoTypes() string {
	var buffer bytes.Buffer
	first := true

	buffer.WriteString("{")

	// for comparison sake, we want the fields in alphabetical order
	sort.Sort(s.fields)

	for idx, _ := range s.fields {
		if !first {
			buffer.WriteString(", ")
		} else {
			first = false
		}

		buffer.WriteString(s.fields[idx].StringNoType())
	}

	if s.bareValue != nil {
		buffer.WriteString(s.bareValue.StringNoType())
	}

	buffer.WriteString("}")
	return (buffer.String())
}

//
// when we have two schemas, inferred from different documents, which are
// equivalent, we want to merge the two into a new schema that adds the matching
// document count of the two, and merges the list of sample values.
// If one schema has a field with a NULL type, use the type from the other
//

func (s *Schema) MergeWith(other *Schema, numSampleValues int32) {

	// can't do anything if the two aren't equal
	if !s.EqualTo(other) {
		fmt.Errorf("Can't merge, schemas not equal")
		return
	}

	// sum the matchingDocCounts
	s.matchingDocCount += other.matchingDocCount

	// go through each field and promote any NULL types, if possible.
	// also merge the subtypes of any OBJECT-type fields

	for idx, _ := range s.fields {
		s.fields[idx].MergeWith(&other.fields[idx], numSampleValues)
	}

	// bare types?
	if s.bareValue != nil && other.bareValue != nil {
		s.bareValue.MergeWith(other.bareValue, numSampleValues)
	}
}

//
// how much do two schemas overlap? We will compute only by comparing top level
// fields, and see what fraction are equivalent. We rely on the fact that the fields
// are store in alphabetical order.
//

func (s *Schema) OverlapsWith(other *Schema) (degree float32) {

	matchingFieldCount := float32(0.0)
	totalFieldCount := 0
	sLen := len(s.fields)
	oLen := len(other.fields)
	if sLen > oLen {
		totalFieldCount = sLen
	} else {
		totalFieldCount = oLen
	}
	sIdx := 0
	oIdx := 0 // indexes to traverse the fields

	// loop through the fields, counting the matches
	for sIdx < len(s.fields) && oIdx < len(other.fields) {
		sField := &s.fields[sIdx]
		oField := &other.fields[oIdx]

		// are the two equal?
		if sField.Name == oField.Name { // same name
			switch {
			// exact equality
			case sField.Kind.EqualTo(&oField.Kind):
				matchingFieldCount++

			// if the two are not equal but are both Object type, and have the
			// same name, we should compute the degree of overlap between the
			// subtypes, and add it to the matchingFieldCount
			case sField.Kind.Type == value.OBJECT && oField.Kind.Type == value.OBJECT:
				matchingFieldCount += sField.Kind.subtype.OverlapsWith(oField.Kind.subtype)

			// if one is NULL count them as the same
			case sField.Kind.Type == value.NULL || oField.Kind.Type == value.NULL:
				matchingFieldCount++

			// if the two have the same name but are different types, count it as half
			default:
				matchingFieldCount += 0.5
			}

			// increment field counters
			sIdx++
			oIdx++

		} else if sField.Name < oField.Name { // our name is less, increment our
			sIdx++
		} else {
			oIdx++
		}
	}

	// bare values?
	if s.bareValue != nil && other.bareValue != nil && s.bareValue.Kind.EqualTo(&other.bareValue.Kind) {
		matchingFieldCount++
		totalFieldCount = 1
	}

	if totalFieldCount == 0 { // no fields in either schema
		degree = 1.0
	} else {
		degree = float32(matchingFieldCount) / float32(totalFieldCount)
	}

	return
}

//
// a SchemaCollection is a group of Schemas, each of which is different. When
// a new schema is added, if it is equal to an existing schema, the two are merged
// (which adds the number of matching documents, and merges the list of sample
// values).
//
// this is implemented as a map, where the key is the hashValue of the schema, and
// the value is a slice of Schemas (since they can sometimes have the same hashValue
// but still be different). So this is effectively a hash table with bucket chaining
//

type SchemaCollection map[uint64][]*Schema

// how many schemas in the collection?
func (c SchemaCollection) Size() (count int) {
	count = 0

	for _, bucketChain := range c {
		count += len(bucketChain)
	}

	return
}

// for convenience, get a random schema from a collection
func (c SchemaCollection) Get() (result *Schema) {
	result = nil

	for idx, _ := range c {
		if len(c[idx]) > 0 {
			return (c[idx][0])
		}
	}

	return
}

// add a new schema, which either merges with an existing schema in the collection,
// if the two are equal, or adds a new one to the end
func (c SchemaCollection) AddSchema(newSchema *Schema, numSampleValues int32) {

	// find any schemas with the same hashValue
	matchArray := c[newSchema.hashValue]

	// if nothing there, make a new slice to hold the schemas
	if matchArray == nil {
		matchArray = make([]*Schema, 1)
		matchArray[0] = newSchema
		if debug {
			fmt.Printf("No matching bucket, creating new bucket for schema %d.\n", newSchema.hashValue)
			if len(matchArray) > 0 && len(matchArray[0].fields) > 0 {
				fmt.Printf("First field: %s\n", matchArray[0].fields[0].String(0))
			}
		}

	} else { // otherwise, look through the slice for an equivalent schema,
		// and if found, merge the two
		found := false // have we seen a match?

		for idx, _ := range matchArray {
			schema := matchArray[idx]
			if newSchema.EqualTo(schema) {
				matchArray[idx].MergeWith(newSchema, numSampleValues)
				found = true
				if debug {
					fmt.Printf("Found matching schema %d, merging, count now: %d\n", schema.hashValue, schema.matchingDocCount)
					fmt.Printf("First field: %s\n", matchArray[idx].fields[0].String(0))
				}
				break
			}
		}

		// if we didn't find it, append it to matchArray
		if !found {
			matchArray = append(matchArray, newSchema)
			if debug {
				fmt.Println("Schema not in bucket, appending to bucket chain")
				if len(matchArray) > 0 && len(matchArray[0].fields) > 0 {
					fmt.Printf("First field: %s\n", matchArray[0].fields[0].String(0))
				}
			}
		}
	}

	// and now make sure the collection holds the new array
	c[newSchema.hashValue] = matchArray
}

func (c SchemaCollection) GetCollectionByteSize() (size uint64) {

	if debug {
		fmt.Printf("Getting byte size with %d distinct schemas.\n", c.Size())
	}

	size = 0

	//
	// with big documents, schemas can get huge. this computes the cumulative size
	// of all the schemas in a collection
	//

	for _, bucketChain := range c {
		for idx, _ := range bucketChain {
			size += bucketChain[idx].byteSize
		}
	}

	return
}

//
// SchemaFlavors - This merges a SchemaCollection into a (possibly smaller)
// set of similar schemas. Thus, schemas with significant overlap are merged, those
// with little overlap are kept separate so we can see what different "flavors" of
// document we have in a collection.
//
// The "similarityMetric" passed in is an indication of how similar two schemas must
// be to be considered the same "flavor". It is a measure of what fraction of the top
// level fields must be the same. Thus, if the similarityMetric is 1.0, then the two
// schemas must be identical to be considered the same flavor. If the similarity is 0.0, then
// you get a "Universal Schema" with a single flavor containing all fields of every document.
// Around 0.6 seems to work well for Couchbase's sample data.
//

type SchemaFlavor struct {
	schema    *Schema
	fieldFreq map[string]int64 // how often does each field occur?
}

type SchemaFlavors []SchemaFlavor

func (c SchemaCollection) GetFlavorsFromCollection(similarityMetric float32, numSampleValues int32,
	dictionary_threshold int32) (flavors SchemaFlavors) {

	if debug {
		fmt.Printf("Starting with %d distinct schemas.\n", c.Size())
	}

	// first, create an array of SchemaCollections where each collection will
	// contain schemas similar to eachother

	similarCollections := make([]SchemaCollection, 0)

	//
	// we have a collection of distinct schemas. Before we test them for similarity,
	// we should collapse any dictionaries (sets of fields with a name and the same
	// object type) into single fields, that way the similarity test will be
	// more accurate
	//

	for _, bucketChain := range c {
		for idx, _ := range bucketChain {
			schema := bucketChain[idx]

			// look for dictionary fields and collaspe them, to make schemas managable
			schema.CollapseDictionaryFields(numSampleValues, dictionary_threshold)

			// also collapse array types by similarity - if two object types in an
			// array are similar, merge them

			schema.CollapseArrayTypes(numSampleValues, similarityMetric)

			//fmt.Printf("Checking schema: \n\n%s\n\n",schema.String())

			// we need to compare this schema to a schema from
			// each collection. If it's sufficiently similar, add it to that
			// collection. If not, add this schema as a new Collection

			foundMatch := false

			for idx, _ := range similarCollections {
				collSchema := similarCollections[idx].Get()
				if collSchema != nil && schema.OverlapsWith(collSchema) > similarityMetric {
					foundMatch = true
					similarCollections[idx].AddSchema(schema, numSampleValues)
					//fmt.Println("  Adding above schema to existing collection.")
					break
				}
			}

			// didn't find a close match, create a new collection
			if !foundMatch {
				//fmt.Println("  Creating new collection for above schema.")
				//fmt.Printf("Making new collection for schema with %d fields and %d docs\n",schema.GetFieldCount(),
				//	schema.GetDocCount())
				newCollection := make(SchemaCollection)
				newCollection.AddSchema(schema, numSampleValues)
				similarCollections = append(similarCollections, newCollection)
			}
		}
	}

	// Now we have an array of similar Collections, let's merge each.

	if debug {
		fmt.Printf("Was able to find %d collections of similar schemas.\n", len(similarCollections))
	}

	flavors = make(SchemaFlavors, len(similarCollections))
	for idx, c := range similarCollections {
		flavors[idx].schema, flavors[idx].fieldFreq = c.Union("", nil, numSampleValues)
		flavors[idx].schema.CollapseDictionaryFields(numSampleValues, dictionary_threshold)
		flavors[idx].schema.RemoveExtraSamples(numSampleValues, similarityMetric)

	}

	// for consistency, make sure the flavors are always in the same order
	sort.Slice(flavors, func(i, j int) bool { return flavors[i].schema.StringNoValues() < flavors[j].schema.StringNoValues() })

	return
}

func (f *SchemaFlavor) GetFieldCount() int {
	return (len(f.schema.fields))
}

func (f SchemaFlavor) String() string {
	return f.schema.StringWithFrequency(0, f.fieldFreq, "", f.schema.GetDocCount())
}

func (sf *SchemaFlavor) MarshalJSON() ([]byte, error) {
	// this is a good time to make sure our field frequencies are up to date
	sf.schema.UpdateFieldFrequencies(sf.fieldFreq, "", sf.schema.matchingDocCount)

	// get the map for use with json.Marshall
	schemaMap := sf.schema.getSchemaMap()

	// flavors also need descriptors - a user-visible label for the flavor, showing any field
	// with only a single value in the flavor, such as 'type = "brewery"'.
	// we look for fields that are string or int, and have only a single value.
	// of course they aren't useful if they get too long, so limit their size.

	maxDescriptorLength := 512
	descriptor := ""

	for _, field := range sf.schema.fields {

		if len(descriptor) < maxDescriptorLength &&
			len(field.sampleValues) == 1 &&
			field.namesake == nil && // ignore fields with namesake types
			(field.numMatchingDocs != nil && *field.numMatchingDocs == sf.schema.matchingDocCount) &&
			(field.Kind.Type == value.BOOLEAN ||
				field.Kind.Type == value.NUMBER ||
				field.Kind.Type == value.STRING) {

			valBytes, _ := field.sampleValues[0].MarshalJSON()
			var valStr string
			// if the value is long, truncate and add "..."
			if len(valBytes) > 32 {
				valStr = string(valBytes[0:32]) + "..."
				if valBytes[0] == '"' {
					valStr = valStr + "\""
				}
			} else {
				valStr = string(valBytes)
			}

			if len(descriptor) > 0 {
				descriptor = descriptor + ", "
			}

			descriptor = descriptor + "`" + field.Name + "` = " + valStr

			// if we hit the max descriptor length, just add elipsis
			if len(descriptor) >= maxDescriptorLength {
				descriptor = descriptor + "..."
			}
		}
	}

	schemaMap["Flavor"] = descriptor

	return json.Marshal(schemaMap)
}

//
// when doing Union, we need to keep track of the number of occurrances of each field
// and subfield. This routine will recursively initialize those numbers for a schema
//

func setFieldFrequency(s *Schema, parentField string, fieldFreq map[string]int64, freq int64,
	setVsAdd bool) {
	for _, field := range s.fields { // how many documents have each field
		setSingleFieldFrequency(&field, parentField, fieldFreq, freq, setVsAdd)
	}
}

func setSingleFieldFrequency(field *Field, parentField string, fieldFreq map[string]int64, freq int64, setVsAdd bool) {
	for field != nil {
		if setVsAdd {
			fieldFreq[parentField+field.NameTypeOnly()] = freq
		} else {
			fieldFreq[parentField+field.NameTypeOnly()] += freq
		}

		// for subobjects, make a recursive call
		if field.Kind.Type == value.OBJECT {
			setFieldFrequency(field.Kind.subtype, parentField+field.Name+".", fieldFreq, freq, setVsAdd)
		}

		// if the field has a namesake, do it as well
		field = field.namesake
	}
}

//
// One way to summarize set of schemas is to make the union of all the fields,
// with an indication for each field indicating how often it appears in the set as
// a whole
//

func (c SchemaCollection) Union(parentField string, fieldFreq map[string]int64, numSampleValues int32) (
	*Schema, map[string]int64) {

	if len(c) == 0 { // nothing here
		return nil, nil
	}

	// we have at least one schema to work with

	var union *Schema = nil
	if fieldFreq == nil {
		fieldFreq = make(map[string]int64)
	}

	for _, bucketChain := range c {
		for idx, _ := range bucketChain {
			schema := bucketChain[idx]

			// start with a copy of the first schema we see
			if union == nil {
				union = schema.Copy()

				// if we are at the beginning, we need to recursively add counts for
				// every field and subfield in the schema

				if len(fieldFreq) == 0 {
					setFieldFrequency(union, "", fieldFreq, schema.matchingDocCount, true)
					if union.bareValue != nil {
						setSingleFieldFrequency(union.bareValue, "", fieldFreq, schema.matchingDocCount, true)
					}
				}

				if debug {
					fmt.Printf("Got original union schema, %d docs\n", union.matchingDocCount)
					fmt.Println(union.StringWithFrequency(2, fieldFreq, "", union.matchingDocCount))
				}
				continue
			} else {
				// on subsequent iterations, merge the current union with each schema, keeping track of
				// field prevalence.
				mergeNewSchemaIntoUnion(union, schema, parentField, fieldFreq, numSampleValues)
			}
		}
	}
	return union, fieldFreq
}

func mergeNewSchemaIntoUnion(union *Schema, schema *Schema, parentField string,
	fieldFreq map[string]int64, numSampleValues int32) (*Schema, map[string]int64) {

	var mergeDebug = false

	// We do it field by field, since the fields are in
	// alphabetical order, we can traverse them alphabetically.
	// Keep 2 pointers, one to to each array of fields, and compare the names
	// - If one field name is less than the other, make sure that name is in the result
	// - If fields are equal, and not object type merge sample values
	// - If fields have the same name and are Object type, Union the subschemas

	unionIdx := 0
	schemaIdx := 0
	newUnionFields := make(Fields, 0)

	union.matchingDocCount += schema.matchingDocCount
	if mergeDebug {
		fmt.Printf("\n\nMerging in schemas with %d docs\n%s\n", schema.matchingDocCount, schema.StringNoValues())
	}

	// if we have fields, merge them
	for unionIdx < len(union.fields) && schemaIdx < len(schema.fields) {
		uField := &union.fields[unionIdx]
		sField := &schema.fields[schemaIdx]

		origType := uField.Kind.Type
		origCount := fieldFreq[parentField+uField.NameTypeOnly()]

		if mergeDebug {
			fmt.Printf(" uField %d (%s-%d) sField %d (%s-%d),",
				unionIdx, parentField+uField.Name, origCount,
				schemaIdx, parentField+sField.Name, schema.matchingDocCount)
		}

		switch {

		// case 0: both fields have same name, or one is a dictionary, and they
		// have an equivalent type, so merge the two.
		// - sum the doc counts
		// - if the union field is null, and the new field is an OBJECT, we need
		//   to set the field counts for the new subobject

		case (uField.Name == sField.Name || uField.isDictionary || sField.isDictionary) &&
			uField.Kind.EqualTo(&sField.Kind):

			uFieldDict := uField.isDictionary // remember if we started out with a dict

			if mergeDebug {
				fmt.Printf(" equal, merge, type was: %s, and %s,", uField.Kind.Type.String(), sField.Kind.Type.String())
			}

			// promote null types, merge sample values
			uField.MergeWith(sField, numSampleValues)

			if mergeDebug {
				fmt.Printf(" new type was: %s\n", uField.Kind.Type.String())
			}

			// update #docs who have this field
			fieldFreq[parentField+uField.NameTypeOnly()] = origCount + schema.matchingDocCount // sum docCounts

			// if we had are merging an OBJECT into a NULL value, make sure
			// to set the field counts for the new fields
			if origType == value.NULL && sField.Kind.Type == value.OBJECT {
				setFieldFrequency(uField.Kind.subtype, parentField+uField.Name+".",
					fieldFreq, schema.matchingDocCount, true)
			}

			// if we are merging two equivalent objects, we to recursively
			// add to the field counts for each subfield
			if origType == value.OBJECT && sField.Kind.Type == value.OBJECT {
				setFieldFrequency(uField.Kind.subtype, parentField+uField.Name+".",
					fieldFreq, schema.matchingDocCount, false)
			}

			// move on to the next fields for both schemas
			unionIdx++
			schemaIdx++

			// but if the union was a dictionary to begin with, and the other field was
			// not, we have merged with the first field from the other schema, and we
			// can skip the equivalent non-dictionary fields from the other schema
			if uFieldDict && !sField.isDictionary {
				for schemaIdx < len(schema.fields) && uField.Kind.EqualTo(&schema.fields[schemaIdx].Kind) {
					schemaIdx++
				}
			}

			// if the union field was not initially a dictionary, but it is now,
			// there may be subsequent fields that should be folded into the union
			if !uFieldDict && sField.isDictionary {
				for unionIdx < len(union.fields) && uField.Kind.EqualTo(&union.fields[unionIdx].Kind) {
					union.fields = append(union.fields[:unionIdx], union.fields[unionIdx+1:]...)
				}
			}

		// case 1: union has a field not in schema, nothing to do, move along
		case uField.Name < sField.Name:
			if mergeDebug {
				fmt.Print(" uName less, get next union\n")
			}
			unionIdx++

		// case 2: schema has field not in union, insert it into the union
		case sField.Name < uField.Name:
			if mergeDebug {
				fmt.Print(" sName less, insert\n")
			}

			newUnionFields = append(newUnionFields, *sField)
			schemaIdx++
			fieldFreq[parentField+sField.NameTypeOnly()] = schema.matchingDocCount

			// if new field is type OBJECT, we need to set field counts for
			// all the subfields
			if sField.Kind.Type == value.OBJECT {
				//fmt.Printf("Adding subfields of %s with count %d\n",parentField + sField.Name,schema.matchingDocCount)
				setFieldFrequency(sField.Kind.subtype, parentField+sField.Name+".",
					fieldFreq, schema.matchingDocCount, true)
			}

		// if we get this far, the two names must be equal, but not the field types
		// case 3: whether the types are equal or not, if names are the same and
		// both are OBJECT type, but are different, so we need
		// to union the two subtypes

		case uField.Kind.Type == value.OBJECT && sField.Kind.Type == value.OBJECT:
			if mergeDebug {
				fmt.Printf(" uField, sField objects, merging w/origCount: %d, subschemas %d and %d\n", origCount,
					uField.Kind.subtype.hashValue, sField.Kind.subtype.hashValue)
			}

			// before we merge, note how many documents matched each field
			uField.Kind.subtype.matchingDocCount = origCount
			sField.Kind.subtype.matchingDocCount = schema.matchingDocCount

			uField.Kind.subtype, _ = mergeNewSchemaIntoUnion(uField.Kind.subtype, sField.Kind.subtype, parentField+uField.Name+".",
				fieldFreq, numSampleValues)

			// update #docs who have this field
			fieldFreq[parentField+uField.NameTypeOnly()] = origCount + schema.matchingDocCount // sum docCounts
			if mergeDebug {
				fmt.Printf("\n\n  Done with merge, new Count: %d\n", fieldFreq[parentField+uField.NameTypeOnly()])
			}

			unionIdx++
			schemaIdx++

		// case 4: types are different, but names are the same. Loop through the existing namesakes
		// for matches, if found, merge, if not make the new type a namesake of the union type

		default:
			if mergeDebug {
				fmt.Printf(" namesakes u: %v\n", uField.String(2))
				fmt.Printf(" namesakes s: %v\n", sField.String(2))
			}

			namesake := uField.namesake
			for namesake != nil {
				if namesake.Kind.EqualTo(&sField.Kind) {
					// update #docs who have this field
					fieldFreq[parentField+uField.NameTypeOnly()] = origCount + schema.matchingDocCount // sum docCounts
					break
				}

				namesake = namesake.namesake
			}

			// if we got to the end of the namesakes without finding a match, add this field as a namesake

			if namesake == nil {
				// make sure this field knows the number of matching docs
				setSingleFieldFrequency(sField, parentField, fieldFreq, schema.matchingDocCount, true)
				sField.namesake = uField.namesake // sfield assumes list of namesakes (if any)
				uField.namesake = sField          // make the sField a namesake of the uField
			}

			// on to the next field
			unionIdx++
			schemaIdx++
		}

	} // end for loop over fields

	// do we have any new fields for the union?

	if len(newUnionFields) > 0 {
		union.fields = append(union.fields, newUnionFields...)
		sort.Sort(union.fields)
	}

	// at this point, we must have hit the end of one or both field lists. If there are still
	// fields in the new schema, add them to the union

	if schemaIdx < len(schema.fields) {
		if mergeDebug {
			fmt.Printf("end of list, appending %d fields from schema\n", len(schema.fields)-schemaIdx)
		}
		//union.fields = append(union.fields, schema.fields[schemaIdx:]...)

		// must also set the field frequency for each of the new fields

		for _, sField := range schema.fields[schemaIdx:] {
			union.fields = append(union.fields, sField)
			fieldFreq[parentField+sField.NameTypeOnly()] = schema.matchingDocCount

			// if new field is type OBJECT, we need to set field counts for
			// all the subfields
			if sField.Kind.Type == value.OBJECT {
				//fmt.Printf("Adding subfields of %s with count %d\n",parentField + sField.Name,schema.matchingDocCount)
				setFieldFrequency(sField.Kind.subtype, parentField+sField.Name+".",
					fieldFreq, schema.matchingDocCount, true)
			}
		}
	}

	// if the schemas have bare values...
	if schema.bareValue != nil {
		setSingleFieldFrequency(schema.bareValue, "", fieldFreq, schema.matchingDocCount, true)
	}

	switch {
	// if types are equivalent, merge
	case union.bareValue != nil && schema.bareValue != nil && union.bareValue.Kind.EqualTo(&schema.bareValue.Kind):
		union.bareValue.MergeWith(schema.bareValue, numSampleValues)

	// types not equivalent, check for namesakes
	case union.bareValue != nil && schema.bareValue != nil && !union.bareValue.Kind.EqualTo(&schema.bareValue.Kind):
		namesake := union.bareValue.namesake
		for namesake != nil {
			if namesake.Kind.EqualTo(&schema.bareValue.Kind) {
				namesake.MergeWith(schema.bareValue, numSampleValues)
				break
			}
			namesake = namesake.namesake
		}

		// if we didn't find a match in the namesakes, add the new type as a namesake
		if namesake == nil {
			schema.bareValue.namesake = union.bareValue
			union.bareValue = schema.bareValue
		}
	}

	return (union), fieldFreq
}

//
// we also might want to find the intersection, that set of fields that is present
// in *every* schema in the collection
//

//func (c SchemaCollection) Intersection() Schema {
//
//}
