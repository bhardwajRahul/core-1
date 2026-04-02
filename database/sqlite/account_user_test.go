package sqlite

import (
	"testing"
	"time"

	"github.com/staticbackendhq/core/model"
)

func TestAddAccountUser(t *testing.T) {
	au := model.AccountUser{
		UserID:    adminToken.ID,
		AccountID: "other-account-id",
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
	au := model.AccountUser{
		UserID:    adminToken.ID,
		AccountID: "get-account-id",
		Email:     adminEmail,
		Role:      0,
		Token:     "assoc-token-get",
		Created:   time.Now(),
	}

	id, err := datastore.AddAccountUser(confDBName, au)
	if err != nil {
		t.Fatal(err)
	}

	found, err := datastore.GetAccountUser(confDBName, adminToken.ID, "get-account-id")
	if err != nil {
		t.Fatal(err)
	} else if found.ID != id {
		t.Errorf("expected id %s got %s", id, found.ID)
	}
}

func TestFindAccountUserByToken(t *testing.T) {
	const tok = "assoc-token-findbytoken"
	au := model.AccountUser{
		UserID:    adminToken.ID,
		AccountID: "find-by-token-account",
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
	const listUserID = "list-acct-user-id"
	au1 := model.AccountUser{
		UserID:    listUserID,
		AccountID: "list-account-1",
		Email:     "list@test.com",
		Role:      0,
		Token:     "list-tok-1",
		Created:   time.Now(),
	}
	au2 := model.AccountUser{
		UserID:    listUserID,
		AccountID: "list-account-2",
		Email:     "list@test.com",
		Role:      0,
		Token:     "list-tok-2",
		Created:   time.Now(),
	}

	if _, err := datastore.AddAccountUser(confDBName, au1); err != nil {
		t.Fatal(err)
	}
	if _, err := datastore.AddAccountUser(confDBName, au2); err != nil {
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
	au := model.AccountUser{
		UserID:    adminToken.ID,
		AccountID: "delete-account-id",
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

	if _, err := datastore.GetAccountUser(confDBName, adminToken.ID, "delete-account-id"); err == nil {
		t.Error("expected error after deletion, got nil")
	}
}

func TestUpdateUserAccount(t *testing.T) {
	newAcctID, err := datastore.CreateAccount(confDBName, "promote@test.com")
	if err != nil {
		t.Fatal(err)
	}

	if err := datastore.UpdateUserAccount(confDBName, adminToken.ID, newAcctID); err != nil {
		t.Fatal(err)
	}

	tok, err := datastore.FindUser(confDBName, adminToken.ID, adminToken.Token)
	if err != nil {
		t.Fatal(err)
	} else if tok.AccountID != newAcctID {
		t.Errorf("expected accountID %s got %s", newAcctID, tok.AccountID)
	}

	// restore original account so other tests are not affected
	if err := datastore.UpdateUserAccount(confDBName, adminToken.ID, adminToken.AccountID); err != nil {
		t.Fatal(err)
	}
}
