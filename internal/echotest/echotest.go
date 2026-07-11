// Package echotest holds shared helpers for tests that exercise Echo handlers.
// It is deliberately separate from internal/testutil (which provides DB/config
// infra) so that importing this package does not pull in a database driver —
// only Echo and net/http/httptest, both of which compile cleanly under js/wasm.
//
// All helpers materialize returned *echo.HTTPError values onto the recorder via
// Context.Error, mirroring what Echo's ServeHTTP does in production. Without
// that, calling a handler directly leaves the recorder at its default 200 even
// when the handler returned a 4xx/5xx error.
package echotest

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

// CallJSON dispatches a request with a JSON body against the given handler and
// returns the recorder so the caller can inspect status/body.
func CallJSON(t *testing.T, method, target, body string, handler echo.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if err := handler(c); err != nil {
		c.Error(err)
	}
	return rec
}

// CallAuthed is like CallJSON but pre-populates the auth context with the given
// user ID, simulating a valid Bearer token without going through JWT parsing.
func CallAuthed(t *testing.T, method, target, body string, userID uint, handler echo.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)
	if err := handler(c); err != nil {
		c.Error(err)
	}
	return rec
}

// ParamResult bundles an echo.Context with its recorder for handlers that read
// path params (Get/Update/Delete), keeping per-row boilerplate out of tests.
type ParamResult struct {
	Context echo.Context
	Rec     *httptest.ResponseRecorder
}

// NewParamRequest builds a request with an optional JSON body and the :id path
// param set to the given task ID, authenticated as userID.
func NewParamRequest(method, body string, userID, taskID uint) ParamResult {
	e := echo.New()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, "/", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	} else {
		req = httptest.NewRequest(method, "/", nil)
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.FormatUint(uint64(taskID), 10))
	c.Set("user_id", userID)
	return ParamResult{Context: c, Rec: rec}
}

// NewContext builds a bare, unauthenticated Echo context bound to the given
// recorder — useful for handlers tested with custom param values (e.g. a
// non-numeric :id to exercise the bad-request path).
func NewContext(method string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, "/", nil)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}
