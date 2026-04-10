package postgresql

import (
	"testing"
	"time"

	"github.com/staticbackendhq/core/model"
)

func TestAddAccountUser(t *testing.T) {
	acctID, err := datastore.CreateAccount(confDBName, "au-add@test.com")
	if err != nil {
		t.Fatal(err)
	}

	au := model.AccountUser{
		UserID:    adminToken.ID,
		AccountID: acctID,
		Email:     adminEmail,
		Role:      0,
		Token:     "assoc-token-1",
		Created:   time.Now(),
	}

	id, err := datastore.AddAccountUser(confDBName, au)
	if err != nil {
		t.Fatal(err)
	} else if len(id) == 0 {
		t.Error("expected a non-empty id")
	}
}

func TestGetAccountUser(t *testing.T) {
	acctID, err := datastore.CreateAccount(confDBName, "au-get@test.com")
	if err != nil {
		t.Fatal(err)
	}

	au := model.AccountUser{
		UserID:    adminToken.ID,
		AccountID: acctID,
		Email:     adminEmail,
		Role:      0,
		Token:     "assoc-token-get",
		Created:   time.Now(),
	}

	id, err := datastore.AddAccountUser(confDBName, au)
	if err != nil {
		t.Fatal(err)
	}

	found, err := datastore.GetAccountUser(confDBName, adminToken.ID, acctID)
	if err != nil {
		t.Fatal(err)
	} else if found.ID != id {
		t.Errorf("expected id %s got %s", id, found.ID)
	}
}

func TestFindAccountUserByToken(t *testing.T) {
	acctID, err := datastore.CreateAccount(confDBName, "au-findbytoken@test.com")
	if err != nil {
		t.Fatal(err)
	}

	const tok = "assoc-token-findbytoken"
	au := model.AccountUser{
		UserID:    adminToken.ID,
		AccountID: acctID,
		Email:     adminEmail,
		Role:      0,
		Token:     tok,
		Created:   time.Now(),
	}

	id, err := datastore.AddAccountUser(confDBName, au)
	if err != nil {
		t.Fatal(err)
	}

	found, err := datastore.FindAccountUserByToken(confDBName, tok)
	if err != nil {
		t.Fatal(err)
	} else if found.ID != id {
		t.Errorf("expected id %s got %s", id, found.ID)
	}
}

func TestListAccountUsers(t *testing.T) {
	// Create a second user to use as the list subject (avoids UNIQUE constraint
	// on the (user_id, account_id) pair with adminToken across tests).
	listUserID, err := datastore.CreateUser(confDBName, model.User{
		AccountID: adminAccount.ID,
		Email:     "au-listuser@test.com",
		Password:  "pw",
		Token:     "au-listuser-token",
		Role:      0,
		Created:   time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	acct1, err := datastore.CreateAccount(confDBName, "au-list1@test.com")
	if err != nil {
		t.Fatal(err)
	}
	acct2, err := datastore.CreateAccount(confDBName, "au-list2@test.com")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := datastore.AddAccountUser(confDBName, model.AccountUser{
		UserID: listUserID, AccountID: acct1, Email: "au-listuser@test.com", Token: "list-tok-1", Created: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := datastore.AddAccountUser(confDBName, model.AccountUser{
		UserID: listUserID, AccountID: acct2, Email: "au-listuser@test.com", Token: "list-tok-2", Created: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	results, err := datastore.ListAccountUsers(confDBName, listUserID)
	if err != nil {
		t.Fatal(err)
	} else if len(results) < 2 {
		t.Errorf("expected at least 2 associations, got %d", len(results))
	}
}

func TestDeleteAccountUser(t *testing.T) {
	acctID, err := datastore.CreateAccount(confDBName, "au-del@test.com")
	if err != nil {
		t.Fatal(err)
	}

	au := model.AccountUser{
		UserID:    adminToken.ID,
		AccountID: acctID,
		Email:     adminEmail,
		Role:      0,
		Token:     "assoc-token-del",
		Created:   time.Now(),
	}

	id, err := datastore.AddAccountUser(confDBName, au)
	if err != nil {
		t.Fatal(err)
	}

	if err := datastore.DeleteAccountUser(confDBName, id); err != nil {
		t.Fatal(err)
	}

	if _, err := datastore.GetAccountUser(confDBName, adminToken.ID, acctID); err == nil {
		t.Error("expected error after deletion, got nil")
	}
}

func TestUpdateUserAccount(t *testing.T) {
	newAcctID, err := datastore.CreateAccount(confDBName, "au-promote@test.com")
	if err != nil {
		t.Fatal(err)
	}

	if err := datastore.UpdateUserAccount(confDBName, adminToken.ID, newAcctID, 50); err != nil {
		t.Fatal(err)
	}

	tok, err := datastore.FindUser(confDBName, adminToken.ID, adminToken.Token)
	if err != nil {
		t.Fatal(err)
	} else if tok.AccountID != newAcctID {
		t.Errorf("expected accountID %s got %s", newAcctID, tok.AccountID)
	}

	// restore original account so other tests are not affected
	if err := datastore.UpdateUserAccount(confDBName, adminToken.ID, adminToken.AccountID, adminToken.Role); err != nil {
		t.Fatal(err)
	}
}
