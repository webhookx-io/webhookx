package accesslog

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/webhookx-io/webhookx/utils"
	"net/http"
	"time"
)

type Entry struct {
	Timestamp time.Time

	// Request
	Username   string
	Method     string
	Path       string
	Proto      string
	RemoteAddr string
	RequestIP  string
	Referer    string
	UserAgent  string

	// Response
	Status   int
	Latency  time.Duration
	BodySize int
}

func NewEntry(r *http.Request) *Entry {
	host, _ := parseHostPort(r.RemoteAddr)
	username, _ := extractBasicAuth(r)
	entry := Entry{
		Username:   username,
		Method:     r.Method,
		Path:       r.URL.Path,
		Proto:      r.Proto,
		RemoteAddr: r.RemoteAddr,
		RequestIP:  host,
		Referer:    r.Referer(),
		UserAgent:  r.UserAgent(),
	}
	return &entry
}

func (m *Entry) MarshalZerologObject(e *zerolog.Event) {
	// TODO
	e.Str("method", m.Method)
	e.Str("path", m.Path)
	e.Str("proto", m.Proto)
	e.Int("status", m.Status)
	e.Str("ip", m.RequestIP)
	e.Str("username", m.Username)
	e.Str("referer", m.Referer)
	e.Str("user_agent", m.UserAgent)
	e.Int64("latency", m.Latency.Milliseconds())
	e.Int("response_size", m.BodySize)
}

func (m *Entry) String() string {
	return fmt.Sprintf(`%s - %s "%s %s %s" %d %d %dms "%s" "%s"`,
		m.RequestIP,
		utils.DefaultIfZero(m.Username, "-"),
		m.Method,
		m.Path,
		m.Proto,
		m.Status,
		m.BodySize,
		m.Latency.Milliseconds(),
		utils.DefaultIfZero(m.Referer, "-"),
		utils.DefaultIfZero(m.UserAgent, "-"))
}
