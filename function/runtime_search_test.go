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
