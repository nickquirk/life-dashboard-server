package config

import (
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
