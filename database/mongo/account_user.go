package mongo

import (
	"log"
	"time"

	"github.com/staticbackendhq/core/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type LocalAccountUser struct {
	ID        primitive.ObjectID `bson:"_id"`
	UserID    primitive.ObjectID `bson:"userId"`
	AccountID primitive.ObjectID `bson:"accountId"`
	Email     string             `bson:"email"`
	Role      int                `bson:"role"`
	Token     string             `bson:"token"`
	Created   time.Time          `bson:"created"`
}

func toLocalAccountUser(au model.AccountUser) (lau LocalAccountUser, err error) {
	uid, err := primitive.ObjectIDFromHex(au.UserID)
	if err != nil {
		return
	}
	aid, err := primitive.ObjectIDFromHex(au.AccountID)
	if err != nil {
		return
	}
	lau = LocalAccountUser{
		UserID:    uid,
		AccountID: aid,
		Email:     au.Email,
		Role:      au.Role,
		Token:     au.Token,
		Created:   au.Created,
	}
	return
}

func fromLocalAccountUser(lau LocalAccountUser) model.AccountUser {
	return model.AccountUser{
		ID:        lau.ID.Hex(),
		UserID:    lau.UserID.Hex(),
		AccountID: lau.AccountID.Hex(),
		Email:     lau.Email,
		Role:      lau.Role,
		Token:     lau.Token,
		Created:   lau.Created,
	}
}

func (mg *Mongo) ensureAccountUserIndexes(db *mongo.Database) {
	col := db.Collection("sb_account_users")
	if _, err := col.Indexes().CreateMany(mg.Ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "token", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "userId", Value: 1}, {Key: "accountId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}); err != nil {
		log.Println("ensureAccountUserIndexes error:", err)
	}
}

func (mg *Mongo) AddAccountUser(dbName string, au model.AccountUser) (id string, err error) {
	db := mg.Client.Database(dbName)
	mg.ensureAccountUserIndexes(db)

	au.Created = time.Now()

	lau, err := toLocalAccountUser(au)
	if err != nil {
		return
	}
	lau.ID = primitive.NewObjectID()

	if _, err = db.Collection("sb_account_users").InsertOne(mg.Ctx, lau); err != nil {
		return
	}

	id = lau.ID.Hex()
	return
}

func (mg *Mongo) GetAccountUser(dbName, userID, accountID string) (au model.AccountUser, err error) {
	db := mg.Client.Database(dbName)

	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return
	}
	aid, err := primitive.ObjectIDFromHex(accountID)
	if err != nil {
		return
	}

	var lau LocalAccountUser
	err = db.Collection("sb_account_users").FindOne(mg.Ctx, bson.M{"userId": uid, "accountId": aid}).Decode(&lau)
	if err != nil {
		return
	}
	au = fromLocalAccountUser(lau)
	return
}

func (mg *Mongo) FindAccountUserByToken(dbName, token string) (au model.AccountUser, err error) {
	db := mg.Client.Database(dbName)

	var lau LocalAccountUser
	err = db.Collection("sb_account_users").FindOne(mg.Ctx, bson.M{"token": token}).Decode(&lau)
	if err != nil {
		return
	}
	au = fromLocalAccountUser(lau)
	return
}

func (mg *Mongo) ListAccountUsers(dbName, userID string) (results []model.AccountUser, err error) {
	db := mg.Client.Database(dbName)

	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return
	}

	cur, err := db.Collection("sb_account_users").Find(mg.Ctx, bson.M{"userId": uid})
	if err != nil {
		return
	}
	defer cur.Close(mg.Ctx)

	for cur.Next(mg.Ctx) {
		var lau LocalAccountUser
		if err = cur.Decode(&lau); err != nil {
			return
		}
		results = append(results, fromLocalAccountUser(lau))
	}

	err = cur.Err()
	return
}

func (mg *Mongo) DeleteAccountUser(dbName, id string) error {
	db := mg.Client.Database(dbName)

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = db.Collection("sb_account_users").DeleteOne(mg.Ctx, bson.M{FieldID: oid})
	return err
}
