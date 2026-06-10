package sqlite

import (
	"fmt"
	"strings"

	"github.com/staticbackendhq/core/internal"
	sbquery "github.com/staticbackendhq/core/internal/query"
	"github.com/staticbackendhq/core/model"
)

func (sl *SQLite) ParseQuery(clauses [][]interface{}) (map[string]interface{}, error) {
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
		list := listValues(clause.Value.Value)
		if len(list) == 0 {
			if clause.Operator == sbquery.OpNotIn {
				return "TRUE", nil
			}
			return "FALSE", nil
		}
		placeholders := make([]string, 0, len(list))
		args := make([]any, 0, len(list))
		for i, item := range list {
			placeholders = append(placeholders, fmt.Sprintf("$%d", startAt+i))
			args = append(args, item)
		}
		not := ""
		if clause.Operator == sbquery.OpNotIn {
			not = "NOT "
		}
		return fmt.Sprintf("%sEXISTS (SELECT 1 FROM json_each(json_extract(data, \"$.%s\")) WHERE value IN (%s))",
			not, clause.Field, strings.Join(placeholders, ", ")), args
	case sbquery.OpContains, sbquery.OpNotContains:
		not := ""
		if clause.Operator == sbquery.OpNotContains {
			not = "NOT "
		}
		return fmt.Sprintf("json_type(data, \"$.%s\") = 'text' AND json_extract(data, \"$.%s\") %sLIKE $%d ESCAPE '\\'",
			clause.Field, clause.Field, not, startAt), []any{escapeLikePattern(fmt.Sprintf("%v", clause.Value.Value))}
	default:
		return "TRUE", nil
	}
}

func fieldExpr(field string, typ sbquery.ValueType) string {
	expr := fmt.Sprintf("json_extract(data, \"$.%s\")", field)
	switch typ {
	case sbquery.TypeNumber:
		return fmt.Sprintf("CAST(%s AS REAL)", expr)
	case sbquery.TypeBoolean:
		return fmt.Sprintf("(CASE WHEN json_type(data, \"$.%s\") IN ('true', 'false') THEN CAST(%s AS INTEGER) END)", field, expr)
	}
	return expr
}

func operandExpr(operand sbquery.Operand, startAt int) (string, []any) {
	if operand.Kind == sbquery.OperandField {
		return fieldExpr(operand.Field, operand.Type), nil
	}
	switch operand.Type {
	case sbquery.TypeNumber:
		return fmt.Sprintf("CAST($%d AS REAL)", startAt), []any{operand.Value}
	case sbquery.TypeBoolean:
		return fmt.Sprintf("CAST($%d AS INTEGER)", startAt), []any{operand.Value}
	}
	return fmt.Sprintf("$%d", startAt), []any{operand.Value}
}

func listValues(v any) []any {
	switch list := v.(type) {
	case []any:
		return list
	case []string:
		values := make([]any, 0, len(list))
		for _, item := range list {
			values = append(values, item)
		}
		return values
	default:
		return []any{v}
	}
}

func applyLegacyFilter(where string, filters map[string]interface{}) string {
	for field, val := range filters {
		if s, ok := val.(string); ok {
			s = strings.ReplaceAll(s, "'", "''")
			where += fmt.Sprintf(" AND %s '%v'", field, s)
		} else if list, ok := val.([]string); ok {
			var s string
			for _, item := range list {
				s += fmt.Sprintf("'%s', ", strings.ReplaceAll(item, "'", "''"))
			}

			s = strings.TrimRight(s, ", ")
			where += fmt.Sprintf(" AND %s", strings.ReplaceAll(field, "_in_", s))
		} else {
			where += fmt.Sprintf(" AND %s %v", field, val)
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
