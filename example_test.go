package env_test

import (
	"fmt"
	"github.com/qiangxue/go-env"
	"os"
)

type Config struct {
	Host string
	Port int
}

func Example_one() {
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
