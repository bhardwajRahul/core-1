package memory

import (
	"errors"
	"fmt"
	"strings"

	"github.com/staticbackendhq/core/model"
)

func (m *Memory) AddFile(dbName string, f model.File) (id string, err error) {
	id = m.NewID()
	f.ID = id
	err = create(m, dbName, "sb_files", id, f)
	return
}

func (m *Memory) GetFileByID(dbName, fileID string) (f model.File, err error) {
	err = getByID(m, dbName, "sb_files", fileID, &f)
	return
}

func (m *Memory) DeleteFile(dbName, fileID string) error {
	key := fmt.Sprintf("%s_sb_files", dbName)

	files, ok := m.DB[key]
	if !ok {
		return errors.New("no files available for delete")
	}

	delete(files, fileID)

	mx.Lock()
	m.DB[key] = files
	mx.Unlock()
	return nil
}

func (m *Memory) ListAllFiles(dbName, accountID string) (results []model.File, err error) {
	files, err := all[model.File](m, dbName, "sb_files")
	if err != nil {
		return
	}

	results = filter(files, func(x model.File) bool {
		return x.AccountID == accountID
	})

	return
}

func (m *Memory) GetTotalFileBytes(dbName, accountID string) (int64, error) {
	files, err := all[model.File](m, dbName, "sb_files")
	if err != nil {
		return 0, err
	}

	var total int64
	for _, file := range files {
		if file.AccountID == accountID {
			total += file.Size
		}
	}

	return total, nil
}

func (m *Memory) ListFiles(dbName, accountID string, params model.ListParams) ([]model.File, int64, error) {
	files, err := all[model.File](m, dbName, "sb_files")
	if err != nil {
		return nil, 0, err
	}

	results := filter(files, func(x model.File) bool {
		return x.AccountID == accountID
	})

	sortBy := strings.ToLower(params.SortBy)
	switch sortBy {
	case "size":
		results = sortSlice(results, func(a, b model.File) bool {
			if params.SortDescending {
				return a.Size > b.Size
			}
			return a.Size < b.Size
		})
	default:
		results = sortSlice(results, func(a, b model.File) bool {
			if params.SortDescending {
				return a.Uploaded.After(b.Uploaded)
			}
			return a.Uploaded.Before(b.Uploaded)
		})
	}

	total := int64(len(results))
	start := (params.Page - 1) * params.Size
	if start >= total {
		return []model.File{}, total, nil
	}

	end := start + params.Size
	if end > total {
		end = total
	}

	return results[start:end], total, nil
}
