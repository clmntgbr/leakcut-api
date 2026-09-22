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
	"go-api/internal/infrastructure/ocr"
	"go-api/internal/infrastructure/persistence/outbox"
	"go-api/internal/infrastructure/persistence/processed"
	"go-api/internal/infrastructure/persistence/write"
	"go-api/internal/infrastructure/storage"

	"gorm.io/gorm"
)

type Container struct {
	Consumer *rabbitmq.Consumer
	Conn     *rabbitmq.Connection
}

func NewContainer(db *gorm.DB, env *config.Config) *Container {
	topology := rabbitmq.DefaultTopology(
		env.RabbitMQExchange,
		env.OCRQueue,
		env.OCRRoutingKey,
		env.RabbitMQRetryTTLMS,
	)

	conn, err := rabbitmq.Connect(env.RabbitMQURL, topology)
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %v", err)
	}

	outboxRepo := outbox.NewRepository(db)
	dedupRepo := processed.NewRepository(db)
	videoWriteRepo := write.NewVideoWriteRepository(db)
	jobWriteRepo := write.NewJobWriteRepository(db)
	frameWriteRepo := write.NewFrameWriteRepository(db)
	ocrWriteRepo := write.NewOCRResultWriteRepository(db)

	minioStorage, err := storage.NewMinIOStorage(env)
	if err != nil {
		log.Fatalf("failed to create storage client: %v", err)
	}

	ocrHandler := videocommand.NewOCRFramesHandler(
		videoWriteRepo,
		jobWriteRepo,
		frameWriteRepo,
		ocrWriteRepo,
		outboxRepo,
		minioStorage,
		ocr.NewClient(env.PaddleOCRURL, env.PaddleOCRTimeout),
		env.OCRBatchSize,
		env.OCRMinConfidence,
		env.OCRLang,
		env.OCRTimeout,
	)

	reg := registry.NewHandlerRegistry()
	reg.Register(domainvideo.EventTypeVideoFramesExtracted, dedup.With(
		dedupRepo,
		"ocr_frames_on_frames_extracted",
		eventvideo.NewOCRFramesOnExtractedHandler(ocrHandler).Handle,
	))

	consumer := rabbitmq.NewConsumer(conn, reg, env.OCRConcurrency, env.WorkerMaxRetries)

	return &Container{
		Consumer: consumer,
		Conn:     conn,
	}
}
