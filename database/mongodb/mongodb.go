package mongodb

import (
	"context"
	"fmt"
	"sort"

	"github.com/calumari/jwalk"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/calumari/poutine"
	"github.com/calumari/poutine/database"
)

type Driver struct {
	db *mongo.Database
}

var (
	_ poutine.Registrar = (*Driver)(nil)
	_ database.Driver   = (*Driver)(nil)
)

func NewDriver(db *mongo.Database) *Driver {
	return &Driver{
		db: db,
	}
}

func (d *Driver) Seed(ctx context.Context, root jwalk.Document) (jwalk.Document, error) {
	cols, err := toBSONCollections(root)
	if err != nil {
		return nil, fmt.Errorf("convert jwalk to bson: %w", err)
	}

	opts := options.BulkWrite().SetOrdered(false)

	err = d.db.Client().UseSession(ctx, func(ctx context.Context) error {
		for name, docs := range cols {
			if len(docs) == 0 {
				continue
			}
			models := make([]mongo.WriteModel, 0, len(docs))
			for _, doc := range docs {
				models = append(models, mongo.NewInsertOneModel().SetDocument(doc))
			}
			if _, err := d.db.Collection(name).BulkWrite(ctx, models, opts); err != nil {
				return fmt.Errorf("bulk write to collection %q: %w", name, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("bulk write error: %w", err)
	}

	return root, nil
}

func (d *Driver) Snapshot(ctx context.Context) (jwalk.Document, error) {
	colNames, err := d.db.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("list collection names: %w", err)
	}

	actual := make(jwalk.Document, 0, len(colNames))

	for _, colName := range colNames {
		err := func() error {
			cur, err := d.db.Collection(colName).Find(ctx, bson.M{})
			if err != nil {
				return fmt.Errorf("find in collection %q: %w", colName, err)
			}
			defer cur.Close(context.Background())

			var docs bson.A
			if err := cur.All(ctx, &docs); err != nil {
				return fmt.Errorf("decode documents in collection %q: %w", colName, err)
			}
			actual = append(actual, jwalk.Entry{Key: colName, Value: toArray(docs)})
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}

	sort.Slice(actual, func(i, j int) bool {
		return actual[i].Key < actual[j].Key
	})

	return actual, nil
}

func (d *Driver) Teardown(ctx context.Context) error {
	return d.db.Client().UseSession(ctx, func(ctx context.Context) error {
		if err := d.db.Drop(ctx); err != nil {
			return fmt.Errorf("drop database: %w", err)
		}
		return nil
	})
}

// RegisterTypes implements poutine.Registrar allowing automatic directive
// registration.
func (d *Driver) RegisterTypes(reg *jwalk.Registry) error {
	return reg.Register(ObjectIDDirective)
}
