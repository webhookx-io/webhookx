package accesslog

import (
	"encoding/base64"
	"net"
	"net/http"
	"strings"
)

func extractBasicAuth(r *http.Request) (username string, password string) {
	if r.URL.User != nil {
		username = r.URL.User.Username()
		password, _ = r.URL.User.Password()
		return
	}

	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Basic ") {
		payload, err := base64.StdEncoding.DecodeString(auth[6:])
		if err != nil {
			return
		}
		parts := strings.SplitN(string(payload), ":", 2)
		if len(parts) != 2 {
			return
		}
		username = parts[0]
		password = parts[1]
		return
	}

	return
}

func parseHostPort(hostport string) (string, string) {
	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport, ""
	}
	return host, port
}
