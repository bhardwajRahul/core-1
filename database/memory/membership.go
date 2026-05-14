package memory

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/staticbackendhq/core/model"
)

func (m *Memory) CreateAccount(dbName, email string) (id string, err error) {
	id = m.NewID()

	acct := model.Account{
		ID:      id,
		Created: time.Now(),
		Email:   email,
	}
	err = create(m, dbName, "sb_accounts", id, acct)
	return
}

func (m *Memory) CreateUser(dbName string, tok model.User) (id string, err error) {
	tok.Created = time.Now()

	id = m.NewID()
	tok.ID = id

	err = create(m, dbName, "sb_tokens", id, tok)
	return
}

func (m *Memory) SetPasswordResetCode(dbName, tokenID, code string) error {
	var tok model.User
	if err := getByID(m, dbName, "sb_tokens", tokenID, &tok); err != nil {
		return err
	}

	tok.ResetCode = code
	return create(m, dbName, "sb_tokens", tok.ID, tok)
}

func (m *Memory) ResetPassword(dbName, email, code, password string) error {
	tok, err := m.FindUserByEmail(dbName, email)
	if err != nil {
		return err
	} else if tok.ResetCode != code {
		return fmt.Errorf("invalid code")
	}

	tok.Password = password
	return create(m, dbName, "sb_tokens", tok.ID, tok)
}

func (m *Memory) SetUserRole(dbName, accountID, email string, role int) error {
	tok, err := m.FindUserByEmail(dbName, email)
	if err == nil && tok.AccountID == accountID {
		tok.Role = role
		return create(m, dbName, "sb_tokens", tok.ID, tok)
	}

	accountUsers, err := all[model.AccountUser](m, dbName, "sb_account_users")
	if err != nil {
		return err
	}

	for _, au := range accountUsers {
		if au.Email == email && au.AccountID == accountID {
			au.Role = role
			return create(m, dbName, "sb_account_users", au.ID, au)
		}
	}

	return fmt.Errorf("user membership not found")
}

func (m *Memory) UserSetPassword(dbName, tokenID, password string) error {
	var tok model.User
	if err := getByID(m, dbName, "sb_tokens", tokenID, &tok); err != nil {
		return err
	}

	tok.Password = password
	return create(m, dbName, "sb_tokens", tok.ID, tok)
}

func (m *Memory) ChangeUserEmail(dbName, userID, accountID, oldEmail, newEmail string) error {
	var tok model.User
	if err := getByID(m, dbName, "sb_tokens", userID, &tok); err != nil {
		return err
	}

	tok.Email = newEmail
	if err := create(m, dbName, "sb_tokens", tok.ID, tok); err != nil {
		return err
	}

	accountUsers, err := all[model.AccountUser](m, dbName, "sb_account_users")
	if err != nil {
		return err
	}
	for _, au := range accountUsers {
		if au.UserID == userID {
			au.Email = newEmail
			if err := create(m, dbName, "sb_account_users", au.ID, au); err != nil {
				return err
			}
		}
	}

	var acct model.Account
	if err := getByID(m, dbName, "sb_accounts", accountID, &acct); err == nil && strings.EqualFold(acct.Email, oldEmail) {
		acct.Email = newEmail
		return create(m, dbName, "sb_accounts", acct.ID, acct)
	}

	return nil
}

func (m *Memory) UpdateUserAccount(dbName, userID, newAccountID string, role int) error {
	var tok model.User
	if err := getByID(m, dbName, "sb_tokens", userID, &tok); err != nil {
		return err
	}
	tok.AccountID = newAccountID
	tok.Role = role
	return create(m, dbName, "sb_tokens", tok.ID, tok)
}

func (m *Memory) RemoveUser(auth model.Auth, dbName, userID string) error {
	key := fmt.Sprintf("%s_sb_tokens", dbName)
	docs, ok := m.DB[key]
	if !ok {
		return errors.New("cannot find repo")
	}

	if _, ok := docs[userID]; !ok {
		return errors.New("user not found: ")
	}

	delete(docs, userID)

	mx.Lock()
	m.DB[key] = docs
	mx.Unlock()
	return nil
}
