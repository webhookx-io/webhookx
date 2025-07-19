package errs

import "errors"

var ErrRequestValidate = errors.New("request validation")

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
	Message string                 `json:"message"`
	Fields  map[string]interface{} `json:"fields"`
}

func NewValidateError(err error) *ValidateError {
	return &ValidateError{
		err:     err,
		Message: err.Error(),
		Fields:  make(map[string]interface{}),
	}
}

func NewValidateFieldsError(err error, fields map[string]interface{}) *ValidateError {
	return &ValidateError{
		err:     err,
		Message: err.Error(),
		Fields:  fields,
	}
}

func (e *ValidateError) Error() string {
	return e.err.Error()
}
