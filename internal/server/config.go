package server

type Config struct {
	Port                int `env:"PORT" envDefault:"8080"`
	SyncIntervalMinutes int `env:"SYNC_INTERVAL_MINUTES" envDefault:"1440"`

	// LTA API
	LTAAccessKey string `env:"LTA_ACCESS_KEY,notEmpty"`
	LTAAPIHost   string `env:"LTA_API_HOST,notEmpty"`

	// Database
	DBDatabase string `env:"DB_DATABASE,notEmpty"`
	DBPassword string `env:"DB_PASSWORD,notEmpty"`
	DBUsername string `env:"DB_USERNAME,notEmpty"`
	DBPort     string `env:"DB_PORT,notEmpty"`
	DBHost     string `env:"DB_HOST,notEmpty"`
	DBSchema   string `env:"DB_SCHEMA,notEmpty"`
}
