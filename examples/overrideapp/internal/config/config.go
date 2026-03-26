package config

type Config struct {
	Name string
}

func Load() (*Config, error) {
	return &Config{
		Name: "overrideapp",
	}, nil
}
