package memory

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

func TestDeleteAccount(t *testing.T) {
	accountID, err := datastore.CreateAccount(confDBName, "delete-account@test.com")
	if err != nil {
		t.Fatal(err)
	}
	otherAccountID, err := datastore.CreateAccount(confDBName, "delete-account-other@test.com")
	if err != nil {
		t.Fatal(err)
	}

	userID, err := datastore.CreateUser(confDBName, model.User{
		AccountID: accountID,
		Token:     "delete-account-token",
		Email:     "delete-account@test.com",
		Password:  "pw",
		Role:      50,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := datastore.CreateUser(confDBName, model.User{
		AccountID: otherAccountID,
		Token:     "delete-account-other-token",
		Email:     "delete-account-other@test.com",
		Password:  "pw",
		Role:      50,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := datastore.AddAccountUser(confDBName, model.AccountUser{
		UserID:    userID,
		AccountID: accountID,
		Email:     "delete-account@test.com",
		Token:     "delete-account-assoc-token",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := datastore.AddAccountUser(confDBName, model.AccountUser{
		UserID:    userID,
		AccountID: otherAccountID,
		Email:     "delete-account@test.com",
		Token:     "delete-account-other-assoc-token",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := datastore.AddFile(confDBName, model.File{AccountID: accountID, Key: "delete-account/file.txt"}); err != nil {
		t.Fatal(err)
	}

	auth := model.Auth{AccountID: accountID, UserID: userID, Role: 50}
	otherAuth := model.Auth{AccountID: otherAccountID, UserID: "delete-account-other-user", Role: 50}
	if _, err := datastore.CreateDocument(auth, confDBName, "delete_account_docs", map[string]interface{}{"name": "deleted"}); err != nil {
		t.Fatal(err)
	}
	if _, err := datastore.CreateDocument(otherAuth, confDBName, "delete_account_docs", map[string]interface{}{"name": "kept"}); err != nil {
		t.Fatal(err)
	}

	if err := datastore.DeleteAccount(confDBName, accountID); err != nil {
		t.Fatal(err)
	}

	if users, err := datastore.ListUsers(confDBName, accountID); err != nil {
		t.Fatal(err)
	} else if len(users) != 0 {
		t.Fatalf("expected deleted account users to be removed, got %d", len(users))
	}
	if files, err := datastore.ListAllFiles(confDBName, accountID); err != nil {
		t.Fatal(err)
	} else if len(files) != 0 {
		t.Fatalf("expected deleted account files to be removed, got %d", len(files))
	}
	if _, err := datastore.GetAccountUser(confDBName, userID, otherAccountID); err == nil {
		t.Fatal("expected deleted user account association to be removed")
	}
	if docs, err := datastore.QueryDocuments(auth, confDBName, "delete_account_docs", map[string]interface{}{}, model.ListParams{Page: 1, Size: 10}); err != nil {
		t.Fatal(err)
	} else if docs.Total != 0 {
		t.Fatalf("expected deleted account docs to be removed, got %d", docs.Total)
	}
	if users, err := datastore.ListUsers(confDBName, otherAccountID); err != nil {
		t.Fatal(err)
	} else if len(users) != 1 {
		t.Fatalf("expected other account user to remain, got %d", len(users))
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

func TestListUsersByRole(t *testing.T) {
	roleZero := model.User{
		AccountID: adminAuth.AccountID,
		Email:     "role-filter-zero@test.com",
		Password:  "role-filter-zero",
		Role:      0,
		Token:     "role-filter-zero-token",
	}
	roleFifty := model.User{
		AccountID: adminAuth.AccountID,
		Email:     "role-filter-fifty@test.com",
		Password:  "role-filter-fifty",
		Role:      50,
		Token:     "role-filter-fifty-token",
	}

	if _, err := datastore.CreateUser(confDBName, roleZero); err != nil {
		t.Fatal(err)
	}
	roleFiftyID, err := datastore.CreateUser(confDBName, roleFifty)
	if err != nil {
		t.Fatal(err)
	}

	users, err := datastore.ListUsers(confDBName, adminAuth.AccountID, 50)
	if err != nil {
		t.Fatal(err)
	}

	foundRoleFifty := false
	for _, user := range users {
		if user.Role != 50 {
			t.Fatalf("expected only role 50 users, got role %d for %s", user.Role, user.Email)
		}
		if user.ID == roleFiftyID {
			foundRoleFifty = true
		}
	}
	if !foundRoleFifty {
		t.Fatal("expected role 50 user in filtered list")
	}
}
