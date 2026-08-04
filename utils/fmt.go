package utils

import (
	"strconv"
	"strings"
	"time"
)

func FormatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	d = d.Abs()
	var sb strings.Builder

	day := 24 * time.Hour
	if d >= day {
		days := d / day
		sb.WriteString(strconv.FormatInt(int64(days), 10))
		sb.WriteByte('d')
		d %= day
	}

	if d >= time.Hour {
		hours := d / time.Hour
		sb.WriteString(strconv.FormatInt(int64(hours), 10))
		sb.WriteByte('h')
		d %= time.Hour
	}

	if d >= time.Minute {
		minutes := d / time.Minute
		sb.WriteString(strconv.FormatInt(int64(minutes), 10))
		sb.WriteByte('m')
		d %= time.Minute
	}

	if d >= time.Second {
		seconds := d / time.Second
		sb.WriteString(strconv.FormatInt(int64(seconds), 10))
		sb.WriteByte('s')
		d %= time.Second
	}

	if d > 0 {
		sb.WriteString(d.String())
	}

	return sb.String()
}
