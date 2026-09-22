package di

import (
	"log"

	videocommand "go-api/internal/application/command/video"
	"go-api/internal/application/event/dedup"
	eventvideo "go-api/internal/application/event/video"
	"go-api/internal/application/registry"
	domainvideo "go-api/internal/domain/video"
	"go-api/internal/infrastructure/config"
	"go-api/internal/infrastructure/messaging/rabbitmq"
	"go-api/internal/infrastructure/persistence/outbox"
	"go-api/internal/infrastructure/persistence/processed"
	"go-api/internal/infrastructure/persistence/write"
	"go-api/internal/infrastructure/storage"
	infraVideo "go-api/internal/infrastructure/video"

	"gorm.io/gorm"
)

type Container struct {
	Consumer *rabbitmq.Consumer
	Conn     *rabbitmq.Connection
}

func NewContainer(db *gorm.DB, env *config.Config) *Container {
	topology := rabbitmq.DefaultTopology(
		env.RabbitMQExchange,
		env.FrameQueue,
		env.FrameRoutingKey,
		env.RabbitMQRetryTTLMS,
	)

	conn, err := rabbitmq.Connect(env.RabbitMQURL, topology)
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %v", err)
	}

	outboxRepo := outbox.NewRepository(db)
	dedupRepo := processed.NewRepository(db)
	videoWriteRepo := write.NewVideoWriteRepository(db)
	scanJobWriteRepo := write.NewScanJobWriteRepository(db)
	frameWriteRepo := write.NewFrameWriteRepository(db)

	minioStorage, err := storage.NewMinIOStorage(env)
	if err != nil {
		log.Fatalf("failed to create storage client: %v", err)
	}

	extractHandler := videocommand.NewExtractFramesHandler(
		videoWriteRepo,
		scanJobWriteRepo,
		frameWriteRepo,
		outboxRepo,
		minioStorage,
		infraVideo.NewFrameExtractor(),
		env.FrameExtractionTimeout,
	)

	reg := registry.NewHandlerRegistry()
	reg.Register(domainvideo.EventTypeVideoUploaded, dedup.With(
		dedupRepo,
		"extract_frames_on_video_uploaded",
		eventvideo.NewExtractFramesOnUploadedHandler(extractHandler).Handle,
	))

	consumer := rabbitmq.NewConsumer(conn, reg, env.FrameConcurrency, env.WorkerMaxRetries)

	return &Container{
		Consumer: consumer,
		Conn:     conn,
	}
}
