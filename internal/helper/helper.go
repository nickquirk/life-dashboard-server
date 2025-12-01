package helper

import cfg "github.com/nickquirk/life-dashboard-server/internal/config"

type app struct {
	config cfg.Config
}

type App interface {
	GetConfig() cfg.Config
}

func (a *app) GetConfig() cfg.Config {
	return a.config
}

func NewApp(configPath string) App {

	c := cfg.NewConfig()
	// Capture the error
	err := c.LoadYaml(configPath)

	// Fail immediately if the file cannot be read
	if err != nil {
		panic("Failed to load config file at " + configPath + ": " + err.Error())
	}

	if c.GetAsString("app.name") == "" {
		panic("Missing app.name in the config file")
	}

	return &app{
		config: c,
	}
}
