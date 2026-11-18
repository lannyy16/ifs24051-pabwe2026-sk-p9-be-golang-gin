package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func run(t *testing.T, fn func(c *gin.Context)) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	fn(c)
	var body map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return w, body
}

func TestSuccessWithAndWithoutData(t *testing.T) {
	w, body := run(t, func(c *gin.Context) { Success(c, 200, "ok", gin.H{"a": 1}) })
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "success", body["status"])
	assert.NotNil(t, body["data"])

	_, body = run(t, func(c *gin.Context) { Success(c, 200, "ok", nil) })
	_, has := body["data"]
	assert.False(t, has)
}

func TestFailAndError(t *testing.T) {
	w, body := run(t, func(c *gin.Context) { Fail(c, 400, "bad") })
	assert.Equal(t, 400, w.Code)
	assert.Equal(t, "fail", body["status"])

	w, body = run(t, func(c *gin.Context) { Error(c, 500, "boom") })
	assert.Equal(t, 500, w.Code)
	assert.Equal(t, "error", body["status"])
}

func TestAbortVariants(t *testing.T) {
	w, body := run(t, func(c *gin.Context) { AbortFail(c, 401, "no") })
	assert.Equal(t, 401, w.Code)
	assert.Equal(t, "fail", body["status"])

	w, body = run(t, func(c *gin.Context) { AbortError(c, 503, "down") })
	assert.Equal(t, 503, w.Code)
	assert.Equal(t, "error", body["status"])
}
