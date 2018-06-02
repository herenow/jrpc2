package jrpc2

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
)

// ServerOptions control the behaviour of a server created by NewServer.
// A nil *ServerOptions provides sensible defaults.
type ServerOptions struct {
	// If not nil, send debug logs here.
	Logger *log.Logger

	// Instructs the server to tolerate requests that do not include the
	// required "jsonrpc" version marker.
	AllowV1 bool

	// Allows up to the specified number of concurrent goroutines to execute
	// when processing requests. A value less than 1 uses runtime.NumCPU().
	Concurrency int

	// If set, this function is called with the encoded request parameters
	// received from the client, before they are delivered to the handler.  Its
	// return value replaces the context and argument values. This allows the
	// server to decode context metadata sent by the client. If unset, ctx and
	// params are used as given.
	DecodeContext func(context.Context, json.RawMessage) (context.Context, json.RawMessage, error)

	// If set, use this value to record server metrics. All servers created
	// from the same options will share the same metrics collector.  If none is
	// set, an empty collector will be created for each new server.
	Metrics *Metrics
}

func (s *ServerOptions) logger() func(string, ...interface{}) {
	if s == nil || s.Logger == nil {
		return func(string, ...interface{}) {}
	}
	logger := s.Logger
	return func(msg string, args ...interface{}) { logger.Output(2, fmt.Sprintf(msg, args...)) }
}

func (s *ServerOptions) allowV1() bool { return s != nil && s.AllowV1 }

func (s *ServerOptions) concurrency() int64 {
	if s == nil || s.Concurrency < 1 {
		return int64(runtime.NumCPU())
	}
	return int64(s.Concurrency)
}

func (s *ServerOptions) decodeContext() func(context.Context, json.RawMessage) (context.Context, json.RawMessage, error) {
	if s == nil || s.DecodeContext == nil {
		return func(ctx context.Context, params json.RawMessage) (context.Context, json.RawMessage, error) {
			return ctx, params, nil
		}
	}
	return s.DecodeContext
}

func (s *ServerOptions) metrics() *Metrics {
	if s == nil || s.Metrics == nil {
		return NewMetrics()
	}
	return s.Metrics
}

// ClientOptions control the behaviour of a client created by NewClient.
// A nil *ClientOptions provides sensible defaults.
type ClientOptions struct {
	// If not nil, send debug logs here.
	Logger *log.Logger

	// Instructs the client to tolerate responses that do not include the
	// required "jsonrpc" version marker.
	AllowV1 bool

	// If set, this function is called with the context and encoded request
	// parameters before the request is sent to the server. Its return value
	// replaces the request parameters. This allows the client to send context
	// metadata along with the request. If unset, the parameters are unchanged.
	EncodeContext func(context.Context, json.RawMessage) (json.RawMessage, error)
}

// ClientLog enables debug logging to the specified writer.
func (c *ClientOptions) logger() func(string, ...interface{}) {
	if c == nil || c.Logger == nil {
		return func(string, ...interface{}) {}
	}
	logger := c.Logger
	return func(msg string, args ...interface{}) { logger.Output(2, fmt.Sprintf(msg, args...)) }
}

func (c *ClientOptions) allowV1() bool { return c != nil && c.AllowV1 }

func (c *ClientOptions) encodeContext() func(context.Context, json.RawMessage) (json.RawMessage, error) {
	if c == nil || c.EncodeContext == nil {
		return func(_ context.Context, params json.RawMessage) (json.RawMessage, error) { return params, nil }
	}
	return c.EncodeContext
}
