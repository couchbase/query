// Copyright 2019-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
// file, in accordance with the Business Source License, use of this software
// will be governed by the Apache License, Version 2.0, included in the file
// licenses/APL2.txt.

package inferencer

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"

	"os"
	"runtime/pprof"
)

//
// Given a keyspace, iterate over the keys to come up with a common
// schema. We can limit it to a certain sample size, or if sampleSize is
// zero use all the documents in the bucket
//

var desc_debug = false

type DescribeOptions struct {
	SampleSize          int
	SimilarityMetric    float32
	NumSampleValues     int32
	DictionaryThreshold int32
	InferTimeout        int32
	MaxSchemaMB         int32
	Flags               Flag
}

func DescribeKeyspace(context datastore.QueryContext, conn *datastore.ValueConnection, retriever DocumentRetriever,
	options *DescribeOptions) (value.Value, errors.Error) {

	if options == nil {
		return nil, errors.NewInferOptionsError()
	}

	var err errors.Error

	if desc_debug {
		fmt.Printf("Inferring keyspace...")
		f, err := os.Create("profile")
		if err != nil {
			fmt.Printf("Error creating profile: %s\n", err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	collection := make(SchemaCollection)

	start := time.Now() // remember when we started

	for {
		// see if we've been requested to stop
		if conn != nil {
			select {
			case <-conn.StopChannel():
				return nil, err
			default:
			}
		}

		// Get the document
		key, doc, err := retriever.GetNextDoc(context)
		if err != nil {
			return nil, err
		}

		if doc == nil { // all done, no more docs
			break
		}

		if options != nil && (options.Flags&INCLUDE_KEY) != 0 {
			if _, ok := doc.Field("~meta"); !ok {
				m := make(map[string]interface{})
				m["id"] = key
				doc.SetField("~meta", value.NewValue(m))
			}
		}

		if desc_debug {
			fmt.Printf("  got document, collection size: %d\n", len(collection))
		}

		// make a schema out of the JSON document
		aSchema := NewSchemaFromValue(doc)

		// add it to the collection

		collection.AddSchema(aSchema, options.NumSampleValues)

		// have we exceeded our timeout time?
		if int32(time.Now().Sub(start)/time.Second) > options.InferTimeout {
			if desc_debug {
				fmt.Printf("  exceeded infer_timeout of %d seconds, finishing document inferencing\n", options.InferTimeout)
			}
			err = errors.NewInferTimeout(options.InferTimeout)
			break
		}

		if desc_debug {
			fmt.Printf("Collection with %d schemas has size: %d\n", collection.Size(), collection.GetCollectionByteSize())
		}

		// have we exceeded our max schema size?
		if int32(collection.GetCollectionByteSize()/1000000) > options.MaxSchemaMB {
			if desc_debug {
				fmt.Printf("  exceeded max schema size of %d MB, finishing document inferencing\n", options.MaxSchemaMB)
			}
			err = errors.NewInferSizeLimit(options.MaxSchemaMB)
			break
		}

	}

	if desc_debug {
		fmt.Printf("Done with first pass\n")
		for _, schemaArray := range collection {
			for _, schema := range schemaArray {
				fmt.Printf("Got schema: \n%v\n", schema.StringIndentNoValues(4))
			}
		}
	}

	// nothing to do if no documents

	if len(collection) == 0 {
		return nil, errors.NewInferNoDocuments()
	}

	// now get the complete description

	flavors := collection.GetFlavorsFromCollection(options.SimilarityMetric, options.NumSampleValues, options.DictionaryThreshold)

	if desc_debug {
		fmt.Printf("Done with second pass\n")
		for _, flavor := range flavors {
			fmt.Printf("Got flavor: \n%v\n", flavor.String())
		}
	}

	//
	// put out each flavor as JSON and return the result
	//

	desc := make([]value.Value, len(flavors))
	for idx, _ := range flavors {
		bytes, jerr := flavors[idx].MarshalJSON()
		if jerr != nil {
			desc[idx] = value.NewValue(jerr.Error())
		} else {
			desc[idx] = value.NewValue(bytes)
		}
	}

	return value.NewValue(desc), err
}

func processWith(context datastore.QueryContext, with value.Value) (*DescribeOptions, errors.Error) {
	// defaults
	options := &DescribeOptions{
		SampleSize:          1000,
		SimilarityMetric:    float32(0.6),
		NumSampleValues:     5,
		DictionaryThreshold: 10,
		InferTimeout:        60, // don't spend more than 60 seconds on any bucket
		MaxSchemaMB:         10, // if the schema is bigger than 10MB, don't return
		Flags:               NO_FLAGS,
	}

	if !context.GetReqDeadline().IsZero() {
		options.InferTimeout = int32(context.GetReqDeadline().Sub(time.Now()).Seconds())
		logging.Debugf("Setting infer_timeout to %v based on context deadline %v",
			options.InferTimeout, context.GetReqDeadline(), context)
	}

	if with == nil {
		return options, nil
	}
	if with.Type() != value.OBJECT {
		return nil, errors.NewInferInvalidOption(fmt.Sprintf("%v", with.Actual()))
	}

	for fieldName, _ := range with.Fields() {
		fv, _ := with.Field(fieldName)
		if fv.Type() != value.NUMBER {
			if fieldName == "flags" {
				if fv.Type() == value.STRING {
					flags_num, err := strconv.ParseInt(fv.Actual().(string), 0, 32)
					if err != nil {
						return nil, errors.NewInferErrorReadingNumber(fieldName, fmt.Sprintf("%v", fv.Actual()))
					}
					options.Flags = Flag(flags_num)
				} else if fv.Type() == value.ARRAY {
					fa := fv.Actual().([]interface{})
					options.Flags = NO_FLAGS
					for _, f := range fa {
						fs := strings.ToLower(f.(value.Value).ToString())
						v, ok := flags_map[fs]
						if !ok {
							//TODO
							return nil, errors.NewWarning(fmt.Sprintf("'flags' array element '%v' is invalid", fs))
						}
						options.Flags |= v
					}
				} else {
					// TODO
					return nil, errors.NewWarning(fmt.Sprintf("'flags' must be a number, a string or an array not: %v", fv.Type()))
				}
				continue
			}
			return nil, errors.NewInferOptionMustBeNumeric(fieldName, fv.Type().String())
		}
		v, ok := fv.Actual().(float64)
		if !ok {
			return nil, errors.NewInferErrorReadingNumber(fieldName, fmt.Sprintf("%v", fv.Actual()))
		}
		switch fieldName {
		case "sample_size":
			options.SampleSize = int(v)
		case "similarity_metric":
			options.SimilarityMetric = float32(v)
		case "num_sample_values":
			options.NumSampleValues = int32(v)
		case "dictionary_threshold":
			options.DictionaryThreshold = int32(v)
		case "infer_timeout":
			options.InferTimeout = int32(v)
		case "max_schema_MB":
			options.MaxSchemaMB = int32(v)
		default:
			return nil, errors.NewInferInvalidOption(fieldName)
		}
	}

	return options, nil
}

func NewDefaultSchemaInferencer(store datastore.Datastore) (datastore.Inferencer, errors.Error) {
	inferencer := new(DefaultInferencer)
	inferencer.store = store
	return inferencer, nil
}

type DefaultInferencer struct {
	store datastore.Datastore
}

func (di *DefaultInferencer) Name() datastore.InferenceType {
	return ("Default")
}

func (di *DefaultInferencer) InferKeyspace(context datastore.QueryContext, ks datastore.Keyspace, with value.Value,
	conn *datastore.ValueConnection) {

	docCount, _ := ks.Count(context)

	defer close(conn.ValueChannel())

	options, err := processWith(context, with)
	if err != nil {
		conn.Error(err)
		return
	}
	if options.Flags == NO_FLAGS {
		options.Flags |= INCLUDE_KEY
	}

	if docCount == 0 {
		conn.Error(errors.NewInferNoDocuments())
		return
	}

	retriever, err := MakeUnifiedDocumentRetriever("infer_"+context.RequestId(), context, ks, options.SampleSize, options.Flags)
	if err != nil {
		if !err.IsWarning() {
			conn.Error(err)
			return
		}
		conn.Warning(err)
	}
	defer retriever.Close()

	schema, err := DescribeKeyspace(context, conn, retriever, options)
	if err != nil {
		if !err.IsWarning() {
			conn.Error(err)
			return
		}
		conn.Warning(err)
	}

	conn.ValueChannel() <- schema
}

func (di *DefaultInferencer) InferExpression(context datastore.QueryContext, expr expression.Expression, with value.Value,
	conn *datastore.ValueConnection) {

	defer close(conn.ValueChannel())

	options, err := processWith(context, with)
	if err != nil {
		conn.Error(err)
		return
	}

	var retriever DocumentRetriever

	retriever = nil
	err = nil

	retriever, err = MakeExpressionDocumentRetriever(context, expr, options.SampleSize)
	if err != nil {
		conn.Error(errors.NewInferCreateRetrieverFailed(err))
		return
	}
	defer retriever.Close()

	schema, err := DescribeKeyspace(context, conn, retriever, options)
	if err != nil {
		if !err.IsWarning() {
			conn.Error(err)
			return
		}
		conn.Warning(err)
	}

	conn.ValueChannel() <- schema
}
