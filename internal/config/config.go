package config

type Config struct {
	IdentityProvider IdentityProvider
	ServiceProvider  ServiceProvider
}

type IdentityProvider struct {
	Port int
}

type ServiceProvider struct {
	Port int
}

func New() (*Config, error) {
	return &Config{
		IdentityProvider: IdentityProvider{
			Port: 8124,
		},
		ServiceProvider: ServiceProvider{
			Port: 8123,
		},
	}, nil
}
