package config

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

type RedisConfig struct {
	Host string
	Port int
}

type NATSConfig struct {
	Host string
	Port int
}

type Config struct {
	Server   map[string]string
	Database map[string]string
	Redis    map[string]string
	NATS     map[string]string
	VRChat   map[string]string
	Discord  map[string]string
}
