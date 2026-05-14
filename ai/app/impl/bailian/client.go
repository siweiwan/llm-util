package bailian

import (
	"bufio"
	"context"
	"llm-util/ai/app/utils"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	config         ClientConfig
	requestBuilder utils.RequestBuilder
}

type requestOptions struct {
	body   any
	header http.Header
}

type requestOption func(*requestOptions)

func withBody(body any) requestOption {
	return func(args *requestOptions) {
		args.body = body
	}
}

func withHeaders(headers map[string]string) requestOption {
	return func(args *requestOptions) {
		for k, v := range headers {
			args.header.Set(k, v)
		}
	}
}

func withContentType(contentType string) requestOption {
	return func(args *requestOptions) {
		args.header.Set("Content-Type", contentType)
	}
}

func (c *Client) newRequest(ctx context.Context, method, url string, setters ...requestOption) (*http.Request, error) {
	// Default Options
	args := &requestOptions{
		body:   nil,
		header: make(http.Header),
	}
	for _, setter := range setters {
		setter(args)
	}
	req, err := c.requestBuilder.Build(ctx, method, url, args.body, args.header)
	if err != nil {
		return nil, err
	}
	c.setCommonHeaders(req)
	return req, nil
}

func (c *Client) setCommonHeaders(req *http.Request) {
	if c.config.Workspace != "" {
		req.Header.Set("X-DashScope-WorkSpace", c.config.Workspace)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.apiKey))
}

func (c *Client) sendRequest(req *http.Request, v Response) error {
	contentType := req.Header.Get("Content-Type")
	if contentType == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if isStream := req.Header.Get("X-DashScope-SSE"); isStream == "enable" {
		req.Header.Del("X-DashScope-SSE")
	}

	res, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if v != nil {
		v.SetHeader(res.Header)
	}

	if isFailureStatusCode(res) {
		return c.handleErrorResp(res)
	}

	return decodeResponse(res.Body, v)
}

func isFailureStatusCode(resp *http.Response) bool {
	return resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest
}

func (c *Client) handleErrorResp(resp *http.Response) error {

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error, reading response body: %w", err)
	}
	var errRes ErrorResponse
	err = json.Unmarshal(body, &errRes.Error)
	if err != nil {
		reqErr := &RequestError{
			HTTPStatusCode: resp.StatusCode,
			Err:            err,
			RequestId:      resp.Header.Get("X-Request-Id"),
		}
		return reqErr
	}

	errRes.Error.HTTPStatusCode = resp.StatusCode

	return errRes.Error
}

func decodeResponse(body io.Reader, v any) error {
	if v == nil {
		return nil
	}

	switch o := v.(type) {
	case *string:
		return decodeString(body, o)
	case *audioTextResponse:
		return decodeString(body, &o.Text)
	default:
		return json.NewDecoder(body).Decode(v)
	}
}

func decodeString(body io.Reader, output *string) error {
	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	*output = string(b)
	return nil
}

func (c *Client) fullURL(appID string) string {
	baseURL := strings.TrimRight(c.config.BaseURL, "/")
	return fmt.Sprintf("%s/%s%s", baseURL, appID, BailianAppInvokeSuffix)
}

func needRetryError(err error) bool {
	apiErr := &APIError{}
	reqErr := &RequestError{}
	if errors.As(err, &apiErr) {
		return apiErr.HTTPStatusCode >= http.StatusInternalServerError || apiErr.HTTPStatusCode == http.StatusTooManyRequests
	} else if errors.Is(err, io.EOF) {
		return true
	} else if errors.As(err, &reqErr) {
		return reqErr.HTTPStatusCode >= http.StatusInternalServerError
	}
	return false
}

func NewClientWithConfig(config ClientConfig) *Client {
	return &Client{
		config:         config,
		requestBuilder: utils.NewRequestBuilder(),
	}
}

func NewClientWithAppIDAPIKey(appID, apiKey string) *Client {
	cfg := NewClientConfig(appID, apiKey)
	return NewClientWithConfig(cfg)
}

func sendRequestStream[T Streamable](client *Client, req *http.Request) (*StreamReader[T], error) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("X-DashScope-SSE", "enable")

	resp, err := client.config.HTTPClient.Do(req) //nolint:bodyclose // body is closed in stream.Close()
	if err != nil {
		return new(StreamReader[T]), err
	}
	if isFailureStatusCode(resp) {
		return new(StreamReader[T]), client.handleErrorResp(resp)
	}

	return &StreamReader[T]{
		EmptyMessagesLimit: client.config.EmptyMessagesLimit,
		Reader:             bufio.NewReader(resp.Body),
		Response:           resp,
		ErrAccumulator:     utils.NewErrorAccumulator(),
		Unmarshaler:        &utils.JSONUnmarshaler{},
		HttpHeader:         HttpHeader(resp.Header),
	}, nil
}
