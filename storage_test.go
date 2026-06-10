package staticbackend

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/middleware"
	"github.com/staticbackendhq/core/model"
)

func TestFileUpload(t *testing.T) {
	pr, pw := io.Pipe()

	writer := multipart.NewWriter(pw)

	go func() {
		defer func() { _ = writer.Close() }()

		part, err := writer.CreateFormFile("file", "upload.test")
		if err != nil {
			t.Error(err)
		}

		if _, err := part.Write([]byte("testing file upload")); err != nil {
			t.Error(err)
		}
	}()

	req := httptest.NewRequest("POST", "/storage/upload", pr)
	req.Header.Add("Content-Type", writer.FormDataContentType())
	req.Header.Set("SB-PUBLIC-KEY", pubKey)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

	stdAuth := []middleware.Middleware{
		middleware.WithDB(backend.DB, backend.Cache, getStripePortalURL),
		middleware.RequireAuth(backend.DB, backend.Cache),
	}

	// prevent DB (SQLite) from being busy
	time.Sleep(35 * time.Millisecond)

	w := httptest.NewRecorder()
	h := middleware.Chain(http.HandlerFunc(upload), stdAuth...)
	h.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var data backend.SavedFile
	if err := parseBody(w.Result().Body, &data); err != nil {
		t.Error(err)
	}
	defer func() { _ = w.Result().Body.Close() }()

	t.Log(data)

	// let's remove the web-based prefix to test if file was saved
	localFilePath := strings.ReplaceAll(data.URL, "http://localhost:8099/localfs", "")

	localFilePath = path.Join(os.TempDir(), localFilePath)

	if _, err := os.Stat(localFilePath); os.IsNotExist(err) {
		t.Errorf("Expected file %s to exists", localFilePath)
	}

	// test the delete file endpoint
	delPath := fmt.Sprintf("/sudostorage/delete?id=%s", data.ID)
	resp := dbReq(t, deleteFile, "DELETE", delPath, nil)
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, go %d", resp.StatusCode)
	}

	// the file should not exists anymore
	if _, err := os.Stat(localFilePath); !os.IsNotExist(err) {
		t.Errorf("Expected file %s to not exists", localFilePath)
	}
}

func TestCleanUpFileName(t *testing.T) {
	fakeNames := make(map[string]string)
	fakeNames[""] = ""
	fakeNames["abc.def"] = "abc"
	fakeNames["ok!.test"] = "ok"
	fakeNames["@file-name_here!.ext"] = "file-name_here"

	for k, v := range fakeNames {
		if clean := internal.CleanUpFileName(k); clean != v {
			t.Errorf("expected %s got %s", v, clean)
		}
	}
}

