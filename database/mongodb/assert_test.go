package mongodb

import (
	"testing"

	"github.com/calumari/jwalk"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func Test_toDocument(t *testing.T) {
	t.Run("valid bson document returns jwalk document", func(t *testing.T) {
		d := bson.D{{Key: "foo", Value: 1}, {Key: "bar", Value: 2}}
		got := toDocument(d)
		assert.Len(t, got, 2)
		assert.Equal(t, "foo", got[0].Key)
		assert.Equal(t, 1, got[0].Value)
	})

	t.Run("empty bson document returns empty jwalk document", func(t *testing.T) {
		got := toDocument(bson.D{})
		assert.Empty(t, got)
	})

	t.Run("nested document and array convert recursively", func(t *testing.T) {
		d := bson.D{
			{Key: "child", Value: bson.D{{Key: "leaf", Value: 7}}},
			{Key: "list", Value: bson.A{1, bson.D{{Key: "deep", Value: 9}}}},
		}
		got := toDocument(d)
		assert.Len(t, got, 2)

		childVal, ok := got[0].Value.(jwalk.Document)
		assert.True(t, ok)
		assert.Equal(t, "leaf", childVal[0].Key)
		assert.Equal(t, 7, childVal[0].Value)

		listVal, ok := got[1].Value.(jwalk.Array)
		assert.True(t, ok)
		assert.Equal(t, 1, listVal[0])
		deepDoc := listVal[1].(jwalk.Document)
		assert.Equal(t, "deep", deepDoc[0].Key)
		assert.Equal(t, 9, deepDoc[0].Value)
	})
}

func Test_toArray(t *testing.T) {
	t.Run("valid bson array returns jwalk array", func(t *testing.T) {
		a := bson.A{1, 2, 3}
		got := toArray(a)
		assert.Equal(t, jwalk.Array{1, 2, 3}, got)
	})

	t.Run("empty bson array returns empty jwalk array", func(t *testing.T) {
		got := toArray(bson.A{})
		assert.Empty(t, got)
	})

	t.Run("nested array with documents converts recursively", func(t *testing.T) {
		a := bson.A{
			1,
			bson.D{{Key: "foo", Value: 2}},
			bson.A{
				bson.D{{Key: "bar", Value: 3}},
				4,
			},
		}
		got := toArray(a)
		assert.Len(t, got, 3)
		assert.Equal(t, 1, got[0])

		doc1 := got[1].(jwalk.Document)
		assert.Equal(t, "foo", doc1[0].Key)
		assert.Equal(t, 2, doc1[0].Value)

		nestedArr := got[2].(jwalk.Array)
		assert.Len(t, nestedArr, 2)
		doc2 := nestedArr[0].(jwalk.Document)
		assert.Equal(t, "bar", doc2[0].Key)
		assert.Equal(t, 3, doc2[0].Value)
		assert.Equal(t, 4, nestedArr[1])
	})
}

func Test_toValue(t *testing.T) {
	t.Run("bson document returns jwalk document", func(t *testing.T) {
		d := bson.D{{Key: "foo", Value: 1}}
		got := toValue(d)
		doc, ok := got.(jwalk.Document)
		assert.True(t, ok)
		assert.Equal(t, "foo", doc[0].Key)
	})

	t.Run("bson array returns jwalk array", func(t *testing.T) {
		a := bson.A{1, 2}
		got := toValue(a)
		arr, ok := got.(jwalk.Array)
		assert.True(t, ok)
		assert.Equal(t, jwalk.Array{1, 2}, arr)
	})

	t.Run("other value returns itself", func(t *testing.T) {
		got := toValue(42)
		assert.Equal(t, 42, got)
	})

	t.Run("nested mixed structure converts recursively", func(t *testing.T) {
		val := bson.D{
			{Key: "arr", Value: bson.A{
				bson.D{{Key: "innerDoc", Value: bson.A{1, 2}}},
				5,
			}},
			{Key: "doc", Value: bson.D{
				{Key: "nested", Value: bson.D{{Key: "x", Value: 8}}},
			}},
		}
		got := toValue(val)
		doc := got.(jwalk.Document)
		assert.Len(t, doc, 2)

		arrVal := doc[0].Value.(jwalk.Array)
		assert.Len(t, arrVal, 2)
		innerDoc := arrVal[0].(jwalk.Document)
		assert.Equal(t, "innerDoc", innerDoc[0].Key)
		innerDocArr := innerDoc[0].Value.(jwalk.Array)
		assert.Equal(t, jwalk.Array{1, 2}, innerDocArr)
		assert.Equal(t, 5, arrVal[1])

		docVal := doc[1].Value.(jwalk.Document)
		nestedDoc := docVal[0].Value.(jwalk.Document)
		assert.Equal(t, "x", nestedDoc[0].Key)
		assert.Equal(t, 8, nestedDoc[0].Value)
	})
}
