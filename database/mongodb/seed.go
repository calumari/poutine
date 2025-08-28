package mongodb

import (
	"fmt"

	"github.com/calumari/jwalk"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type unwrappable interface { // TODO: think of a better way to do this
	UnwrapValue() any
}

// toBSONCollections converts a top-level jwalk.Document where each field is an
// array of documents into a map of collection name -> slice of bson documents.
func toBSONCollections(rootDoc jwalk.Document) (map[string]bson.A, error) {
	collections := make(map[string]bson.A, len(rootDoc))
	for _, topField := range rootDoc {
		array, ok := topField.Value.(jwalk.Array)
		if !ok {
			return nil, fmt.Errorf("toBSON: collection %q expects jwalk.Array, got %T", topField.Key, topField.Value)
		}

		bsonDocs := make(bson.A, 0, len(array))
		for i, element := range array {
			doc, ok := element.(jwalk.Document)
			if !ok {
				return nil, fmt.Errorf("toBSON: collection %q index %d expects jwalk.Document, got %T", topField.Key, i, element)
			}
			bsonDocs = append(bsonDocs, toBSONDocument(doc))
		}
		collections[topField.Key] = bsonDocs
	}
	return collections, nil
}

// toBSONDocument converts a jwalk.Document to bson.D.
func toBSONDocument(doc jwalk.Document) bson.D {
	bdoc := make(bson.D, 0, len(doc))
	for _, f := range doc {
		bdoc = append(bdoc, bson.E{Key: f.Key, Value: toBSONValue(f.Value)})
	}
	return bdoc
}

// toBSONArray converts a jwalk.Array to bson.A.
func toBSONArray(arr jwalk.Array) bson.A {
	barr := make(bson.A, 0, len(arr))
	for _, v := range arr {
		barr = append(barr, toBSONValue(v))
	}
	return barr
}

// toBSONValue converts nested jwalk values to their bson equivalents.
func toBSONValue(v any) any {
	switch val := v.(type) {
	case jwalk.Document:
		return toBSONDocument(val)
	case jwalk.Array:
		return toBSONArray(val)
	case unwrappable:
		return val.UnwrapValue()
	default:
		return v
	}
}
