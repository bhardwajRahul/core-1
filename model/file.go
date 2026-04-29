package model

import (
	"io"
	"time"
)

type UploadFileData struct {
	FileKey  string
	File     io.ReadSeeker
	Size     int64
	Mimetype string
}

type File struct {
	ID        string    `json:"id"`
	AccountID string    `json:"accountId"`
	Key       string    `json:"key"`
	URL       string    `json:"url"`
	Size      int64     `json:"size"`
	Uploaded  time.Time `json:"uploaded"`
}

type FileUsage struct {
	Bytes int64   `json:"bytes"`
	GB    float64 `json:"gb"`
}

type FileListResult struct {
	Page    int64  `json:"page"`
	Size    int64  `json:"size"`
	Total   int64  `json:"total"`
	Results []File `json:"results"`
}
