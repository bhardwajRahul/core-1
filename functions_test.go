package staticbackend

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/model"
)

func TestFunctionsExecuteDBOperations(t *testing.T) {
	code := `
	function handle(body) {
		log(body);

		sendMail({
			from: "me@backend.com",
			to: "user1@domain.com",
			subject: "Begin test",
			htmlBody: "<h1>Hello</h1>...",
			textBody: "Hello\n\n...",
		  });

		var o = {
			from: body.from,
			desc: "yep", 
			done: false, 
			subobj: {
				yep: "working", 
				status: true
			}
		};
		var result = create("jsexec", o);
		if (!result.ok) {
			log("ERROR: creating doc");
			log(result.content);
			return;
		}
		var getRes = getById("jsexec", result.content.id)
		if (!getRes.ok) {
			log("ERROR: getting doc by id");
			log("id:");
			log(getRes.content.id);
			log("end id");
			return;
		} else if (getRes.content.from != "val from unit test") {
			log("ERROR: asserting data from request body");
			log(getRes.content);
			return;			
		}

		var updata = getRes.content;
		updata.done = true;
		var upres = update("jsexec", updata.id, updata);
		if (!upres.ok) {
			log("ERROR: updating doc");
			log(upres.content);
			return;
		}

		var qres = query("jsexec", [["done", "==", true]]);
		if (!qres.ok) {
			log("ERROR: querying documents");
			log(qres.content);
			return;
		}

		if (qres.content.results.length != 1) {
			log("ERROR");
			log("expected results to have 1 doc, got: " + qres.content.results.length);
			log(qres);
			return;
		}

		if (upres.content.id != qres.content.results[0].id) {
			log("ERROR");
			log("expected updated doc's id to equal the query result");
			log("updated id: " + upres.content.id);
			log("query doc id: " + qres.content.results[0].id);
			return;
		}

		var getRes = fetch("https://echo.free.beeceptor.com/sb=fn");
		if (!getRes.ok) {
			log("ERROR: sending GET request");
			log(getRes.content);
			return;
		}

		var postRes = fetch("https://echo.free.beeceptor.com/sb=fn", {
			method: "POST",
			headers: {
				"Content-Type" : "application/json"
			}, 
			body: {
				"test": "test msg"
			}
		});
		if (!postRes.ok) {
			log("ERROR: sending POST request");
			log(postRes.content);
			return;
		}

		var putRes = fetch("https://echo.free.beeceptor.com/sb=fn", {
			method: "PUT",
			headers: {
				"Content-Type" : "application/json"
			}, 
			body: {
				"test": "test msg"
			}
		});
		if (!putRes.ok) {
			log("ERROR: sending PUT request");
			log(putRes.content);
			return;
		}
		var patchRes = fetch("https://echo.free.beeceptor.com/sb=fn", {
			method: "PATCH",
			headers: {
				"Content-Type" : "application/json"
			}, 
			body: {
				"test": "test msg"
			}
		});
		if (!patchRes.ok) {
			log("ERROR: sending PATCH request");
			log(patchRes.content);
			return;
		}
		var delRes = fetch("https://echo.free.beeceptor.com/sb=fn", {
			method: "DELETE",
			headers: {
				"Content-Type" : "application/json"
			}, 
			body: {
				"test": "test msg"
			}
		});
		if (!delRes.ok) {
			log("ERROR: sending DELETE request");
			log(delRes.content);
			return;
		}

		sendMail({
			from: "me@backend.com",
			to: "user1@domain.com",
			subject: "End test",
			htmlBody: "<h1>Bye</h1>...",
			textBody: "Bye\n\n...",
		  });

	}`
	data := model.ExecData{
		FunctionName: "unittest",
		Code:         code,
		TriggerTopic: "web",
	}
	addResp := dbReq(t, funexec.add, "POST", "/", data, true)
	if addResp.StatusCode != http.StatusOK {
		b, err := io.ReadAll(addResp.Body)
		if err != nil {
			t.Fatal(err)
		}
		defer addResp.Body.Close()

		t.Log(string(b))
		t.Errorf("add: expected status 200 got %s", addResp.Status)
	}

	val := url.Values{}
	val.Add("from", "val from unit test")

	execResp := dbReq(t, funexec.exec, "POST", "/fn/exec/unittest", val, false, true)
	if execResp.StatusCode != http.StatusOK {
		b, err := io.ReadAll(execResp.Body)
		if err != nil {
			t.Fatal(err)
		}
		defer execResp.Body.Close()

		t.Log(string(b))
		t.Errorf("expected status 200 got %s", execResp.Status)
	}

	infoResp := dbReq(t, funexec.info, "GET", "/fn/info/unittest", nil, true)
	defer infoResp.Body.Close()

	if infoResp.StatusCode >= 299 {
		b, err := io.ReadAll(infoResp.Body)
		if err != nil {
			t.Fatal(err)
		}

		t.Fatalf("expected 200 status got %d - %s", infoResp.StatusCode, string(b))
	}

	var checkFn model.ExecData
	if err := parseBody(infoResp.Body, &checkFn); err != nil {
		t.Fatal(err)
	}
	defer infoResp.Body.Close()

	var errorLines []string
	foundError := false
	for _, h := range checkFn.History {
		for _, line := range h.Output {
			if strings.Contains(line, "ERROR") {
				errorLines = h.Output
				foundError = true
				break
			}
		}

		if foundError {
			break
		}
	}

	if foundError {
		t.Errorf("found error in function exec log: %v", errorLines)
	}

	time.Sleep(500 * time.Millisecond)
}

