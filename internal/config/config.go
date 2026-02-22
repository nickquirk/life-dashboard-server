package config

import (
	"fmt"
	"log"
	"os"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type AppConfig struct {
	App struct {
		Name string
	}
}

type Config interface {
	LoadJson(path string) error
	LoadYaml(path string) error
	GetAsString(key string) string
}

type koanfConfig struct {
	conf *koanf.Koanf
}

func NewConfig() Config {
	return &koanfConfig{}
}

func (k *koanfConfig) LoadJson(path string) error {
	// delimit by .
	k.conf = koanf.New(".")
	f := file.Provider(path)

	err := k.conf.Load(f, json.Parser())
	if err != nil {
		return err
	}
	// poss merge markers here
	return nil
}

func (k *koanfConfig) LoadYaml(path string) error {
	k.conf = koanf.New(".")
	f := file.Provider(path)

	err := k.conf.Load(f, yaml.Parser())
	if err != nil {
		return err
	}
	return nil
}

func (k *koanfConfig) GetAsString(key string) string {
	return k.conf.String(key)
}

func LoadConfig() Config {
	// Determine Path
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.dev.yaml"
	}

	// Initialize
	k := &koanfConfig{
		conf: koanf.New("."),
	}

	// Check if the file exists before attempting to load it
	if _, err := os.Stat(configPath); err == nil {
		f := file.Provider(configPath)
		if err := k.conf.Load(f, yaml.Parser()); err != nil {
			panic(fmt.Sprintf("Failed to load config file at %s: %v", configPath, err))
		}

		// Validate
		if k.GetAsString("app.name") == "" {
			panic("Missing app.name in the config file")
		}
	} else {
		// File does not exist.
		// If in production, log a warning and rely on environment variables.
		if os.Getenv("ENV") == "prod" {
			log.Printf("Warning: Config file '%s' not found. Relying on environment variables.", configPath)
		} else {
			// If we are developing locally, failing fast is still helpful
			panic(fmt.Sprintf("Config file not found at %s. Please create one.", configPath))
		}
	}

	return k
}
