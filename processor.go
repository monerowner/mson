package mson

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func processTag(field reflect.Value, value interface{}, msonTag string, fieldName string) error {
	setNewValue := func(field reflect.Value) {
		value = field.Interface()
	}

	for _, opt := range splitIgnoreQuoted(msonTag, ';') {
		parts := splitIgnoreQuoted(opt, ',')
		var inner reflect.Value

		switch parts[0] {
		case "duration":
			inner = stripPointer(field)

			if field.Type() != reflect.TypeOf(time.Time{}) {
				return fmt.Errorf("mson: invalid custom tag %s for type %v", parts[0], field.Type())
			}

			var unit string

			if len(parts) > 1 {
				unit = parts[1]
			} else {
				unit = "seconds"
			}

			duration, err := parseDuration(fmt.Sprint(value), unit)

			if err != nil {
				return fmt.Errorf("mson: %w, conversion of field %s to time.Duration failed", err, fieldName)
			}

			field.SetInt(int64(duration))
			setNewValue(field)
		case "duration+":
			inner = stripPointer(field)

			if field.Type() != reflect.TypeOf(time.Time{}) {
				return fmt.Errorf("mson: invalid custom tag %s for type %v", parts[0], field.Type())
			}

			var unit string

			if len(parts) > 1 {
				unit = parts[1]
			} else {
				unit = "seconds"
			}

			duration, err := parseDuration(fmt.Sprint(value), unit)

			if err != nil {
				return fmt.Errorf("mson: %w, conversion of field %s to time.Time failed", err, fieldName)
			}

			field.Set(reflect.ValueOf(time.Now().Add(duration)))
			setNewValue(field)
		case "unix":
			inner = stripPointer(field)

			if field.Type() != reflect.TypeOf(time.Time{}) {
				return fmt.Errorf("mson: invalid custom tag %s for type %v", parts[0], field.Type())
			}

			var unit string

			if len(parts) > 1 {
				unit = parts[1]
			} else {
				unit = "seconds"
			}

			t, err := parseTime(fmt.Sprint(value), unit)

			if err != nil {
				return fmt.Errorf("mson: %w, conversion of field %s to time.Time failed", err, fieldName)
			}

			field.Set(reflect.ValueOf(t))
			setNewValue(field)
		case "nilslice":
			inner = stripPointer(field)

			if field.Type().Kind() != reflect.Slice {
				return fmt.Errorf("mson: invalid custom tag %s for type %v", parts[0], field.Type())
			}

			if field.IsNil() {
				field.Set(reflect.MakeSlice(field.Type(), 0, 0))
				setNewValue(field)
			}
		case "compare":
			inner = stripPointer(field)

			if field.Type() != reflect.TypeOf(true) {
				return fmt.Errorf("mson: invalid custom tag %s for type %v", parts[0], field.Type())
			}

			if len(parts) > 1 {
				arg, err := strconv.Unquote(parts[1])

				if err != nil {
					arg = parts[1]
				}

				field.SetBool(compareInterfaceValue(value, arg))
			} else {
				field.SetBool(field.IsZero())
			}

			setNewValue(field)
		case "contains":
			inner = stripPointer(field)

			if field.Type() != reflect.TypeOf(true) {
				return fmt.Errorf("mson: invalid custom tag %s for type %v", parts[0], field.Type())
			}

			field.SetBool(true)
			setNewValue(field)
		case "empty":
			if field.Type().Kind() != reflect.Ptr {
				return fmt.Errorf("mson: invalid custom tag %s for type %v", parts[0], field.Type())
			}

			v := reflect.ValueOf(value)
			var empty bool

			if len(parts) > 1 {
				isZero := v.MethodByName(parts[1])

				if !isZero.IsValid() {
					return fmt.Errorf("mson: invalid function name %s provided as argument to empty", parts[1])
				}

				if isZero.Type().NumIn() > 0 {
					return fmt.Errorf("mson: invalid function %s provided as argument to empty, must have 0 parameters", parts[1])
				}

				if isZero.Type().NumOut() != 1 && isZero.Type().Out(0) != reflect.TypeOf(true) {
					return fmt.Errorf("mson: invalid function %s provided as argument to empty, must return one boolean value", parts[1])
				}

				empty = isZero.Call(nil)[0].Bool()
			} else {
				empty = v.IsZero()
			}

			if empty {
				field.Set(reflect.Zero(field.Type()))
				return nil
			}

			inner = stripPointer(field)

			if inner.Type().Kind() == v.Type().Kind() {
				inner.Set(v)
				setNewValue(inner)
			}
		}
	}

	return nil
}

func processField(field reflect.Value, metaData reflect.StructField, data map[string]interface{}) error {
	jsonTag := splitIgnoreQuoted(metaData.Tag.Get("json"), ',')
	if containsOption(jsonTag, "-") {
		return nil
	}

	var fieldName string

	if len(jsonTag) > 0 && jsonTag[0] != "" {
		fieldName = jsonTag[0]
	} else {
		fieldName = metaData.Name
	}

	value, ok := data[strings.ToLower(fieldName)]
	if !ok {
		field.Set(reflect.Zero(field.Type()))
		return nil
	}

	if containsOption(jsonTag, "string") {
		strValue, ok := value.(string)
		if !ok {
			return fmt.Errorf("mson: field %s is not a string", fieldName)
		}
		if err := json.Unmarshal([]byte(strValue), &value); err != nil {
			return fmt.Errorf("mson: %w, conversion of field %s to %v failed", err, fieldName, field.Type())
		}
	}

	return processTag(field, value, metaData.Tag.Get("mson"), fieldName)
}

func Unmarshal(data []byte, v any) error {
	var parsedData map[string]interface{}

	err := json.Unmarshal(data, &parsedData)

	if err != nil {
		return err
	}

	for k, v := range parsedData {
		delete(parsedData, k)
		parsedData[strings.ToLower(k)] = v
	}

	rv := reflect.ValueOf(v).Elem()
	rt := rv.Type()

	for i := 0; i < rt.NumField(); i++ {
		field := rv.Field(i)

		if field.CanSet() {
			err = processField(field, rt.Field(i), parsedData)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
