package memory

import (
	"fmt"
	"strings"

	"github.com/staticbackendhq/core/model"
)

func (m *Memory) AddAccountUser(dbName string, au model.AccountUser) (id string, err error) {
	id = m.NewID()
	au.ID = id
	err = create(m, dbName, "sb_account_users", id, au)
	return
}

func (m *Memory) GetAccountUser(dbName, userID, accountID string) (au model.AccountUser, err error) {
	all, err := all[model.AccountUser](m, dbName, "sb_account_users")
	if err != nil {
		return
	}

	matches := filter(all, func(a model.AccountUser) bool {
		return a.UserID == userID && a.AccountID == accountID
	})

	if len(matches) == 0 {
		err = fmt.Errorf("account user association not found")
		return
	}
	au = matches[0]
	return
}

func (m *Memory) FindAccountUserByToken(dbName, token string) (au model.AccountUser, err error) {
	all, err := all[model.AccountUser](m, dbName, "sb_account_users")
	if err != nil {
		return
	}

	matches := filter(all, func(a model.AccountUser) bool {
		return strings.EqualFold(a.Token, token)
	})

	if len(matches) == 0 {
		err = fmt.Errorf("account user association not found for token")
		return
	}
	au = matches[0]
	return
}

func (m *Memory) ListAccountUsers(dbName, userID string) ([]model.AccountUser, error) {
	all, err := all[model.AccountUser](m, dbName, "sb_account_users")
	if err != nil {
		return nil, err
	}

	return filter(all, func(a model.AccountUser) bool {
		return a.UserID == userID
	}), nil
}

func (m *Memory) DeleteAccountUser(dbName, id string) error {
	key := fmt.Sprintf("%s_sb_account_users", dbName)

	docs, ok := m.DB[key]
	if !ok {
		return fmt.Errorf("cannot find repo")
	}

	if _, ok := docs[id]; !ok {
		return fmt.Errorf("account user association not found")
	}

	delete(docs, id)

	mx.Lock()
	m.DB[key] = docs
	mx.Unlock()
	return nil
}
