// Copyright 2019 Qiang Xue. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package env

import (
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type (
	// Loader loads a struct with values returned by a lookup function.
	Loader struct {
		log    LogFunc
		prefix string
		lookup LookupFunc
	}

	// LogFunc logs a message.
	LogFunc func(format string, args ...interface{})

	// LookupFunc looks up a name and returns the corresponding value and a flag indicating if the name is found.
	LookupFunc func(name string) (string, bool)

	// Setter sets the object with a string value.
	Setter interface {
		// Set sets the object with a string value.
		Set(value string) error
	}
)

var (
	// ErrStructPointer represents the error that a pointer to a struct is expected.
	ErrStructPointer = errors.New("must be a pointer to a struct")
	// ErrNilPointer represents the error that a nil pointer is received
	ErrNilPointer = errors.New("the pointer should not be nil")
	// TagName specifies the tag name for customizing struct field names when loading environment variables
	TagName = "env"

	// nameRegex is used to convert a string from camelCase into snake format
	nameRegex = regexp.MustCompile(`([^A-Z_])([A-Z])`)
	// loader is the default loader used by the "Load" function at the package level.
	loader = New("APP_", log.Printf)
)

// New creates a new environment variable loader.
// The prefix will be used to prefix the struct field names when they are used to read from environment variables.
func New(prefix string, log LogFunc) *Loader {
	return &Loader{prefix: prefix, lookup: os.LookupEnv, log: log}
}

// NewWithLookup creates a new loader using the given lookup function.
// The prefix will be used to prefix the struct field names when they are used to read from environment variables.
func NewWithLookup(prefix string, lookup LookupFunc, log LogFunc) *Loader {
	return &Loader{prefix: prefix, lookup: lookup, log: log}
}

// Load populates a struct with the values read from the corresponding environment variables.
// Load uses "APP_" as the prefix for environment variable names. It uses log.Printf() to log the data population
// of each struct field.
// For more details on how Load() works, please refer to Loader.Load().
func Load(structPtr interface{}) error {
	return loader.Load(structPtr)
}

// Load populates a struct with the values read returned by the specified lookup function.
// The struct must be specified as a pointer.
//
// Load calls a lookup function for each public struct field. If the function returns a value, it is parsed according
// to the field type and assigned to the field.
//
// Load uses the following rules to determine what name should be used to look up the value for a struct field:
// - If the field has an "env" tag, use the tag value as the name, unless the tag is "-" in which case it means
//   the field should be skipped.
// - If the field has no "env" tag, turn the field name into snake format and use that as the name.
// - Names are turned into upper case and prefixed with the specified prefix.
//
// The following types of struct fields are supported:
// - types implementing Setter, TextUnmarshaler, BinaryUnmarshaler: the corresponding interface method will be used
//   to populate the field with a string
// - primary types (e.g. int, string): appropriate parsing functions will be called to parse a string value
// - other types (e.g. array, struct): the string value is assumed to be in JSON format and is decoded/assigned to the field.
//
// Load will log every field that is populated. In case when a field is tagged with `env:",secret"`, the value being
// logged will be masked for security purpose.
func (l *Loader) Load(structPtr interface{}) error {
	rval := reflect.ValueOf(structPtr)
	if rval.Kind() != reflect.Ptr || !rval.IsNil() && rval.Elem().Kind() != reflect.Struct {
		return ErrStructPointer
	}
	if rval.IsNil() {
		return ErrNilPointer
	}

	rval = rval.Elem()
	rtype := rval.Type()

	for i := 0; i < rval.NumField(); i++ {
		f := rval.Field(i)
		if !f.CanSet() {
			continue
		}

		ft := rtype.Field(i)

		if ft.Anonymous {
			f = indirect(f)
			if f.Kind() == reflect.Struct {
				// populate embedded struct
				if err := l.Load(f.Addr().Interface()); err != nil {
					return err
				}
			}
			continue
		}

		name, secret := getName(ft.Tag.Get(TagName), ft.Name)
		if name == "-" {
			continue
		}

		name = l.prefix + strings.ToUpper(name)

		if value, ok := l.lookup(name); ok {
			logValue := value
			if l.log != nil {
				if secret {
					l.log("set %v with $%v=\"***\"", ft.Name, name)
				} else {
					l.log("set %v with $%v=\"%v\"", ft.Name, name, logValue)
				}
			}
			if err := setValue(f, value); err != nil {
				return fmt.Errorf("error reading \"%v\": %v", ft.Name, err)
			}
		}
	}
	return nil
}

// indirect dereferences pointers and returns the actual value it points to.
// If a pointer is nil, it will be initialized with a new value.
func indirect(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	return v
}

// getName generates the environment variable name from a struct field tag and the field name.
func getName(tag string, field string) (string, bool) {
	name := strings.TrimSuffix(tag, ",secret")
	nameLen := len(name)

	// If the `,secret` suffix was found, it would have been trimmed, so the length should be different.
	secret := nameLen < len(tag)

	if nameLen == 0 {
		name = camelCaseToSnake(field)
	}
	return name, secret
}

// camelCaseToSnake converts a name from camelCase format into snake format.
func camelCaseToSnake(name string) string {
	return nameRegex.ReplaceAllString(name, "${1}_$2")
}

// setValue assigns a string value to a reflection value using appropriate string parsing and conversion logic.
func setValue(rval reflect.Value, value string) error {
	rval = indirect(rval)
	rtype := rval.Type()

	if !rval.CanAddr() {
		return errors.New("the value is unaddressable")
	}

	// if the reflection value implements supported interface, use the interface to set the value
	pval := rval.Addr().Interface()
	if p, ok := pval.(Setter); ok {
		return p.Set(value)
	}
	if p, ok := pval.(encoding.TextUnmarshaler); ok {
		return p.UnmarshalText([]byte(value))
	}
	if p, ok := pval.(encoding.BinaryUnmarshaler); ok {
		return p.UnmarshalBinary([]byte(value))
	}

	// parse the string according to the type of the reflection value and assign it
	switch rtype.Kind() {
	case reflect.String:
		rval.SetString(value)
		break
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, err := strconv.ParseInt(value, 0, rtype.Bits())
		if err != nil {
			return err
		}

		rval.SetInt(val)
		break
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(value, 0, rtype.Bits())
		if err != nil {
			return err
		}
		rval.SetUint(val)
		break
	case reflect.Bool:
		val, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		rval.SetBool(val)
		break
	case reflect.Float32, reflect.Float64:
		val, err := strconv.ParseFloat(value, rtype.Bits())
		if err != nil {
			return err
		}
		rval.SetFloat(val)
		break
	case reflect.Slice:
		if rtype.Elem().Kind() == reflect.Uint8 {
			sl := reflect.ValueOf([]byte(value))
			rval.Set(sl)
			return nil
		}
		fallthrough
	default:
		// assume the string is in JSON format for non-basic types
		return json.Unmarshal([]byte(value), rval.Addr().Interface())
	}

	return nil
}
