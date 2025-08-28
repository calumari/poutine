package mongodb_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/calumari/jwalk"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/calumari/poutine/database/mongodb"
)

type MongoSuite struct {
	suite.Suite
	mongoContainer testcontainers.Container
	client         *mongo.Client
}

func (s *MongoSuite) SetupSuite() {
	t := s.T()

	mongoContainer, err := testcontainers.GenericContainer(t.Context(), testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "mongo:7",
			ExposedPorts: []string{"27017/tcp"},
			WaitingFor: wait.ForAll(
				wait.ForLog("Waiting for connections"),
				wait.ForListeningPort("27017/tcp"),
			),
		},
		Started: true,
	})
	require.NoError(t, err)

	host, err := mongoContainer.Host(t.Context())
	require.NoError(t, err)
	port, err := mongoContainer.MappedPort(t.Context(), "27017/tcp")
	require.NoError(t, err)
	uri := fmt.Sprintf("mongodb://%s:%s", host, port.Port())
	s.mongoContainer = mongoContainer

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	require.NoError(t, err)
	err = client.Ping(t.Context(), nil)
	require.NoError(t, err)
	s.client = client
}

func (s *MongoSuite) TearDownSuite() {
	_ = s.mongoContainer.Terminate(context.Background())
}

func (s *MongoSuite) TearDownTest() {
}

func TestPetStoreRepoSuite(t *testing.T) {
	suite.Run(t, new(MongoSuite))
}

// helper to create a new driver with a unique database per subtest
func (s *MongoSuite) newDriver(t *testing.T) (*mongodb.Driver, *mongo.Database) {
	t.Helper()
	// MongoDB database names must be <= 63 chars; use short prefix + 8 char suffix
	dbName := fmt.Sprintf("pmdt_%s", uuid.NewString()[:8])
	db := s.client.Database(dbName)
	return mongodb.NewDriver(db), db
}

func (s *MongoSuite) TestDriver_Seed() {
	s.Run("invailid document returns error", func() {
		t := s.T()
		driver, _ := s.newDriver(t)
		root := jwalk.Document{
			{Key: "users", Value: "not an array"},
		}
		_, err := driver.Seed(t.Context(), root)
		require.Error(t, err)
	})

	s.Run("inserts documents", func() {
		t := s.T()
		driver, db := s.newDriver(t)

		root := jwalk.Document{
			{Key: "users", Value: jwalk.Array{
				jwalk.Document{{Key: "_id", Value: "u1"}, {Key: "name", Value: "Alice"}},
				jwalk.Document{{Key: "_id", Value: "u2"}, {Key: "name", Value: "Bob"}},
			}},
		}

		seeded, err := driver.Seed(t.Context(), root)
		require.NoError(t, err)
		assert.Equal(t, root, seeded)

		// verify via mongo client
		cur, err := db.Collection("users").Find(t.Context(), bson.M{})
		require.NoError(t, err)
		defer cur.Close(t.Context())
		var docs []bson.D
		err = cur.All(t.Context(), &docs)
		require.NoError(t, err)
		require.Len(t, docs, 2)
		assert.Equal(t, bson.D{{Key: "_id", Value: "u1"}, {Key: "name", Value: "Alice"}}, docs[0])
		assert.Equal(t, bson.D{{Key: "_id", Value: "u2"}, {Key: "name", Value: "Bob"}}, docs[1])
	})

	s.Run("seed empty document creates no collections", func() {
		t := s.T()
		driver, db := s.newDriver(t)
		_, err := driver.Seed(t.Context(), jwalk.Document{})
		require.NoError(t, err)
		cols, err := db.ListCollectionNames(t.Context(), bson.M{})
		require.NoError(t, err)
		assert.Empty(t, cols)
	})
}

func (s *MongoSuite) TestDriver_Snapshot() {
	s.Run("reads existing collections", func() {
		t := s.T()
		driver, db := s.newDriver(t)

		_, err := db.Collection("posts").InsertOne(t.Context(), bson.D{{Key: "_id", Value: "p1"}, {Key: "title", Value: "Hello"}})
		require.NoError(t, err)
		_, err = db.Collection("users").InsertMany(t.Context(), []any{
			bson.D{{Key: "_id", Value: "u1"}, {Key: "name", Value: "Alice"}},
			bson.D{{Key: "_id", Value: "u2"}, {Key: "name", Value: "Bob"}},
		})
		require.NoError(t, err)

		got, err := driver.Snapshot(t.Context())
		require.NoError(t, err)

		want := jwalk.Document{
			{Key: "posts", Value: jwalk.Array{
				jwalk.Document{{Key: "_id", Value: "p1"}, {Key: "title", Value: "Hello"}},
			}},
			{Key: "users", Value: jwalk.Array{
				jwalk.Document{{Key: "_id", Value: "u1"}, {Key: "name", Value: "Alice"}},
				jwalk.Document{{Key: "_id", Value: "u2"}, {Key: "name", Value: "Bob"}},
			}},
		}
		assert.Equal(t, want, got)
	})
}

func (s *MongoSuite) TestDriver_Teardown() {
	s.Run("teardown drops database", func() {
		t := s.T()
		driver, db := s.newDriver(t)

		_, err := db.Collection("posts").InsertOne(t.Context(), bson.D{{Key: "_id", Value: "p1"}, {Key: "title", Value: "Hello"}})
		require.NoError(t, err)

		// ensure collection exists
		cols, err := db.ListCollectionNames(t.Context(), bson.M{})
		require.NoError(t, err)
		assert.Equal(t, []string{"posts"}, cols)

		err = driver.Teardown(t.Context())
		require.NoError(t, err)

		colsAfter, err := db.ListCollectionNames(t.Context(), bson.M{})
		require.NoError(t, err)
		assert.Empty(t, colsAfter)
	})
}

func (s *MongoSuite) TestDriver_RegisterTypes() {
	s.Run("register types registers ObjectID directive", func() {
		t := s.T()
		driver, _ := s.newDriver(t)
		reg, err := jwalk.NewRegistry()
		require.NoError(t, err)

		err = driver.RegisterTypes(reg)
		require.NoError(t, err)

		// second registration should fail (duplicate directive)
		err = driver.RegisterTypes(reg)
		assert.Error(t, err)
	})
}
