package utils

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	storage "github.com/supabase-community/storage-go"
)

// UploadToSupabase upload bất kỳ loại file nào lên bucket "uploads"
func UploadToSupabase(file interface{}, filename string, fileID string, folder string, contentType string) (string, error) {
	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseKey := os.Getenv("SUPABASE_KEY")

	storageClient := storage.NewClient(supabaseURL+"/storage/v1", supabaseKey, nil)

	var reader io.Reader
	var ext string

	// Nếu là file upload qua form-data
	if fh, ok := file.(*multipart.FileHeader); ok {
		f, err := fh.Open()
		if err != nil {
			return "", err
		}
		defer f.Close()
		reader = f
		ext = filepath.Ext(fh.Filename)
		if contentType == "" {
			contentType = fh.Header.Get("Content-Type")
		}
		// Đảm bảo con trỏ file được reset về đầu
		if _, err := f.Seek(0, 0); err != nil {
			return "", err
		}
	}

	// Nếu là []byte (ví dụ file sinh ra từ AI)
	if data, ok := file.([]byte); ok {
		reader = bytes.NewReader(data)
		ext = filepath.Ext(filename)
	}

	// Path trong bucket
	objectPath := fmt.Sprintf("%s%s", fileID, ext)
	if folder != "" {
		objectPath = fmt.Sprintf("%s/%s%s", folder, fileID, ext)
	}

	upsert := true
	options := storage.FileOptions{
		ContentType: &contentType,
		Upsert:      &upsert,
	}

	if _, err := storageClient.UploadFile("uploadfile_survey", objectPath, reader, options); err != nil {
		return "", err
	}

	publicURL := storageClient.GetPublicUrl("uploadfile_survey", objectPath)
	return publicURL.SignedURL, nil

}
