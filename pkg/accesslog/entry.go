package accesslog

import (
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/webhookx-io/webhookx/utils"
)

type Entry struct {
	Username string        `json:"username"`
	Latency  time.Duration `json:"latency"`
	ClientIP string        `json:"client_ip"`

	Request  Request  `json:"request"`
	Response Response `json:"response"`
}

type Request struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Proto   string            `json:"proto"`
	Headers map[string]string `json:"headers"`
}

type Response struct {
	Status int `json:"status"`
	Size   int `json:"size"`
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
	e.Any("request", m.Request)
	e.Any("response", m.Response)
	e.Int64("latency", m.Latency.Milliseconds())
}

func (m *Entry) String() string {
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
		utils.DefaultIfZero(m.Request.Headers["user-agent"], "-"))
}
