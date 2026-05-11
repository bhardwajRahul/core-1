package postgresql

import (
	"fmt"
	"strings"

	"github.com/staticbackendhq/core/internal"
	sbquery "github.com/staticbackendhq/core/internal/query"
	"github.com/staticbackendhq/core/model"
)

func (pg *PostgreSQL) ParseQuery(clauses [][]interface{}) (map[string]interface{}, error) {
	q, err := sbquery.Parse(clauses)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{sbquery.FilterKey: q}, nil
}

func applyFilter(where string, filters map[string]interface{}, startAt int) (string, []any) {
	q, ok := sbquery.FromFilter(filters)
	if !ok {
		return applyLegacyFilter(where, filters), nil
	}

	args := make([]any, 0, len(q))
	next := startAt
	for _, clause := range q {
		fragment, values := buildClause(clause, next)
		next += len(values)
		args = append(args, values...)
		where += " AND " + fragment
	}
	return where, args
}

func buildClause(clause sbquery.Clause, startAt int) (string, []any) {
	left := fieldExpr(clause.Field, clause.Value.Type)

	switch clause.Operator {
	case sbquery.OpEqual, sbquery.OpNotEqual, sbquery.OpGreater, sbquery.OpLower, sbquery.OpGreaterEq, sbquery.OpLowerEq:
		right, args := operandExpr(clause.Value, startAt)
		return fmt.Sprintf("%s %s %s", left, clause.Operator, right), args
	case sbquery.OpIn, sbquery.OpNotIn:
		if clause.Operator == sbquery.OpNotIn {
			return fmt.Sprintf("NOT data->'%s' ? $%d", clause.Field, startAt), []any{fmt.Sprintf("%v", clause.Value.Value)}
		}
		return fmt.Sprintf("data->'%s' ? $%d", clause.Field, startAt), []any{fmt.Sprintf("%v", clause.Value.Value)}
	case sbquery.OpContains, sbquery.OpNotContains:
		not := ""
		if clause.Operator == sbquery.OpNotContains {
			not = "NOT "
		}
		return fmt.Sprintf("jsonb_typeof(data->'%s') = 'string' AND data->>'%s' %sILIKE $%d ESCAPE '\\'", clause.Field, clause.Field, not, startAt),
			[]any{escapeLikePattern(fmt.Sprintf("%v", clause.Value.Value))}
	default:
		return "TRUE", nil
	}
}

func fieldExpr(field string, typ sbquery.ValueType) string {
	expr := fmt.Sprintf("data->>'%s'", field)
	switch typ {
	case sbquery.TypeNumber:
		return numericFieldExpr(field)
	case sbquery.TypeBoolean:
		return booleanFieldExpr(field)
	}
	return expr
}

func stringFieldExpr(field string) string {
	return fmt.Sprintf("data->>'%s'", field)
}

func numericFieldExpr(field string) string {
	expr := stringFieldExpr(field)
	return fmt.Sprintf("(CASE WHEN %s ~ '^-?[0-9]+(\\.[0-9]+)?$' THEN (%s)::numeric END)", expr, expr)
}

func booleanFieldExpr(field string) string {
	expr := stringFieldExpr(field)
	return fmt.Sprintf("(CASE WHEN jsonb_typeof(data->'%s') = 'boolean' OR lower(%s) IN ('true', 'false') THEN (%s)::boolean END)", field, expr, expr)
}

func operandExpr(operand sbquery.Operand, startAt int) (string, []any) {
	if operand.Kind == sbquery.OperandField {
		return fieldExpr(operand.Field, operand.Type), nil
	}
	switch operand.Type {
	case sbquery.TypeNumber:
		return fmt.Sprintf("$%d::numeric", startAt), []any{operand.Value}
	case sbquery.TypeBoolean:
		return fmt.Sprintf("$%d::boolean", startAt), []any{operand.Value}
	}
	return fmt.Sprintf("$%d", startAt), []any{fmt.Sprintf("%v", operand.Value)}
}

func applyLegacyFilter(where string, filters map[string]interface{}) string {
	for field, val := range filters {
		switch v := val.(type) {
		case string:
			where += fmt.Sprintf(" AND %s '%s'", field, strings.ReplaceAll(v, "'", "''"))
		default:
			where += fmt.Sprintf(" AND %s '%v'", field, val)
		}
	}
	return where
}

func escapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return "%" + s + "%"
}

func secureRead(auth model.Auth, col string) string {
	if strings.HasPrefix(col, "pub_") || auth.Role == 100 {
		return "WHERE $1=$1 AND $2=$2 "
	}

	switch internal.ReadPermission(col) {
	case internal.PermGroup:
		return "WHERE account_id = $1 AND $2=$2 "
	case internal.PermOwner:
		return "WHERE account_id = $1 AND owner_id = $2 "
	default:
		//for read permission to everyone i.e. col-name_774_
		return "WHERE $1=$1 AND $2=$2 "
	}
}

func secureWrite(auth model.Auth, col string) string {
	if strings.HasPrefix(col, "pub_") || auth.Role == 100 {
		return "WHERE $1=$1 AND $2=$2 "
	}

	switch internal.WritePermission(col) {
	case internal.PermGroup:
		return "WHERE account_id = $1 AND $2=$2 "
	case internal.PermOwner:
		return "WHERE account_id = $1 AND owner_id = $2 "
	default:
		//for write permission to everyone i.e. col-name_776_
		// This should probably get more warning in the doc.
		// All logged-in users can update/delete data.
		// There's use cases for that, and it's certainly opt-in
		// but it's not recommended.
		return "WHERE $1=$1 AND $2=$2 "
	}
}

func setPaging(params model.ListParams) string {
	if len(params.SortBy) == 0 {
		params.SortBy = "created"
	}

	direction := "ASC"
	if params.SortDescending {
		direction = "DESC"
	}

	orderBy := fmt.Sprintf("ORDER BY %s %s", params.SortBy, direction)

	offset := (params.Page - 1) * params.Size
	return fmt.Sprintf("%s\nLIMIT %d OFFSET %d", orderBy, params.Size, offset)
}
