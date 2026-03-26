package config

type Config struct {
	Name string
}

func Load() (*Config, error) {
	return &Config{
		Name: "hello",
	}, nil
}
