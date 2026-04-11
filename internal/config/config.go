package config

import (
	"os"
	"strconv"
)

type Config struct {
	LibraryPath string
	DataPath    string
	BackupPath  string
	Host        string
	Port        int
}

func Load() *Config {
	return &Config{
		LibraryPath: getEnv("FOLIO_LIBRARY_PATH", "./library"),
		DataPath:    getEnv("FOLIO_DATA_PATH", "./data"),
		BackupPath:  getEnv("FOLIO_BACKUP_PATH", "./backup"),
		Host:        getEnv("FOLIO_HOST", "0.0.0.0"),
		Port:        getEnvInt("FOLIO_PORT", 3000),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// Address returns the full server address string.
func (c *Config) Address() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}
