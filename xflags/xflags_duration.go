package xflags

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Duration is a time.Duration that parses the day and week unit suffixes "d"
// and "w" in addition to the units time.ParseDuration accepts (ns, us, ms, s,
// m, h). A bare number with no unit is interpreted as seconds, matching common
// Prometheus-style flag input. It implements encoding.TextUnmarshaler and
// encoding.TextMarshaler, so it works as an xflags option type and round-trips
// through text.
type Duration time.Duration

// Day and week durations, which time.Duration does not define.
const (
	Day  = Duration(24 * time.Hour)
	Week = 7 * Day
)

// durationUnits lists the extra suffixes this type understands beyond
// time.ParseDuration, from longest to shortest suffix.
var durationUnits = []struct {
	suffix string
	unit   time.Duration
}{
	{"w", time.Duration(Week)},
	{"d", time.Duration(Day)},
}

// ParseDuration parses a duration string. It accepts every unit
// time.ParseDuration does (ns, us, ms, s, m, h) plus "d" (24h) and "w" (7d).
// A bare number with no unit is interpreted as seconds. It reports an error for
// empty or malformed input.
func ParseDuration(s string) (Duration, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return 0, fmt.Errorf("empty duration")
	}

	for _, u := range durationUnits {
		if num, ok := strings.CutSuffix(trimmed, u.suffix); ok {
			value, err := strconv.ParseFloat(strings.TrimSpace(num), 64)
			if err != nil {
				return 0, fmt.Errorf("invalid duration %q", s)
			}
			return scaleDuration(value, u.unit, s)
		}
	}

	// A bare number is seconds; delegate anything with a known unit to the
	// standard parser so fractional and compound values keep working.
	if value, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return scaleDuration(value, time.Second, s)
	}

	d, err := time.ParseDuration(trimmed)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q", s)
	}
	return Duration(d), nil
}

const (
	maxDuration = time.Duration(^uint64(0) >> 1)
	minDuration = -maxDuration - 1
)

// scaleDuration multiplies a parsed number by a unit, rejecting NaN and values
// that fall outside the time.Duration range. s is the original input, used for
// error messages.
func scaleDuration(value float64, unit time.Duration, s string) (Duration, error) {
	scaled := value * float64(unit)
	if math.IsNaN(scaled) || scaled > float64(maxDuration) || scaled < float64(minDuration) {
		return 0, fmt.Errorf("duration overflow %q", s)
	}
	return Duration(scaled), nil
}

// UnmarshalText parses a duration, implementing encoding.TextUnmarshaler.
func (d *Duration) UnmarshalText(text []byte) error {
	v, err := ParseDuration(string(text))
	if err != nil {
		return err
	}
	*d = v
	return nil
}

// MarshalText renders the duration, implementing encoding.TextMarshaler. It uses
// the same format as time.Duration.String.
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// String renders the duration using time.Duration's format.
func (d Duration) String() string {
	return time.Duration(d).String()
}
