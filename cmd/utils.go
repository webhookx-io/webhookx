package cmd

import (
	"fmt"
	"io"
	"strings"
)

var ANSWERS = map[string]bool{
	"y":   true,
	"yes": true,
	"n":   false,
	"no":  false,
}

func prompt(w io.Writer, q string) bool {
	_, _ = w.Write([]byte("> " + q + " [Y/N] "))
	var answer string
	_, _ = fmt.Scan(&answer)
	return ANSWERS[strings.ToLower(answer)]
}