func TestStorageUsage(t *testing.T) {
	conf, err := backend.DB.FindDatabase(pubKey)
	if err != nil {
		t.Fatal(err)
	}

	email := fmt.Sprintf("storage-usage-%d@test.com", time.Now().UnixNano())
	token, user, err := backend.Membership(conf).CreateAccountAndUser(email, password, 0)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	_, err = backend.DB.AddFile(dbName, model.File{
		AccountID: user.AccountID,
		Key:       "usage-1",
		URL:       "https://test/usage-1",
		Size:      1_500_000_000,
		Uploaded:  now,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = backend.DB.AddFile(dbName, model.File{
		AccountID: user.AccountID,
		Key:       "usage-2",
		URL:       "https://test/usage-2",
		Size:      250_000_000,
		Uploaded:  now.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}

	otherAccountID, err := backend.DB.CreateAccount(dbName, fmt.Sprintf("storage-usage-other-%d@test.com", time.Now().UnixNano()))
	if err != nil {
		t.Fatal(err)
	}

	_, err = backend.DB.AddFile(dbName, model.File{
		AccountID: otherAccountID,
		Key:       "usage-other",
		URL:       "https://test/usage-other",
		Size:      999_000_000,
		Uploaded:  now.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}

	resp := storageReq(t, storageUsage, string(token), "GET", "/storage/usage", nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatal(GetResponseBody(t, resp))
	}

	var usage model.FileUsage
	if err := parseBody(resp.Body, &usage); err != nil {
		t.Fatal(err)
	} else if usage.Bytes != 1_750_000_000 {
		t.Fatalf("expected 1750000000 bytes got %d", usage.Bytes)
	} else if usage.GB != 1.75 {
		t.Fatalf("expected 1.75 GB got %v", usage.GB)
	}
}

func TestListFiles(t *testing.T) {
	conf, err := backend.DB.FindDatabase(pubKey)
	if err != nil {
		t.Fatal(err)
	}

	email := fmt.Sprintf("storage-list-%d@test.com", time.Now().UnixNano())
	token, user, err := backend.Membership(conf).CreateAccountAndUser(email, password, 0)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	for i := 0; i < 30; i++ {
		_, err = backend.DB.AddFile(dbName, model.File{
			AccountID: user.AccountID,
			Key:       fmt.Sprintf("list-%02d", i),
			URL:       fmt.Sprintf("https://test/list-%02d", i),
			Size:      int64(i + 1),
			Uploaded:  now.Add(time.Duration(i) * time.Minute),
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	otherAccountID, err := backend.DB.CreateAccount(dbName, fmt.Sprintf("storage-list-other-%d@test.com", time.Now().UnixNano()))
	if err != nil {
		t.Fatal(err)
	}

	_, err = backend.DB.AddFile(dbName, model.File{
		AccountID: otherAccountID,
		Key:       "list-other",
		URL:       "https://test/list-other",
		Size:      5000,
		Uploaded:  now.Add(31 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}

	resp := storageReq(t, listFiles, string(token), "GET", "/storage/files", nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatal(GetResponseBody(t, resp))
	}

	var result model.FileListResult
	if err := parseBody(resp.Body, &result); err != nil {
		t.Fatal(err)
	} else if result.Page != 1 || result.Size != 25 {
		t.Fatalf("expected page 1 size 25 got page=%d size=%d", result.Page, result.Size)
	} else if result.Total != 30 {
		t.Fatalf("expected 30 files for authenticated account got %d", result.Total)
	} else if len(result.Results) != 25 {
		t.Fatalf("expected 25 results got %d", len(result.Results))
	} else if result.Results[0].AccountID != user.AccountID {
		t.Fatalf("expected only authenticated account files got account %s", result.Results[0].AccountID)
	} else if result.Results[0].Size != 30 {
		t.Fatalf("expected latest upload first with size 30 got %d", result.Results[0].Size)
	}

	resp = storageReq(t, listFiles, string(token), "GET", "/storage/files?page=2", nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatal(GetResponseBody(t, resp))
	}

	if err := parseBody(resp.Body, &result); err != nil {
		t.Fatal(err)
	} else if result.Page != 2 {
		t.Fatalf("expected page 2 got %d", result.Page)
	} else if len(result.Results) != 5 {
		t.Fatalf("expected 5 results on page 2 got %d", len(result.Results))
	} else if result.Results[0].Size != 5 {
		t.Fatalf("expected page 2 to start at size 5 got %d", result.Results[0].Size)
	}

	resp = storageReq(t, listFiles, string(token), "GET", "/storage/files?sort=size", nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatal(GetResponseBody(t, resp))
	}

	if err := parseBody(resp.Body, &result); err != nil {
		t.Fatal(err)
	} else if result.Results[0].Size != 30 {
		t.Fatalf("expected largest file first got %d", result.Results[0].Size)
	}
}

func storageReq(t *testing.T, hf func(http.ResponseWriter, *http.Request), token, method, path string, v interface{}) *http.Response {
	payload := []byte("null")
	if v != nil {
		var err error
		payload, err = json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest(method, path, strings.NewReader(string(payload)))
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
