package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	MongoDB    MongoDBConfig
	API        APIConfig
	Scheduler  SchedulerConfig
	Processing ProcessingConfig
	Storage    StorageConfig
	Logging    LoggingConfig
	App        AppConfig
}

type MongoDBConfig struct {
	URI                    string
	Database               string
	ReferenceDatabase      string
	MaxPoolSize            uint64
	MinPoolSize            uint64
	MaxConnIdleTime        time.Duration
	ServerSelectionTimeout time.Duration
}

type APIConfig struct {
	Endpoint      string
	Timeout       time.Duration
	RetryAttempts int
	RetryDelay    time.Duration
}

type SchedulerConfig struct {
	CronSchedule string
}

type ProcessingConfig struct {
	BatchSize  int
	MaxWorkers int
}

type StorageConfig struct {
	AFSTTLDays       int
	EnableXMLArchive bool
	ArchivePath      string
}

type LoggingConfig struct {
	Level string
	Path  string
}

type AppConfig struct {
	Env  string
	Port string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if exists (ignore error in production)
	_ = godotenv.Load()

	cfg := &Config{
		MongoDB: MongoDBConfig{
			URI:                    getEnv("MONGO_URI", "mongodb://localhost:27017"),
			Database:               getEnv("MONGO_DATABASE", "afs_db"),
			ReferenceDatabase:      getEnv("MONGO_REFERENCE_DATABASE", "master_reference"),
			MaxPoolSize:            getEnvAsUint64("MONGO_MAX_POOL_SIZE", 10),
			MinPoolSize:            getEnvAsUint64("MONGO_MIN_POOL_SIZE", 2),
			MaxConnIdleTime:        getEnvAsDuration("MONGO_MAX_CONN_IDLE_TIME", 5*time.Minute),
			ServerSelectionTimeout: getEnvAsDuration("MONGO_SERVER_SELECTION_TIMEOUT", 5*time.Second),
		},
		API: APIConfig{
			Endpoint:      getEnv("API_ENDPOINT", "http://api-receiver:3001/api/schedules"),
			Timeout:       getEnvAsDuration("API_TIMEOUT", 30*time.Second),
			RetryAttempts: getEnvAsInt("RETRY_ATTEMPTS", 3),
			RetryDelay:    getEnvAsDuration("RETRY_DELAY_MS", 60*time.Second),
		},
		Scheduler: SchedulerConfig{
			CronSchedule: getEnv("CRON_SCHEDULE", "0 0 * * *"),
		},
		Processing: ProcessingConfig{
			BatchSize:  getEnvAsInt("BATCH_SIZE", 100),
			MaxWorkers: getEnvAsInt("MAX_WORKERS", 4),
		},
		Storage: StorageConfig{
			AFSTTLDays:       getEnvAsInt("AFS_TTL_DAYS", 7),
			EnableXMLArchive: getEnv("ENABLE_XML_ARCHIVE", "true") == "true",
			ArchivePath:      getEnv("ARCHIVE_PATH", "/app/archive"),
		},
		Logging: LoggingConfig{
			Level: getEnv("LOG_LEVEL", "info"),
			Path:  getEnv("LOG_PATH", "/app/logs"),
		},
		App: AppConfig{
			Env:  getEnv("NODE_ENV", "production"),
			Port: getEnv("PORT", "3000"),
		},
	}

	// Setup logging
	setupLogging(cfg.Logging)

	log.WithFields(log.Fields{
		"env":                cfg.App.Env,
		"mongo_uri":          cfg.MongoDB.URI,
		"afs_database":       cfg.MongoDB.Database,
		"reference_database": cfg.MongoDB.ReferenceDatabase,  // ← ADD THIS
		"api":                cfg.API.Endpoint,
		"schedule":           cfg.Scheduler.CronSchedule,
	}).Info("Configuration loaded")

	return cfg, nil
}

func setupLogging(cfg LoggingConfig) {
	// Set log level
	level, err := log.ParseLevel(cfg.Level)
	if err != nil {
		level = log.InfoLevel
	}
	log.SetLevel(level)

	// Set JSON formatter for structured logging
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// Log to stdout
	log.SetOutput(os.Stdout)
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsUint64(key string, defaultValue uint64) uint64 {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseUint(valueStr, 10, 64); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := getEnv(key, "")
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	return defaultValue
}