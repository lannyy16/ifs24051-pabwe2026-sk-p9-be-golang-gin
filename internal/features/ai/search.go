package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxSources = 5

// SearchClient melakukan penelusuran fakta terkini ke Search Engine Endpoint.
type SearchClient struct {
	Endpoint string
	Key      string
	HTTP     *http.Client
}

// NewSearchClient membuat SearchClient dengan timeout 20 detik.
func NewSearchClient(endpoint, key string) *SearchClient {
	return &SearchClient{Endpoint: endpoint, Key: key, HTTP: &http.Client{Timeout: 20 * time.Second}}
}

type searchItem struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Content string `json:"content"`
}

// Bentuk balasan yang didukung: [...], {"data":[...]}, {"results":[...]}, atau {"data":{"results":[...]}}.
type searchResponse struct {
	Results []searchItem    `json:"results"`
	Data    json.RawMessage `json:"data"`
}

// extractItems mengurai balasan Search Engine ke daftar item, apa pun bentuknya.
func extractItems(body json.RawMessage) []searchItem {
	var list []searchItem
	if json.Unmarshal(body, &list) == nil && len(list) > 0 {
		return list
	}
	var wrapped searchResponse
	if json.Unmarshal(body, &wrapped) != nil {
		return nil
	}
	if len(wrapped.Results) > 0 {
		return wrapped.Results
	}
	var nested searchResponse
	if json.Unmarshal(wrapped.Data, &list) == nil && len(list) > 0 {
		return list
	}
	if json.Unmarshal(wrapped.Data, &nested) == nil {
		return nested.Results
	}
	return nil
}

// Search menjalankan GET {endpoint}?q=<query>&limit=5 dan memetakan hasilnya ke Source.
func (s *SearchClient) Search(ctx context.Context, query string) ([]Source, error) {
	sep := "?"
	if strings.Contains(s.Endpoint, "?") {
		sep = "&"
	}
	target := s.Endpoint + sep + url.Values{"q": {query}, "limit": {"5"}}.Encode()

	var raw json.RawMessage
	if err := doJSON(ctx, s.HTTP, http.MethodGet, target, s.Key, nil, &raw); err != nil {
		return nil, err
	}
	items := extractItems(raw)
	sources := make([]Source, 0, maxSources)
	for _, it := range items {
		if len(sources) == maxSources {
			break
		}
		snippet := it.Snippet
		if snippet == "" {
			snippet = it.Content
		}
		sources = append(sources, Source{Title: it.Title, URL: it.URL, Snippet: snippet})
	}
	return sources, nil
}
