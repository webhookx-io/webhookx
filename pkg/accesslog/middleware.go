package accesslog

import (
	"net/http"
	"time"
)

type middleware struct {
	logger AccessLogger
}

func NewMiddleware(logger AccessLogger) func(http.Handler) http.Handler {
	h := middleware{
		logger: logger,
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r, next)
		})
	}
}

func (m *middleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.Handler) {
	entry := NewEntry(r)

	now := time.Now()
	rw := &responseWriter{ResponseWriter: w}
	next.ServeHTTP(rw, r)

	entry.Latency = time.Since(now)
	entry.Response.Status = rw.statusCode
	entry.Response.Size = rw.bytesWritten

	m.logger.Log(r.Context(), entry)
}

type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += n
	return n, err
}
