package domain

import "time"

type Response interface {
	Validate(*ResponseValidation) []string
	GetBody() interface{}
	GetStatusCode() int
	GetHeaders() []Header
}

type ResponseValidation struct {
	ExpectedStatusCode int               `json:"expectedStatusCode,omitempty"`
	ExpectedHeaders    map[string]string `json:"expectedHeaders,omitempty"`
	JSONSchema         string            `json:"jsonSchema,omitempty"`
	CustomAssertions   []string          `json:"customAssertions,omitempty"`
}

type RequestResult struct {
	Request            Request            `json:"request"`
	Response           Response           `json:"response"`
	Error              string             `json:"error,omitempty"`
	PerformanceMetrics PerformanceMetrics `json:"performance_metrics,omitempty"`
}

type PerformanceMetrics struct {
	DNSLookup        time.Duration `json:"dns_lookup"`
	TCPConnection    time.Duration `json:"tcp_connection"`
	TLSHandshake     time.Duration `json:"tls_handshake"`
	ServerProcessing time.Duration `json:"server_processing"`
	ContentTransfer  time.Duration `json:"content_transfer"`
	Total            time.Duration `json:"total"`
}
