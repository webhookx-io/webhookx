package accesslog

import (
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/webhookx-io/webhookx/utils"
	"go.opentelemetry.io/otel/trace"
)

type Entry struct {
	Username string
	Latency  time.Duration
	ClientIP string
	Request  Request
	Response Response
}

type Request struct {
	Method  string
	Path    string
	Proto   string
	Headers map[string]string
}

type Response struct {
	Status int
	Size   int
}

func NewEntry(r *http.Request) *Entry {
	host, _ := parseHostPort(r.RemoteAddr)
	username, _, _ := r.BasicAuth()
	entry := Entry{
		Username: username,
		ClientIP: host,
		Request: Request{
			Method: r.Method,
			Path:   r.URL.Path,
			Proto:  r.Proto,
			Headers: map[string]string{
				"user-agent": r.UserAgent(),
				"referer":    r.Referer(),
			},
		},
	}
	return &entry
}

func (m *Entry) MarshalZerologObject(e *zerolog.Event) {
	e.Str("client_ip", m.ClientIP)
	e.Str("username", m.Username)
	e.Dict("request", zerolog.Dict().
		Str("method", m.Request.Method).
		Str("path", m.Request.Path).
		Str("proto", m.Request.Proto).
		Dict("headers", zerolog.Dict().
			Str("user-agent", m.Request.Headers["user-agent"]).
			Str("referer", m.Request.Headers["referer"])),
	)
	e.Dict("response",
		zerolog.Dict().
			Int("status", m.Response.Status).
			Int("size", m.Response.Size),
	)
	e.Int64("latency", m.Latency.Milliseconds())

	sc := trace.SpanContextFromContext(e.GetCtx())
	if sc.IsValid() {
		e.Str("trace_id", sc.TraceID().String())
	}
}

func (m *Entry) String(e *zerolog.Event) string {
	return fmt.Sprintf(`%s - %s "%s %s %s" %d %d %dms "%s" "%s"`,
		m.ClientIP,
		utils.DefaultIfZero(m.Username, "-"),
		m.Request.Method,
		m.Request.Path,
		m.Request.Proto,
		m.Response.Status,
		m.Response.Size,
		m.Latency.Milliseconds(),
		utils.DefaultIfZero(m.Request.Headers["referer"], "-"),
		utils.DefaultIfZero(m.Request.Headers["user-agent"], "-"),
	)
}
