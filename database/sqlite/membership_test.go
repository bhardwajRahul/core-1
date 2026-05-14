package sqlite

import (
	"testing"
	"time"

	"github.com/staticbackendhq/core/model"
)

func TestCreateUserAccountAndToken(t *testing.T) {
	acctID, err := datastore.CreateAccount(confDBName, "unit@test.com")
	if err != nil {
		t.Fatal(err)
	}

	tok := model.User{
		AccountID: acctID,
		Token:     "123",
		Email:     "unit@test.com",
		Password:  "4321",
		Role:      0,
		Created:   time.Now(),
	}

	tokID, err := datastore.CreateUser(confDBName, tok)
	if err != nil {
		t.Fatal(err)
	} else if len(tokID) < 10 {
		t.Errorf("expected id to be len > 10 got %s", tokID)
	}
}

func TestGetFirstTokenFromAccountID(t *testing.T) {
	tok, err := datastore.GetFirstUserFromAccountID(confDBName, adminToken.AccountID)
	if err != nil {
		t.Fatal(err)
	} else if tok.ID != adminToken.ID {
		t.Errorf("wrong token, expected %s got %s", adminToken.ID, tok.ID)
	}
}

func TestSetPasswordResetCode(t *testing.T) {
	expected := "from_unit_test"

	if err := datastore.SetPasswordResetCode(confDBName, adminToken.ID, expected); err != nil {
		t.Fatal(err)
	}

	tok, err := datastore.FindUser(confDBName, adminToken.ID, adminToken.Token)
	if err != nil {
		t.Fatal(err)
	} else if tok.ResetCode != expected {
		t.Errorf("expected reset code to be %s got %s", expected, tok.ResetCode)
	}

	newpw := "changed_from_test"
	if err := datastore.ResetPassword(confDBName, adminEmail, expected, newpw); err != nil {
		t.Fatal(err)
	}

	tok2, err := datastore.FindUser(confDBName, adminToken.ID, adminToken.Token)
	if err != nil {
		t.Fatal(err)
	} else if tok2.Password != newpw {
		t.Errorf("expected password to be %s got %s", newpw, tok2.Password)
	}
}

func TestSetUserRole(t *testing.T) {
	newTok := model.User{
		AccountID: adminAccount.ID,
		Token:     "normal-user-token",
		Email:     "normal@test.com",
		Password:  "normal",
		Role:      1,
		ResetCode: "none",
		Created:   time.Now(),
	}

	newID, err := datastore.CreateUser(confDBName, newTok)
	if err != nil {
		t.Fatal(err)
	}

	if err := datastore.SetUserRole(confDBName, newTok.AccountID, newTok.Email, 90); err != nil {
		t.Fatal(err)
	}

	tok, err := datastore.FindUser(confDBName, newID, newTok.Token)
	if err != nil {
		t.Fatal(err)
	} else if tok.Role != 90 {
		t.Errorf("expected role to be 90 got %d", tok.Role)
	}
}

func TestUserSetPassword(t *testing.T) {
	expected := "pw_changed"
	if err := datastore.UserSetPassword(confDBName, adminToken.ID, expected); err != nil {
		t.Fatal(err)
	}

	tok, err := datastore.FindUser(confDBName, adminToken.ID, adminToken.Token)
	if err != nil {
		t.Fatal(err)
	} else if tok.Password != expected {
		t.Errorf("expected password to be %s got %s", expected, tok.Password)
	}
}

func TestChangeUserEmail(t *testing.T) {
	oldEmail := "change-email-old@test.com"
	newEmail := "change-email-new@test.com"

	acctID, err := datastore.CreateAccount(confDBName, oldEmail)
	if err != nil {
		t.Fatal(err)
	}
	assocAcctID, err := datastore.CreateAccount(confDBName, "change-email-assoc@test.com")
	if err != nil {
		t.Fatal(err)
	}

	userID, err := datastore.CreateUser(confDBName, model.User{
		AccountID: acctID,
		Token:     "change-email-token",
		Email:     oldEmail,
		Password:  "pw",
		Role:      50,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := datastore.AddAccountUser(confDBName, model.AccountUser{
		UserID:    userID,
		AccountID: assocAcctID,
		Email:     oldEmail,
		Role:      10,
		Token:     "change-email-assoc-token",
	}); err != nil {
		t.Fatal(err)
	}

	if err := datastore.ChangeUserEmail(confDBName, userID, acctID, oldEmail, newEmail); err != nil {
		t.Fatal(err)
	}

	tok, err := datastore.FindUser(confDBName, userID, "change-email-token")
	if err != nil {
		t.Fatal(err)
	} else if tok.Email != newEmail {
		t.Fatalf("expected user email %s got %s", newEmail, tok.Email)
	}

	assoc, err := datastore.GetAccountUser(confDBName, userID, assocAcctID)
	if err != nil {
		t.Fatal(err)
	} else if assoc.Email != newEmail {
		t.Fatalf("expected association email %s got %s", newEmail, assoc.Email)
	}

	accounts, err := datastore.ListAccounts(confDBName)
	if err != nil {
		t.Fatal(err)
	}
	for _, account := range accounts {
		if account.ID == acctID && account.Email != newEmail {
			t.Fatalf("expected account email %s got %s", newEmail, account.Email)
		}
	}
}

func TestUserAddRemoveFromAccount(t *testing.T) {
	u := model.User{
		AccountID: adminAuth.AccountID,
		Email:     "user2@test.com",
		Password:  "1234user2",
		Role:      0,
		Token:     "user2-token",
	}

	newUserID, err := datastore.CreateUser(confDBName, u)
	if err != nil {
		t.Fatal(err)
	}

	users, err := datastore.ListUsers(confDBName, adminAuth.AccountID)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, user := range users {
		if user.ID == newUserID {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected new user id to be in account user")
	}

	if err := datastore.RemoveUser(adminAuth, confDBName, newUserID); err != nil {
		t.Fatal(err)
	}

	users, err = datastore.ListUsers(confDBName, adminAuth.AccountID)
	if err != nil {
		t.Fatal(err)
	}

	found = false
	for _, user := range users {
		if user.ID == newUserID {
			found = true
			break
		}
	}

	if found {
		t.Error("new user is still in account users?")
	}
}
