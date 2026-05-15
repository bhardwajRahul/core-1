package function

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/database/memory"
	"github.com/staticbackendhq/core/logger"
	"github.com/staticbackendhq/core/model"
	"github.com/staticbackendhq/core/search"
)

func TestRuntimeDatabaseHelpers(t *testing.T) {
	code := `
	function handle(body) {
		var created = create("contacts", {
			firstName: "Alice",
			lastName: "Example",
			company: "StaticBackend",
			email: "alice@example.com"
		});
		if (!created.ok) {
			log("ERROR creating contact: " + created.content);
			return;
		}

		var listed = list("contacts", { Page: 1, Size: 25 });
		if (!listed.ok) {
			log("ERROR listing contacts: " + listed.content);
			return;
		}
		if (listed.content.results.length !== 1) {
			log("ERROR expected one listed contact, got " + listed.content.results.length);
			return;
		}

		var found = query("contacts", [["email", "==", "alice@example.com"]], { Page: 1, Size: 25 });
		if (!found.ok) {
			log("ERROR querying contact: " + found.content);
			return;
		}
		if (found.content.results.length !== 1) {
			log("ERROR expected one queried contact, got " + found.content.results.length);
			return;
		}

		var updated = update("contacts", created.content.id, { company: "Acme" });
		if (!updated.ok) {
			log("ERROR updating contact: " + updated.content);
			return;
		}
		if (updated.content.company !== "Acme") {
			log("ERROR expected updated company to be Acme, got " + updated.content.company);
			return;
		}

		var deleted = del("contacts", created.content.id);
		if (!deleted.ok) {
			log("ERROR deleting contact: " + deleted.content);
			return;
		}
	}`

	ctx := newRuntimeTestContext(t, "crm-db", code)
	if err := ctx.env.Execute(map[string]any{}); err != nil {
		t.Fatal(err)
	}

	assertFunctionCompleted(t, ctx.datastore, ctx.fn.ID)
}

