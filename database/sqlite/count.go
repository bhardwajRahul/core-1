package sqlite

import (
	"fmt"

	"github.com/staticbackendhq/core/model"
)

func (sl *SQLite) Count(auth model.Auth, dbName, col string, filters map[string]interface{}) (count int64, err error) {
	where := secureRead(auth, col)
	where, filterArgs := applyFilter(where, filters, 3)

	query := fmt.Sprintf(`
    SELECT COUNT(*)
    FROM %s_%s
    %s;
    `, dbName, model.CleanCollectionName(col), where)

	args := append([]any{auth.AccountID, auth.UserID}, filterArgs...)
	err = sl.DB.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return -1, err
	}

	return count, nil
}
