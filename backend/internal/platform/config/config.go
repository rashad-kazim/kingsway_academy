package config

import (
	"bufio"
	"os"
	"strings"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	AppEnv        string `env:"APP_ENV" envDefault:"local"`
	HTTPAddr      string `env:"HTTP_ADDR" envDefault:":8080"`
	JWTSecret     string `env:"JWT_SECRET" envDefault:"dev-only-change-me"`
	DataStore     string `env:"DATA_STORE" envDefault:"postgres"`
	RunMigrations bool   `env:"RUN_MIGRATIONS" envDefault:"true"`

	RateLimitEnabled   bool `env:"RATE_LIMIT_ENABLED" envDefault:"true"`
	RateLimitPerMinute int  `env:"RATE_LIMIT_PER_MINUTE" envDefault:"600"`

	PostgresHost     string `env:"POSTGRES_HOST" envDefault:"localhost"`
	PostgresPort     int    `env:"POSTGRES_PORT" envDefault:"5432"`
	PostgresDB       string `env:"POSTGRES_DB" envDefault:"kingsway"`
	PostgresUser     string `env:"POSTGRES_USER" envDefault:"kingsway"`
	PostgresPassword string `env:"POSTGRES_PASSWORD" envDefault:"kingsway"`

	RedisAddr   string `env:"REDIS_ADDR" envDefault:"localhost:6379"`
	RabbitURL   string `env:"RABBITMQ_URL" envDefault:"amqp://kingsway:kingsway@localhost:5672/"`
	MinIOHost   string `env:"MINIO_ENDPOINT" envDefault:"localhost:9000"`
	MinIOKey    string `env:"MINIO_ACCESS_KEY" envDefault:"kingsway"`
	MinIOSecret string `env:"MINIO_SECRET_KEY" envDefault:"kingsway-secret"`

	StandardBucket string `env:"MINIO_BUCKET_STANDARD" envDefault:"kingsway-standard"`
	SpecialBucket  string `env:"MINIO_BUCKET_SPECIAL" envDefault:"kingsway-special"`
	MinIOUseSSL    bool   `env:"MINIO_USE_SSL" envDefault:"false"`

	RetentionWorkerEnabled  bool `env:"RETENTION_WORKER_ENABLED" envDefault:"true"`
	RetentionWorkerInterval int  `env:"RETENTION_WORKER_INTERVAL_SECONDS" envDefault:"3600"`
	RetentionWorkerLimit    int  `env:"RETENTION_WORKER_LIMIT" envDefault:"100"`

	OutboxDispatcherEnabled  bool `env:"OUTBOX_DISPATCHER_ENABLED" envDefault:"true"`
	OutboxDispatcherInterval int  `env:"OUTBOX_DISPATCHER_INTERVAL_SECONDS" envDefault:"5"`
	OutboxDispatcherBatch    int  `env:"OUTBOX_DISPATCHER_BATCH" envDefault:"50"`
	OutboxMaxAttempts        int  `env:"OUTBOX_MAX_ATTEMPTS" envDefault:"10"`

	NotificationWorkersEnabled      bool `env:"NOTIFICATION_WORKERS_ENABLED" envDefault:"true"`
	PaymentReminderIntervalSeconds  int  `env:"PAYMENT_REMINDER_INTERVAL_SECONDS" envDefault:"21600"`
	PaymentReminderHorizonDays      int  `env:"PAYMENT_REMINDER_HORIZON_DAYS" envDefault:"7"`
	FileRetentionNoticeIntervalSecs int  `env:"FILE_RETENTION_NOTICE_INTERVAL_SECONDS" envDefault:"21600"`
	FileRetentionNoticeHorizonDays  int  `env:"FILE_RETENTION_NOTICE_HORIZON_DAYS" envDefault:"7"`
}

func Load() (Config, error) {
	_ = loadDotEnv(".env")
	return env.ParseAs[Config]()
}

func (c Config) Postgres() PostgresConfig {
	return PostgresConfig{
		Host:     c.PostgresHost,
		Port:     c.PostgresPort,
		Database: c.PostgresDB,
		User:     c.PostgresUser,
		Password: c.PostgresPassword,
	}
}

type PostgresConfig struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}

	return scanner.Err()
}
