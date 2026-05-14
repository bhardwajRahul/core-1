package sqlite

import (
	"fmt"
	"time"

	"github.com/staticbackendhq/core/model"
)

func (sl *SQLite) CreateAccount(dbName, email string) (id string, err error) {
	id = sl.NewID()

	qry := fmt.Sprintf(`
		INSERT INTO %s_sb_accounts(id, email, created)
		VALUES($1, $2, $3);
	`, dbName)

	_, err = sl.DB.Exec(qry, id, email, time.Now())
	return
}

func (sl *SQLite) CreateUser(dbName string, tok model.User) (id string, err error) {
	tok.Created = time.Now()

	id = sl.NewID()

	qry := fmt.Sprintf(`
		INSERT INTO %s_sb_tokens(id, account_id, email, password, token, role, reset_code, created)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8);
	`, dbName)

	_, err = sl.DB.Exec(
		qry,
		id,
		tok.AccountID,
		tok.Email,
		tok.Password,
		tok.Token,
		tok.Role,
		tok.ResetCode,
		tok.Created,
	)
	return
}

func (sl *SQLite) UserEmailExists(dbName, email string) (exists bool, err error) {
	qry := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM %s_sb_tokens
		WHERE email = $1;
	`, dbName)

	var count int
	err = sl.DB.QueryRow(qry, email).Scan(&count)

	exists = count > 0
	return
}

func (sl *SQLite) SetUserRole(dbName, accountID, email string, role int) error {
	qry := fmt.Sprintf(`
		UPDATE %s_sb_tokens SET role = $3
		WHERE email = $1 AND account_id = $2;
	`, dbName)

	res, err := sl.DB.Exec(qry, email, accountID, role)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows > 0 {
		return nil
	}

	qry = fmt.Sprintf(`
		UPDATE %s_sb_account_users SET role = $3
		WHERE email = $1 AND account_id = $2;
	`, dbName)

	res, err = sl.DB.Exec(qry, email, accountID, role)
	if err != nil {
		return err
	}

	rows, err = res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("user membership not found")
	}
	return nil
}

func (sl *SQLite) UserSetPassword(dbName, tokenID, password string) error {
	qry := fmt.Sprintf(`
		UPDATE %s_sb_tokens SET password = $2
		WHERE id = $1;
	`, dbName)

	if _, err := sl.DB.Exec(qry, tokenID, password); err != nil {
		return err
	}
	return nil
}

func (sl *SQLite) ChangeUserEmail(dbName, userID, accountID, oldEmail, newEmail string) error {
	tx, err := sl.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qry := fmt.Sprintf(`
		UPDATE %s_sb_tokens SET email = $2
		WHERE id = $1;
	`, dbName)
	if _, err := tx.Exec(qry, userID, newEmail); err != nil {
		return err
	}

	qry = fmt.Sprintf(`
		UPDATE %s_sb_account_users SET email = $2
		WHERE user_id = $1;
	`, dbName)
	if _, err := tx.Exec(qry, userID, newEmail); err != nil {
		return err
	}

	qry = fmt.Sprintf(`
		UPDATE %s_sb_accounts SET email = $3
		WHERE id = $1 AND email = $2;
	`, dbName)
	if _, err := tx.Exec(qry, accountID, oldEmail, newEmail); err != nil {
		return err
	}

	return tx.Commit()
}

func (sl *SQLite) GetFirstUserFromAccountID(dbName, accountID string) (tok model.User, err error) {
	qry := fmt.Sprintf(`
		SELECT * 
		FROM %s_sb_tokens 
		WHERE account_id = $1
		ORDER BY created ASC
		LIMIT 1
	`, dbName)

	row := sl.DB.QueryRow(qry, accountID)

	err = scanToken(row, &tok)
	return
}

func (sl *SQLite) SetPasswordResetCode(dbName, userID, code string) error {
	qry := fmt.Sprintf(`
	UPDATE %s_sb_tokens SET
		reset_code = $2
	WHERE id = $1
`, dbName)

	_, err := sl.DB.Exec(qry, userID, code)
	if err != nil {
		return err
	}
	return nil
}

func (sl *SQLite) ResetPassword(dbName, email, code, password string) error {
	qry := fmt.Sprintf(`
		UPDATE %s_sb_tokens SET
			password = $3
		WHERE email = $1 AND reset_code = $2
	`, dbName)

	if _, err := sl.DB.Exec(qry, email, code, password); err != nil {
		return err
	}
	return nil
}

func (sl *SQLite) UpdateUserAccount(dbName, userID, newAccountID string, role int) error {
	qry := fmt.Sprintf(`
		UPDATE %s_sb_tokens SET account_id = $2, role = $3 WHERE id = $1;
	`, dbName)

	_, err := sl.DB.Exec(qry, userID, newAccountID, role)
	return err
}

func (sl *SQLite) RemoveUser(auth model.Auth, dbName, userID string) error {
	qry := fmt.Sprintf(`
	DELETE FROM %s_sb_tokens
	WHERE account_id = $1 AND id = $2;
	`, dbName)

	if _, err := sl.DB.Exec(qry, auth.AccountID, userID); err != nil {
		return err
	}
	return nil
}
