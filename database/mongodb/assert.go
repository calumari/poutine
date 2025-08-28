package mongodb

import (
	"github.com/calumari/jwalk"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func toDocument(d bson.D) jwalk.Document {
	doc := make(jwalk.Document, 0, len(d))
	for _, e := range d {
		doc = append(doc, jwalk.Entry{Key: e.Key, Value: toValue(e.Value)})
	}
	return doc
}

func toArray(a bson.A) jwalk.Array {
	arr := make(jwalk.Array, 0, len(a))
	for _, v := range a {
		arr = append(arr, toValue(v))
	}
	return arr
}

func toValue(v any) any {
	switch val := v.(type) {
	case bson.D:
		return toDocument(val)
	case bson.A:
		return toArray(val)
	default:
		return v
	}
}
