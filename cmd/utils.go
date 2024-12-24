package cmd

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var ANSWERS = map[string]bool{
	"y":   true,
	"yes": true,
	"n":   false,
	"no":  false,
}

func prompt(q string) bool {
	fmt.Print("> " + q + " [Y/N] ")
	var answer string
	_, _ = fmt.Scan(&answer)
	return ANSWERS[strings.ToLower(answer)]
}

func sendHTTPRequest(req *http.Request, timeout time.Duration) (string, error) {
	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("invalid status code: %d %s", resp.StatusCode, string(b))
	}

	return string(b), nil
}
