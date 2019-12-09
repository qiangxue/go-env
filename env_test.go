// Copyright 2016 Qiang Xue. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package env

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func Test_indirect(t *testing.T) {
	var a int
	assert.Equal(t, reflect.ValueOf(a).Kind(), indirect(reflect.ValueOf(a)).Kind())
	var b *int
	bi := indirect(reflect.ValueOf(&b))
	assert.Equal(t, reflect.ValueOf(a).Kind(), bi.Kind())
	if assert.NotNil(t, b) {
		assert.Equal(t, 0, *b)
	}
}

type mySet bool

func (v *mySet) Set(value string) error {
	r, err := strconv.ParseBool(strings.ToUpper(value))
	if err != nil {
		return err
	}
	*v = mySet(r)
	return nil
}

type myInt int64

func (v *myInt) UnmarshalText(data []byte) error {
	var x int64
	x, err := strconv.ParseInt(string(data), 10, 0)
	if err != nil {
		return err
	}
	*v = myInt(x)
	return err
}

type myString string

func (v *myString) UnmarshalBinary(data []byte) error {
	*v = myString(string(data) + "ok")
	return nil
}

func Test_setValue(t *testing.T) {
	cfg := struct {
		str1   string
		str2   *string
		int1   int
		uint1  uint64
		bool1  bool
		float1 float32
		slice1 []byte
		slice2 []int
		slice3 []string
		map1   map[string]int
		myint1 myInt
		myint2 *myInt
		mystr1 myString
		mystr2 *myString
		myset  mySet
	}{}

	tests := []struct {
		tag      string
		rval     reflect.Value
		value    string
		expected interface{}
		equal    bool
		err      bool
	}{
		{"t0.1", reflect.ValueOf(&cfg.str1), "abc", "abc", true, false},
		{"t0.2", reflect.ValueOf(&cfg.str2), "abc", "abc", true, false},
		{"t1.1", reflect.ValueOf(&cfg.int1), "1", int(1), true, false},
		{"t1.2", reflect.ValueOf(&cfg.int1), "1", int64(1), false, false},
		{"t1.3", reflect.ValueOf(&cfg.int1), "a1", int(1), true, true},
		{"t2.1", reflect.ValueOf(&cfg.uint1), "1", uint64(1), true, false},
		{"t2.2", reflect.ValueOf(&cfg.uint1), "1", uint32(1), false, false},
		{"t2.3", reflect.ValueOf(&cfg.uint1), "a1", uint64(1), true, true},
		{"t3.1", reflect.ValueOf(&cfg.bool1), "1", true, true, false},
		{"t3.2", reflect.ValueOf(&cfg.bool1), "TRuE", true, true, true},
		{"t3.3", reflect.ValueOf(&cfg.bool1), "TRUE", true, true, false},
		{"t4.1", reflect.ValueOf(&cfg.float1), "12.1", float32(12.1), true, false},
		{"t4.2", reflect.ValueOf(&cfg.float1), "12.1", float64(12.1), false, false},
		{"t4.3", reflect.ValueOf(&cfg.float1), "a12.1", float32(12.1), true, true},
		{"t5.1", reflect.ValueOf(&cfg.slice1), "abc", []byte("abc"), true, false},
		{"t5.2", reflect.ValueOf(&cfg.slice2), "[1,2]", []int{1, 2}, true, false},
		{"t5.3", reflect.ValueOf(&cfg.slice3), "[\"1\",\"2\"]", []string{"1", "2"}, true, false},
		{"t5.4", reflect.ValueOf(&cfg.map1), "{\"a\":1,\"b\":2}", map[string]int{"a": 1, "b": 2}, true, false},
		{"t5.5", reflect.ValueOf(&cfg.map1), "a:1,b:2", "", true, true},
		{"t6.1", reflect.ValueOf(&cfg.myint1), "1", myInt(1), true, false},
		{"t6.2", reflect.ValueOf(&cfg.myint2), "1", myInt(1), true, false},
		{"t6.3", reflect.ValueOf(&cfg.mystr1), "1", myString("1ok"), true, false},
		{"t6.4", reflect.ValueOf(&cfg.mystr2), "1", myString("1ok"), true, false},
		{"t7.1", reflect.ValueOf(&cfg.myset), "1", mySet(true), true, false},
		{"t7.2", reflect.ValueOf(&cfg.myset), "TRuE", mySet(true), true, false},
		{"t7.3", reflect.ValueOf(&cfg.myset), "TRUE", mySet(true), true, false},
		{"t8.1", reflect.ValueOf("test"), "test", "test", true, true},
	}

	for _, test := range tests {
		err := setValue(test.rval, test.value)
		if test.err {
			assert.NotNil(t, err, test.tag)
		} else if assert.Nil(t, err, test.tag) {
			actual := indirect(test.rval)
			if test.equal {
				assert.True(t, reflect.DeepEqual(test.expected, actual.Interface()), test.tag)
			} else {
				assert.False(t, reflect.DeepEqual(test.expected, actual.Interface()), test.tag)
			}
		}
	}
}

