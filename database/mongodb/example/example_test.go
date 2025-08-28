package example_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/calumari/poutine"
	"github.com/calumari/poutine/database/mongodb"
	"github.com/calumari/poutine/testine"
)

type RepositorySuite struct {
	suite.Suite
	mongoContainer testcontainers.Container
	client         *mongo.Client
	repository     *Repository
	poutine        *poutine.Poutine
	testine        *testine.T
}

func (s *RepositorySuite) SetupSuite() {
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

func (s *RepositorySuite) TearDownSuite() {
	_ = s.mongoContainer.Terminate(context.Background())
}

func (s *RepositorySuite) SetupSubTest() {
	t := s.T()

	dbName := fmt.Sprintf("poutine_example_test_%s", uuid.NewString())
	fmt.Println("Using database:", dbName)
	db := s.client.Database(dbName)

	s.poutine = poutine.New(mongodb.NewDriver(db))
	pt, err := testine.New(s.poutine)
	require.NoError(t, err)
	s.testine = pt

	s.repository = NewRepository(db)
}

func (s *RepositorySuite) TearDownTest() {
	err := s.poutine.Teardown(context.Background())
	require.NoError(s.T(), err)
}

func (s *RepositorySuite) TestCreatePet() {
	s.Run("create pet adds pet", func() {
		t := s.T()
		err := s.repository.Create(t.Context(), &Pet{Name: "Luna", Type: "cat"})
		require.NoError(t, err)

		s.testine.Assert(t, s.testine.LoadJSON(t, "test_data/pets_after_create.json"))
	})
}

func (s *RepositorySuite) TestDeletePet() {
	s.Run("delete non-existing pet keeps snapshot unchanged", func() {
		t := s.T()
		snapshot := s.testine.Seed(t, s.testine.LoadJSON(t, "test_data/pets_seed.json"))

		err := s.repository.Delete(t.Context(), "NonExistent")
		require.NoError(t, err)

		snapshot.Assert(t)
	})

	s.Run("delete existing pet removes pet", func() {
		t := s.T()
		_ = s.testine.Seed(t, s.testine.LoadJSON(t, "test_data/pets_seed.json"))

		err := s.repository.Delete(t.Context(), "Max")
		require.NoError(t, err)

		s.testine.Assert(t, s.testine.LoadJSON(t, "test_data/pets_after_delete.json"))
	})
}

func TestRepositorySuite(t *testing.T) {
	suite.Run(t, new(RepositorySuite))
}

// example repository and model

type Pet struct {
	ID   bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Name string        `bson:"name" json:"name"`
	Type string        `bson:"type" json:"type"`
}

type Repository struct {
	collection *mongo.Collection
}

func NewRepository(db *mongo.Database) *Repository {
	return &Repository{collection: db.Collection("pets")}
}

func (s *Repository) Create(ctx context.Context, pet *Pet) error {
	_, err := s.collection.InsertOne(ctx, bson.M{"name": pet.Name, "type": pet.Type})
	return err
}

func (s *Repository) Delete(ctx context.Context, name string) error {
	_, err := s.collection.DeleteOne(ctx, bson.M{"name": name})
	return err
}
