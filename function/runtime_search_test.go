package function

import (
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

func TestRuntimeCanDeleteIndexedDocument(t *testing.T) {
	cfg := config.AppConfig{}
	log := logger.Get(cfg)
	pubsub := cache.NewDevCache(log)
	datastore := memory.New(pubsub.PublishDocument)

	src, err := search.New(filepath.Join(t.TempDir(), "test.fts"), pubsub)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(src.Close)

	code := `
	function handle(body) {
		var indexed = indexDocument("contacts", "contact_1", "Alice Example alice@example.com");
		if (!indexed.ok) {
			log("ERROR indexing contact: " + indexed.content);
			return;
		}

		var removed = deleteIndexDocument("contacts", "contact_1");
		if (!removed.ok) {
			log("ERROR deleting indexed contact: " + removed.content);
			return;
		}
	}`

	fn := model.ExecData{
		FunctionName: "delete-indexed-document",
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

	if err := env.Execute(map[string]any{}); err != nil {
		t.Fatal(err)
	}

	results, err := src.Search("crm", "contacts", "alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(results.IDs) != 0 {
		t.Fatalf("expected deleted indexed document to be absent, got ids %v", results.IDs)
	}

	assertFunctionCompleted(t, datastore, fn.ID)
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
