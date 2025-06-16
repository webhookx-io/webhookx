package api

// TODO delete me
type ErrorResponse struct {
	Message string      `json:"message"`
	Error   interface{} `json:"error,omitempty"`
}

var (
	MsgNotFound    = "Not found"
	MsgInavlidUUID = "Invalid uuid"
)
