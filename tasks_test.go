package staticbackend

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/model"
)

func TestTasksAPICRUD(t *testing.T) {
	scheduler := backend.Scheduler
	backend.Scheduler = nil
	defer func() {
		backend.Scheduler = scheduler
	}()

	h := tasks{}
	task := model.Task{
		Name:     fmt.Sprintf("api-task-%d", time.Now().UnixNano()),
		Type:     model.TaskTypeFunction,
		Value:    "daily",
		Meta:     `{"data":"{}"}`,
		Interval: "0 1 * * *",
	}

	addResp := dbReq(t, h.listAdd, http.MethodPost, "/task", task, true)
	if addResp.StatusCode != http.StatusOK {
		t.Fatal(GetResponseBody(t, addResp))
	}

	var created model.Task
	if err := parseBody(addResp.Body, &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatal("expected created task id")
	}
	if created.BaseName != dbName {
		t.Fatalf("expected base %q got %q", dbName, created.BaseName)
	}

	listResp := dbReq(t, h.listAdd, http.MethodGet, "/task", nil, true)
	if listResp.StatusCode != http.StatusOK {
		t.Fatal(GetResponseBody(t, listResp))
	}
	var listed []model.Task
	if err := parseBody(listResp.Body, &listed); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range listed {
		if item.ID == created.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected task %q in list", created.ID)
	}

	infoResp := dbReq(t, h.item, http.MethodGet, "/task/"+created.ID, nil, true)
	if infoResp.StatusCode != http.StatusOK {
		t.Fatal(GetResponseBody(t, infoResp))
	}

	created.Name += "-updated"
	created.Interval = "0 2 * * *"
	updateResp := dbReq(t, h.item, http.MethodPost, "/task/"+created.ID, created, true)
	if updateResp.StatusCode != http.StatusOK {
		t.Fatal(GetResponseBody(t, updateResp))
	}
	var updated model.Task
	if err := parseBody(updateResp.Body, &updated); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(updated.Name, "-updated") || updated.Interval != "0 2 * * *" {
		t.Fatalf("task was not updated: %+v", updated)
	}

	delResp := dbReq(t, h.item, http.MethodDelete, "/task/"+created.ID, nil, true)
	if delResp.StatusCode != http.StatusOK {
		t.Fatal(GetResponseBody(t, delResp))
	}

	missingResp := dbReq(t, h.item, http.MethodGet, "/task/"+created.ID, nil, true)
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deleted task lookup to return 404, got %d", missingResp.StatusCode)
	}
}
