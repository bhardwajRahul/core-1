package mongo

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func TestParseQueryFieldReference(t *testing.T) {
	filters, err := (&Mongo{}).ParseQuery([][]interface{}{
		{"inventory", "<=", map[string]any{"$field": "inventoryThreshold", "$type": "number"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	expr, ok := filters["$expr"].(bson.M)
	if !ok {
		t.Fatalf("expected $expr got %#v", filters)
	}
	values, ok := expr["$lte"].(bson.A)
	if !ok {
		t.Fatalf("expected $lte expression got %#v", expr)
	}
	if len(values) != 2 || values[0] != "$inventory" || values[1] != "$inventoryThreshold" {
		t.Fatalf("unexpected expression values: %#v", values)
	}
}
