// Package server provides support routines for running jrpc2 servers.
package server

import (
	"io"
	"log"
	"net"
	"sync"

	"bitbucket.org/creachadair/jrpc2"
	"bitbucket.org/creachadair/jrpc2/channel"
)

// Loop obtains connections from lst and starts a server for each with the
// given assigner and options, running in a new goroutine. If accept reports an
// error, the loop will terminate and the error will be reported once all the
// servers currently active have returned.
func Loop(lst net.Listener, assigner jrpc2.Assigner, opts *LoopOptions) error {
	var wg sync.WaitGroup
	for {
		conn, err := lst.Accept()
		if err != nil {
			log.Printf("Error accepting new connection: %v", err)
			wg.Wait()
			return err
		}
		ch := opts.newChannel(conn)
		wg.Add(1)
		go func() {
			defer wg.Done()
			srv := jrpc2.NewServer(assigner, opts.serverOpts()).Start(ch)
			if err := srv.Wait(); err != nil && err != io.EOF {
				log.Printf("Server exit: %v", err)
			}
		}()
	}
}

// LoopOptions control the behaviour of the Loop function.  A nil *LoopOptions
// provides default values as described.
type LoopOptions struct {
	// If non-nil, this function is used to convert a stream connection to an
	// RPC channel. If this field is nil, channel.Raw is used.
	NewChannel func(io.ReadWriteCloser) jrpc2.Channel

	// If non-nil, these options are used when constructing the server to
	// handle requests on an inbound connection.
	ServerOptions *jrpc2.ServerOptions
}

func (o *LoopOptions) serverOpts() *jrpc2.ServerOptions {
	if o == nil {
		return nil
	}
	return o.ServerOptions
}

func (o *LoopOptions) newChannel(conn net.Conn) jrpc2.Channel {
	if o == nil || o.NewChannel == nil {
		return channel.Raw(conn)
	}
	return o.NewChannel(conn)
}
