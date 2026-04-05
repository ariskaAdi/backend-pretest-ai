package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	R2       R2Config
	Genkit   GenkitConfig
	JWT      JWTConfig
	Mail     MailerConfig
}

type AppConfig struct {
	Port              string
	Env               string
	LynkWebhookSecret string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type R2Config struct {
	AccountID  string
	AccessKey  string
	SecretKey  string
	BucketName string
	PublicURL  string
}

type GenkitConfig struct {
	BaseURL        string
	TimeoutSeconds int
}

type JWTConfig struct {
	Secret      string
	ExpireHours int
}

type MailerConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

var Cfg *Config

func Load() {
	// Load .env if exist (it save to skip,)
	if err := godotenv.Load(); err != nil {
		log.Println("[config] .env file not found, using system env")
	}

	Cfg = &Config{
		App: AppConfig{
			Port:              getEnv("APP_PORT", "8080"),
			Env:               getEnv("APP_ENV", "development"),
			LynkWebhookSecret: getEnv("LYNK_WEBHOOK_SECRET", "secret"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "pretestai"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		R2: R2Config{
			AccountID:  mustGetEnv("R2_ACCOUNT_ID"),
			AccessKey:  mustGetEnv("R2_ACCESS_KEY"),
			SecretKey:  mustGetEnv("R2_SECRET_KEY"),
			BucketName: mustGetEnv("R2_BUCKET_NAME"),
			PublicURL:  mustGetEnv("R2_PUBLIC_URL"),
		},
		Genkit: GenkitConfig{
			BaseURL:        getEnv("GENKIT_URL", "http://localhost:3400"),
			TimeoutSeconds: getEnvInt("GENKIT_TIMEOUT_SECONDS", 60),
		},
		JWT: JWTConfig{
			Secret:      mustGetEnv("JWT_SECRET"),
			ExpireHours: getEnvInt("JWT_EXPIRE_HOURS", 24),
		},
		Mail: MailerConfig{
			Host:     mustGetEnv("MAIL_HOST"),
			Port:     mustGetEnv("MAIL_PORT"),
			Username: mustGetEnv("MAIL_USERNAME"),
			Password: mustGetEnv("MAIL_PASSWORD"),
			From:     mustGetEnv("MAIL_FROM"),
		},
	}

	log.Println("[config] configuration loaded successfully")
}

// getEnv returns env value or fallback default
func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// mustGetEnv panics if env not set — untuk required vars
func mustGetEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("[config] required env variable %q is not set", key)
	}
	return val
}

// getEnvInt returns env value as int or fallback
func getEnvInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		log.Printf("[config] invalid int value for %q, using default %d", key, fallback)
		return fallback
	}
	return n
}