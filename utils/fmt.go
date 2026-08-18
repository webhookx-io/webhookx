package utils

import (
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var dayDurationPattern = regexp.MustCompile(`(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)d`)

// ParseDuration parses a duration using Go's duration syntax with the addition
// of "d" as a 24-hour day unit.
func ParseDuration(value string) (time.Duration, error) {
	value = dayDurationPattern.ReplaceAllStringFunc(value, func(dayValue string) string {
		number := strings.TrimSuffix(dayValue, "d")
		days, _ := new(big.Rat).SetString(number)
		hours := days.Mul(days, big.NewRat(24, 1))

		precision := 0
		if dot := strings.IndexByte(number, '.'); dot >= 0 {
			precision = len(number) - dot - 1
		}
		return hours.FloatString(precision) + "h"
	})

	return time.ParseDuration(value)
}

func FormatDuration(d time.Duration) string {
	if d == 0 {
		return "0"
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
