package memory

import (
	"github.com/staticbackendhq/core/internal"
	sbquery "github.com/staticbackendhq/core/internal/query"
	"github.com/staticbackendhq/core/model"
)

func (m *Memory) ParseQuery(clauses [][]interface{}) (filter map[string]any, err error) {
	q, err := sbquery.Parse(clauses)
	if err != nil {
		return nil, err
	}

	return map[string]any{sbquery.FilterKey: q}, nil
}

func secureRead(auth model.Auth, col string, list []map[string]any) []map[string]any {
	var filtered []map[string]any

	filter := make(map[string]string)

	switch internal.ReadScope(auth, col) {
	case internal.RowScopeAccount:
		filter[FieldAccountID] = auth.AccountID
	case internal.RowScopeOwner:
		filter[FieldAccountID] = auth.AccountID
		filter[FieldOwnerID] = auth.UserID
	}

	for _, doc := range list {
		matches := 0
		for k, v := range filter {
			if doc[k] == v {
				matches++
			}
		}

		if matches == len(filter) {
			filtered = append(filtered, doc)
		}

	}

	return filtered
}

func canWrite(auth model.Auth, col string, doc map[string]any) bool {
	switch internal.WriteScope(auth, col, false) {
	case internal.RowScopeAccount:
		return doc[FieldAccountID] == auth.AccountID
	case internal.RowScopeOwner:
		return doc[FieldAccountID] == auth.AccountID && doc[FieldOwnerID] == auth.UserID
	}

	return true
}
