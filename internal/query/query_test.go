package query

import "testing"

func TestParseFieldReferenceWithNumberType(t *testing.T) {
	q, err := Parse([][]interface{}{
		{"inventory", "<=", map[string]any{"$field": "inventoryThreshold", "$type": "number"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(q) != 1 {
		t.Fatalf("expected 1 clause got %d", len(q))
	}
	if q[0].Field != "inventory" || q[0].Operator != OpLowerEq {
		t.Fatalf("unexpected clause: %#v", q[0])
	}
	if q[0].Value.Kind != OperandField || q[0].Value.Field != "inventoryThreshold" || q[0].Value.Type != TypeNumber {
		t.Fatalf("unexpected operand: %#v", q[0].Value)
	}
}

func TestParseTypedLiteral(t *testing.T) {
	q, err := Parse([][]interface{}{
		{"lowStockThreshold", ">", map[string]any{"$value": 0, "$type": "number"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if q[0].Value.Kind != OperandLiteral || q[0].Value.Value != 0 || q[0].Value.Type != TypeNumber {
		t.Fatalf("unexpected operand: %#v", q[0].Value)
	}
}

func TestParseInfersTypedLiterals(t *testing.T) {
	q, err := Parse([][]interface{}{
		{"count", "==", 23},
		{"done", "==", true},
		{"visible", "==", map[string]any{"$value": false}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if q[0].Value.Type != TypeNumber {
		t.Fatalf("expected number type got %q", q[0].Value.Type)
	}
	if q[1].Value.Type != TypeBoolean {
		t.Fatalf("expected boolean type got %q", q[1].Value.Type)
	}
	if q[2].Value.Type != TypeBoolean {
		t.Fatalf("expected boolean $value type got %q", q[2].Value.Type)
	}
}

func TestParseBooleanFieldReference(t *testing.T) {
	q, err := Parse([][]interface{}{
		{"enabled", "==", map[string]any{"$field": "active", "$type": "boolean"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if q[0].Value.Kind != OperandField || q[0].Value.Field != "active" || q[0].Value.Type != TypeBoolean {
		t.Fatalf("unexpected operand: %#v", q[0].Value)
	}
}

func TestParseRejectsFieldReferenceForContains(t *testing.T) {
	_, err := Parse([][]interface{}{
		{"title", "contains", map[string]any{"$field": "otherTitle"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
