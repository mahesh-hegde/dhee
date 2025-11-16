package common

import (
	"fmt"
)

type UserVisibleError struct {
	HttpCode int
	Message  string
}

func (e *UserVisibleError) Error() string {
	return fmt.Sprintf("Error %d: %s", e.HttpCode, e.Message)
}

func NewUserVisibleError(httpCode int, message string) *UserVisibleError {
	return &UserVisibleError{
		HttpCode: httpCode,
		Message:  message,
	}
}

func WrapErrorForResponse(err error, message string) error {
	if e, ok := err.(*UserVisibleError); ok {
		return &UserVisibleError{
			HttpCode: e.HttpCode,
			Message:  fmt.Sprintf("%s: %s", message, e.Message),
		}
	}
	return err
}
