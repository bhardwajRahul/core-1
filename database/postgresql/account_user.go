package postgresql

import (
	"fmt"
	"time"

	"github.com/staticbackendhq/core/model"
)

func (pg *PostgreSQL) AddAccountUser(dbName string, au model.AccountUser) (id string, err error) {
	au.Created = time.Now()

	qry := fmt.Sprintf(`
		INSERT INTO %s.sb_account_users(user_id, account_id, email, role, token, created)
		VALUES($1, $2, $3, $4, $5, $6)
		RETURNING id;
	`, dbName)

	err = pg.DB.QueryRow(qry,
		au.UserID,
		au.AccountID,
		au.Email,
		au.Role,
		au.Token,
		au.Created,
	).Scan(&id)
	return
}

func (pg *PostgreSQL) GetAccountUser(dbName, userID, accountID string) (au model.AccountUser, err error) {
	qry := fmt.Sprintf(`
		SELECT id, user_id, account_id, email, role, token, created
		FROM %s.sb_account_users
		WHERE user_id = $1 AND account_id = $2;
	`, dbName)

	err = pg.DB.QueryRow(qry, userID, accountID).Scan(
		&au.ID, &au.UserID, &au.AccountID, &au.Email, &au.Role, &au.Token, &au.Created,
	)
	return
}

func (pg *PostgreSQL) FindAccountUserByToken(dbName, token string) (au model.AccountUser, err error) {
	qry := fmt.Sprintf(`
		SELECT id, user_id, account_id, email, role, token, created
		FROM %s.sb_account_users
		WHERE token = $1;
	`, dbName)

	err = pg.DB.QueryRow(qry, token).Scan(
		&au.ID, &au.UserID, &au.AccountID, &au.Email, &au.Role, &au.Token, &au.Created,
	)
	return
}

func (pg *PostgreSQL) ListAccountUsers(dbName, userID string) (results []model.AccountUser, err error) {
	qry := fmt.Sprintf(`
		SELECT id, user_id, account_id, email, role, token, created
		FROM %s.sb_account_users
		WHERE user_id = $1
		ORDER BY created ASC;
	`, dbName)

	rows, err := pg.DB.Query(qry, userID)
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

func (pg *PostgreSQL) DeleteAccountUser(dbName, id string) error {
	qry := fmt.Sprintf(`
		DELETE FROM %s.sb_account_users WHERE id = $1;
	`, dbName)

	_, err := pg.DB.Exec(qry, id)
	return err
}
