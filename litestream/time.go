package litestream

import (
	"math"
	"strings"
	"time"

	"github.com/ncruces/go-sqlite3/util/sql3util"
)

func parseTimeDelta(s string) (years, months, days int, duration time.Duration, ok bool) {
	duration, err := time.ParseDuration(s)
	if err == nil {
		return 0, 0, 0, duration, true
	}

	if strings.EqualFold(s, "now") {
		return 0, 0, 0, 0, true
	}

	ss := strings.TrimSuffix(strings.ToLower(s), "s")
	switch {
	case strings.HasSuffix(ss, " year"):
		years, duration, ok = parseDateUnit(ss, " year", 365*86400)

	case strings.HasSuffix(ss, " month"):
		months, duration, ok = parseDateUnit(ss, " month", 30*86400)

	case strings.HasSuffix(ss, " day"):
		months, duration, ok = parseDateUnit(ss, " day", 86400)

	case strings.HasSuffix(ss, " hour"):
		duration, ok = parseTimeUnit(ss, " hour", time.Hour)

	case strings.HasSuffix(ss, " minute"):
		duration, ok = parseTimeUnit(ss, " minute", time.Minute)

	case strings.HasSuffix(ss, " second"):
		duration, ok = parseTimeUnit(ss, " second", time.Second)

	default:
		return sql3util.ParseTimeShift(s)
	}
	return
}

func parseDateUnit(s, unit string, seconds float64) (int, time.Duration, bool) {
	f, ok := sql3util.ParseFloat(s[:len(s)-len(unit)])
	if !ok {
		return 0, 0, false
	}

	i, f := math.Modf(f)
	if math.MinInt <= i && i <= math.MaxInt {
		return int(i), time.Duration(f * seconds * float64(time.Second)), true
	}
	return 0, 0, false
}

func parseTimeUnit(s, unit string, scale time.Duration) (time.Duration, bool) {
	f, ok := sql3util.ParseFloat(s[:len(s)-len(unit)])
	return time.Duration(f * float64(scale)), ok
}
