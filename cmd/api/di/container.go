package di

import (
	"log"

	authcmd "go-api/internal/application/command/auth"
	identitycmd "go-api/internal/application/command/identity"
	usercmd "go-api/internal/application/command/user"
	videocommand "go-api/internal/application/command/video"
	queryuser "go-api/internal/application/query/user"
	queryvideo "go-api/internal/application/query/video"
	"go-api/internal/infrastructure/centrifugo"
	infraClerk "go-api/internal/infrastructure/clerk"
	"go-api/internal/infrastructure/config"
	"go-api/internal/infrastructure/persistence/outbox"
	"go-api/internal/infrastructure/persistence/read"
	"go-api/internal/infrastructure/persistence/write"
	"go-api/internal/infrastructure/storage"
	httphandler "go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/middleware"

	"gorm.io/gorm"
)

type Container struct {
	AuthenticateMiddleware   *middleware.AuthenticateMiddleware
	UserWebhookMiddleware    *middleware.UserWebhookMiddleware
	StorageWebhookMiddleware *middleware.StorageWebhookMiddleware
	VideoWebhookMiddleware   *middleware.VideoWebhookMiddleware
	UserWebhookHandler       *httphandler.UserWebhookHandler
	StorageWebhookHandler    *httphandler.StorageWebhookHandler
	VideoWebhookHandler      *httphandler.VideoWebhookHandler
	UserHandler              *httphandler.UserHandler
	VideoHandler             *httphandler.VideoHandler
	RealtimeHandler          *httphandler.RealtimeHandler
}

func NewContainer(db *gorm.DB, env *config.Config) *Container {
	jwksProvider, err := infraClerk.NewJWKSProvider(env)
	if err != nil {
		log.Fatalf("failed to create JWKS provider: %v", err)
	}

	userWriteRepo := write.NewUserWriteRepository(db)
	userReadRepo := read.NewUserReadRepository(db)
	outboxRepo := outbox.NewRepository(db)

	createUserHandler := usercmd.NewCreateUserHandler(userWriteRepo, outboxRepo)
	updateUserHandler := usercmd.NewUpdateUserHandler(userWriteRepo, outboxRepo)
	getUserByExternalIDHandler := usercmd.NewGetUserByExternalIDHandler(userWriteRepo)
	deleteUserByExternalIDHandler := usercmd.NewDeleteUserByExternalIDHandler(userWriteRepo, outboxRepo)
	validateTokenHandler := authcmd.NewValidateTokenHandler(jwksProvider, userWriteRepo)
	fetchUserHandler := identitycmd.NewFetchUserHandler(infraClerk.NewUserGateway(env.ClerkSecretKey))
	getUserByIDHandler := queryuser.NewGetUserByIDHandler(userReadRepo)

	minioStorage, err := storage.NewMinIOStorage(env)
	if err != nil {
		log.Fatalf("failed to create storage client: %v", err)
	}

	videoWriteRepo := write.NewVideoWriteRepository(db)
	videoReadRepo := read.NewVideoReadRepository(db)
	jobWriteRepo := write.NewJobWriteRepository(db)

	requestUploadURLHandler := videocommand.NewRequestUploadURLHandler(
		videoWriteRepo,
		outboxRepo,
		minioStorage,
		env.UploadURLTTL,
	)
	confirmUploadHandler := videocommand.NewConfirmUploadHandler(videoWriteRepo, jobWriteRepo, outboxRepo, env.VideoMaxSizeBytes)
	requestIngestHandler := videocommand.NewRequestIngestHandler(videoWriteRepo, outboxRepo)
	getVideoByIDHandler := queryvideo.NewGetVideoByIDHandler(videoReadRepo, minioStorage)
	listVideosHandler := queryvideo.NewListVideosHandler(videoReadRepo, minioStorage)

	return &Container{
		AuthenticateMiddleware: middleware.NewAuthenticateMiddleware(
			validateTokenHandler,
			fetchUserHandler,
			createUserHandler,
		),
		UserWebhookMiddleware:    middleware.NewUserWebhookMiddleware(env.ClerkWebhookSecret),
		StorageWebhookMiddleware: middleware.NewStorageWebhookMiddleware(env.MinIOWebhookSecret),
		VideoWebhookMiddleware:   middleware.NewVideoWebhookMiddleware(env.VideoWebhookSecret),
		UserWebhookHandler: httphandler.NewUserWebhookHandler(
			getUserByExternalIDHandler,
			createUserHandler,
			updateUserHandler,
			deleteUserByExternalIDHandler,
		),
		StorageWebhookHandler: httphandler.NewStorageWebhookHandler(env.StorageBucket, confirmUploadHandler),
		VideoWebhookHandler:   httphandler.NewVideoWebhookHandler(requestIngestHandler),
		UserHandler:           httphandler.NewUserHandler(getUserByIDHandler),
		VideoHandler:          httphandler.NewVideoHandler(requestUploadURLHandler, getVideoByIDHandler, listVideosHandler),
		RealtimeHandler:       httphandler.NewRealtimeHandler(centrifugo.NewConnectionInfoCreator(env)),
	}
}
