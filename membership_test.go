package staticbackend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/middleware"
	"github.com/staticbackendhq/core/model"
)

func TestGetCurrentAuthUser(t *testing.T) {
	resp := dbReq(t, mship.me, "GET", "/me", nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	var me model.Auth
	if err := parseBody(resp.Body, &me); err != nil {
		t.Fatal(err)
	} else if !strings.EqualFold(me.Email, admEmail) {
		t.Errorf("expected email to be %s got %s", admEmail, me.Email)
	} else if me.Role != 100 {
		t.Errorf("expected role to be 100 got %d", me.Role)
	}
}

func TestSetRoleAllowsRole50ToGrantRole50(t *testing.T) {
	conf, err := backend.DB.FindDatabase(pubKey)
	if err != nil {
		t.Fatal(err)
	}

	adminToken, admin, err := backend.Membership(conf).CreateUser(testAccountID, "set-role-admin-50@test.com", userPassword, 50)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = backend.DB.RemoveUser(model.Auth{AccountID: testAccountID, UserID: admin.ID, Role: 100}, dbName, admin.ID)
	})

	_, user, err := backend.Membership(conf).CreateUser(testAccountID, "set-role-target-50@test.com", userPassword, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = backend.DB.RemoveUser(model.Auth{AccountID: testAccountID, UserID: user.ID, Role: 100}, dbName, user.ID)
	})

	resp := authReqWithToken(t, string(adminToken), mship.setRole, "POST", "/setrole", map[string]interface{}{
		"accountId": testAccountID,
		"email":     user.Email,
		"role":      50,
	})
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	updated, err := backend.DB.GetUserByID(dbName, testAccountID, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Role != 50 {
		t.Fatalf("expected role 50 got %d", updated.Role)
	}
}

func TestChangeEmail(t *testing.T) {
	conf, err := backend.DB.FindDatabase(pubKey)
	if err != nil {
		t.Fatal(err)
	}

	oldEmail := "change-http-old@test.com"
	newEmail := "change-http-new@test.com"
	token, user, err := backend.Membership(conf).CreateUser(testAccountID, oldEmail, userPassword, 0)
	if err != nil {
		t.Fatal(err)
	}

	resp := authReqWithToken(t, string(token), mship.changeEmail, "POST", "/me/email", map[string]string{"email": newEmail})
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	updated, err := backend.DB.FindUserByEmail(dbName, newEmail)
	if err != nil {
		t.Fatal(err)
	} else if updated.Email != newEmail {
		t.Fatalf("expected persisted email %s got %s", newEmail, updated.Email)
	}

	var cached model.Auth
	if err := backend.Cache.GetTyped(fmt.Sprintf("%s|%s", user.ID, user.Token), &cached); err == nil {
		t.Fatalf("expected auth cache to be cleared, found cached email %s", cached.Email)
	}

	resp = authReqWithToken(t, string(token), mship.me, "GET", "/me", nil)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	var me model.Auth
	if err := parseBody(resp.Body, &me); err != nil {
		t.Fatal(err)
	} else if me.Email != newEmail {
		t.Fatalf("expected cached auth to refresh email to %s got %s", newEmail, me.Email)
	}
}

func TestChangeEmailConflict(t *testing.T) {
	conf, err := backend.DB.FindDatabase(pubKey)
	if err != nil {
		t.Fatal(err)
	}

	token, _, err := backend.Membership(conf).CreateUser(testAccountID, "change-conflict-user@test.com", userPassword, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := backend.Membership(conf).CreateUser(testAccountID, "change-conflict-existing@test.com", userPassword, 0); err != nil {
		t.Fatal(err)
	}

	resp := authReqWithToken(t, string(token), mship.changeEmail, "POST", "/me/email", map[string]string{"email": "change-conflict-existing@test.com"})
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 got %d: %s", resp.StatusCode, GetResponseBody(t, resp))
	}
}

func TestChangeEmailInvalidEmail(t *testing.T) {
	resp := dbReq(t, mship.changeEmail, "POST", "/me/email", map[string]string{"email": "invalid"})
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", resp.StatusCode)
	}
}

func TestDeleteAccount(t *testing.T) {
	conf, err := backend.DB.FindDatabase(pubKey)
	if err != nil {
		t.Fatal(err)
	}

	token, user, err := backend.Membership(conf).CreateAccountAndUser("delete-account-http@test.com", userPassword, 50)
	if err != nil {
		t.Fatal(err)
	}

	resp := authReqWithToken(t, string(token), mship.deleteAccount, "DELETE", "/account", nil)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	if _, err := backend.DB.GetUserByID(conf.Name, user.AccountID, user.ID); err == nil {
		t.Fatal("expected deleted account user to be removed")
	}
}

func TestDeleteAccountRequiresAdmin(t *testing.T) {
	resp := authReqWithToken(t, userToken, mship.deleteAccount, "DELETE", "/account", nil)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", resp.StatusCode)
	}
}

func TestDeleteAccountRequiresDeleteMethod(t *testing.T) {
	resp := dbReq(t, mship.deleteAccount, "POST", "/account", nil)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 got %d", resp.StatusCode)
	}
}

