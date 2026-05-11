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
