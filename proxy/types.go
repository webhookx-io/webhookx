package proxy

type HttpError struct {
	Code    int
	Message string
	Err     error
}

func (e *HttpError) Error() string {
	return e.Message
}

type Response struct {
	Headers map[string]string
	Code    int
	Body    []byte
}
