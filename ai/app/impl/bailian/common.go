package bailian

import (
	"net/http"
	"time"
)

const (
	BailianAppInvokeSuffix = "/completion"

	ErrorRetryBaseDelay = 500 * time.Millisecond
	ErrorRetryMaxDelay  = 8 * time.Second
)

type Response interface {
	SetHeader(http.Header)
	GetHeader() http.Header
}

type HttpHeader http.Header

func (h *HttpHeader) SetHeader(header http.Header) {
	*h = HttpHeader(header)
}

func (h *HttpHeader) GetHeader() http.Header {
	return http.Header(*h)
}

func (h *HttpHeader) Header() http.Header {
	return http.Header(*h)
}
