package config

type Config struct {
	Addr string
	Name string
}

func Load() (*Config, error) {
	return &Config{
		Addr: ":9090",
		Name: "modularapp",
	}, nil
}
