package function

import (
	"strings"
	"testing"
	"time"

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/database/memory"
	"github.com/staticbackendhq/core/logger"
	"github.com/staticbackendhq/core/model"
)

func TestSubscriberDBTriggerRecordsHistoryAndLastRun(t *testing.T) {
	baseName := "subscriber_db_trigger"
	log := logger.Get(config.LoadConfig())
	vol := cache.NewDevCache(log)
	ds := memory.New(vol.PublishDocument)

	fnID, err := ds.AddFunction(baseName, model.ExecData{
		FunctionName: "index-contact",
		TriggerTopic: "db-contacts",
		Code: `function handle(channel, type, data) {
			if (type != "db_created") return;
			log("indexed " + data.name);
		}`,
		Version: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	sub := &Subscriber{
		PubSub: vol,
		Log:    log,
		GetExecEnv: func(msg model.Command) (*ExecutionEnvironment, error) {
			return &ExecutionEnvironment{
				Auth:      msg.Auth,
				BaseName:  msg.Base,
				DataStore: ds,
				Volatile:  vol,
				Log:       log,
			}, nil
		},
	}

	sub.handleRealtimeEvents(model.Command{
		Channel: "db-contacts",
		Type:    model.MsgTypeDBCreated,
		Data:    `{"id":"contact-1","name":"Ada"}`,
		Base:    baseName,
		Auth: model.Auth{
			AccountID: "account-1",
			UserID:    "user-1",
			Role:      100,
			Token:     "token-1",
		},
	})

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		fn, err := ds.GetFunctionByID(baseName, fnID)
		if err != nil {
			t.Fatal(err)
		}
		if len(fn.History) == 0 {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		if fn.LastRun.IsZero() {
			t.Fatal("expected db-triggered function to set last run time")
		}
		history := fn.History[len(fn.History)-1]
		if !history.Success {
			t.Fatalf("expected successful function execution history, got %+v", history)
		}
		if !strings.Contains(strings.Join(history.Output, "\n"), "indexed Ada") {
			t.Fatalf("expected function output to include index log, got %+v", history.Output)
		}
		return
	}

	t.Fatal("timed out waiting for db-triggered function execution history")
}
