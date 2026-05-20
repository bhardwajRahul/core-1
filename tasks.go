package staticbackend

import (
	"net/http"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/middleware"
	"github.com/staticbackendhq/core/model"
)

type tasks struct{}

func (t tasks) listAdd(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		t.list(w, r)
	case http.MethodPost:
		t.add(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (t tasks) item(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		t.info(w, r)
	case http.MethodPost:
		t.update(w, r)
	case http.MethodDelete:
		t.del(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (t tasks) list(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	results, err := backend.DB.ListTasksByBase(conf.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, results)
}

func (t tasks) info(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	task, err := backend.DB.GetTask(conf.Name, getURLPart(r.URL.Path, 2))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	respond(w, http.StatusOK, task)
}

func (t tasks) add(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var task model.Task
	if err := parseBody(r.Body, &task); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	task.BaseName = conf.Name

	id, err := backend.DB.AddTask(conf.Name, task)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	task.ID = id
	if backend.Scheduler != nil {
		backend.Scheduler.AddOnTheFly(task)
	}

	respond(w, http.StatusOK, task)
}

func (t tasks) update(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var task model.Task
	if err := parseBody(r.Body, &task); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	task.ID = getURLPart(r.URL.Path, 2)
	task.BaseName = conf.Name

	if err := backend.DB.UpdateTask(conf.Name, task); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if backend.Scheduler != nil {
		_ = backend.Scheduler.CancelTask(task.ID)
		backend.Scheduler.AddOnTheFly(task)
	}

	respond(w, http.StatusOK, task)
}

func (t tasks) del(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	id := getURLPart(r.URL.Path, 2)
	if err := backend.DB.DeleteTask(conf.Name, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if backend.Scheduler != nil {
		_ = backend.Scheduler.CancelTask(id)
	}

	w.WriteHeader(http.StatusOK)
}
