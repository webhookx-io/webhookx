package utils

import (
	"fmt"
)

const (
	ColorDarkGray = 90
	ColorDarkBlue = 94
)

func Colorize(s interface{}, color int, enabled bool) string {
	if !enabled || color == 0 {
		return fmt.Sprintf("%s", s)
	}

	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", color, s)
}
