package postgresql

import (
	"fmt"

	"github.com/staticbackendhq/core/model"
)

func (pg *PostgreSQL) Count(auth model.Auth, dbName, col string, filters map[string]interface{}) (count int64, err error) {
	where := secureRead(auth, col)
	where, filterArgs := applyFilter(where, filters, 3)

	query := fmt.Sprintf(`
    SELECT COUNT(*)
    FROM %s.%s
    %s;
    `, dbName, model.CleanCollectionName(col), where)

	args := append([]any{auth.AccountID, auth.UserID}, filterArgs...)
	err = pg.DB.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return -1, err
	}

	return count, nil
}
