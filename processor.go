package mson

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func processTag(field reflect.Value, value interface{}, options []string, fieldName string) error {
	inner := stripPointer(field)

	for _, opt := range options {
		parts := splitIgnoreQuoted(opt, ',')

		var modified string
		var inverted bool

		if len(parts[0]) > 0 && rune(parts[0][len(parts[0])-1]) == '!' {
			inverted = true
			modified = parts[0][:len(parts[0])-1]
		}

		switch modified {
		case "duration":
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

			if inverted {
				value = time.Now().Add(duration)
			} else {
				value = int64(duration)
			}
		case "unix":
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

			value = reflect.ValueOf(t)
		case "nilslice":
			if value == nil {
				if !inverted {
					if inner.Kind() != reflect.Slice {
						return fmt.Errorf("mson: cannot convert field %s to a new slice; field is of kind %s, not a slice", fieldName, inner.Kind())
					}

					value = reflect.SliceOf(inner.Type().Elem())
				}
			} else if inverted {
				v := reflect.ValueOf(value)

				if v.Kind() == reflect.Slice && v.Len() == 0 {
					value = nil
				}
			}
		case "nilmap":
			if value == nil {
				if !inverted {
					if inner.Kind() != reflect.Map {
						return fmt.Errorf("mson: cannot convert field %s to a new map; field is of kind %s, not a map", fieldName, inner.Kind())
					}

					value = reflect.MapOf(inner.Type().Key(), inner.Type().Elem())
				}
			} else if inverted {
				v := reflect.ValueOf(value)

				if v.Kind() == reflect.Map && v.Len() == 0 {
					value = nil
				}
			}
		case "equals":
			if len(parts) > 1 {
				arg, err := strconv.Unquote(parts[1])

				if err != nil {
					arg = parts[1]
				}

				value = reflect.ValueOf(compareInterfaceValue(value, arg) == (!inverted))
			} else {
				value = reflect.ValueOf(inner.IsZero() == (!inverted))
			}
		case "contains":
			// Sets value to true if the field contains the argument, false otherwise
			// There is no 'contains!' alternative because mson ignores non-existent fields
			value = true
		case "empty":
			var empty bool

			if v := reflect.ValueOf(value); len(parts) > 1 {
				isZero := v.MethodByName(parts[1])

				if !isZero.IsValid() || isZero.Type().NumIn() > 0 || isZero.Type().NumOut() != 1 || isZero.Type().Out(0) != reflect.TypeOf(true) {
					panic(fmt.Errorf("mson: invalid function %s provided as argument to empty; function must exist on the type %s, take zero parameters, and return one boolean value", parts[1], v.Type().String()))
				}
				empty = isZero.Call(nil)[0].Bool()
			} else {
				empty = v.IsZero()
			}

			if (empty && !inverted) || (!empty && inverted) {
				field.Set(reflect.Zero(field.Type()))
				return nil
			}
		case "fromstring":
			strValue, ok := value.(string)
			if ok {
				if inverted {
					return fmt.Errorf("mson: field %s is already a string", fieldName)
				}
				if err := json.Unmarshal([]byte(strValue), &value); err != nil {
					return fmt.Errorf("%w, unquoting of field %s to %v failed", fmt.Errorf(strings.Replace(err.Error(), "json", "mson", 1)), fieldName, field.Type())
				}
			} else {
				if !inverted {
					return fmt.Errorf("mson: field %s is not a string", fieldName)
				}

				value = fmt.Sprintf("%v", value)
			}
		case "add", "subtract", "multiply", "divide":
			v, err := performArithmeticOperation(value, parts, inverted, fieldName)

			if err != nil {
				return err
			}

			value = v
		case "round", "floor", "ceil":
			v, err := performNumericalOperation(value, parts, inverted, fieldName)

			if err != nil {
				return err
			}

			value = v
		default:
			panic(fmt.Errorf("mson: unknown tag option %s", parts[0]))
		}
	}

	inner.Set(reflect.ValueOf(value))

	return nil
}

func processField(field reflect.Value, metaData reflect.StructField, data map[string]interface{}) error {
	msonTag := splitIgnoreQuoted(metaData.Tag.Get("json"), ',')

	if len(msonTag) == 0 || msonTag[0] == "-" {
		return nil
	}

	if msonTag[0] == "_" {

	}

	fieldName := msonTag[0]

	if fieldName == "" {
		fieldName = metaData.Name
	} else if fieldName == "_" {
		fieldName = metaData.Name

		

	}

	if value, ok := data[strings.ToLower(fieldName)]; ok {
		return processTag(field, value, msonTag[1:], fieldName)
	}

	field.Set(reflect.Zero(field.Type()))
	return nil
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