func TestFunctionSudoExecUsesRootAuth(t *testing.T) {
	code := `
	function handle() {
		var listed = list("contacts", { Page: 1, Size: 25 });
		if (!listed.ok) {
			log("ERROR: " + listed.content);
			return;
		}
		log("contacts total: " + listed.content.total);
		log("contacts listed: " + listed.content.results.length);
	}
	`

	fn := model.ExecData{
		FunctionName: "fn-test-sudoexec-root-auth",
		Code:         code,
		TriggerTopic: "web",
	}
	addResp := dbReq(t, funexec.add, "POST", "/", fn, true)
	defer addResp.Body.Close()
	if addResp.StatusCode != http.StatusOK {
		t.Fatal(GetResponseBody(t, addResp))
	}

	accountUserID, err := backend.DB.CreateUser(dbName, model.User{
		AccountID: testAccountID,
		Email:     "sudoexec-account-user@test.com",
		Token:     backend.DB.NewID(),
		Role:      0,
		Created:   time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	otherAccountID, err := backend.DB.CreateAccount(dbName, "sudoexec-other-account@test.com")
	if err != nil {
		t.Fatal(err)
	}
	otherUserID, err := backend.DB.CreateUser(dbName, model.User{
		AccountID: otherAccountID,
		Email:     "sudoexec-other-user@test.com",
		Token:     backend.DB.NewID(),
		Role:      0,
		Created:   time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	contacts := []model.Auth{
		{
			AccountID: testAccountID,
			UserID:    accountUserID,
			Role:      0,
		},
		{
			AccountID: otherAccountID,
			UserID:    otherUserID,
			Role:      0,
		},
	}
	for i, auth := range contacts {
		if _, err := backend.DB.CreateDocument(auth, dbName, "contacts", map[string]interface{}{
			"firstName": "SudoExec",
			"lastName":  i,
		}); err != nil {
			t.Fatal(err)
		}
	}

	execResp := dbReq(t, funexec.exec, "POST", "/fn/sudoexec/fn-test-sudoexec-root-auth", map[string]interface{}{}, true)
	defer execResp.Body.Close()
	if execResp.StatusCode != http.StatusOK {
		t.Fatal(GetResponseBody(t, execResp))
	}

	var checkFn model.ExecData
	for i := 0; i < 50; i++ {
		infoResp := dbReq(t, funexec.info, "GET", "/fn/info/fn-test-sudoexec-root-auth", nil, true)
		if infoResp.StatusCode != http.StatusOK {
			t.Fatal(GetResponseBody(t, infoResp))
		}
		if err := parseBody(infoResp.Body, &checkFn); err != nil {
			t.Fatal(err)
		}
		infoResp.Body.Close()

		if len(checkFn.History) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(checkFn.History) == 0 {
		t.Fatal("function run history was not recorded")
	}

	output := strings.Join(checkFn.History[len(checkFn.History)-1].Output, "\n")
	if !strings.Contains(output, "contacts listed: 2") {
		t.Fatalf("expected root function to list contacts from both accounts, got output:\n%s", output)
	}
}

func TestFunctionTriggerByDBChanges(t *testing.T) {
	code := `
	function handle(channel, type, data) {
		// we only want db_created in this function
		if (type != "db_created") return;

		data.FromFn = "yes this works";
		log("ch: " + channel);
		log("id: " + data.id);
		const res = update("coltrigger", data.id, data);;
		if (!res.ok) {
			log("ERROR: " + res.content);
			return;
		}
		log("document updated via function triggered by db_created msg");
	}
	`

	data := model.ExecData{
		FunctionName: "fn-test-trigger",
		Code:         code,
		TriggerTopic: "db-coltrigger",
	}
	addResp := dbReq(t, funexec.add, "POST", "/", data, true)
	defer addResp.Body.Close()
	if addResp.StatusCode != http.StatusOK {
		t.Fatal(GetResponseBody(t, addResp))
	}

	// this should trigger the function
	v := new(struct {
		ID     string `json:"id"`
		Name   string
		FromFn string
	})
	v.Name = "test"

	dbResp := dbReq(t, db.add, "POST", "/db/coltrigger", v)
	defer dbResp.Body.Close()
	if dbResp.StatusCode >= 299 {
		t.Fatal(GetResponseBody(t, dbResp))
	} else if err := parseBody(dbResp.Body, &v); err != nil {
		t.Fatal(err)
	}

	// give sometimes for the event to propagate
	time.Sleep(650 * time.Millisecond)

	infoResp := dbReq(t, funexec.info, "GET", "/fn/info/fn-test-trigger", nil, true)
	defer infoResp.Body.Close()

	if infoResp.StatusCode >= 299 {
		t.Fatal(GetResponseBody(t, infoResp))
	}

	var checkFn model.ExecData
	if err := parseBody(infoResp.Body, &checkFn); err != nil {
		t.Fatal(err)
	}

	if len(checkFn.History) == 0 {
		t.Fatal("expected db-triggered function history to be recorded")
	}
	if checkFn.LastRun.IsZero() {
		t.Fatal("expected db-triggered function last run time to be set")
	}

	var errorLines []string
	foundError := false
	for _, h := range checkFn.History {
		for _, line := range h.Output {
			if strings.Contains(line, "ERROR") {
				errorLines = h.Output
				foundError = true
				break
			}
		}

		if foundError {
			break
		}
	}

	if foundError {
		t.Errorf("found error in function exec log: %v", errorLines)
	}

	chkResp := dbReq(t, db.get, "GET", "/db/coltrigger/"+v.ID, nil)
	defer chkResp.Body.Close()

	if chkResp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, chkResp))
	} else if err := parseBody(chkResp.Body, &v); err != nil {
		t.Fatal(err)
	} else if v.FromFn != "yes this works" {
		t.Errorf("expected FromFn to be 'yes this works' got %s", v.FromFn)
	}

	time.Sleep(500 * time.Millisecond)
}

func TestFunctionTriggerBySystemAccountCreated(t *testing.T) {
	code := `
	function handle(channel, type, data) {
		if (channel != "sys-sb_accounts" || type != "db_created") return;

		const res = create("system_onboarding", {
			sourceAccountId: data.id,
			sourceEmail: data.email,
			sourceUserId: data.userId,
			sourceUserRole: data.userRole
		});
		if (!res.ok) {
			log("ERROR: " + res.content);
			return;
		}
		log("created onboarding row for " + data.email);
	}
	`

	data := model.ExecData{
		FunctionName: "fn-test-system-account-created",
		Code:         code,
		TriggerTopic: "sys-sb_accounts",
	}
	addResp := dbReq(t, funexec.add, "POST", "/", data, true)
	defer addResp.Body.Close()
	if addResp.StatusCode != http.StatusOK {
		t.Fatal(GetResponseBody(t, addResp))
	}

	usrSvc := backend.Membership(model.DatabaseConfig{
		ID:   pubKey,
		Name: dbName,
	})
	_, newUser, err := usrSvc.CreateAccountAndUser("system-trigger@test.com", "system-trigger-pass", 50)
	if err != nil {
		t.Fatal(err)
	}

	newUserAuth := model.Auth{
		AccountID: newUser.AccountID,
		UserID:    newUser.ID,
		Email:     newUser.Email,
		Role:      newUser.Role,
		Token:     newUser.Token,
	}
	var docs []map[string]interface{}
	for i := 0; i < 10; i++ {
		result, err := backend.DB.ListDocuments(newUserAuth, dbName, "system_onboarding", model.ListParams{Page: 1, Size: 25})
		if err != nil {
			t.Fatal(err)
		}
		for _, doc := range result.Results {
			if doc["sourceAccountId"] == newUser.AccountID {
				docs = append(docs, doc)
			}
		}
		if len(docs) > 0 {
			break
		}
		time.Sleep(150 * time.Millisecond)
	}
	if len(docs) == 0 {
		t.Fatal("expected system account trigger to create an onboarding document")
	}
	if docs[0]["accountId"] != newUser.AccountID {
		t.Errorf("expected onboarding document to be created with new user account %s got %v", newUser.AccountID, docs[0]["accountId"])
	}

	docID, ok := docs[0]["id"].(string)
	if !ok {
		t.Fatalf("expected onboarding document id to be a string, got %T", docs[0]["id"])
	}
	docs[0]["updatedByNewUser"] = true
	if _, err := backend.DB.UpdateDocument(newUserAuth, dbName, "system_onboarding", docID, docs[0]); err != nil {
		t.Fatalf("expected new user to update onboarding document: %v", err)
	}

	infoResp := dbReq(t, funexec.info, "GET", "/fn/info/fn-test-system-account-created", nil, true)
	defer infoResp.Body.Close()
	if infoResp.StatusCode >= 299 {
		t.Fatal(GetResponseBody(t, infoResp))
	}

	var checkFn model.ExecData
	if err := parseBody(infoResp.Body, &checkFn); err != nil {
		t.Fatal(err)
	}
	for _, h := range checkFn.History {
		for _, line := range h.Output {
			if strings.Contains(line, "ERROR") {
				t.Fatalf("found error in function exec log: %v", h.Output)
			}
		}
	}
}

func TestFunctionTriggerByPublishingMsg(t *testing.T) {
	code := `
	function handle(channel, type, data) {
		// we only want db_created in this function
		if (type != "do-something-custom") return;

		data.FromFn = "yes this works";
		log("ch: " + channel);
		log("id: " + data.id);
		const res = update("coltriggerpub", data.id, data);;
		if (!res.ok) {
			log("ERROR: " + res.content);
			return;
		}
		log("document updated via function triggered by db_created msg");
	}
	`

	data := model.ExecData{
		FunctionName: "fn-pubmsg-trigger",
		Code:         code,
		TriggerTopic: "custom-channel",
	}
	addResp := dbReq(t, funexec.add, "POST", "/", data, true)
	defer addResp.Body.Close()
	if addResp.StatusCode != http.StatusOK {
		t.Fatal(GetResponseBody(t, addResp))
	}

	v := new(struct {
		ID     string `json:"id"`
		Name   string
		FromFn string
	})
	v.Name = "test"

	dbResp := dbReq(t, db.add, "POST", "/db/coltriggerpub", v)
	defer dbResp.Body.Close()
	if dbResp.StatusCode >= 299 {
		t.Fatal(GetResponseBody(t, dbResp))
	} else if err := parseBody(dbResp.Body, &v); err != nil {
		t.Fatal(err)
	}

	pubData := new(struct {
		Channel string `json:"channel"`
		Type    string `json:"type"`
		Data    string `json:"data"`
	})
	pubData.Channel = "custom-channel"
	pubData.Type = "do-something-custom"

	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}

	pubData.Data = string(b)

	pubResp := dbReq(t, publishMessage, "POST", "/publish", pubData)
	defer pubResp.Body.Close()
	if pubResp.StatusCode >= 299 {
		t.Fatal(GetResponseBody(t, pubResp))
	}

	// give sometimes for the event to propagate
	time.Sleep(650 * time.Millisecond)

	infoResp := dbReq(t, funexec.info, "GET", "/fn/info/fn-pubmsg-trigger", nil, true)
	defer infoResp.Body.Close()

	if infoResp.StatusCode >= 299 {
		t.Fatal(GetResponseBody(t, infoResp))
	}

	var checkFn model.ExecData
	if err := parseBody(infoResp.Body, &checkFn); err != nil {
		t.Fatal(err)
	}

	t.Log(checkFn.History)

	var errorLines []string
	foundError := false
	for _, h := range checkFn.History {
		for _, line := range h.Output {
			if strings.Contains(line, "ERROR") {
				errorLines = h.Output
				foundError = true
				break
			}
		}

		if foundError {
			break
		}
	}

	if foundError {
		t.Errorf("found error in function exec log: %v", errorLines)
	}

	chkResp := dbReq(t, db.get, "GET", "/db/coltriggerpub/"+v.ID, nil)
	defer chkResp.Body.Close()

	if chkResp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, chkResp))
	} else if err := parseBody(chkResp.Body, &v); err != nil {
		t.Fatal(err)
	} else if v.FromFn != "yes this works" {
		t.Errorf("expected FromFn to be 'yes this works' got %s", v.FromFn)
	}

	time.Sleep(500 * time.Millisecond)
}

