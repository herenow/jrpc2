/*
Package jrpc2 implements a server and a client for the JSON-RPC 2.0 protocol
defined by http://www.jsonrpc.org/specification.

Servers

The *Server type implements a JSON-RPC server. A server communicates with a
client over a channel.Channel, and dispatches client requests to user-defined
method handlers.  Handlers satisfy the jrpc2.Handler interface by exporting a
Handle method with this signature:

   Handle(ctx Context.Context, req *jrpc2.Request) (interface{}, error)

The jrpc2.NewHandler function helps adapt existing functions to this interface.
A server finds the handler for a request by looking up its method name in a
jrpc2.Assigner provided when the server is set up.

For example, suppose we have defined the following Add function, and would like
to export it as a JSON-RPC method:

   // Add returns the sum of a slice of integers.
   func Add(ctx context.Context, values []int) (int, error) {
      sum := 0
      for _, v := range values {
         sum += v
      }
      return sum, nil
   }

To convert Add to a jrpc2.Handler, call the jrpc2.NewHandler function, which
uses reflection to lift its argument into the jrpc2.Handler interface:

   h := jrpc2.NewHandler(Add)  // h is a jrpc2.Handler that invokes Add

We will advertise this function under the name "Add".  For static assignments
we can use a jrpc2.MapAssigner, which finds methods by looking them up in a Go
map:

   assigner := jrpc2.MapAssigner{
      "Add": jrpc2.NewHandler(Add),
   }

Equipped with an Assigner we can now construct a Server:

   srv := jrpc2.NewServer(assigner, nil)  // nil for default options

To serve requests, we will next need a channel.Channel. The channel package
exports functions that can adapt various input and output streams.  For this
example, we'll use a channel that delimits messages by newlines, and
communicates on os.Stdin and os.Stdout:

   ch := channel.Line(os.Stdin, os.Stdout)
   srv.Start(ch)

Once started, the running server will handle incoming requests until the
channel closes, or until it is stopped explicitly by calling srv.Stop(). To
wait for the server to finish, call:

   err := srv.Wait()

This will report the error that led to the server exiting. A working
implementation of this example can found in examples/adder/adder.go:

    $ go run examples/adder/adder.go

You can interact with this server on the command line.


Clients

The *Client type implements a JSON-RPC client. A client communicates with a
server over a channel.Channel, and is safe for concurrent use by multiple
goroutines. It supports batched requests and may have arbitrarily many pending
requests in flight simultaneously.

To establish a client we first need a channel:

   import "net"

   conn, err := net.Dial("tcp", "localhost:8080")
   ...
   ch := channel.RawJSON(conn, conn)
   cli := jrpc2.NewClient(ch, nil)  // nil for default options

To send a single RPC, use the Call method:

   rsp, err := cli.Call(ctx, "Add", []int{1, 3, 5, 7})

This blocks until the response is received. Any error returned by the server,
including cancellation or deadline exceeded, has concrete type *jrpc2.Error.

To issue a batch of requests all at once, use the Batch method:

   rsps, err := cli.Batch(ctx, []jrpc2.Spec{
      {"Math.Add", []int{1, 2, 3}},
      {"Math.Mul", []int{4, 5, 6}},
      {"Math.Max", []int{-1, 5, 3, 0, 1}},
   })

The Batch method waits until all the responses are received.  The caller must
check each response separately for errors. The responses will be returned in
the same order as the Spec values, save that notifications are omitted.

To decode the result from a successful response use its UnmarshalResult method:

   var result int
   if err := rsp.UnmarshalResult(&result); err != nil {
      log.Fatalln("UnmarshalResult:", err)
   }

To shut down a client and discard all its pending work, call cli.Close().


Notifications

The JSON-RPC protocol also supports a kind of request called a notification.
Notifications differ from ordinary calls in that they are one-way: The client
sends them to the server, but the server does not reply.

A jrpc2.Client supports sending notifications as follows:

   type alert struct { M string `json:"message"` }
   err := cli.Notify(ctx, "Alert", alert{M: "a fire is burning!"})

Unlike ordinary requests, there are no pending calls for notifications; the
notification is complete once it has been sent.

On the server side, notifications are identical to ordinary requests, save that
their return value is discarded once the handler returns. If a handler does not
want to do anything for a notification, it can query the request:

   if req.IsNotification() {
      return 0, nil  // ignore notifications
   }

Cancellation

The *Client and *Server types support a nonstandard cancellation protocol, that
consists of a notification method "rpc.cancel" taking an array of request IDs
to be cancelled. Upon receiving this notification, the server will cancel the
context of each method handler whose ID is named.

When the context associated with a client request is cancelled, the client will
send an "rpc.cancel" notification to the server for that request's ID:

   ctx, cancel := context.WithCancel(ctx)
   p, err := cli.Call(ctx, "MethodName", params)
   ...
   cancel()
   rsp := p.Wait()

The "rpc.cancel" method is automatically handled by the *Server implementation
from this package.

Services with Multiple Methods

The examples above show a server with only one method using NewHandler; you
will often want to expose more than one. The NewService function supports this
by applying NewHandler to all the exported methods of a concrete value to
produce a MapAssigner for those methods:

   type math struct{}

   func (math) Add(ctx context.Context, vals ...int) (int, error) { ... }
   func (math) Mul(ctx context.Context, vals []int) (int, error) { ... }

   assigner := jrpc2.NewService(math{})

This assigner maps the name "Add" to the Add method, and the name "Mul" to the
Mul method, of the math value.

This may be further combined with the ServiceMapper type to allow different
services to work together:

   type status struct{}

   func (status) Get(context.Context) (string, error) {
      return "all is well", nil
   }

   assigner := jrpc2.ServiceMapper{
      "Math":   jrpc2.NewService(math{}),
      "Status": jrpc2.NewService(status{}),
   }

This assigner dispatches "Math.Add" and "Math.Mul" to the math value's methods,
and "Status.Get" to the status value's method. A ServiceMapper splits the
method name on the first period ("."), and you may nest ServiceMappers more
deeply if you require a more complex hierarchy.

See the "caller" package for a convenient way to generate client call wrappers.
*/
package jrpc2

// Version is the version string for the JSON-RPC protocol understood by this
// implementation, defined at http://www.jsonrpc.org/specification.
const Version = "2.0"
