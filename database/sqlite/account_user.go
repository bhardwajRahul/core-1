package sqlite

import (
	"fmt"
	"time"

	"github.com/staticbackendhq/core/model"
)

func (sl *SQLite) AddAccountUser(dbName string, au model.AccountUser) (id string, err error) {
	au.Created = time.Now()
	id = sl.NewID()

	qry := fmt.Sprintf(`
		INSERT INTO %s_sb_account_users(id, user_id, account_id, email, role, token, created)
		VALUES($1, $2, $3, $4, $5, $6, $7);
	`, dbName)

	_, err = sl.DB.Exec(qry, id, au.UserID, au.AccountID, au.Email, au.Role, au.Token, au.Created)
	return
}

func (sl *SQLite) GetAccountUser(dbName, userID, accountID string) (au model.AccountUser, err error) {
	qry := fmt.Sprintf(`
		SELECT id, user_id, account_id, email, role, token, created
		FROM %s_sb_account_users
		WHERE user_id = $1 AND account_id = $2;
	`, dbName)

	err = sl.DB.QueryRow(qry, userID, accountID).Scan(
		&au.ID, &au.UserID, &au.AccountID, &au.Email, &au.Role, &au.Token, &au.Created,
	)
	return
}

func (sl *SQLite) FindAccountUserByToken(dbName, token string) (au model.AccountUser, err error) {
	qry := fmt.Sprintf(`
		SELECT id, user_id, account_id, email, role, token, created
		FROM %s_sb_account_users
		WHERE token = $1;
	`, dbName)

	err = sl.DB.QueryRow(qry, token).Scan(
		&au.ID, &au.UserID, &au.AccountID, &au.Email, &au.Role, &au.Token, &au.Created,
	)
	return
}

func (sl *SQLite) ListAccountUsers(dbName, userID string) (results []model.AccountUser, err error) {
	qry := fmt.Sprintf(`
		SELECT id, user_id, account_id, email, role, token, created
		FROM %s_sb_account_users
		WHERE user_id = $1
		ORDER BY created ASC;
	`, dbName)

	rows, err := sl.DB.Query(qry, userID)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var au model.AccountUser
		if err = rows.Scan(&au.ID, &au.UserID, &au.AccountID, &au.Email, &au.Role, &au.Token, &au.Created); err != nil {
			return
		}
		results = append(results, au)
	}

	err = rows.Err()
	return
}

func (sl *SQLite) DeleteAccountUser(dbName, id string) error {
	qry := fmt.Sprintf(`
		DELETE FROM %s_sb_account_users WHERE id = $1;
	`, dbName)

	_, err := sl.DB.Exec(qry, id)
	return err
}
