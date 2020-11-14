package zapmw

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
)

// TestZapMW is not so much a test as it is a bare-essentials harness for visually
// verifying that the library works. I'll do a better job later, maybe.
func TestZapMW(t *testing.T) {
	// init a dev logger
	l := zaptest.NewLogger(t)
	defer l.Sync()

	// create zap middleware
	mw := New(l)

	// create test response writer, request, and handler
	w := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodGet, "http://nowhere.void/path/to/nothing", nil)
	if err != nil {
		t.Fatal(err)
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Extract(r.Context()).Info("handler message", zap.String("handler_field", "testing is fun"))
		w.WriteHeader(http.StatusOK)
	})
	mw(next).ServeHTTP(w, r)

	next = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Extract(r.Context()).Info("handler message", zap.String("handler_field", "testing is fun"))
		w.WriteHeader(http.StatusInternalServerError)
	})
	mw(next).ServeHTTP(w, r)

	// log successful requests at info level
	mw = New(l, WithSuccessLevel(zapcore.InfoLevel), WithRemoteAddr())
	next = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Extract(r.Context()).Info("handler message", zap.String("handler_field", "testing is fun"))
		w.WriteHeader(http.StatusOK)
	})
	mw(next).ServeHTTP(w, r)
}
