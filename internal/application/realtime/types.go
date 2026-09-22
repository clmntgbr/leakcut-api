package realtime

const (
	ActionCreated = "created"
	ActionUpdated = "updated"
	ActionDeleted = "deleted"

	EntityUser  = "user"
	EntityVideo = "video"
)

func EventType(entity, action string) string {
	return entity + "." + action
}
