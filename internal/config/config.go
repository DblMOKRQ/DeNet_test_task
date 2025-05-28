package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Storage `yaml:"storage" env-required:"true"`
	Rest    `yaml:"rest" env-required:"true"`
	JWT     `yaml:"jwt" env-required:"true"`
}

type Storage struct {
	User     string `yaml:"user" env-required:"true"`
	Password string `yaml:"password" env-required:"true"`
	Host     string `yaml:"host" env-required:"true"`
	Port     string `yaml:"port" env-required:"true"`
	DBName   string `yaml:"dbname" env-required:"true"`
	Sslmode  string `yaml:"sslmode" env-default:"false"`
}
type Rest struct {
	Host string `yaml:"host" env-required:"true"`
	Port string `yaml:"port" env-required:"true"`
}
type JWT struct {
	SecretKey     string        `yaml:"secretkey" env-required:"true"`
	TokenDuration time.Duration `yaml:"tokenduration" env-required:"true"`
}

// MustLoad загружает конфигурацию из файла YAML.
// Паникует при возникновении ошибок загрузки или парсинга.
func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "../config/config.yaml"
	}
	file, err := os.Open(configPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	decoder := yaml.NewDecoder(file)
	config := &Config{}
	err = decoder.Decode(config)
	if err != nil {
		panic(err)
	}

	return config
}
