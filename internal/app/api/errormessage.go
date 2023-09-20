package api

type ErrorMessage struct {
	Error []string `json:"error"`
}

func NewSingleMessageError(err string) ErrorMessage {
	return ErrorMessage{Error: []string{err}}
}
