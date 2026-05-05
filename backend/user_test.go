package backend_test

import (
	"testing"

	"github.com/staticbackendhq/core/backend"
)

func TestAuthenticateCreatesMissingAccountAssociation(t *testing.T) {
	const (
		email    = "login-association@test.com"
		password = "test1234!"
	)

	usr := backend.Membership(base)
	_, tok, err := usr.CreateAccountAndUser(email, password, 50)
	if err != nil {
		t.Fatal(err)
	}

	otherAccountID, err := backend.DB.CreateAccount(base.Name, "login-association-account@test.com")
	if err != nil {
		t.Fatal(err)
	}

	exists, err := backend.DB.AssociationExists(base.Name, tok.ID, otherAccountID)
	if err != nil {
		t.Fatal(err)
	} else if exists {
		t.Fatal("association should not exist before login")
	}

	jwtToken, err := usr.Authenticate(email, password, otherAccountID)
	if err != nil {
		t.Fatal(err)
	} else if jwtToken == "" {
		t.Fatal("expected a session token")
	}

	assoc, err := backend.DB.GetAccountUser(base.Name, tok.ID, otherAccountID)
	if err != nil {
		t.Fatal(err)
	}
	if assoc.Role != 0 {
		t.Errorf("expected role 0 got %d", assoc.Role)
	}
	if assoc.Email != email {
		t.Errorf("expected email %s got %s", email, assoc.Email)
	}
	if assoc.Token == "" {
		t.Error("expected a non-empty association token")
	}
}
