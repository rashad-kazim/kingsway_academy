package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"

	"kingsway/backend/internal/academic"
	"kingsway/backend/internal/admin"
	"kingsway/backend/internal/auth"
	"kingsway/backend/internal/files"
	"kingsway/backend/internal/finance"
	"kingsway/backend/internal/httpapi"
	"kingsway/backend/internal/notification"
	"kingsway/backend/internal/platform/cache"
	"kingsway/backend/internal/platform/config"
	"kingsway/backend/internal/platform/database"
	"kingsway/backend/internal/platform/logger"
	"kingsway/backend/internal/platform/outbox"
	"kingsway/backend/internal/platform/queue"
	"kingsway/backend/internal/platform/storage"
	"kingsway/backend/internal/store"
)

type applicationStore interface {
	auth.UserStore
	academic.Store
	admin.Store
	finance.Store
	files.Store
	notification.Store
}

type runtimeDependencies struct {
	appStore         applicationStore
	financeOptions   []finance.Option
	fileOptions      []files.Option
	eventConsumer    *queue.TopicConsumer
	outboxDispatcher *outbox.Dispatcher
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	log, err := logger.New(cfg.AppEnv)
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()

	startupCtx, startupCancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer startupCancel()

	deps, cleanup := buildDependencies(startupCtx, cfg, log)
	defer cleanup()

	authService := auth.NewService(deps.appStore, cfg.JWTSecret)
	academicService := academic.NewService(deps.appStore, authService)
	financeService := finance.NewService(deps.appStore, deps.financeOptions...)
	fileService := files.NewService(deps.appStore, deps.fileOptions...)
	notificationService := notification.NewService(deps.appStore)
	adminService := admin.NewService(deps.appStore)
	workerCtx, stopWorkers := context.WithCancel(context.Background())
	defer stopWorkers()
	if cfg.RetentionWorkerEnabled && !strings.EqualFold(cfg.DataStore, "memory") {
		fileService.StartRetentionWorker(workerCtx, files.RetentionWorkerOptions{
			Interval: time.Duration(cfg.RetentionWorkerInterval) * time.Second,
			Limit:    cfg.RetentionWorkerLimit,
			Logger:   log,
		})
		log.Info(
			"retention worker started",
			zap.Int("interval_seconds", cfg.RetentionWorkerInterval),
			zap.Int("limit", cfg.RetentionWorkerLimit),
		)
	}
	if cfg.OutboxDispatcherEnabled && deps.outboxDispatcher != nil {
		deps.outboxDispatcher.Start(workerCtx, outbox.DispatcherOptions{
			Interval:    time.Duration(cfg.OutboxDispatcherInterval) * time.Second,
			BatchSize:   cfg.OutboxDispatcherBatch,
			MaxAttempts: cfg.OutboxMaxAttempts,
			Logger:      log,
		})
		log.Info("outbox dispatcher started", zap.Int("interval_seconds", cfg.OutboxDispatcherInterval))
	}
	if cfg.NotificationWorkersEnabled && !strings.EqualFold(cfg.DataStore, "memory") {
		notificationService.StartPaymentReminderWorker(workerCtx, notification.WorkerOptions{
			Interval: time.Duration(cfg.PaymentReminderIntervalSeconds) * time.Second,
			Horizon:  time.Duration(cfg.PaymentReminderHorizonDays) * 24 * time.Hour,
			Logger:   log,
		})
		notificationService.StartFileRetentionNoticeWorker(workerCtx, notification.WorkerOptions{
			Interval: time.Duration(cfg.FileRetentionNoticeIntervalSecs) * time.Second,
			Horizon:  time.Duration(cfg.FileRetentionNoticeHorizonDays) * 24 * time.Hour,
			Logger:   log,
		})
		log.Info("notification workers started")
	}
	if deps.eventConsumer != nil {
		if err := deps.eventConsumer.Start(workerCtx, notificationService.HandleEvent, log); err != nil {
			log.Fatal("rabbitmq notification consumer failed", zap.Error(err))
		}
		log.Info("rabbitmq notification consumer started")
	}

	api := httpapi.New(
		authService,
		academicService,
		financeService,
		fileService,
		notificationService,
		adminService,
		log,
		httpapi.Options{
			RateLimitEnabled:   cfg.RateLimitEnabled,
			RateLimitPerMinute: cfg.RateLimitPerMinute,
		},
	)
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("api server listening", zap.String("addr", cfg.HTTPAddr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("api server failed", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	stopWorkers()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Error("api server shutdown failed", zap.Error(err))
	}
}

func buildDependencies(ctx context.Context, cfg config.Config, log *zap.Logger) (runtimeDependencies, func()) {
	cleanupFns := make([]func(), 0)
	cleanup := func() {
		for i := len(cleanupFns) - 1; i >= 0; i-- {
			cleanupFns[i]()
		}
	}

	if strings.EqualFold(cfg.DataStore, "memory") {
		log.Info("using in-memory store")
		return runtimeDependencies{appStore: store.NewMemory()}, cleanup
	}

	if !strings.EqualFold(cfg.DataStore, "postgres") {
		log.Fatal("unsupported data store", zap.String("data_store", cfg.DataStore))
	}

	pool, err := database.Connect(ctx, database.DSN(cfg.Postgres()))
	if err != nil {
		log.Fatal("postgres connection failed", zap.Error(err))
	}
	cleanupFns = append(cleanupFns, pool.Close)

	if err := pool.Ping(ctx); err != nil {
		log.Fatal("postgres ping failed", zap.Error(err))
	}
	if cfg.RunMigrations {
		if err := database.ApplyMigrations(ctx, pool, "migrations"); err != nil {
			log.Fatal("postgres migrations failed", zap.Error(err))
		}
	}

	redisClient := cache.NewRedis(cfg.RedisAddr)
	cleanupFns = append(cleanupFns, func() { _ = redisClient.Close() })
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatal("redis ping failed", zap.Error(err))
	}

	rabbitConn, err := queue.DialRabbitMQ(cfg.RabbitURL)
	if err != nil {
		log.Fatal("rabbitmq connection failed", zap.Error(err))
	}
	cleanupFns = append(cleanupFns, func() { _ = rabbitConn.Close() })

	publisher := queue.NewTopicPublisher(rabbitConn, "kingsway.events")
	if err := publisher.EnsureExchange(ctx); err != nil {
		log.Fatal("rabbitmq exchange setup failed", zap.Error(err))
	}
	outboxDispatcher := outbox.NewDispatcher(pool, publisher)
	eventConsumer := queue.NewTopicConsumer(rabbitConn, "kingsway.events", "kingsway.notifications", []string{
		"finance.#",
		"files.retention.#",
	})

	minioClient, err := storage.NewMinIO(cfg.MinIOHost, cfg.MinIOKey, cfg.MinIOSecret, cfg.MinIOUseSSL)
	if err != nil {
		log.Fatal("minio client setup failed", zap.Error(err))
	}
	objectStore := storage.NewObjectStore(minioClient)
	if err := objectStore.EnsureBucket(ctx, cfg.StandardBucket); err != nil {
		log.Fatal("minio standard bucket setup failed", zap.Error(err))
	}
	if err := objectStore.EnsureBucket(ctx, cfg.SpecialBucket); err != nil {
		log.Fatal("minio special bucket setup failed", zap.Error(err))
	}

	financeOptions := []finance.Option{
		finance.WithCache(cache.NewJSONCache(redisClient, "kingsway:")),
	}
	fileOptions := []files.Option{
		files.WithObjectStorage(objectStore, cfg.StandardBucket, cfg.SpecialBucket),
	}

	log.Info("using postgres store with local infrastructure")
	return runtimeDependencies{
		appStore:         store.NewPostgres(pool),
		financeOptions:   financeOptions,
		fileOptions:      fileOptions,
		eventConsumer:    eventConsumer,
		outboxDispatcher: outboxDispatcher,
	}, cleanup
}
