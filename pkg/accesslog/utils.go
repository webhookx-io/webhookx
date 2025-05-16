package accesslog

import (
	"net"
)

func parseHostPort(hostport string) (string, string) {
	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport, ""
	}
	return host, port
}
