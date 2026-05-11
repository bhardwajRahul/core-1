package postgresql

import (
	"strings"
	"testing"

	sbquery "github.com/staticbackendhq/core/internal/query"
)

func TestApplyFilterFieldReferenceWithNumberType(t *testing.T) {
	filters := map[string]interface{}{
		sbquery.FilterKey: sbquery.Query{
			{
				Field:    "inventory",
				Operator: sbquery.OpLowerEq,
				Value: sbquery.Operand{
					Kind:  sbquery.OperandField,
					Field: "inventoryThreshold",
					Type:  sbquery.TypeNumber,
				},
			},
		},
	}

	where, args := applyFilter("WHERE $1=$1 AND $2=$2 ", filters, 3)
	if len(args) != 0 {
		t.Fatalf("expected no args got %v", args)
	}
	if !strings.Contains(where, "data->>'inventory'") || !strings.Contains(where, "data->>'inventoryThreshold'") || !strings.Contains(where, "::numeric") {
		t.Fatalf("expected numeric field comparison, got %s", where)
	}
}

func TestApplyFilterInfersNumberLiteral(t *testing.T) {
	filters, err := (&PostgreSQL{}).ParseQuery([][]interface{}{
		{"count", ">=", 23},
	})
	if err != nil {
		t.Fatal(err)
	}

	where, args := applyFilter("WHERE $1=$1 AND $2=$2 ", filters, 3)
	if len(args) != 1 || args[0] != 23 {
		t.Fatalf("unexpected args: %v", args)
	}
	if !strings.Contains(where, "data->>'count'") || !strings.Contains(where, "::numeric") || !strings.Contains(where, "$3::numeric") {
		t.Fatalf("expected numeric literal comparison, got %s", where)
	}
}

func TestApplyFilterInfersBooleanLiteral(t *testing.T) {
	filters, err := (&PostgreSQL{}).ParseQuery([][]interface{}{
		{"done", "==", true},
	})
	if err != nil {
		t.Fatal(err)
	}

	where, args := applyFilter("WHERE $1=$1 AND $2=$2 ", filters, 3)
	if len(args) != 1 || args[0] != true {
		t.Fatalf("unexpected args: %v", args)
	}
	if !strings.Contains(where, "jsonb_typeof(data->'done') = 'boolean'") || !strings.Contains(where, "::boolean") || !strings.Contains(where, "$3::boolean") {
		t.Fatalf("expected boolean literal comparison, got %s", where)
	}
}

func TestApplyFilterFieldReferenceWithBooleanType(t *testing.T) {
	filters := map[string]interface{}{
		sbquery.FilterKey: sbquery.Query{
			{
				Field:    "enabled",
				Operator: sbquery.OpEqual,
				Value: sbquery.Operand{
					Kind:  sbquery.OperandField,
					Field: "active",
					Type:  sbquery.TypeBoolean,
				},
			},
		},
	}

	where, args := applyFilter("WHERE $1=$1 AND $2=$2 ", filters, 3)
	if len(args) != 0 {
		t.Fatalf("expected no args got %v", args)
	}
	if !strings.Contains(where, "jsonb_typeof(data->'enabled') = 'boolean'") || !strings.Contains(where, "jsonb_typeof(data->'active') = 'boolean'") {
		t.Fatalf("expected boolean field comparison, got %s", where)
	}
}
