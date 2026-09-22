package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL           string
	ClerkWebhookSecret    string
	Port                  string
	Environment           string
	ClerkSecretKey        string
	ClerkFrontendAPI      string
	CORSAllowedOrigins    []string
	CORSAllowCredentials  bool
	CORSAllowMethods      []string
	CORSAllowHeaders      []string
	CORSMaxAge            int
	RateLimitMax          int
	RabbitMQURL           string
	RabbitMQExchange      string
	RabbitMQQueue         string
	RabbitMQRoutingKey    string
	RabbitMQRetryTTLMS    int
	WorkerMaxRetries      int
	OutboxPollInterval    time.Duration
	WorkerConcurrency     int
	CentrifugoURL         string
	CentrifugoAPIKey      string
	CentrifugoTokenSecret string
	CentrifugoPublicWSURL string

	StorageEndpoint         string
	StorageInternalEndpoint string
	StorageRegion           string
	StorageAccessKey        string
	StorageSecretKey        string
	StorageBucket           string
	StorageUsePathStyle     bool
	MinIOWebhookSecret      string
	VideoWebhookSecret      string
	VideoIngestAllowedHosts []string
	UploadURLTTL            time.Duration
	FrameQueue              string
	FrameRoutingKey         string
	FrameConcurrency        int
	FrameExtractionTimeout  time.Duration
	FrameMaxWidthPx         int
	ExpireUploadsInterval   time.Duration
	VideoMaxSizeBytes       int64
	OCRQueue                string
	OCRRoutingKey           string
	OCRMinConfidence        float64
	OCRLang                 string
	ClassifyQueue           string
	ClassifyRoutingKey      string
	ClassifyConcurrency     int
	ClassifyEngineTimeout   time.Duration
	ClassifyTimeout         time.Duration
	ClassifyThreshold       float64
	AIGatewayURL            string
	AIGatewayAPIKey         string
	JevModel                string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using environment variables")
	}

	return &Config{
		DatabaseURL:             getEnv("DATABASE_URL"),
		ClerkWebhookSecret:      getEnv("CLERK_WEBHOOK_SECRET"),
		Port:                    getEnv("PORT"),
		Environment:             getEnv("GO_ENV"),
		ClerkSecretKey:          getEnv("CLERK_SECRET_KEY"),
		ClerkFrontendAPI:        getEnv("CLERK_FRONTEND_API"),
		CORSAllowedOrigins:      strings.Split(getEnv("CORS_ALLOWED_ORIGINS"), ","),
		CORSAllowCredentials:    getEnvBool("CORS_ALLOW_CREDENTIALS"),
		CORSAllowMethods:        strings.Split(getEnv("CORS_ALLOW_METHODS"), ","),
		CORSAllowHeaders:        strings.Split(getEnv("CORS_ALLOW_HEADERS"), ","),
		CORSMaxAge:              getEnvInt("CORS_MAX_AGE"),
		RateLimitMax:            getEnvInt("RATE_LIMIT_MAX"),
		RabbitMQURL:             getEnv("RABBITMQ_URL"),
		RabbitMQExchange:        getEnvOrDefault("RABBITMQ_EXCHANGE", "domain.events"),
		RabbitMQQueue:           getEnvOrDefault("RABBITMQ_QUEUE", "domain.events"),
		RabbitMQRoutingKey:      getEnvOrDefault("RABBITMQ_ROUTING_KEY", "user.#,video.#"),
		RabbitMQRetryTTLMS:      getEnvIntOrDefault("RABBITMQ_RETRY_TTL_MS", 30000),
		WorkerMaxRetries:        getEnvIntOrDefault("WORKER_MAX_RETRIES", 3),
		OutboxPollInterval:      getEnvDuration("OUTBOX_POLL_INTERVAL", 2*time.Second),
		WorkerConcurrency:       getEnvIntOrDefault("WORKER_CONCURRENCY", 4),
		CentrifugoURL:           getEnv("CENTRIFUGO_URL"),
		CentrifugoAPIKey:        getEnv("CENTRIFUGO_API_KEY"),
		CentrifugoTokenSecret:   getEnv("CENTRIFUGO_TOKEN_SECRET"),
		CentrifugoPublicWSURL:   getEnvOrDefault("CENTRIFUGO_PUBLIC_WS_URL", ""),
		StorageEndpoint:         getEnv("STORAGE_ENDPOINT"),
		StorageInternalEndpoint: getEnvOrDefault("STORAGE_INTERNAL_ENDPOINT", ""),
		StorageRegion:           getEnvOrDefault("STORAGE_REGION", "us-east-1"),
		StorageAccessKey:        getEnv("STORAGE_ACCESS_KEY"),
		StorageSecretKey:        getEnv("STORAGE_SECRET_KEY"),
		StorageBucket:           getEnv("STORAGE_BUCKET"),
		StorageUsePathStyle:     getEnvBoolOrDefault("STORAGE_USE_PATH_STYLE", true),
		MinIOWebhookSecret:      getEnvOrDefault("MINIO_WEBHOOK_SECRET", ""),
		VideoWebhookSecret:      getEnvOrDefault("VIDEO_WEBHOOK_SECRET", ""),
		VideoIngestAllowedHosts: splitCSV(getEnvOrDefault("VIDEO_INGEST_ALLOWED_HOSTS", "")),
		UploadURLTTL:            getEnvDuration("UPLOAD_URL_TTL", 15*time.Minute),
		FrameQueue:              getEnvOrDefault("FRAME_QUEUE", "frame"),
		FrameRoutingKey:         getEnvOrDefault("FRAME_ROUTING_KEY", "video.uploaded.v1"),
		FrameConcurrency:        getEnvIntOrDefault("FRAME_CONCURRENCY", 2),
		FrameExtractionTimeout:  getEnvDuration("FRAME_EXTRACTION_TIMEOUT", 10*time.Minute),
		FrameMaxWidthPx:         getEnvIntOrDefault("FRAME_MAX_WIDTH_PX", 1280),
		ExpireUploadsInterval:   getEnvDuration("EXPIRE_UPLOADS_INTERVAL", time.Minute),
		VideoMaxSizeBytes:       getEnvInt64OrDefault("VIDEO_MAX_SIZE_BYTES", 2*1024*1024*1024),
		OCRQueue:                getEnvOrDefault("OCR_QUEUE", "ocr"),
		OCRRoutingKey:           getEnvOrDefault("OCR_ROUTING_KEY", "video.ocr_frame_requested.v1"),
		OCRMinConfidence:        getEnvFloatOrDefault("OCR_MIN_CONFIDENCE", 0.5),
		OCRLang:                 getEnvOrDefault("OCR_LANG", "fr+en"),
		ClassifyQueue:           getEnvOrDefault("CLASSIFY_QUEUE", "classify"),
		ClassifyRoutingKey:      getEnvOrDefault("CLASSIFY_ROUTING_KEY", "video.frames_ocr_completed.v1"),
		ClassifyConcurrency:     getEnvIntOrDefault("CLASSIFY_CONCURRENCY", 4),
		ClassifyEngineTimeout:   getEnvDuration("CLASSIFY_ENGINE_TIMEOUT", 30*time.Second),
		ClassifyTimeout:         getEnvDuration("CLASSIFY_TIMEOUT", 10*time.Minute),
		ClassifyThreshold:       getEnvFloatOrDefault("CLASSIFY_THRESHOLD", 0.7),
		AIGatewayURL:            getEnvOrDefault("AI_GATEWAY_URL", "https://ai-gateway.vercel.sh"),
		AIGatewayAPIKey:         firstNonEmpty(os.Getenv("AI_GATEWAY_API_KEY"), os.Getenv("JEV_API_KEY")),
		JevModel:                getEnvOrDefault("JEV_MODEL", "typesafe-ai/jev"),
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func getEnv(key string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	log.Panicf("required environment variable %s is not set", key)
	return ""
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string) bool {
	value := os.Getenv(key)
	if value == "" {
		return false
	}

	return value == "true"
}

func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value == "true"
}

func getEnvInt(key string) int {
	value := os.Getenv(key)
	if value == "" {
		log.Panicf("required environment variable %s is not set", key)
		return 0
	}

	parsedValue, err := strconv.Atoi(value)
	if err != nil {
		log.Panicf("invalid integer for %s: %q", key, value)
		return 0
	}

	return parsedValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsedValue, err := strconv.Atoi(value)
	if err != nil {
		log.Panicf("invalid integer for %s: %q", key, value)
		return 0
	}
	return parsedValue
}

func getEnvInt64OrDefault(key string, defaultValue int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsedValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		log.Panicf("invalid integer for %s: %q", key, value)
		return 0
	}
	return parsedValue
}

func getEnvFloatOrDefault(key string, defaultValue float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsedValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		log.Panicf("invalid float for %s: %q", key, value)
		return 0
	}
	return parsedValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		log.Panicf("invalid duration for %s: %q", key, value)
		return 0
	}
	return parsed
}