func TestRuntimeBulkDatabaseHelpers(t *testing.T) {
	code := `
	function fail(message) {
		throw new Error(message);
	}

	function expectOK(result, name) {
		if (!result.ok) {
			fail(name + " failed: " + result.content);
		}
		return result.content;
	}

	function handle(body) {
		var generated = expectOK(newId(), "newId");
		if (typeof generated !== "string" || generated.length === 0) {
			fail("newId returned an empty id");
		}

		expectOK(createBulk("runtime_bulk_contacts", [
			{ email: "ada@example.com", company: "StaticBackend", score: 1, segment: "lead" },
			{ email: "grace@example.com", company: "StaticBackend", score: 2, segment: "lead" },
			{ email: "linus@example.com", company: "Kernel", score: 3, segment: "customer" },
			{ email: "dennis@example.com", company: "Bell", score: 4, segment: "archive" }
		]), "createBulk");

		var listed = expectOK(query("runtime_bulk_contacts", []), "query all");
		if (listed.results.length !== 4) {
			fail("expected 4 created docs, got " + listed.results.length);
		}

		var ids = [listed.results[0].id, listed.results[1].id];
		var byIds = expectOK(getByIds("runtime_bulk_contacts", ids), "getByIds");
		if (byIds.length !== 2) {
			fail("expected 2 docs by ids, got " + byIds.length);
		}

		var staticBackendCount = expectOK(count("runtime_bulk_contacts", [["company", "=", "StaticBackend"]]), "count");
		if (staticBackendCount !== 2) {
			fail("expected count 2, got " + staticBackendCount);
		}

		var updatedMany = expectOK(updateMany("runtime_bulk_contacts", [["segment", "=", "lead"]], { segment: "qualified" }), "updateMany");
		if (updatedMany !== 2) {
			fail("expected updateMany count 2, got " + updatedMany);
		}

		var updatedBulk = expectOK(updateBulk("runtime_bulk_contacts", [["company", "=", "Kernel"]], { segment: "qualified" }), "updateBulk");
		if (updatedBulk !== 1) {
			fail("expected updateBulk count 1, got " + updatedBulk);
		}

		var qualified = expectOK(query("runtime_bulk_contacts", [["segment", "=", "qualified"]], { Page: 1, Size: 25 }), "query qualified");
		if (qualified.results.length !== 3) {
			fail("expected 3 qualified docs, got " + qualified.results.length);
		}

		expectOK(incrementValue("runtime_bulk_contacts", qualified.results[0].id, "score", 5), "incrementValue");
		var incremented = expectOK(getById("runtime_bulk_contacts", qualified.results[0].id), "getById incremented");
		if (incremented.score !== qualified.results[0].score + 5) {
			fail("expected incremented score, got " + incremented.score);
		}

		var deletedMany = expectOK(deleteMany("runtime_bulk_contacts", [["company", "=", "Bell"]]), "deleteMany");
		if (deletedMany !== 1) {
			fail("expected deleteMany count 1, got " + deletedMany);
		}

		var deletedBulk = expectOK(deleteBulk("runtime_bulk_contacts", [["segment", "=", "qualified"]]), "deleteBulk");
		if (deletedBulk !== 3) {
			fail("expected deleteBulk count 3, got " + deletedBulk);
		}
	}`

	ctx := newRuntimeTestContext(t, "crm-bulk-db", code)
	if err := ctx.env.Execute(map[string]any{}); err != nil {
		t.Fatal(err)
	}

	result, err := ctx.datastore.ListDocuments(ctx.env.Auth, ctx.env.BaseName, "runtime_bulk_contacts", model.ListParams{
		Page: 1,
		Size: 25,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 0 {
		t.Fatalf("expected all runtime_bulk_contacts docs to be deleted, got %d", result.Total)
	}

	assertFunctionCompleted(t, ctx.datastore, ctx.fn.ID)
}

func TestRuntimeSearchHelpers(t *testing.T) {
	code := `
	function handle(body) {
		var indexed = indexDocument("contacts", "contact_1", "Alice Example alice@example.com");
		if (!indexed.ok) {
			log("ERROR indexing contact: " + indexed.content);
			return;
		}

		var found = search("contacts", "alice");
		if (!found.ok) {
			log("ERROR searching contact: " + found.content);
			return;
		}
		if (found.content.length !== 1) {
			log("ERROR expected one searched contact, got " + found.content.length);
			return;
		}

		var removed = deleteIndexDocument("contacts", "contact_1");
		if (!removed.ok) {
			log("ERROR deleting indexed contact: " + removed.content);
			return;
		}
	}`

	ctx := newRuntimeTestContext(t, "crm-search", code)
	if _, err := ctx.datastore.CreateDocument(ctx.env.Auth, "crm-search", "contacts", map[string]interface{}{
		"id":        "contact_1",
		"firstName": "Alice",
		"lastName":  "Example",
		"email":     "alice@example.com",
	}); err != nil {
		t.Fatal(err)
	}

	if err := ctx.env.Execute(map[string]any{}); err != nil {
		t.Fatal(err)
	}

	results, err := ctx.search.Search("crm-search", "contacts", "alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(results.IDs) != 0 {
		t.Fatalf("expected deleted indexed document to be absent, got ids %v", results.IDs)
	}

	assertFunctionCompleted(t, ctx.datastore, ctx.fn.ID)
}

type runtimeTestContext struct {
	datastore *memory.Memory
	env       *ExecutionEnvironment
	fn        model.ExecData
	search    *search.Search
}

func newRuntimeTestContext(t *testing.T, dbName, code string) runtimeTestContext {
	t.Helper()

	cfg := config.AppConfig{}
	log := logger.Get(cfg)
	pubsub := cache.NewDevCache(log)
	datastore := memory.New(pubsub.PublishDocument).(*memory.Memory)

	src, err := search.New(filepath.Join(t.TempDir(), "test.fts"), pubsub)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(src.Close)

	_, err = datastore.CreateDatabase(model.DatabaseConfig{ID: dbName, Name: dbName})
	if err != nil {
		t.Fatal(err)
	}

	fn := model.ExecData{
		FunctionName: fmt.Sprintf("%s-runtime-test", dbName),
		Code:         code,
		TriggerTopic: "web",
	}
	fn.ID, err = datastore.AddFunction("crm", fn)
	if err != nil {
		t.Fatal(err)
	}

	env := &ExecutionEnvironment{
		Auth: model.Auth{
			Role: 100,
		},
		BaseName:  "crm",
		DataStore: datastore,
		Volatile:  pubsub,
		Search:    src,
		Data:      fn,
		Log:       log,
	}

	return runtimeTestContext{
		datastore: datastore,
		env:       env,
		fn:        fn,
		search:    src,
	}
}

func assertFunctionCompleted(t *testing.T, datastore any, fnID string) {
	t.Helper()

	ds, ok := datastore.(interface {
		GetFunctionByID(dbName, id string) (model.ExecData, error)
	})
	if !ok {
		t.Fatal("datastore does not expose GetFunctionByID")
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		fn, err := ds.GetFunctionByID("crm", fnID)
		if err != nil {
			t.Fatal(err)
		}
		if len(fn.History) > 0 {
			if !fn.History[len(fn.History)-1].Success {
				t.Fatalf("function did not complete successfully: %#v", fn.History[len(fn.History)-1])
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("timed out waiting for function execution history")
}
