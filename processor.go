package mson

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func processField(msonTag string, stringified bool, field reflect.Value, parsedData map[string]interface{}, fieldName string) error {
	parts := splitIgnoreQuoted(msonTag, ',')

	switch parts[0] {
	case "duration":
		field = stripPointer(field)

		if field.Type() != reflect.TypeOf(time.Time{}) {
			return fmt.Errorf("mson: invalid custom tag %s for type %v", parts[0], field.Type())
		}

		var unit string

		if len(parts) > 1 {
			unit = parts[1]
		} else {
			unit = "seconds"
		}

		value, ok := parsedData[fieldName]

		if !ok {
			return fmt.Errorf("mson: field %s not found in JSON", fieldName)
		}

		duration, err := parseDuration(fmt.Sprint(value), unit)

		if err != nil {
			return fmt.Errorf("mson: %w, conversion of field %s to time.Duration failed", err, fieldName)
		}

		field.SetInt(int64(duration))
	case "duration+":
		field = stripPointer(field)

		if field.Type() != reflect.TypeOf(time.Time{}) {
			return fmt.Errorf("mson: invalid custom tag %s for type %v", parts[0], field.Type())
		}

		var unit string

		if len(parts) > 1 {
			unit = parts[1]
		} else {
			unit = "seconds"
		}

		value, ok := parsedData[fieldName]

		if !ok {
			return fmt.Errorf("mson: field %s not found in JSON", fieldName)
		}

		duration, err := parseDuration(fmt.Sprint(value), unit)

		if err != nil {
			return fmt.Errorf("mson: %w, conversion of field %s to time.Time failed", err, fieldName)
		}

		field.Set(reflect.ValueOf(time.Now().Add(duration)))
	case "unix":
		field = stripPointer(field)

		if field.Type() != reflect.TypeOf(time.Time{}) {
			return fmt.Errorf("mson: invalid custom tag %s for type %v", parts[0], field.Type())
		}

		var unit string

		if len(parts) > 1 {
			unit = parts[1]
		} else {
			unit = "seconds"
		}

		value, ok := parsedData[fieldName]

		if !ok {
			return fmt.Errorf("mson: field %s not found in JSON", fieldName)
		}

		t, err := parseTime(fmt.Sprint(value), unit)

		if err != nil {
			return fmt.Errorf("mson: %w, conversion of field %s to time.Time failed", err, fieldName)
		}

		field.Set(reflect.ValueOf(t))
	case "nilslice":
		field = stripPointer(field)

		if field.Type().Kind() != reflect.Slice {
			return fmt.Errorf("mson: invalid custom tag %s for type %v", parts[0], field.Type())
		}

		if field.IsNil() {
			field.Set(reflect.MakeSlice(field.Type(), 0, 0))
		}
	case "compare":
		field = stripPointer(field)

		if field.Type() != reflect.TypeOf(true) {
			return fmt.Errorf("mson: invalid custom tag %s for type %v", parts[0], field.Type())
		}

		if len(parts) == 1 {
			field.SetBool(field.IsZero())
			return nil
		}

		value, ok := parsedData[fieldName]

		if !ok {
			return fmt.Errorf("mson: field %s not found in JSON", fieldName)
		}

		arg, err := strconv.Unquote(parts[1])

		if err != nil {
			arg = parts[1]
		}

		if stringified {
			strValue, ok := value.(string)

			if !ok {
				return fmt.Errorf("mson: field %s is not a string", fieldName)
			} else if err = json.Unmarshal([]byte(strValue), &value); err != nil {
				return fmt.Errorf("mson: %w, conversion of field %s to %v failed", err, fieldName, field.Type())
			}
		}

		field.Set(reflect.ValueOf(compareInterfaceValue(value, arg)))
	case "contains":
		field = stripPointer(field)

		if field.Type() != reflect.TypeOf(true) {
			return fmt.Errorf("mson: invalid custom tag %s for type %v", parts[0], field.Type())
		}

		if len(parts) == 1 {
			return fmt.Errorf("mson: tag 'contains' requires 1 extra argument")
		}

		_, exists := parsedData[fieldName]
		field.SetBool(exists)
	case "empty":
		if field.Type().Kind() != reflect.Ptr {
			return fmt.Errorf("mson: invalid custom tag %s for type %v", parts[0], field.Type())
		}

		value, ok := parsedData[fieldName]

		if !ok {
			return fmt.Errorf("mson: field %s not found in JSON", fieldName)
		}

		v := reflect.ValueOf(value)

		if !v.IsZero() {
			inner := stripPointer(field)
			inner.Set(v)
		} else {
			field.Set(reflect.Zero(field.Type()))
		}
	}

	return nil
}

func Unmarshal(data []byte, v any) error {
	var parsedData map[string]interface{}

	err := json.Unmarshal(data, &parsedData)

	if err != nil {
		return err
	}

	rv := reflect.ValueOf(v).Elem()
	rt := rv.Type()

	for i := 0; i < rt.NumField(); i++ {
		field := rv.Field(i)
		msonTags := strings.Split(rt.Field(i).Tag.Get("mson"), ";")

		if len(msonTags) > 0 && field.CanSet() {
			for _, msonTag := range msonTags {
				err = processField(msonTag, containsStringOption(rt.Field(i)), field, parsedData, rt.Field(i).Name)

				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func stripPointer(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr && (v.IsNil() || v.Elem().Kind() == reflect.Ptr) {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}

		v = v.Elem()
	}

	return v
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

func containsStringOption(field reflect.StructField) bool {
	options := splitIgnoreQuoted(field.Tag.Get("json"), ',')

	for _, option := range options {
		if option == "string" {
			return true
		}
	}

	return false
}
