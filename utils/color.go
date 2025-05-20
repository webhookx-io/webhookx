package utils

import (
	"fmt"
	"os"
)

const (
	ColorDarkGray = 90
)

func Colorize(s interface{}, c int) string {
	if os.Getenv("NO_COLOR") != "" || c == 0 {
		return fmt.Sprintf("%s", s)
	}

	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", c, s)
}
