package staticbackend

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/model"
)

func TestUserAddRemoveFromAccount(t *testing.T) {
	u := model.Login{Email: "newuser@test.com", Password: "newuser1234"}
	resp := dbReq(t, acct.addUser, "POST", "/account/users", u)
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	// adding user with same email should return an error
	resp2 := dbReq(t, acct.addUser, "POST", "/account/users", u)
	defer resp2.Body.Close()

	if resp2.StatusCode <= 299 {
		t.Fatal(GetResponseBody(t, resp2))
	}

	// check if users is created
	users, err := backend.DB.ListUsers(dbName, testAccountID)
	if err != nil {
		t.Fatal(err)
	}

	newUserID := ""
	for _, user := range users {
		if user.Email == "newuser@test.com" {
			newUserID = user.ID
			if user.Created.Format("2006-01-02") != time.Now().Format("2006-01-02") {
				t.Errorf("expected user to have a recent creation date, got %v", user.Created)
			}
			break
		}
	}

	if len(newUserID) == 0 {
		t.Fatal("unable to find new user")
	}

	resp3 := dbReq(t, acct.deleteUser, "DELETE", "/account/users/"+newUserID, nil)
	defer resp3.Body.Close()

	if resp3.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp3))
	}

	users, err = backend.DB.ListUsers(dbName, testAccountID)
	if err != nil {
		t.Fatal(err)
	}

	for _, user := range users {
		if user.ID == newUserID {
			t.Fatal("deleted user was found?")
			break
		}
	}
}

func TestAddNewDatabase(t *testing.T) {
	resp := dbReq(t, acct.addDatabase, "GET", "/account/add-db", nil)
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}
}

func TestListAssociations(t *testing.T) {
	resp := dbReq(t, acct.listAssociations, "GET", "/account/associations", nil)
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	// result may be nil/null when there are no associations — that is valid
	var result []model.AccountUser
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal("decoding response:", err)
	}
}

func TestGetUserAccounts(t *testing.T) {
	resp := dbReq(t, acct.getUserAccounts, "GET", "/account/user-accounts?email="+admEmail, nil, true)
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	type accountEntry struct {
		AccountID string `json:"accountId"`
		Role      int    `json:"role"`
		Home      bool   `json:"home"`
	}
	var result []accountEntry
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal("decoding response:", err)
	}
	if len(result) == 0 {
		t.Error("expected at least the home account entry")
	}

	found := false
	for _, entry := range result {
		if entry.Home {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected one entry with home=true")
	}
}

func TestGetUserAccountsMissingEmail(t *testing.T) {
	resp := dbReq(t, acct.getUserAccounts, "GET", "/account/user-accounts", nil, true)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 got %d", resp.StatusCode)
	}
}

func TestCrossAccountUserAssociation(t *testing.T) {
	// Create a user that lives in a *different* account so that addUser
	// triggers the cross-account association path instead of the
	// "email already in use in this account" rejection.
	const crossEmail = "cross-account-user@test.com"

	otherAcctID, err := backend.DB.CreateAccount(dbName, "cross-owner@test.com")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := backend.DB.CreateUser(dbName, model.User{
		AccountID: otherAcctID,
		Email:     crossEmail,
		Password:  "doesnotmatter",
		Token:     backend.DB.NewID(),
		Role:      0,
		Created:   time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	// addUser as the admin (testAccountID) with an email from a different account
	// should create a cross-account association, not a new user record
	resp := dbReq(t, acct.addUser, "POST", "/account/users", model.Login{Email: crossEmail})
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	var assoc model.AccountUser
	if err := json.NewDecoder(resp.Body).Decode(&assoc); err != nil {
		t.Fatal("decoding response:", err)
	}
	if len(assoc.Token) == 0 {
		t.Error("expected a non-empty association token")
	}
	if assoc.Email != crossEmail {
		t.Errorf("expected email %s got %s", crossEmail, assoc.Email)
	}

	// clean up
	if err := backend.DB.DeleteAccountUser(dbName, assoc.ID); err != nil {
		t.Fatal("cleanup failed:", err)
	}
}
