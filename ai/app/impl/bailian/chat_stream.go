package bailian

import (
	"context"
	"llm-util/ai/app/utils"
	"net/http"
)

type ChatCompletionStreamResponse struct {
	RequestID string                        `json:"request_id"`
	Output    *ChatCompletionResponseOutput `json:"output"`
	Usage     *ChatCompletionResponseUsage  `json:"usage,omitempty"`
}

type ChatCompletionResponseOutput struct {
	SessionID    string                  `json:"session_id"`
	FinishReason string                  `json:"finish_reason"`
	Text         string                  `json:"text"`
	Thoughts     []ResponseOutputThought `json:"thoughts"`
}

type ChatCompletionResponseUsage struct {
	Models []*ChatCompletionResponseUsageModel `json:"models"`
}
type ChatCompletionResponseUsageModel struct {
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	ModelID      string `json:"model_id"`
}

type ChatCompletionStream struct {
	*StreamReader[ChatCompletionStreamResponse]
}

func (c *Client) CreateChatCompletionStream(ctx context.Context, request ChatCompletionRequest,
) (stream *ChatCompletionStream, err error) {

	if request.Parameters == nil {
		trueVal := true
		request.Parameters = &RequestParameters{IncrementalOutput: &trueVal}
	}

	err = utils.Retry(
		ctx,
		utils.RetryPolicy{
			MaxAttempts:    c.config.RetryTimes,
			InitialBackoff: ErrorRetryBaseDelay,
			MaxBackoff:     ErrorRetryMaxDelay,
		},
		func() bool { return true },
		func() error {
			req, inErr := c.newRequest(
				ctx,
				http.MethodPost,
				c.fullURL(c.config.AppID),
				withBody(request),
			)
			if inErr != nil {
				return inErr
			}

			resp, err := sendRequestStream[ChatCompletionStreamResponse](c, req)
			stream = &ChatCompletionStream{
				StreamReader: resp,
			}
			return err
		},
		nil,
		needRetryError,
	)

	return
}
