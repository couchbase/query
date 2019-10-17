package inferencer

import (
	"fmt"
	"strings"
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
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

func DescribeKeyspace(conn *datastore.ValueConnection, retriever DocumentRetriever,
	similarityMetric float32, numSampleValues, dictionary_threshold, infer_timeout, max_schema_MB int32) (result value.Value, error_msg *string, warning_msg *string) {

	result = nil
	error_msg = nil
	warning_msg = nil

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
				return
			default:
			}
		}

		// Get the document
		doc, err := retriever.GetNextDoc()
		if err != nil {
			message := fmt.Sprintf("Error getting documents for infer.\n%s", *err)
			error := map[string]interface{}{"error": message}
			error["internal Error"] = *err
			result = value.NewValue(error)
			error_msg = &message
			return
		}

		if doc == nil { // all done, no more docs
			break
		}

		if desc_debug {
			fmt.Printf("   got document, collection size: %d\n", len(collection))
		}

		// make a schema out of the JSON document
		aSchema := NewSchemaFromValue(doc)

		// add it to the collection

		collection.AddSchema(aSchema, numSampleValues)

		// have we exceeded our timeout time?
		if int32(time.Now().Sub(start)/time.Second) > infer_timeout {
			if desc_debug {
				fmt.Printf("   exceeded infer_timeout of %d seconds, finishing document inferencing\n", infer_timeout)
			}
			tmp := fmt.Sprintf("Schema may be incomplete. Stopped schema inferencing after exceeding infer_timeout of %d seconds.", infer_timeout)
			warning_msg = &tmp
			break
		}

		if desc_debug {
			fmt.Printf("Collection with %d schemas has size: %d\n", collection.Size(), collection.GetCollectionByteSize())
		}

		// have we exceeded our max schema size?
		if int32(collection.GetCollectionByteSize()/1000000) > max_schema_MB {
			if desc_debug {
				fmt.Printf("   exceeded max schema size of %d MB, finishing document inferencing\n", max_schema_MB)
			}
			tmp := fmt.Sprintf("Schema may be incomplete. Stopped schema inferencing after schema size exceeded max_schema_MB of %d MB.", max_schema_MB)
			warning_msg = &tmp
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

	//fmt.Printf("Count was %d.",count)

	// nothing to do if no documents

	if len(collection) == 0 {
		message := fmt.Sprintf("No documents found, unable to infer schema.")
		error := map[string]interface{}{"error": message}
		result = value.NewValue(error)
		error_msg = &message
		return
	}

	// now get the complete description

	flavors := collection.GetFlavorsFromCollection(similarityMetric, numSampleValues, dictionary_threshold)

	if desc_debug {
		fmt.Printf("Done with second pass\n")
		for _, flavor := range flavors {
			fmt.Printf("Got flavor: \n%v\n", flavor.String())
		}
	}

	//
	// put out each flavor as JSON and return the result
	//

	schema_size := 0

	desc := make([]value.Value, len(flavors))
	for idx, _ := range flavors {
		bytes, jerr := flavors[idx].MarshalJSON()
		if jerr != nil {
			desc[idx] = value.NewValue(jerr.Error())
		} else {
			desc[idx] = value.NewValue(bytes)
			schema_size = schema_size + len(bytes)
		}
	}

	result = value.NewValue(desc)

	return
}

//
// Here is an inferencer that cbq-engine can use to infer schemas
//

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

//
// here
//

