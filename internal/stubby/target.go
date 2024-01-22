package stubby

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type Target struct {
	URL    url.URL `json:"url"`
	Prefix string  `json:"prefix,omitempty"`
}

func (t *Target) Matches(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, t.Prefix)
}

func (t *Target) Rewrite(r *http.Request) {
	r.Host = t.URL.Host
	r.RequestURI = strings.TrimPrefix(r.RequestURI, t.Prefix)
	r.URL.Path = strings.TrimPrefix(r.URL.Path, t.Prefix)
	r.URL.Host = t.URL.Host
	r.URL.Scheme = t.URL.Scheme
}

func (t *Target) UnmarshalJSON(data []byte) error {
	type Alias Target
	aux := &struct {
		URL string `json:"url"`
		*Alias
	}{
		Alias: (*Alias)(t),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	parsedURL, err := url.Parse(aux.URL)
	if err != nil {
		return err
	}

	t.URL = url.URL{
		Scheme: parsedURL.Scheme,
		Host:   parsedURL.Host,
	}

	return nil
}
