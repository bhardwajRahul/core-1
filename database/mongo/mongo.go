package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/database"
	"github.com/staticbackendhq/core/logger"
	"github.com/staticbackendhq/core/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Mongo struct {
	Client          *mongo.Client
	Ctx             context.Context
	PublishDocument cache.PublishDocumentEvent
	log             *logger.Logger
}

func New(client *mongo.Client, pubdoc cache.PublishDocumentEvent, log *logger.Logger) database.Persister {
	return &Mongo{
		Client:          client,
		Ctx:             context.Background(),
		PublishDocument: pubdoc,
		log:             log,
	}
}

func (mg *Mongo) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return mg.Client.Ping(ctx, readpref.Primary())
}

func (mg *Mongo) CreateIndex(dbName, col, field string) error {
	db := mg.Client.Database(dbName)

	idx := mongo.IndexModel{
		Keys: bson.M{field: 1},
	}

	dbCol := db.Collection(model.CleanCollectionName(col))

	if _, err := dbCol.Indexes().CreateOne(mg.Ctx, idx); err != nil {
		return err
	}
	return nil
}

func (mg *Mongo) CreateTypedIndex(dbName, col, field string, typ database.IndexType) error {
	if !database.IsSupportedIndexType(typ) {
		return fmt.Errorf("index type %q is not supported", typ)
	}
	return mg.CreateIndex(dbName, col, field)
}
