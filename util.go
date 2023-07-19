package mson

import (
	"fmt"
	"math"
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

func performArithmeticOperation(value interface{}, parts []string, inverted bool, fieldName string) (interface{}, error) {
	var op1 func(int64, int64) int64
	var op2 func(float64, float64) float64

	switch parts[0] {
	case "add":
		op1 = func(a, b int64) int64 { return a + b }
		op2 = func(a, b float64) float64 { return a + b }
	case "subtract":
		op1 = func(a, b int64) int64 { return a - b }
		op2 = func(a, b float64) float64 { return a - b }
	case "multiply":
		op1 = func(a, b int64) int64 { return a * b }
		op2 = func(a, b float64) float64 { return a * b }
	case "divide":
		op1 = func(a, b int64) int64 { return a / b }
		op2 = func(a, b float64) float64 { return a / b }
	default:
		panic(fmt.Errorf("mson: unknown numerical operation %s", parts[0]))
	}

	if len(parts) < 2 {
		panic(fmt.Errorf("mson: tag option '%s' requires at least one argument", parts[0]))
	}

	if conv, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
		switch v := value.(type) {
		case int:
			value = op1(int64(v), conv)
		case float64:
			value = op2(v, float64(conv))
		default:
			return nil, fmt.Errorf("mson: field %s is not a number", fieldName)
		}

		return value, nil
	}

	if conv, err := strconv.ParseFloat(parts[1], 64); err == nil {
		switch v := value.(type) {
		case int:
			value = op2(float64(v), conv)
		case float64:
			value = op2(v, conv)
		default:
			return nil, fmt.Errorf("mson: field %s is not a number", fieldName)
		}

		return value, nil
	}

	return nil, fmt.Errorf("mson: tag option '%s' received invalid argument %s", parts[0], parts[1])
}

func performNumericalOperation(value interface{}, parts []string, inverted bool, fieldName string) (interface{}, error) {
	var op func(float64) float64

	switch parts[0] {
	case "round":
		op = math.Round
	case "floor":
		op = math.Floor
	case "ceil":
		op = math.Ceil
	default:
		panic(fmt.Errorf("mson: unknown numerical operation %s", parts[0]))
	}

	var places uint8

	if len(parts) > 1 {
		p, err := strconv.ParseUint(parts[1], 10, 8)
		if err != nil {
			return nil, fmt.Errorf("mson: tag option 'round' received invalid argument %s", parts[1])
		}
		places = uint8(p)
	}

	switch v := value.(type) {
	case float64:
		if places == 0 {
			value = int64(op(v))
		} else if inverted {
			for i := uint8(0); i < places; i++ {
				v /= 10
			}
			v = op(v)
			for ; places > 0; places-- {
				v *= 10
			}
		} else {
			for i := uint8(0); i < places; i++ {
				v *= 10
			}
			v = op(v)
			for ; places > 0; places-- {
				v /= 10
			}
		}

		value = v
	case int:
		if inverted && places > 0 {
			conv := float64(v)
			for i := uint8(0); i < places; i++ {
				conv /= 10
			}
			conv = op(conv)
			for ; places > 0; places-- {
				conv *= 10
			}
			value = conv
		}
	default:
		return nil, fmt.Errorf("mson: field %s is not a number", fieldName)
	}

	return value, nil
}