func (di *DefaultInferencer) InferKeyspace(ks datastore.Keyspace, with value.Value, conn *datastore.ValueConnection) {

	var ok bool

	docCount, _ := ks.Count(datastore.NULL_QUERY_CONTEXT)
	sample_size := 1000
	similarity_metric := float64(0.6)
	num_sample_values := int32(5)
	dictionary_threshold := int32(10)
	infer_timeout := int32(60) // don't spend more than 60 seconds on any bucket
	max_schema_MB := int32(10) // if the schema is bigger than 10MB, don't return

	defer close(conn.ValueChannel())

	// did we get any options to do something other than our defaults?
	if with != nil {
		if with.Type() != value.OBJECT {
			conn.Error(errors.NewWarning(fmt.Sprintf(`Unrecognized infer option '%v', options must be JSON (e.g., '{"sample_size":1000,"similarity_metric":0.6}'`, with.Actual())))
			return
		}

		unrecognizedNames := make([]string, 0)
		for fieldName, _ := range with.Fields() {
			if !strings.EqualFold(fieldName, "sample_size") &&
				!strings.EqualFold(fieldName, "dictionary_threshold") &&
				!strings.EqualFold(fieldName, "similarity_metric") &&
				!strings.EqualFold(fieldName, "num_sample_values") &&
				!strings.EqualFold(fieldName, "max_schema_MB") &&
				!strings.EqualFold(fieldName, "infer_timeout") {
				unrecognizedNames = append(unrecognizedNames, fieldName)
			}
		}

		if len(unrecognizedNames) > 0 {
			conn.Error(errors.NewWarning(fmt.Sprintf(`Unrecognized infer options: '%v'`, unrecognizedNames)))
			return
		}

		//////////////////////////////////////////////////////////////////////
		// sample_size parameter
		sample_size_val, sample_size_found := with.Field("sample_size")
		if sample_size_found {
			if sample_size_val.Type() != value.NUMBER {
				conn.Error(errors.NewWarning(fmt.Sprintf("'sample_size' option must be a number, not %s", sample_size_val.Type().String())))
				return
			}
			sample_size_num, ok := sample_size_val.Actual().(float64)
			if !ok {
				conn.Error(errors.NewWarning(fmt.Sprintf("Error reading 'sample_size' %v", sample_size_val.Actual())))
				return
			}
			sample_size = int(sample_size_num)
		}

		//////////////////////////////////////////////////////////////////////
		// similarity_metric parameter
		similarity_metric_val, similarity_metric_found := with.Field("similarity_metric")
		if similarity_metric_found {
			if similarity_metric_val.Type() != value.NUMBER {
				conn.Error(errors.NewWarning(fmt.Sprintf("'similarity_metric' option must be a number, not %s", similarity_metric_val.Type().String())))
				return
			}
			similarity_metric, ok = similarity_metric_val.Actual().(float64)
			if !ok {
				conn.Error(errors.NewWarning(fmt.Sprintf("Error reading 'similarity_metric' %v", similarity_metric_val.Actual())))
				return
			}
		}

		//////////////////////////////////////////////////////////////////////
		// num_sample_values parameter
		num_sample_values_val, num_sample_values_found := with.Field("num_sample_values")
		if num_sample_values_found {
			if num_sample_values_val.Type() != value.NUMBER {
				conn.Error(errors.NewWarning(fmt.Sprintf("'num_sample_values' option must be a number, not %s", num_sample_values_val.Type().String())))
				return
			}
			num_sample_values_num, ok := num_sample_values_val.Actual().(float64)
			if !ok {
				conn.Error(errors.NewWarning(fmt.Sprintf("Error reading 'num_sample_values' %v", num_sample_values_val.Actual())))
				return
			}
			num_sample_values = int32(num_sample_values_num)
		}

		//////////////////////////////////////////////////////////////////////
		// dictionary_threshold parameter
		dictionary_threshold_val, dictionary_threshold_found := with.Field("dictionary_threshold")
		if dictionary_threshold_found {
			if dictionary_threshold_val.Type() != value.NUMBER {
				conn.Error(errors.NewWarning(fmt.Sprintf("'dictionary_threshold' option must be a number, not %s", dictionary_threshold_val.Type().String())))
				return
			}
			dictionary_threshold_num, ok := dictionary_threshold_val.Actual().(float64)
			if !ok {
				conn.Error(errors.NewWarning(fmt.Sprintf("Error reading 'dictionary_threshold' %v", dictionary_threshold_val.Actual())))
				return
			}
			dictionary_threshold = int32(dictionary_threshold_num)
		}

		//////////////////////////////////////////////////////////////////////
		// infer_timeout parameter - how many seconds to allow
		infer_timeout_val, infer_timeout_found := with.Field("infer_timeout")
		if infer_timeout_found {
			if infer_timeout_val.Type() != value.NUMBER {
				conn.Error(errors.NewWarning(fmt.Sprintf("'infer_timeout' option must be a number, not %s", infer_timeout_val.Type().String())))
				return
			}
			infer_timeout_num, ok := infer_timeout_val.Actual().(float64)
			if !ok {
				conn.Error(errors.NewWarning(fmt.Sprintf("Error reading 'infer_timeout' %v", infer_timeout_val.Actual())))
				return
			}
			infer_timeout = int32(infer_timeout_num)
		}

		//////////////////////////////////////////////////////////////////////
		// max_schema_MB parameter - how many megabytes of schema before we give up.
		max_schema_MB_val, max_schema_MB_found := with.Field("max_schema_MB")
		if max_schema_MB_found {
			if max_schema_MB_val.Type() != value.NUMBER {
				conn.Error(errors.NewWarning(fmt.Sprintf("'max_schema_MB' option must be a number, not %s", max_schema_MB_val.Type().String())))
				return
			}
			max_schema_MB_num, ok := max_schema_MB_val.Actual().(float64)
			if !ok {
				conn.Error(errors.NewWarning(fmt.Sprintf("Error reading 'max_schema_MB' %v", max_schema_MB_val.Actual())))
				return
			}
			max_schema_MB = int32(max_schema_MB_num)
		}

	}

	//
	// Can't do anything if zero documents
	//

	if docCount == 0 {
		conn.Error(errors.NewWarning(fmt.Sprintf("Keyspace %s has no documents, schema inference not possible", ks.Name())))
		return
	}

	//	fmt.Printf("    Inferring keyspace for NamespaceId: %s Id: %s Name: %s, Count: %d\n",
	//		ks.NamespaceId(), ks.Id(), ks.Name(), docCount)
	//	if with != nil {
	//		fmt.Printf("      With: %v (%v)\n", with, with.Type())
	//	}

	//
	// we have two choices to get documents: either random (if supported) or primary key
	// traversal (if a primary index exists). Random is generally best, unless the sample
	// size is close to or greater than the number of documents. If neither is available
	// we can't do anything.
	//

	var retriever DocumentRetriever
	var err, err2 *string

	retriever = nil
	err = nil

	// does the Keyspace support random document retrieval?
	_, random_ok := ks.(datastore.RandomEntryProvider)

	// if the keyspace supports random document retrieval, and the sample size is small enough
	// relative to the docCount, a random document retriever is our first choice, if that
	// fails use the primary index (if any)

	if random_ok && float64(sample_size) < float64(docCount)*0.75 {
		retriever, err = MakeKeyspaceRandomDocumentRetriever(ks, sample_size)

		// if that failed, try to make a PrimaryKey retriever
		if err != nil {
			retriever, err2 = MakePrimaryIndexDocumentRetriever(ks, sample_size)
			if err2 != nil {
				conn.Error(errors.NewWarning(fmt.Sprintf("Unable to create either random or primary document retriever for keyspace: %s\n%s\n%s\n", ks.Name(), err, err2)))
				return
			}
		}

		//
		// otherwise, a primary index retriever is our first choice, but if it fails try random
		//
	} else {

		// if the number of documents to too small relative to the sample size,
		// we need to get the all documents from a sequential scan
		if float64(sample_size) >= float64(docCount)*0.75 {
			retriever, err = MakePrimaryIndexDocumentRetriever(ks, int(docCount))
		} else {
			retriever, err = MakePrimaryIndexDocumentRetriever(ks, sample_size)
		}

		// if this fails, perhaps there's no primary key. Try random
		if err != nil {
			retriever, err2 = MakeKeyspaceRandomDocumentRetriever(ks, sample_size)
			if err2 != nil {
				conn.Error(errors.NewWarning(fmt.Sprintf("Unable to create either primary or random document retriever for keyspace: %s\n%s\n%s\n", ks.Name(), err, err2)))
				return
			}
		}
	}

	//
	// get the

	schema, error_msg, warning_msg := DescribeKeyspace(conn, retriever, float32(similarity_metric), num_sample_values, dictionary_threshold, infer_timeout, max_schema_MB)

	if error_msg != nil {
		conn.Error(errors.NewWarning(*error_msg))
		return
	}

	if warning_msg != nil {
		conn.Warning(errors.NewWarning(*warning_msg))
	}
	conn.ValueChannel() <- schema
}

//
// in verbose mode, output messages on the command line
//

func log(message string) {
	if false {
		fmt.Println(message)
	}
}
