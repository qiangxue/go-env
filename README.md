# go-env

[![GoDoc](https://godoc.org/github.com/qiangxue/go-env?status.png)](http://godoc.org/github.com/qiangxue/go-env)
[![Build Status](https://travis-ci.org/qiangxue/go-env.svg?branch=master)](https://travis-ci.org/qiangxue/go-env)
[![Coverage Status](https://coveralls.io/repos/github/qiangxue/go-env/badge.svg?branch=master)](https://coveralls.io/github/qiangxue/go-env?branch=master)
[![Go Report](https://goreportcard.com/badge/github.com/qiangxue/go-env)](https://goreportcard.com/report/github.com/qiangxue/go-env)

## Description

go-env is a Go library that can populate a struct with environment variable values. A common use of go-env is
to load a configuration struct with values set in the environment variables.

## Requirements

Go 1.13 or above.


## Getting Started

### Installation

Run the following command to install the package:

```
go get github.com/qiangxue/go-env
```

### Loading From Environment Variables

The easiest way of using go-env is to call `env.Load()`, like the following:

```go
package main

import (
	"fmt"
	"github.com/qiangxue/go-env"
	"os"
)

type Config struct {
	Host string
	Port int
}

func main() {
	_ = os.Setenv("APP_HOST", "127.0.0.1")
	_ = os.Setenv("APP_PORT", "8080")

	var cfg Config
	if err := env.Load(&cfg); err != nil {
		panic(err)
	}
	fmt.Println(cfg.Host)
	fmt.Println(cfg.Port)
	// Output:
	// 127.0.0.1
	// 8080
}
```

### Environment Variable Names

When go-env populates a struct from environment variables, it uses the following rules to match
a struct field with an environment variable:
- Only public struct fields will be populated
- If the field has an `env` tag, use the tag value as the name, unless the tag value is `-` in which case it means
  the field should NOT be populated.
- If the field has no `env` tag, turn the field name into UPPER_SNAKE_CASE format and use that as the name. For example,
  a field name `HostName` will be turned into `HOST_NAME`, and `MyURL` becomes `MY_URL`.
- Names are prefixed with the specified prefix when they are used to look up in the environment variables.

By default, prefix `APP_` will be used. You can customize the prefix by using `env.New()` to create
a customized loader. For example,

```go
package main

import (
	"fmt"
	"github.com/qiangxue/go-env"
	"log"
	"os"
)

type Config struct {
	Host     string `env:"ES_HOST"`
	Port     int    `env:"ES_PORT"`
	Password string `env:"ES_PASSWORD,secret"`
}

func main() {
	_ = os.Setenv("API_ES_HOST", "127.0.0.1")
	_ = os.Setenv("API_ES_PORT", "8080")
	_ = os.Setenv("API_ES_PASSWORD", "test")

	var cfg Config
	loader := env.New("API_", log.Printf)
	if err := loader.Load(&cfg); err != nil {
		panic(err)
	}
	fmt.Println(cfg.Host)
	fmt.Println(cfg.Port)
	fmt.Println(cfg.Password)
	// Output:
	// 127.0.0.1
	// 8080
	// test
}
```

In the above code, the `Password` field is tagged as `secret`. The log function respects this flag by masking
the field value when logging it in order not to reveal sensitive information.

By setting the prefix to an empty string, you can disable the name prefix completely.


### Data Parsing Rules

Because the values of environment variables are strings, if the corresponding struct fields are of different types,
go-env will convert the string values into appropriate types before assigning them to the struct fields.

- If a struct contains embedded structs, the fields of the embedded structs will be populated like they are directly
under the containing struct.

- If a struct field type implements `env.Setter`, `env.TextMarshaler`, or `env.BinaryMarshaler` interface,
the corresponding interface method will be used to load a string value into the field.

- If a struct field is of a primary type, such as `int`, `string`, `bool`, etc., a string value will be parsed
accordingly and assigned to the field. For example, the string value `TRUE` can be parsed correctly into a
boolean `true` value, while `TrUE` will cause a parsing error.

- If a struct field is of a complex type, such as map, slice, struct, the string value will be treated as a JSON
string, and `json.Unmarshal()` will be called to populate the struct field from the JSON string.
