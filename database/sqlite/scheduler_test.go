package sqlite

import (
	"testing"
	"time"

	"github.com/staticbackendhq/core/model"
)

func TestListTasks(t *testing.T) {
	_, err := datastore.ListTasks()
	if err != nil {
		t.Fatal(err)
	}
}

func TestTaskCRUD(t *testing.T) {
	task := model.Task{
		Name:     "sqlite-task-crud",
		Type:     model.TaskTypeFunction,
		Value:    "daily",
		Meta:     `{"data":"{}"}`,
		Interval: "0 1 * * *",
		LastRun:  time.Now().UTC(),
	}

	id, err := datastore.AddTask(confDBName, task)
	if err != nil {
		t.Fatal(err)
	}

	got, err := datastore.GetTask(confDBName, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != task.Name {
		t.Fatalf("expected task name %q got %q", task.Name, got.Name)
	}
	if got.BaseName != confDBName {
		t.Fatalf("expected task base %q got %q", confDBName, got.BaseName)
	}

	got.Name = "sqlite-task-updated"
	got.Interval = "0 2 * * *"
	if err := datastore.UpdateTask(confDBName, got); err != nil {
		t.Fatal(err)
	}

	updated, err := datastore.GetTask(confDBName, id)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != got.Name || updated.Interval != got.Interval {
		t.Fatalf("task was not updated: %+v", updated)
	}
	if updated.BaseName != confDBName {
		t.Fatalf("expected updated task base %q got %q", confDBName, updated.BaseName)
	}

	list, err := datastore.ListTasks()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, listed := range list {
		if listed.ID == id {
			found = true
			if listed.BaseName != confDBName {
				t.Fatalf("expected listed task base %q got %q", confDBName, listed.BaseName)
			}
		}
	}
	if !found {
		t.Fatalf("expected task %q in ListTasks result", id)
	}

	if err := datastore.DeleteTask(confDBName, id); err != nil {
		t.Fatal(err)
	}
}
