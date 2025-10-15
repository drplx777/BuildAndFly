package config

import (
	"os"
	"strings"

	"github.com/KoNekoD/dotenv/pkg/dotenv"
	"github.com/pkg/errors"
)

type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
	ExposeHeaders    []string
}

type DBConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
}
type S3Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	BaseURL   string
}
type Config struct {
	Port      string
	JWTSecret string
	DB        DBConfig
	CORS      CORSConfig
	S3        S3Config
}

func MustLoad() *Config {
	if err := dotenv.LoadEnv(".env"); err != nil {
		panic(errors.Wrap(err, "failed to load .env"))
	}

	useSSL := getEnv("S3_USE_SSL", "true") == "true"

	return &Config{
		Port:      getEnv("PORT", "3000"),
		JWTSecret: mustGetEnv("JWT_SECRET"),
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			Name:     mustGetEnv("DB_NAME"),
			User:     mustGetEnv("DB_USER"),
			Password: mustGetEnv("DB_PASSWORD"),
		},
		CORS: CORSConfig{
			AllowOrigins:     strings.Split(getEnv("ALLOW_ORIGINS", "http://localhost:3000,https://my-samovar-to-do-list.duckdns.org"), ","),
			AllowMethods:     strings.Split(getEnv("ALLOW_METHODS", "GET,POST,PUT,DELETE,OPTIONS"), ","),
			AllowHeaders:     strings.Split(getEnv("ALLOW_HEADERS", "Origin,Content-Type,Accept,Authorization"), ","),
			AllowCredentials: getEnv("ALLOW_CREDENTIALS", "true") == "true",
			ExposeHeaders:    strings.Split(getEnv("EXPOSE_HEADERS", "Authorization"), ","),
		},
		S3: S3Config{
			Endpoint:  getEnv("S3_ENDPOINT", "localhost:9000"),
			AccessKey: mustGetEnv("S3_ACCESS_KEY"),
			SecretKey: mustGetEnv("S3_SECRET_KEY"),
			Bucket:    getEnv("S3_BUCKET", "buildandfly"),
			UseSSL:    useSSL,
			BaseURL:   getEnv("S3_BASE_URL", ""), // optional
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func mustGetEnv(key string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	panic("environment variable " + key + " is required")
}

func BuildDBConnectionString(cfg DBConfig) string {
	return "postgres://" + cfg.User + ":" + cfg.Password +
		"@" + cfg.Host + ":" + cfg.Port + "/" + cfg.Name +
		"?sslmode=disable"
}
