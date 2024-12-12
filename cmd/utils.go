package cmd

import (
	"fmt"
	"strings"
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
