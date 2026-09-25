package di

import (
	"log"

	videocommand "go-api/internal/application/command/video"
	"go-api/internal/application/event/dedup"
	eventvideo "go-api/internal/application/event/video"
	"go-api/internal/application/registry"
	domainvideo "go-api/internal/domain/video"
	"go-api/internal/infrastructure/classify"
	"go-api/internal/infrastructure/config"
	"go-api/internal/infrastructure/messaging/rabbitmq"
	"go-api/internal/infrastructure/persistence/outbox"
	"go-api/internal/infrastructure/persistence/processed"
	"go-api/internal/infrastructure/persistence/write"

	"gorm.io/gorm"
)

type Container struct {
	Consumer *rabbitmq.Consumer
	Conn     *rabbitmq.Connection
}

func NewContainer(db *gorm.DB, env *config.Config) *Container {
	topology := rabbitmq.DefaultTopology(
		env.RabbitMQExchange,
		env.ClassifyQueue,
		env.ClassifyRoutingKey,
		env.RabbitMQRetryTTLMS,
	)

	conn, err := rabbitmq.Connect(env.RabbitMQURL, topology)
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %v", err)
	}

	outboxRepo := outbox.NewRepository(db)
	dedupRepo := processed.NewRepository(db)

	log.Printf("classify engine=%s", env.ClassifyEngine)

	classifyHandler := videocommand.NewClassifyFramesHandler(
		write.NewVideoWriteRepository(db),
		write.NewJobWriteRepository(db),
		write.NewFrameWriteRepository(db),
		write.NewOCRWriteRepository(db),
		write.NewClassificationWriteRepository(db),
		outboxRepo,
		classify.NewClassifier(
			env.ClassifyEngine,
			env.AIGatewayURL,
			env.AIGatewayAPIKey,
			env.JevModel,
			env.ClassifyEngineTimeout,
			env.ClassifyConcurrency,
		),
		env.ClassifyThreshold,
		env.ClassifyTimeout,
	)

	reg := registry.NewHandlerRegistry()
	reg.Register(domainvideo.EventTypeVideoFramesOCRCompleted, dedup.With(
		dedupRepo,
		"classify_frames_on_ocr_completed",
		eventvideo.NewClassifyFramesOnOCRCompletedHandler(classifyHandler).Handle,
	))

	consumer := rabbitmq.NewConsumer(conn, reg, env.ClassifyConcurrency, env.WorkerMaxRetries)

	return &Container{
		Consumer: consumer,
		Conn:     conn,
	}
}
