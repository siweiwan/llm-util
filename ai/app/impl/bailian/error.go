package bailian

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
)

type APIError struct {
	HTTPStatusCode int
	RequestId      string `json:"request_id"`
	Code           string `json:"code"`
	Message        string `json:"message"`
}

// RequestError provides information about generic request errors.
type RequestError struct {
	HTTPStatusCode int
	Err            error
	RequestId      string `json:"request_id"`
}

func NewRequestError(httpStatusCode int, rawErr error, requestID string) *RequestError {
	return &RequestError{
		HTTPStatusCode: httpStatusCode,
		Err:            rawErr,
		RequestId:      requestID,
	}
}

type ErrorResponse struct {
	Error *APIError `json:"error,omitempty"`
}

func (e *APIError) Error() string {
	s, _ := json.Marshal(e)
	return fmt.Sprintf("Error code: %d - %s", e.HTTPStatusCode, string(s))
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("RequestError code: %d, err: %v, request_id: %s", e.HTTPStatusCode, e.Err, e.RequestId)
}

func (e *RequestError) Unwrap() error {
	return e.Err
}

var (
	ErrTooManyEmptyStreamMessages = errors.New("stream has sent too many empty messages")
)
