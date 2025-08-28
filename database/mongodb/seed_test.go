package mongodb

import (
	"testing"

	"github.com/calumari/jwalk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type unwrapValue struct {
	value any
}

func (u unwrapValue) UnwrapValue() any {
	return u.value
}

func Test_toBSONCollections(t *testing.T) {
	t.Run("valid jwalk document with collections returns map of bson arrays", func(t *testing.T) {
		root := jwalk.Document{
			{Key: "users", Value: jwalk.Array{
				jwalk.Document{{Key: "name", Value: "Alice"}},
				jwalk.Document{{Key: "name", Value: "Bob"}},
			}},
			{Key: "posts", Value: jwalk.Array{
				jwalk.Document{{Key: "title", Value: "Hello"}},
			}},
		}
		got, err := toBSONCollections(root)
		require.NoError(t, err)
		require.Len(t, got, 2)

		users, ok := got["users"]
		require.True(t, ok)
		require.Len(t, users, 2)
		user1 := users[0].(bson.D)
		user2 := users[1].(bson.D)
		assert.Equal(t, bson.E{Key: "name", Value: "Alice"}, user1[0])
		assert.Equal(t, bson.E{Key: "name", Value: "Bob"}, user2[0])

		posts, ok := got["posts"]
		require.True(t, ok)
		require.Len(t, posts, 1)
		post1 := posts[0].(bson.D)
		assert.Equal(t, bson.E{Key: "title", Value: "Hello"}, post1[0])
	})

	t.Run("non-array value for collection returns error", func(t *testing.T) {
		root := jwalk.Document{
			{Key: "users", Value: "not an array"},
		}
		_, err := toBSONCollections(root)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expects jwalk.Array")
	})

	t.Run("array with non-document value returns error", func(t *testing.T) {
		root := jwalk.Document{
			{Key: "users", Value: jwalk.Array{"not a document"}},
		}
		_, err := toBSONCollections(root)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expects jwalk.Document")
	})
}

func Test_toBSONDocument(t *testing.T) {
	t.Run("valid jwalk document returns bson.D", func(t *testing.T) {
		doc := jwalk.Document{
			{Key: "name", Value: "Alice"},
			{Key: "age", Value: 30},
		}
		got := toBSONDocument(doc)
		assert.Equal(t, bson.D{
			{Key: "name", Value: "Alice"},
			{Key: "age", Value: 30},
		}, got)
	})

	t.Run("nested document converts recursively", func(t *testing.T) {
		doc := jwalk.Document{
			{Key: "user", Value: jwalk.Document{
				{Key: "name", Value: "Bob"},
			}},
		}
		got := toBSONDocument(doc)
		assert.Equal(t, bson.D{
			{Key: "user", Value: bson.D{
				{Key: "name", Value: "Bob"},
			}},
		}, got)
	})

	t.Run("nested array converts recursively", func(t *testing.T) {
		doc := jwalk.Document{
			{Key: "tags", Value: jwalk.Array{"go", "mongodb"}},
		}
		got := toBSONDocument(doc)
		assert.Equal(t, bson.D{
			{Key: "tags", Value: bson.A{"go", "mongodb"}},
		}, got)
	})
}

func Test_toBSONArray(t *testing.T) {
	t.Run("valid jwalk array returns bson.A", func(t *testing.T) {
		arr := jwalk.Array{1, "two", 3.0}
		got := toBSONArray(arr)
		assert.Equal(t, bson.A{1, "two", 3.0}, got)
	})

	t.Run("nested document converts recursively", func(t *testing.T) {
		arr := jwalk.Array{
			jwalk.Document{{Key: "name", Value: "Alice"}},
		}
		got := toBSONArray(arr)
		assert.Equal(t, bson.A{
			bson.D{{Key: "name", Value: "Alice"}},
		}, got)
	})

	t.Run("nested array converts recursively", func(t *testing.T) {
		arr := jwalk.Array{
			jwalk.Array{1, 2},
		}
		got := toBSONArray(arr)
		assert.Equal(t, bson.A{
			bson.A{1, 2},
		}, got)
	})
}

func Test_toBSONVal(t *testing.T) {
	t.Run("jwalk document converts to bson.D", func(t *testing.T) {
		val := jwalk.Document{
			{Key: "name", Value: "Alice"},
		}
		got := toBSONValue(val)
		assert.Equal(t, bson.D{
			{Key: "name", Value: "Alice"},
		}, got)
	})

	t.Run("jwalk array converts to bson.A", func(t *testing.T) {
		val := jwalk.Array{1, 2, 3}
		got := toBSONValue(val)
		assert.Equal(t, bson.A{1, 2, 3}, got)
	})

	t.Run("primitive value returns itself", func(t *testing.T) {
		val := "just a string"
		got := toBSONValue(val)
		assert.Equal(t, val, got)
	})

	t.Run("unwraps value implementing UnwrapValue", func(t *testing.T) {
		val := unwrapValue{value: 42}
		got := toBSONValue(val)
		assert.Equal(t, 42, got)
	})
}
