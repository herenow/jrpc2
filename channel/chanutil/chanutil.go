// Package chanutil exports helper functions for working with channels and
// framing defined by the github.com/herenow/jrpc2/channel package.
package chanutil

import "github.com/herenow/jrpc2/channel"

// Framing returns a channel.Framing described by the specified name, or nil if
// the name is unknown. The framing types currently understood are:
//
//    json   -- corresponds to channel.JSON
//    line   -- corresponds to channel.Line
//    lsp    -- corresponds to channel.LSP
//    raw    -- corresponds to channel.RawJSON
//    varint -- corresponds to channel.Varint
//
func Framing(name string) channel.Framing { return framings[name] }

var framings = map[string]channel.Framing{
	"json":   channel.JSON,
	"line":   channel.Line,
	"lsp":    channel.LSP,
	"raw":    channel.RawJSON,
	"varint": channel.Varint,
}
