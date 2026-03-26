package config

type Config struct {
	Addr string
}

func Load() (*Config, error) {
	return &Config{
		Addr: ":8081",
	}, nil
}