func Test_camelCaseToSnake(t *testing.T) {
	tests := []struct {
		tag      string
		input    string
		expected string
	}{
		{"t1", "test", "test"},
		{"t2", "MyName", "My_Name"},
		{"t3", "My2Name", "My2_Name"},
		{"t4", "MyID", "My_ID"},
		{"t5", "My_Name", "My_Name"},
		{"t6", "MyFullName", "My_Full_Name"},
		{"t7", "URLName", "URLName"},
		{"t8", "MyURLName", "My_URLName"},
	}

	for _, test := range tests {
		output := camelCaseToSnake(test.input)
		assert.Equal(t, test.expected, output, test.tag)
	}
}

func Test_getName(t *testing.T) {
	tests := []struct {
		tag    string
		tg     string
		field  string
		name   string
		secret bool
	}{
		{"t1", "", "Name", "Name", false},
		{"t2", "", "MyName", "My_Name", false},
		{"t3", "NaME", "Name", "NaME", false},
		{"t4", "NaME,secret", "Name", "NaME", true},
		{"t5", ",secret", "Name", "Name", true},
		{"t6", "NaME,", "Name", "NaME", false},
	}

	for _, test := range tests {
		name, secret := getName(test.tg, test.field)
		assert.Equal(t, test.name, name, test.tag)
		assert.Equal(t, test.secret, secret, test.tag)
	}
}

type myLogger struct {
	logs []string
}

func (l *myLogger) Log(format string, args ...interface{}) {
	l.logs = append(l.logs, fmt.Sprintf(format, args...))
}

func mockLog(format string, args ...interface{}) {
}

func mockLookup(name string) (string, bool) {
	data := map[string]string{
		"HOST":     "localhost",
		"PORT":     "8080",
		"URL":      "http://example.com",
		"PASSWORD": "xyz",
	}
	value, ok := data[name]
	return value, ok
}

func mockLookup2(name string) (string, bool) {
	data := map[string]string{
		"APP_HOST":     "localhost",
		"APP_PORT":     "8080",
		"APP_URL":      "http://example.com",
		"APP_PASSWORD": "xyz",
	}
	value, ok := data[name]
	return value, ok
}

func mockLookup3(name string) (string, bool) {
	data := map[string]string{
		"PORT": "a8080",
	}
	value, ok := data[name]
	return value, ok
}

type Embedded struct {
	URL  string
	Port int
}

type Config1 struct {
	Host string
	Port int
	Embedded
}

type Config2 struct {
	host     string
	Prt      int    `env:"PORT"`
	URL      string `env:"-"`
	Password string `env:",secret"`
}

type Config3 struct {
	Embedded
}

func TestLoader_Load(t *testing.T) {
	l := NewWithLookup("", mockLookup, nil)

	var cfg Config1
	err := l.Load(&cfg)
	if assert.Nil(t, err) {
		assert.Equal(t, "localhost", cfg.Host)
		assert.Equal(t, 8080, cfg.Port)
		assert.Equal(t, "http://example.com", cfg.URL)
	}

	err = l.Load(cfg)
	assert.Equal(t, ErrStructPointer, err)
	var cfg1 *Config1
	err = l.Load(cfg1)
	assert.Equal(t, ErrNilPointer, err)

	logger := &myLogger{}
	l = NewWithLookup("", mockLookup, logger.Log)
	var cfg2 Config2
	err = l.Load(&cfg2)
	if assert.Nil(t, err) {
		assert.Equal(t, "", cfg2.host)
		assert.Equal(t, 8080, cfg2.Prt)
		assert.Equal(t, "", cfg2.URL)
		assert.Equal(t, "xyz", cfg2.Password)
		assert.Equal(t, []string{`set Prt with $PORT="8080"`, `set Password with $PASSWORD="***"`}, logger.logs)
	}

	var cfg3 Config1
	l = NewWithLookup("", mockLookup3, nil)
	err = l.Load(&cfg3)
	assert.NotNil(t, err)

	var cfg4 Config3
	l = NewWithLookup("", mockLookup3, nil)
	err = l.Load(&cfg4)
	assert.NotNil(t, err)
}

func TestNew(t *testing.T) {
	l := New("T_", mockLog)
	assert.Equal(t, "T_", l.prefix)
}

func TestNewWithLookup(t *testing.T) {
	l := NewWithLookup("T_", mockLookup, mockLog)
	assert.Equal(t, "T_", l.prefix)
}

func TestLoad(t *testing.T) {
	var cfg Config1
	oldLookup := loader.lookup
	loader.lookup = mockLookup2
	oldLog := loader.log
	loader.log = nil
	err := Load(&cfg)
	if assert.Nil(t, err) {
		assert.Equal(t, "localhost", cfg.Host)
		assert.Equal(t, 8080, cfg.Port)
		assert.Equal(t, "http://example.com", cfg.URL)
	}
	loader.lookup = oldLookup
	loader.log = oldLog
}
