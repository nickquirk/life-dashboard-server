package helper

import cfg "github.com/nickquirk/life-dashboard-server/internal/config"

type app struct {
	config cfg.Config
}

type App interface {
	GetConfig() cfg.Config
}

func NewApp(configPath string) App {

	c := cfg.NewConfig()
	c.LoadYaml(configPath)

	if c.GetAsString("app.name") == "" {
		panic("Missing app.name in the config file")
	}

	return &app{
		config: c,
	}
}

func (a *app) GetConfig() cfg.Config {
	return a.config
}
