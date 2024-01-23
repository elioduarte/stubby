package stubby

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

type Matcher struct {
	records map[string][]*Record
	matches map[string]int
}

func (m *Matcher) Match(r *http.Request) (*Record, bool) {
	if record, ok := m.matchKey(m.exactQueryKey(r.URL.Host, r.Method, r.URL.Path, r.URL.Query().Encode())); ok {
		return record, true
	}

	if record, ok := m.matchKey(m.anyQueryKey(r.URL.Host, r.Method, r.URL.Path)); ok {
		return record, true
	}

	return nil, false
}

func (m *Matcher) matchKey(key string) (*Record, bool) {
	if records, ok := m.records[key]; ok {
		record := records[min(len(records)-1, m.matches[key])]
		m.matches[key]++
		return record, true
	}

	return nil, false
}

func (m *Matcher) exactQueryKey(host, method, pathname, query string) string {
	return fmt.Sprintf("#%s#%s#%s#%s#", host, method, pathname, query)
}

func (m *Matcher) emptyQueryKey(host, method, pathname string) string {
	return m.exactQueryKey(host, method, pathname, "")
}

func (m *Matcher) anyQueryKey(host, method, pathname string) string {
	return m.exactQueryKey(host, method, pathname, "*")
}

func (m *Matcher) addFile(f File) error {
	var errs []error

	for _, record := range f.Records {
		err := m.addRecord(record)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (m *Matcher) addRecord(r *Record) error {
	if r.Request.Query == nil {
		m.setRecord(m.anyQueryKey(r.Request.Host, r.Request.Method, r.Request.Pathname), r)
		return nil
	}

	if len(r.Request.Query) == 0 {
		m.setRecord(m.emptyQueryKey(r.Request.Host, r.Request.Method, r.Request.Pathname), r)
		return nil
	}

	rawQuery, err := mapToString(r.Request.Query)
	if err != nil {
		return err
	}
	m.setRecord(m.exactQueryKey(r.Request.Host, r.Request.Method, r.Request.Pathname, rawQuery), r)

	return nil
}

func (m *Matcher) setRecord(k string, r *Record) {
	records := m.records[k]
	m.records[k] = append(records, r)
}

func NewMatcher(dirPath string) (*Matcher, error) {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dirPath, err)
	}

	matcher := &Matcher{records: make(map[string][]*Record), matches: make(map[string]int)}
	var errs []error

	for _, fileName := range files {
		filePath := filepath.Join(dirPath, fileName.Name())
		file, err := os.Open(filePath)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to read profile %s : %w", fileName.Name(), err))
			continue
		}

		var fileJSON File
		err = json.NewDecoder(file).Decode(&fileJSON)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed unmarshal stub file %s : %w", fileName.Name(), err))
			continue
		}

		err = matcher.addFile(fileJSON)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed addRecord file to matcher %s : %w", fileName.Name(), err))
		}
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return matcher, nil
}

func mapToString(input map[string]interface{}) (string, error) {
	if input == nil {
		return "", nil
	}

	values := url.Values{}

	for key, value := range input {
		switch v := value.(type) {
		case string:
			values.Add(key, v)
		case []string:
			for _, str := range v {
				values.Add(key, str)
			}
		case int, int32, int64, float64, bool:
			values.Add(key, fmt.Sprintf("%v", v))
		default:
			return "", fmt.Errorf("unsupported value type for key %s: %T", key, v)
		}
	}

	return values.Encode(), nil
}
