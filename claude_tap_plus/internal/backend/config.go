package backend

import "fmt"

type Config struct {
	Host   string
	Port   int
	DBPath string
}

func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func DefaultConfig() Config {
	return Config{
		Host:   "127.0.0.1",
		Port:   8080,
		DBPath: "backend.db",
	}
}