func TestFunctionWithVolatilizerHelpers(t *testing.T) {
	// Remove the counter
	toRemove, err := backend.Cache.Inc("some-counter", 1)
	if err != nil {
		t.Fatal(err)
	} else if toRemove > 0 {
		if _, err := backend.Cache.Dec("some-counter", toRemove); err != nil {
			t.Fatal(err)
		}
	}

	code := `
	function handle(channel, type, body) {
		// set some value in the cache
		let res = cacheSet("ok-unit-test", "init value");
		if (!res.ok) {
			log(res.content);
			return;
		}

		res = cacheGet("ok-unit-test");
		if (!res.ok) {
			log(res.content);
			return;
		} else if (res.content != "init value") {
			log("error, cache value isn't :init value:");
			return;
		}

		// cacheSet()

		res = inc("some-counter", 10);
		if (!res.ok) {
			log(res.content);
			return;
		}

		res = dec("some-counter", 2);
		if (!res.ok) {
			log(res.content);
			return;
		}

		res = publish("test-channel", "some-type", {a: "which data"});
		if (!res.ok) {
			log(res.content);
			return;
		}
	}
	`

	data := model.ExecData{
		FunctionName: "fn-cache-tests",
		Code:         code,
		TriggerTopic: "trigger-from-unit-test",
	}
	addResp := dbReq(t, funexec.add, "POST", "/", data, true)
	defer addResp.Body.Close()
	if addResp.StatusCode != http.StatusOK {
		t.Fatal(GetResponseBody(t, addResp))
	}

	// let's send this message to trigger the function
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}

	msg := model.Command{
		SID:     "unit-test",
		Type:    "manual-msg",
		Data:    string(b),
		Channel: "trigger-from-unit-test",
		Token:   adminToken,
		Auth:    model.Auth{}, // don't need auth for this test
		Base:    dbName,
	}

	if err := backend.Cache.Publish(msg); err != nil {
		t.Fatal(err)
	}

	// give sometimes for the event to propagate
	time.Sleep(650 * time.Millisecond)

	infoResp := dbReq(t, funexec.info, "GET", "/fn/info/fn-cache-tests", nil, true)
	defer infoResp.Body.Close()

	if infoResp.StatusCode >= 299 {
		t.Fatal(GetResponseBody(t, infoResp))
	}

	var checkFn model.ExecData
	if err := parseBody(infoResp.Body, &checkFn); err != nil {
		t.Fatal(err)
	}

	t.Log(checkFn.History)

	var errorLines []string
	foundError := false
	for _, h := range checkFn.History {
		for _, line := range h.Output {
			if strings.Contains(line, "ERROR") {
				errorLines = h.Output
				foundError = true
				break
			}
		}

		if foundError {
			break
		}
	}

	if foundError {
		t.Errorf("found error in function exec log: %v", errorLines)
	}

	var total int64
	if err := backend.Cache.GetTyped("some-counter", &total); err != nil {
		t.Fatal(err)
	} else if total != 8 {
		t.Errorf("expected total to be 8 got %d", total)
	}
}
