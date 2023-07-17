package mson

import (
	"reflect"
	"strconv"
	"strings"
	"time"
)

func splitIgnoreQuoted(s string, sep byte) []string {
	var parts []string
	var start, end int
	var inQuotes bool

	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			inQuotes = !inQuotes
		case sep:
			if !inQuotes {
				parts = append(parts, strings.TrimSpace(s[start:end]))
				start = i + 1
			}
		}

		end = i + 1
	}

	parts = append(parts, strings.TrimSpace(s[start:end]))
	return parts
}

func parseDuration(value, unit string) (time.Duration, error) {
	seconds, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}

	switch unit {
	case "nanoseconds":
		return time.Duration(seconds), nil
	case "microseconds":
		return time.Duration(seconds * float64(time.Microsecond)), nil
	case "milliseconds":
		return time.Duration(seconds * float64(time.Millisecond)), nil
	case "minutes":
		return time.Duration(seconds * float64(time.Minute)), nil
	case "hours":
		return time.Duration(seconds * float64(time.Hour)), nil
	case "seconds":
		fallthrough
	default:
		return time.Duration(seconds * float64(time.Second)), nil
	}
}

func parseTime(value, unit string) (time.Time, error) {
	unixTime, err := strconv.ParseFloat(value, 64)

	if err != nil {
		return time.Time{}, err
	}

	switch unit {
	case "nanoseconds":
		return time.Unix(0, int64(unixTime)), nil
	case "microseconds":
		return time.Unix(0, int64(unixTime*1000)), nil
	case "milliseconds":
		return time.UnixMilli(int64(unixTime)), nil
	case "minutes":
		return time.Unix(int64(unixTime*60), 0), nil
	case "hours":
		return time.Unix(int64(unixTime*3600), 0), nil
	case "seconds":
		fallthrough
	default:
		return time.Unix(int64(unixTime), 0), nil
	}
}

func compareInterfaceValue(value interface{}, arg string) bool {
	switch v := value.(type) {
	case bool:
		return (arg == "true" && v) || (arg == "false" && !v)
	case int:
		argInt, err := strconv.Atoi(arg)
		return err == nil && v == argInt
	case float64:
		argFloat, err := strconv.ParseFloat(arg, 64)
		return err == nil && v == argFloat
	}

	return false
}

func containsOption(options []string, target string) bool {
	for _, option := range options {
		if option == target {
			return true
		}
	}

	return false
}

func stripPointer(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}

		v = v.Elem()
	}

	return v
}