func TestMagicLink(t *testing.T) {
	// note, even though the magic link route allow public (un-unthenticated) req
	// I'm using dbReq (which enforce the stdauth) for ease of re-using the
	// function.

	data := new(struct {
		FromEmail string `json:"fromEmail"`
		FromName  string `json:"fromName"`
		Email     string `json:"email"`
		Subject   string `json:"subject"`
		Body      string `json:"body"`
		MagicLink string `json:"link"`
	})

	data.FromEmail = "unit@test.com"
	data.FromName = "unit test"
	data.Email = admEmail
	data.Subject = "Magic link from unit test"
	data.Body = "<p>Hello</p><p>Please click on the following link to sign-in</p><p>[ink]</p>"
	data.MagicLink = "https://mycustom.link/with-code"

	resp := dbReq(t, mship.magicLink, "POST", "/login/magic", data)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	// in dev mode, the code is always 666333
	u := fmt.Sprintf("/login/magic?email=%s&code=666333", admEmail)
	resp2 := dbReq(t, mship.magicLink, "GET", u, nil)
	defer func() { _ = resp2.Body.Close() }()

	if resp2.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp2))
	}
}

func TestGetAuthTokenByUserID(t *testing.T) {
	resp := dbReq(t, mship.getAuthTokenByUserID, "GET", "/sudogetauthtokenbyuserid/"+testAccountID+"/"+adminAuthUserID(t), nil, true)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	var token string
	if err := parseBody(resp.Body, &token); err != nil {
		t.Fatal(err)
	}
	if len(token) == 0 {
		t.Fatal("expected token to be returned")
	}
}

func TestGetAuthTokenByUserIDForAccountAssociation(t *testing.T) {
	conf, err := backend.DB.FindDatabase(pubKey)
	if err != nil {
		t.Fatal(err)
	}

	homeAccountID, err := backend.DB.CreateAccount(dbName, "sudo-associated-home@test.com")
	if err != nil {
		t.Fatal(err)
	}

	userID, err := backend.DB.CreateUser(dbName, model.User{
		AccountID: homeAccountID,
		Email:     "sudo-associated-user@test.com",
		Password:  "unused",
		Token:     backend.DB.NewID(),
		Role:      0,
		Created:   time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	assoc := model.AccountUser{
		UserID:    userID,
		AccountID: testAccountID,
		Email:     "sudo-associated-user@test.com",
		Role:      25,
		Token:     backend.DB.NewID(),
		Created:   time.Now(),
	}
	if _, err := backend.DB.AddAccountUser(dbName, assoc); err != nil {
		t.Fatal(err)
	}

	resp := dbReq(t, mship.getAuthTokenByUserID, "GET", "/sudogetauthtokenbyuserid/"+testAccountID+"/"+userID, nil, true)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	var token string
	if err := parseBody(resp.Body, &token); err != nil {
		t.Fatal(err)
	}
	if len(token) == 0 {
		t.Fatal("expected token to be returned")
	}

	resp = authReqWithToken(t, token, mship.me, "GET", "/me", nil)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	var auth model.Auth
	if err := parseBody(resp.Body, &auth); err != nil {
		t.Fatal(err)
	}
	if auth.UserID != userID {
		t.Fatalf("expected user id %s got %s", userID, auth.UserID)
	}
	if auth.AccountID != testAccountID {
		t.Fatalf("expected associated account id %s got %s", testAccountID, auth.AccountID)
	}
	if auth.Role != assoc.Role {
		t.Fatalf("expected role %d got %d", assoc.Role, auth.Role)
	}

	mship := backend.Membership(conf)
	homeUser, err := mship.GetUserByID(homeAccountID, userID)
	if err != nil {
		t.Fatal(err)
	}
	if homeUser.AccountID != homeAccountID {
		t.Fatalf("expected home account id %s got %s", homeAccountID, homeUser.AccountID)
	}
}

func TestGetAuthTokenByUserIDMissingParams(t *testing.T) {
	resp := dbReq(t, mship.getAuthTokenByUserID, "GET", "/sudogetauthtokenbyuserid/"+testAccountID, nil, true)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", resp.StatusCode)
	}
}

func TestGetUserByID(t *testing.T) {
	userID := adminAuthUserID(t)
	resp := dbReq(t, mship.getUserByID, "GET", "/sudogetuserbyid/"+testAccountID+"/"+userID, nil, true)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	var user model.User
	if err := parseBody(resp.Body, &user); err != nil {
		t.Fatal(err)
	}
	if user.ID != userID {
		t.Fatalf("expected user id %s got %s", userID, user.ID)
	}
	if !strings.EqualFold(user.Email, admEmail) {
		t.Fatalf("expected email %s got %s", admEmail, user.Email)
	}
}

func TestGetUserByIDMissingParams(t *testing.T) {
	resp := dbReq(t, mship.getUserByID, "GET", "/sudogetuserbyid/"+testAccountID, nil, true)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", resp.StatusCode)
	}
}

func adminAuthUserID(t *testing.T) string {
	t.Helper()

	user, err := backend.DB.FindUserByEmail(dbName, admEmail)
	if err != nil {
		t.Fatal(err)
	}

	return user.ID
}

func authReqWithToken(t *testing.T, token string, hf func(http.ResponseWriter, *http.Request), method, path string, v interface{}) *http.Response {
	t.Helper()

	var payload []byte
	if v != nil {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal("error marshaling post data:", err)
		}
		payload = b
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	w := httptest.NewRecorder()
	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("SB-PUBLIC-KEY", pubKey)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	stdAuth := []middleware.Middleware{
		middleware.WithDB(backend.DB, backend.Cache, getStripePortalURL),
		middleware.RequireAuth(backend.DB, backend.Cache),
	}
	h := middleware.Chain(http.HandlerFunc(hf), stdAuth...)
	h.ServeHTTP(w, req)

	return w.Result()
}
