package mongodb

import (
	"fmt"

	"github.com/calumari/jwalk"
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/calumari/poutine/exp"
)

var ObjectIDDirective = jwalk.NewDirective("oid", unmarshalOIDPattern)

func unmarshalOIDPattern(dec *jsontext.Decoder) (exp.Pattern[bson.ObjectID], error) {
	var raw any
	if err := json.UnmarshalDecode(dec, &raw); err != nil {
		return exp.Pattern[bson.ObjectID]{}, err
	}
	switch v := raw.(type) {
	case bool:
		if !v {
			return exp.Pattern[bson.ObjectID]{}, fmt.Errorf("$oid bool must be true to indicate wildcard")
		}
		// wildcard presence (no explicit user value)
		return exp.Any(bson.NewObjectID()), nil
	case string:
		oid, err := bson.ObjectIDFromHex(v)
		if err != nil {
			return exp.Pattern[bson.ObjectID]{}, err
		}
		return exp.Value(oid), nil
	default:
		return exp.Pattern[bson.ObjectID]{}, fmt.Errorf("invalid $oid payload type %T", v)
	}
}
