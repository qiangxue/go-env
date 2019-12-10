package env_test

import (
	"fmt"
	"github.com/qiangxue/go-env"
	"log"
	"os"
)

type Config struct {
	Host     string
	Port     int
	Password string `env:",secret"`
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

func Example_two() {
	_ = os.Setenv("API_HOST", "127.0.0.1")
	_ = os.Setenv("API_PORT", "8080")
	_ = os.Setenv("API_PASSWORD", "test")

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
