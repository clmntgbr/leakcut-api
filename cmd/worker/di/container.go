package di

import (
	"log"
	"time"

	videocommand "go-api/internal/application/command/video"
	"go-api/internal/application/event/dedup"
	eventuser "go-api/internal/application/event/user"
	eventvideo "go-api/internal/application/event/video"
	"go-api/internal/application/registry"
	domainuser "go-api/internal/domain/user"
	domainvideo "go-api/internal/domain/video"
	"go-api/internal/infrastructure/centrifugo"
	"go-api/internal/infrastructure/config"
	"go-api/internal/infrastructure/messaging/rabbitmq"
	"go-api/internal/infrastructure/notification"
	"go-api/internal/infrastructure/persistence/outbox"
	"go-api/internal/infrastructure/persistence/processed"
	"go-api/internal/infrastructure/persistence/write"
	"go-api/internal/infrastructure/remote"
	"go-api/internal/infrastructure/storage"

	"gorm.io/gorm"
)

type Container struct {
	Relay                 *outbox.Relay
	Consumer              *rabbitmq.Consumer
	Conn                  *rabbitmq.Connection
	ExpireStaleUploads    *videocommand.ExpireStaleUploadsHandler
	ExpireUploadsInterval time.Duration
}

func NewContainer(db *gorm.DB, env *config.Config) *Container {
	topology := rabbitmq.DefaultTopology(
		env.RabbitMQExchange,
		env.RabbitMQQueue,
		env.RabbitMQRoutingKey,
		env.RabbitMQRetryTTLMS,
	)

	conn, err := rabbitmq.Connect(env.RabbitMQURL, topology)
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %v", err)
	}

	publisher := rabbitmq.NewPublisher(conn, env.RabbitMQExchange)
	outboxRepo := outbox.NewRepository(db)
	relay := outbox.NewRelay(outboxRepo, publisher, env.OutboxPollInterval, 50)

	dedupRepo := processed.NewRepository(db)
	notifier := notification.NewLogNotifier()
	realtimePublisher := centrifugo.NewPublisher(env)
	publishUserRealtime := eventuser.NewPublishRealtimeHandler(realtimePublisher)
	publishVideoRealtime := eventvideo.NewPublishRealtimeHandler(realtimePublisher)
	reg := registry.NewHandlerRegistry()

	reg.Register(domainuser.EventTypeUserCreated, dedup.With(
		dedupRepo,
		"user_created",
		eventuser.NewUserCreatedHandler().Handle,
	))
	reg.Register(domainuser.EventTypeUserCreated, dedup.With(
		dedupRepo,
		"notify_user_on_created",
		eventuser.NewNotifyUserOnCreatedHandler(notifier).Handle,
	))
	reg.Register(domainuser.EventTypeUserCreated, dedup.With(
		dedupRepo,
		"publish_user_created_realtime",
		publishUserRealtime.OnCreated,
	))
	reg.Register(domainuser.EventTypeUserUpdated, dedup.With(
		dedupRepo,
		"user_updated",
		eventuser.NewUserUpdatedHandler().Handle,
	))
	reg.Register(domainuser.EventTypeUserUpdated, dedup.With(
		dedupRepo,
		"publish_user_updated_realtime",
		publishUserRealtime.OnUpdated,
	))
	reg.Register(domainuser.EventTypeUserDeleted, dedup.With(
		dedupRepo,
		"user_deleted",
		eventuser.NewUserDeletedHandler().Handle,
	))
	reg.Register(domainuser.EventTypeUserDeleted, dedup.With(
		dedupRepo,
		"publish_user_deleted_realtime",
		publishUserRealtime.OnDeleted,
	))

	videoWriteRepo := write.NewVideoWriteRepository(db)
	jobWriteRepo := write.NewJobWriteRepository(db)
	minioStorage, err := storage.NewMinIOStorage(env)
	if err != nil {
		log.Fatalf("failed to create storage client: %v", err)
	}

	confirmUploadHandler := videocommand.NewConfirmUploadHandler(videoWriteRepo, jobWriteRepo, outboxRepo, env.VideoMaxSizeBytes)
	ingestRemoteHandler := videocommand.NewIngestRemoteHandler(
		videoWriteRepo,
		remote.NewFetcher(env.VideoIngestAllowedHosts, env.Environment == "development"),
		minioStorage,
		confirmUploadHandler,
		env.VideoMaxSizeBytes,
	)

	reg.Register(domainvideo.EventTypeVideoIngestRequested, dedup.With(
		dedupRepo,
		"ingest_remote_video",
		eventvideo.NewIngestRemoteOnRequestedHandler(ingestRemoteHandler).Handle,
	))
	reg.Register(domainvideo.EventTypeVideoCreated, dedup.With(
		dedupRepo,
		"publish_video_created_realtime",
		publishVideoRealtime.OnCreated,
	))
	reg.Register(domainvideo.EventTypeVideoUploaded, dedup.With(
		dedupRepo,
		"publish_video_uploaded_realtime",
		publishVideoRealtime.OnUploaded,
	))
	reg.Register(domainvideo.EventTypeVideoExtracting, dedup.With(
		dedupRepo,
		"publish_video_extracting_realtime",
		publishVideoRealtime.OnExtracting,
	))
	reg.Register(domainvideo.EventTypeVideoFramesExtracted, dedup.With(
		dedupRepo,
		"publish_video_frames_extracted_realtime",
		publishVideoRealtime.OnFramesExtracted,
	))
	reg.Register(domainvideo.EventTypeVideoFrameExtractionFailed, dedup.With(
		dedupRepo,
		"publish_video_extraction_failed_realtime",
		publishVideoRealtime.OnExtractionFailed,
	))

	consumer := rabbitmq.NewConsumer(conn, reg, env.WorkerConcurrency, env.WorkerMaxRetries)

	return &Container{
		Relay:                 relay,
		Consumer:              consumer,
		Conn:                  conn,
		ExpireStaleUploads:    videocommand.NewExpireStaleUploadsHandler(videoWriteRepo, outboxRepo, env.UploadURLTTL),
		ExpireUploadsInterval: env.ExpireUploadsInterval,
	}
}
