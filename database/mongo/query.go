package mongo

import (
	"fmt"
	"regexp"

	"github.com/staticbackendhq/core/internal"
	sbquery "github.com/staticbackendhq/core/internal/query"
	"github.com/staticbackendhq/core/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (mg *Mongo) ParseQuery(clauses [][]interface{}) (map[string]interface{}, error) {
	q, err := sbquery.Parse(clauses)
	if err != nil {
		return nil, err
	}
	return buildFilter(q), nil
}

func buildFilter(q sbquery.Query) bson.M {
	filter := bson.M{}
	var exprs []bson.M

	for _, clause := range q {
		if clause.Value.Kind == sbquery.OperandField {
			exprs = append(exprs, fieldExpr(clause))
			continue
		}

		switch clause.Operator {
		case sbquery.OpEqual:
			filter[clause.Field] = clause.Value.Value
		case sbquery.OpNotEqual:
			filter[clause.Field] = bson.M{"$ne": clause.Value.Value}
		case sbquery.OpGreater:
			filter[clause.Field] = bson.M{"$gt": clause.Value.Value}
		case sbquery.OpLower:
			filter[clause.Field] = bson.M{"$lt": clause.Value.Value}
		case sbquery.OpGreaterEq:
			filter[clause.Field] = bson.M{"$gte": clause.Value.Value}
		case sbquery.OpLowerEq:
			filter[clause.Field] = bson.M{"$lte": clause.Value.Value}
		case sbquery.OpIn:
			filter[clause.Field] = bson.M{"$in": clause.Value.Value}
		case sbquery.OpNotIn:
			filter[clause.Field] = bson.M{"$nin": clause.Value.Value}
		case sbquery.OpContains:
			filter[clause.Field] = bson.M{"$type": "string", "$regex": regexp.QuoteMeta(fmt.Sprintf("%v", clause.Value.Value)), "$options": "i"}
		case sbquery.OpNotContains:
			filter[clause.Field] = bson.M{"$type": "string", "$not": primitive.Regex{Pattern: regexp.QuoteMeta(fmt.Sprintf("%v", clause.Value.Value)), Options: "i"}}
		}
	}

	if len(exprs) == 1 {
		filter["$expr"] = exprs[0]
	} else if len(exprs) > 1 {
		filter["$and"] = exprFilter(exprs)
	}

	return filter
}

func fieldExpr(clause sbquery.Clause) bson.M {
	op := "$eq"
	switch clause.Operator {
	case sbquery.OpNotEqual:
		op = "$ne"
	case sbquery.OpGreater:
		op = "$gt"
	case sbquery.OpLower:
		op = "$lt"
	case sbquery.OpGreaterEq:
		op = "$gte"
	case sbquery.OpLowerEq:
		op = "$lte"
	}
	return bson.M{op: bson.A{"$" + clause.Field, "$" + clause.Value.Field}}
}

func exprFilter(exprs []bson.M) bson.A {
	items := make(bson.A, 0, len(exprs))
	for _, expr := range exprs {
		items = append(items, bson.M{"$expr": expr})
	}
	return items
}

func secureRead(acctID, userID primitive.ObjectID, role int, col string, filter bson.M) {
	switch internal.ReadScope(model.Auth{Role: role}, col) {
	case internal.RowScopeAccount:
		filter[FieldAccountID] = acctID
	case internal.RowScopeOwner:
		filter[FieldAccountID] = acctID
		filter[FieldOwnerID] = userID
	}
}

func secureWrite(acctID, userID primitive.ObjectID, role int, col string, filter bson.M) {
	switch internal.WriteScope(model.Auth{Role: role}, col, false) {
	case internal.RowScopeAccount:
		filter[FieldAccountID] = acctID
	case internal.RowScopeOwner:
		filter[FieldAccountID] = acctID
		filter[FieldOwnerID] = userID
	}
}
