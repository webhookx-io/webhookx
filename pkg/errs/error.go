package errs

type Error struct {
	err error
}

func NewError(err error) *Error {
	return &Error{
		err: err,
	}
}

func (e *Error) Error() string {
	return e.err.Error()
}

type ValidateError struct {
	err     error
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields"`
}

func NewValidateError(err error) *ValidateError {
	return &ValidateError{
		err:     err,
		Message: err.Error(),
		Fields:  make(map[string]string),
	}
}

func (e *ValidateError) Error() string {
	return e.err.Error()
}
