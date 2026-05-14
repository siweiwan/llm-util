package bailian

import (
	"bufio"
	"bytes"
	"llm-util/ai/app/utils"
	"fmt"
	"io"
	"net/http"
)

var (
	headerID         = []byte("id:")
	headerEvent      = []byte("event:")
	headerHttpStatus = []byte(":HTTP_STATUS")
	headerData       = []byte("data:")
	errorPrefix      = []byte(`data: {"error":`)
)

type Streamable interface {
	ChatCompletionStreamResponse
}

type StreamReader[T Streamable] struct {
	EmptyMessagesLimit uint
	IsFinished         bool
	Reader             *bufio.Reader
	Response           *http.Response
	ErrAccumulator     utils.ErrorAccumulator
	Unmarshaler        utils.Unmarshaler

	HttpHeader
}

func (stream *StreamReader[T]) Recv() (response T, err error) {
	if stream.IsFinished {
		err = io.EOF
		return
	}

	response, err = stream.processLines()
	return
}

//nolint:gocognit
func (stream *StreamReader[T]) processLines() (T, error) {
	var (
		emptyMessagesCount uint
		hasErrorPrefix     bool
	)

	for {
		rawLine, readErr := stream.Reader.ReadBytes('\n')

		if readErr != nil || hasErrorPrefix {
			respErr := stream.unmarshalError()
			if respErr != nil {
				return *new(T), fmt.Errorf("error, %w", respErr.Error)
			}
			return *new(T), readErr
		}

		if bytes.HasPrefix(rawLine, headerID) || bytes.HasPrefix(rawLine, headerEvent) || bytes.HasPrefix(rawLine, headerHttpStatus) {
			continue
		}

		noSpaceLine := bytes.TrimSpace(rawLine)
		if bytes.HasPrefix(noSpaceLine, errorPrefix) {
			hasErrorPrefix = true
		}
		if !bytes.HasPrefix(noSpaceLine, headerData) || hasErrorPrefix {
			if hasErrorPrefix {
				noSpaceLine = bytes.TrimPrefix(noSpaceLine, headerData)
			}
			writeErr := stream.ErrAccumulator.Write(noSpaceLine)
			if writeErr != nil {
				return *new(T), writeErr
			}

			if !bytes.HasPrefix(noSpaceLine, headerID) || !bytes.HasPrefix(noSpaceLine, headerEvent) {

			}
			emptyMessagesCount++
			if emptyMessagesCount > stream.EmptyMessagesLimit {
				return *new(T), ErrTooManyEmptyStreamMessages
			}

			continue
		}

		noPrefixLine := bytes.TrimPrefix(noSpaceLine, headerData)

		var response T
		unmarshalErr := stream.Unmarshaler.Unmarshal(noPrefixLine, &response)
		if unmarshalErr != nil {
			return *new(T), unmarshalErr
		}

		switch v := any(response).(type) {
		case ChatCompletionStreamResponse:
			if v.Output != nil && v.Output.FinishReason == "stop" {
				stream.IsFinished = true
			}
		}

		return response, nil
	}
}

func (stream *StreamReader[T]) unmarshalError() (errResp *ErrorResponse) {
	errBytes := stream.ErrAccumulator.Bytes()
	if len(errBytes) == 0 {
		return
	}

	err := stream.Unmarshaler.Unmarshal(errBytes, &errResp)
	if err != nil {
		errResp = nil
	}

	return
}

func (stream *StreamReader[T]) Close() error {
	return stream.Response.Body.Close()
}
