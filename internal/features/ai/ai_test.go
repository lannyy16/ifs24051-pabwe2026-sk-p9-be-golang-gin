package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errBoom = errors.New("boom")

type fakeSearch struct {
	res []Source
	err error
}

func (f fakeSearch) Search(context.Context, string) ([]Source, error) { return f.res, f.err }

type fakeChat struct {
	out  string
	err  error
	last []Message
}

func (f *fakeChat) Complete(_ context.Context, m []Message) (string, error) {
	f.last = m
	return f.out, f.err
}

const goodJSON = "Berikut hasilnya:\n```json\n{\"title\":\" Judul AI \",\"category\":\"Teknologi\",\"excerpt\":\" ringkas \",\"content\":\"# Isi\"}\n```"

func TestGenerateWithSearch(t *testing.T) {
	chat := &fakeChat{out: goodJSON}
	svc := NewService(fakeSearch{res: []Source{{Title: "T", URL: "http://u", Snippet: "s"}}}, chat)

	res, err := svc.Generate(context.Background(), GenerateRequest{Topic: "Golang", Category: "Bisnis", IncludeWebSearch: true})
	require.NoError(t, err)
	assert.Equal(t, "Judul AI", res.Title)
	assert.Equal(t, "Bisnis", res.Category) // kategori permintaan menang
	assert.Equal(t, "ringkas", res.Excerpt)
	assert.True(t, res.UsedWebSearch)
	assert.Len(t, res.Sources, 1)
	assert.Contains(t, chat.last[1].Content, "[1] T — s (http://u)")
	assert.Contains(t, chat.last[1].Content, "Kategori yang diinginkan: Bisnis")
}

func TestGenerateWithoutSearchOrWithSearchFailure(t *testing.T) {
	chat := &fakeChat{out: goodJSON}

	res, err := NewService(fakeSearch{err: errBoom}, chat).Generate(context.Background(), GenerateRequest{Topic: "Golang", IncludeWebSearch: true})
	require.NoError(t, err)
	assert.False(t, res.UsedWebSearch)
	assert.Equal(t, "Teknologi", res.Category) // kategori dari AI
	assert.NotNil(t, res.Sources)
	assert.NotContains(t, chat.last[1].Content, "Kategori yang diinginkan")

	res, err = NewService(fakeSearch{res: []Source{{Title: "x"}}}, chat).Generate(context.Background(), GenerateRequest{Topic: "Golang"})
	require.NoError(t, err)
	assert.False(t, res.UsedWebSearch)
}

func TestGenerateErrors(t *testing.T) {
	_, err := NewService(nil, &fakeChat{err: errBoom}).Generate(context.Background(), GenerateRequest{Topic: "abc"})
	assert.ErrorIs(t, err, ErrUpstream)

	for _, out := range []string{"bukan json", `{"title":"","content":"x"}`, `{"title":"x","content":" "}`} {
		_, err = NewService(nil, &fakeChat{out: out}).Generate(context.Background(), GenerateRequest{Topic: "abc"})
		assert.ErrorIs(t, err, ErrInvalidAIResponse, out)
	}
}

func TestExtractJSON(t *testing.T) {
	assert.Equal(t, `{"a":1}`, extractJSON("x {\"a\":1} y"))
	assert.Equal(t, "tanpa kurung", extractJSON("tanpa kurung"))
}

func TestChatClient(t *testing.T) {
	var gotAuth, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotPath = r.Header.Get("Authorization"), r.URL.Path
		var req chatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		switch req.Model {
		case "empty":
			_, _ = w.Write([]byte(`{"choices":[]}`))
		case "fail":
			w.WriteHeader(500)
		case "badjson":
			_, _ = w.Write([]byte(`{`))
		default:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"halo"}}]}`))
		}
	}))
	defer srv.Close()

	c := NewChatClient(srv.URL+"/", "kunci", "ok")
	out, err := c.Complete(context.Background(), []Message{{Role: "user", Content: "hi"}})
	require.NoError(t, err)
	assert.Equal(t, "halo", out)
	assert.Equal(t, "Bearer kunci", gotAuth)
	assert.Equal(t, "/chat/completions", gotPath)

	c.Model = "empty"
	_, err = c.Complete(context.Background(), nil)
	assert.ErrorIs(t, err, ErrEmptyCompletion)
	c.Model = "fail"
	_, err = c.Complete(context.Background(), nil)
	assert.Error(t, err)
	c.Model = "badjson"
	_, err = c.Complete(context.Background(), nil)
	assert.Error(t, err)

	// tanpa kunci => tanpa header Authorization
	c = NewChatClient(srv.URL, "", "ok")
	_, err = c.Complete(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "", gotAuth)

	// transport error & URL tidak valid
	c = NewChatClient("http://127.0.0.1:1", "", "ok")
	_, err = c.Complete(context.Background(), nil)
	assert.Error(t, err)
	c = NewChatClient("http://bad host", "", "ok")
	_, err = c.Complete(context.Background(), nil)
	assert.Error(t, err)
}

func TestSearchClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("q") {
		case "top":
			_, _ = w.Write([]byte(`{"results":[{"title":"A","url":"u1","snippet":"s1"},{"title":"B","url":"u2","content":"c2"}]}`))
		case "nested":
			_, _ = w.Write([]byte(`{"data":{"results":[{"title":"N","url":"u","snippet":"s"}]}}`))
		case "many":
			items := make([]string, 8)
			for i := range items {
				items[i] = `{"title":"t","url":"u","snippet":"s"}`
			}
			_, _ = w.Write([]byte(`{"results":[` + strings.Join(items, ",") + `]}`))
		case "fail":
			w.WriteHeader(503)
		}
	}))
	defer srv.Close()

	s := NewSearchClient(srv.URL, "k")
	got, err := s.Search(context.Background(), "top")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "s1", got[0].Snippet)
	assert.Equal(t, "c2", got[1].Snippet)

	got, err = s.Search(context.Background(), "nested")
	require.NoError(t, err)
	assert.Len(t, got, 1)

	got, err = s.Search(context.Background(), "many")
	require.NoError(t, err)
	assert.Len(t, got, maxSources)

	_, err = s.Search(context.Background(), "fail")
	assert.Error(t, err)

	// endpoint yang sudah memiliki query string
	s = NewSearchClient(srv.URL+"?lang=id", "")
	got, err = s.Search(context.Background(), "nested")
	require.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	run := func(svc *Service, body string) (int, map[string]any) {
		r := gin.New()
		r.POST("/g", NewHandler(svc).Generate)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("POST", "/g", bytes.NewBufferString(body)))
		var out map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out
	}

	code, out := run(NewService(nil, &fakeChat{out: goodJSON}), `{"topic":"Golang"}`)
	assert.Equal(t, 200, code)
	assert.Equal(t, "success", out["status"])

	code, _ = run(NewService(nil, &fakeChat{}), `{"topic":"x"}`)
	assert.Equal(t, 400, code)

	code, out = run(NewService(nil, &fakeChat{err: errBoom}), `{"topic":"Golang"}`)
	assert.Equal(t, 500, code)
	assert.Equal(t, "error", out["status"])
}
