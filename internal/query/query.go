package query

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

const FilterKey = "__sb_query__"

type Operator string

const (
	OpEqual       Operator = "="
	OpNotEqual    Operator = "!="
	OpGreater     Operator = ">"
	OpLower       Operator = "<"
	OpGreaterEq   Operator = ">="
	OpLowerEq     Operator = "<="
	OpIn          Operator = "in"
	OpNotIn       Operator = "!in"
	OpContains    Operator = "contains"
	OpNotContains Operator = "!contains"
)

type ValueType string

const (
	TypeDefault ValueType = ""
	TypeNumber  ValueType = "number"
	TypeBoolean ValueType = "boolean"
)

type OperandKind int

const (
	OperandLiteral OperandKind = iota
	OperandField
)

type Operand struct {
	Kind  OperandKind
	Field string
	Value any
	Type  ValueType
}

type Clause struct {
	Field    string
	Operator Operator
	Value    Operand
}

type Query []Clause

var fieldNameRE = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

func Parse(clauses [][]interface{}) (Query, error) {
	result := make(Query, 0, len(clauses))

	for i, clause := range clauses {
		if len(clause) != 3 {
			return nil, fmt.Errorf("the %d query clause did not contains the required 3 parameters (field, operator, value)", i+1)
		}

		field, ok := clause[0].(string)
		if !ok {
			return nil, fmt.Errorf("the %d query clause's field parameter must be a string: %v", i+1, clause[0])
		}
		if err := ValidateField(field); err != nil {
			return nil, fmt.Errorf("the %d query clause's field parameter is invalid: %w", i+1, err)
		}

		op, ok := clause[1].(string)
		if !ok {
			return nil, fmt.Errorf("the %d query clause's operator must be a string: %v", i+1, clause[1])
		}
		operator, err := ParseOperator(op)
		if err != nil {
			return nil, fmt.Errorf("the %d query clause's operator: %s is not supported at the moment", i+1, op)
		}

		operand, err := ParseOperand(clause[2])
		if err != nil {
			return nil, fmt.Errorf("the %d query clause's value parameter is invalid: %w", i+1, err)
		}

		if operand.Kind == OperandField && !supportsFieldOperand(operator) {
			return nil, fmt.Errorf("the %d query clause's operator: %s does not support field values", i+1, op)
		}

		result = append(result, Clause{
			Field:    field,
			Operator: operator,
			Value:    operand,
		})
	}

	return result, nil
}

func FromFilter(filter map[string]interface{}) (Query, bool) {
	v, ok := filter[FilterKey]
	if !ok {
		return nil, false
	}
	q, ok := v.(Query)
	return q, ok
}

func ParseOperator(op string) (Operator, error) {
	switch op {
	case "=", "==":
		return OpEqual, nil
	case "!=", "<>":
		return OpNotEqual, nil
	case ">":
		return OpGreater, nil
	case "<":
		return OpLower, nil
	case ">=":
		return OpGreaterEq, nil
	case "<=":
		return OpLowerEq, nil
	case "in":
		return OpIn, nil
	case "!in", "nin":
		return OpNotIn, nil
	case "contains":
		return OpContains, nil
	case "!contains":
		return OpNotContains, nil
	default:
		return "", errors.New("unsupported operator")
	}
}

func ParseOperand(v any) (Operand, error) {
	m, ok := v.(map[string]interface{})
	if !ok {
		if ma, ok := v.(map[string]any); ok {
			m = ma
			ok = true
		}
	}
	if !ok {
		return Operand{Kind: OperandLiteral, Value: v, Type: InferValueType(v)}, nil
	}

	typ, err := parseType(m["$type"])
	if err != nil {
		return Operand{}, err
	}

	if field, ok := m["$field"]; ok {
		fieldName, ok := field.(string)
		if !ok {
			return Operand{}, errors.New("$field must be a string")
		}
		if err := ValidateField(fieldName); err != nil {
			return Operand{}, err
		}
		return Operand{Kind: OperandField, Field: fieldName, Type: typ}, nil
	}

	if value, ok := m["$value"]; ok {
		if typ == TypeDefault {
			typ = InferValueType(value)
		}
		return Operand{Kind: OperandLiteral, Value: value, Type: typ}, nil
	}

	return Operand{Kind: OperandLiteral, Value: v, Type: InferValueType(v)}, nil
}

func ValidateField(field string) error {
	if !fieldNameRE.MatchString(field) {
		return fmt.Errorf("%q must match %s", field, fieldNameRE.String())
	}
	return nil
}

func IsComparison(op Operator) bool {
	switch op {
	case OpEqual, OpNotEqual, OpGreater, OpLower, OpGreaterEq, OpLowerEq:
		return true
	default:
		return false
	}
}

func parseType(v any) (ValueType, error) {
	if v == nil {
		return TypeDefault, nil
	}
	s, ok := v.(string)
	if !ok {
		return "", errors.New("$type must be a string")
	}
	switch ValueType(s) {
	case TypeDefault, TypeNumber, TypeBoolean:
		return ValueType(s), nil
	default:
		return "", fmt.Errorf("$type %q is not supported", s)
	}
}

func InferValueType(v any) ValueType {
	switch val := v.(type) {
	case bool:
		return TypeBoolean
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return TypeNumber
	case json.Number:
		if _, err := strconv.ParseFloat(string(val), 64); err == nil {
			return TypeNumber
		}
	}
	return TypeDefault
}

func supportsFieldOperand(op Operator) bool {
	return IsComparison(op)
}
