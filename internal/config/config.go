package config

type Config struct {
	IdentityProvider IdentityProvider
	ServiceProvider  ServiceProvider
	JSONRepo         JSONRepo
}

type IdentityProvider struct {
	Port int
}

type ServiceProvider struct {
	Port int
}

type JSONRepo struct {
	Path string
}

func New() (*Config, error) {
	return &Config{
		IdentityProvider: IdentityProvider{
			Port: 8124,
		},
		ServiceProvider: ServiceProvider{
			Port: 8123,
		},
		JSONRepo: JSONRepo{
			Path: "data/repo.json",
		},
	}, nil
}
