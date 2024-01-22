package stubby

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Request struct {
	Pathname string                 `json:"pathname"`
	Method   string                 `json:"method"`
	Query    map[string]interface{} `json:"query"`
}

type Response struct {
	StatusCode int         `json:"statusCode"`
	Body       interface{} `json:"body"`
}

type Record struct {
	Profile  string   `json:"-"`
	Request  Request  `json:"request"`
	Response Response `json:"response"`
}

func (r *Record) Filepath() string {
	lowered := strings.ToLower(r.Request.Pathname)
	normalized := strings.TrimPrefix(strings.ReplaceAll(lowered, "/", "--"), "--")
	return filepath.Join(r.Profile, normalized+".json")
}

type File struct {
	Records []*Record `json:"stubs"`
}

func (f *File) add(r *Record) {
	if r != nil {
		f.Records = append(f.Records, r)
	}
}

func WriteToFile(filePath string, record *Record) error {
	if record == nil {
		return fmt.Errorf("record is null")
	}

	dirPath := path.Dir(filePath)

	_, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		err = os.MkdirAll(dirPath, 0o700)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
		}
	}

	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}

	var content File
	err = json.NewDecoder(file).Decode(&content)
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("failed to unmarshal file %s: %w", filePath, err)
	}
	content.add(record)

	err = file.Truncate(0)
	if err != nil {
		return fmt.Errorf("failed to rewrite file %s: %w", filePath, err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "\t")
	err = encoder.Encode(content)
	if err != nil {
		return fmt.Errorf("failed to marshal new content %w", err)
	}

	err = file.Close()
	if err != nil {
		return fmt.Errorf("failed to close file %s: %w", filePath, err)
	}

	return nil
}
