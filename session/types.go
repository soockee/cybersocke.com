package session

type ctxKey string

const (
	IdTokenKey ctxKey = "id_token"
	SessionKey ctxKey = "session"

	FlashError string = "flash_error"
)
