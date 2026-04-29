package staticbackend

import (
	"net/http"
	"strings"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/middleware"
	"github.com/staticbackendhq/core/model"
)

func upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, h, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// check for file size
	// TODO: This should be based on current plan
	maxFileSizeMB := int64(350)
	if h.Size/(1024*1024) > maxFileSizeMB {
		http.Error(w, "file size exeeded your limit", http.StatusBadRequest)
		return
	}

	name := r.Form.Get("name")

	fileSvc := backend.Storage(auth, conf)
	savedFile, err := fileSvc.Save(h.Filename, name, file, h.Size)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, savedFile)
}

func deleteFile(w http.ResponseWriter, r *http.Request) {
	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fileID := r.URL.Query().Get("id")

	fileSvc := backend.Storage(auth, conf)
	if err := fileSvc.Delete(fileID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}

func storageUsage(w http.ResponseWriter, r *http.Request) {
	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fileSvc := backend.Storage(auth, conf)
	usage, err := fileSvc.Usage()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, usage)
}

func listFiles(w http.ResponseWriter, r *http.Request) {
	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	page, _ := getPagination(r.URL)

	params := model.ListParams{
		Page:           page,
		Size:           25,
		SortBy:         "uploaded",
		SortDescending: true,
	}

	if strings.EqualFold(r.URL.Query().Get("sort"), "size") {
		params.SortBy = "size"
	}

	fileSvc := backend.Storage(auth, conf)
	result, err := fileSvc.ListFiles(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, result)
}
