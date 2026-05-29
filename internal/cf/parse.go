package cf

import (
	"fmt"
	"strings"
	"time"

	"github.com/Arpeeee/ncterm/internal/nc"
)

// AxisType identifies the physical role of a dimension variable.
type AxisType int

const (
	AxisUnknown AxisType = iota
	AxisTime
	AxisLat
	AxisLon
	AxisLevel
)

func (a AxisType) String() string {
	switch a {
	case AxisTime:
		return "time"
	case AxisLat:
		return "lat"
	case AxisLon:
		return "lon"
	case AxisLevel:
		return "level"
	default:
		return "unknown"
	}
}

// DetectAxis identifies the axis type using CF convention attributes,
// falling back to common variable name patterns.
func DetectAxis(v nc.Variable) AxisType {
	if axis, ok := v.Attrs["axis"].(string); ok {
		switch strings.ToUpper(axis) {
		case "T":
			return AxisTime
		case "Y":
			return AxisLat
		case "X":
			return AxisLon
		case "Z":
			return AxisLevel
		}
	}

	if stdName, ok := v.Attrs["standard_name"].(string); ok {
		switch stdName {
		case "latitude":
			return AxisLat
		case "longitude":
			return AxisLon
		case "time":
			return AxisTime
		case "air_pressure", "altitude", "depth", "height",
			"atmosphere_sigma_coordinate",
			"atmosphere_hybrid_sigma_pressure_coordinate":
			return AxisLevel
		}
	}

	switch strings.ToLower(v.Name) {
	case "lat", "latitude":
		return AxisLat
	case "lon", "longitude":
		return AxisLon
	case "time":
		return AxisTime
	case "lev", "level", "plev", "depth", "sigma", "eta":
		return AxisLevel
	}

	return AxisUnknown
}

// ParseTimeUnits parses a CF units string such as "days since 1900-01-01 00:00:00".
// Returns the base time and duration of one unit step.
func ParseTimeUnits(units string) (base time.Time, step time.Duration, err error) {
	parts := strings.SplitN(units, " since ", 2)
	if len(parts) != 2 {
		return time.Time{}, 0, fmt.Errorf("not a CF time units string: %q", units)
	}

	stepStr := strings.TrimSpace(parts[0])
	baseStr := strings.TrimSpace(parts[1])

	switch stepStr {
	case "days", "day", "d":
		step = 24 * time.Hour
	case "hours", "hour", "hr", "h":
		step = time.Hour
	case "minutes", "minute", "min":
		step = time.Minute
	case "seconds", "second", "sec", "s":
		step = time.Second
	default:
		return time.Time{}, 0, fmt.Errorf("unknown time unit: %q", stepStr)
	}

	for _, f := range []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	} {
		if t, e := time.Parse(f, baseStr); e == nil {
			return t, step, nil
		}
	}

	return time.Time{}, 0, fmt.Errorf("cannot parse base date: %q", baseStr)
}

// FormatTime formats a numeric CF time value as a human-readable date string.
func FormatTime(base time.Time, step time.Duration, value float64) string {
	t := base.Add(time.Duration(value * float64(step)))
	return t.UTC().Format("2006-01-02 15:04")
}
