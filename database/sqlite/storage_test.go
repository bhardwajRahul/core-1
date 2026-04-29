package sqlite

import (
	"fmt"
	"testing"
	"time"

	"github.com/staticbackendhq/core/model"
)

func TestFileStorage(t *testing.T) {
	f := model.File{
		AccountID: adminAccount.ID,
		Key:       "key",
		URL:       "https://test",
		Size:      123456,
		Uploaded:  time.Now(),
	}

	f1 := model.File{
		AccountID: adminAccount.ID,
		Key:       "key1",
		URL:       "https://test1",
		Size:      123456,
		Uploaded:  time.Now(),
	}

	id, err := datastore.AddFile(confDBName, f)
	if err != nil {
		t.Fatal(err)
	}

	_, err = datastore.AddFile(confDBName, f1)
	if err != nil {
		t.Fatal(err)
	}

	list, err := datastore.ListAllFiles(confDBName, f.AccountID)
	if err != nil {
		t.Fatal(err)
	} else if len(list) > 2 || len(list) < 2 {
		t.Errorf("expected list length to be 2 got %d", len(list))
	}

	f2, err := datastore.GetFileByID(confDBName, id)
	if err != nil {
		t.Fatal(err)
	} else if f2.Key != f.Key {
		t.Errorf("expected key to be %s got %s", f.Key, f2.Key)
	}

	if err := datastore.DeleteFile(confDBName, id); err != nil {
		t.Fatal(err)
	}

	check, err := datastore.GetFileByID(confDBName, id)
	if err == nil {
		t.Errorf("error should not be nil")
	} else if check.ID == id {
		t.Errorf("deleted file id returned? %v", check)
	}
}

func TestFileUsageAndListFiles(t *testing.T) {
	accountID, err := datastore.CreateAccount(confDBName, fmt.Sprintf("fileusage-%d@test.com", time.Now().UnixNano()))
	if err != nil {
		t.Fatal(err)
	}

	otherAccountID, err := datastore.CreateAccount(confDBName, fmt.Sprintf("fileusage-other-%d@test.com", time.Now().UnixNano()))
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	var expectedTotal int64

	for i := 0; i < 30; i++ {
		size := int64(i + 1)
		expectedTotal += size

		_, err = datastore.AddFile(confDBName, model.File{
			AccountID: accountID,
			Key:       fmt.Sprintf("acct-file-%02d", i),
			URL:       fmt.Sprintf("https://test/%02d", i),
			Size:      size,
			Uploaded:  now.Add(time.Duration(i) * time.Minute),
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	_, err = datastore.AddFile(confDBName, model.File{
		AccountID: otherAccountID,
		Key:       "other-account-file",
		URL:       "https://test/other",
		Size:      9999,
		Uploaded:  now.Add(31 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}

	total, err := datastore.GetTotalFileBytes(confDBName, accountID)
	if err != nil {
		t.Fatal(err)
	} else if total != expectedTotal {
		t.Fatalf("expected total bytes %d got %d", expectedTotal, total)
	}

	files, count, err := datastore.ListFiles(confDBName, accountID, model.ListParams{
		Page:           1,
		Size:           25,
		SortDescending: true,
	})
	if err != nil {
		t.Fatal(err)
	} else if count != 30 {
		t.Fatalf("expected count 30 got %d", count)
	} else if len(files) != 25 {
		t.Fatalf("expected 25 files got %d", len(files))
	} else if files[0].Size != 30 {
		t.Fatalf("expected latest file first with size 30 got %d", files[0].Size)
	}

	files, count, err = datastore.ListFiles(confDBName, accountID, model.ListParams{
		Page:           2,
		Size:           25,
		SortDescending: true,
	})
	if err != nil {
		t.Fatal(err)
	} else if count != 30 {
		t.Fatalf("expected count 30 got %d", count)
	} else if len(files) != 5 {
		t.Fatalf("expected 5 files on page 2 got %d", len(files))
	} else if files[0].Size != 5 {
		t.Fatalf("expected page 2 to start at size 5 got %d", files[0].Size)
	}

	files, count, err = datastore.ListFiles(confDBName, accountID, model.ListParams{
		Page:           1,
		Size:           25,
		SortBy:         "size",
		SortDescending: true,
	})
	if err != nil {
		t.Fatal(err)
	} else if count != 30 {
		t.Fatalf("expected count 30 got %d", count)
	} else if files[0].Size != 30 {
		t.Fatalf("expected largest file first got %d", files[0].Size)
	} else if files[len(files)-1].Size != 6 {
		t.Fatalf("expected last item on first page to have size 6 got %d", files[len(files)-1].Size)
	}
}
