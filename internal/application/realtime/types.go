package realtime

const (
	ActionCreated  = "created"
	ActionUpdated  = "updated"
	ActionDeleted  = "deleted"
	ActionUploaded = "uploaded"

	EntityUser  = "user"
	EntityVideo = "video"
	EntityJob   = "job"
)

func EventType(entity, action string) string {
	return entity + "." + action
}
