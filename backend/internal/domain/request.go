package domain

import (
	"context"
	"net/http"
	"time"
)

type RequestType string

const (
	RequestTypeREST      RequestType = "rest"
	RequestTypeWebSocket RequestType = "websocket"
	RequestTypeGraphQL   RequestType = "graphql"
)

type Request interface {
	Validate() error
	Execute(ctx context.Context, env *Environment, transport *http.Transport) (Response, error)
	GetID() string
	SetID(string)
	GetName() string
	GetType() RequestType
	GetAuth() *Auth
	GetHeaders() []Header
	SetHeaders([]Header)
	GetQueryParams() map[string]string
	SetQueryParams(map[string]string)
	GetDependsOn() []string
	GetExtractVariables() map[string]string
	GetTimeout() *time.Duration
	GetURL() string
	SetURL(string)
	GetBodyOrMessage() string
	SetBodyOrMessage(string)
	GetContext() context.Context
}

type Header struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
