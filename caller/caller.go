/*
Package caller provides a function to construct JRPC2 client call wrappers.

The New function reflectively constructs wrapper functions for calls through a
*jrpc2.Client. This makes it easier to provide a natural function signature for
the remote method, that handles the details of creating the request and
decoding the response internally.

The caller.New function takes the name of a method, a request type X and a
return type Y, and returns a function having the signature:

   func(context.Context, *jrpc2.Client, X) (Y, error)

The result can be asserted to this type and used as a normal function:

   Add := caller.New("Math.Add", caller.Options{
      Params: []int(nil),  // parameter type []int
      Result: int(0),      // result type int
   }).(func(context.Context, *jrpc2.Client, []int) (int, error))
   ...
   sum, err := Add(ctx, cli, []int{1, 3, 5, 7})
   ...

NewCaller can also optionally generate a variadic function:

   Mul := caller.New("Math.Mul", caller.Options{
      Params:   int(0),
      Result:   int(0),
      Variadic: true,
   }).(func(context.Context, *jrpc2.Client, ...int) (int, error))
   ...
   prod, err := Mul(ctx, cli, 1, 2, 3, 4, 5)
   ...

It can also generate a function with no request parameter (with X == nil), or a
function with no result (with Y == nil).

   Status := caller.New("Status", caller.Options{
      Result: string(""),  // result type string, no request parameters
   }).(func(context.Context, *jrpc2.Client) (string, error))
*/
package caller

import (
	"context"
	"reflect"

	"github.com/herenow/jrpc2"
)

// RPCServerInfo calls the built-in rpc.serverInfo method exported by servers
// in the jrpc2 package.
var RPCServerInfo = New("rpc.serverInfo", Options{
	Result: (*jrpc2.ServerInfo)(nil),
}).(func(context.Context, *jrpc2.Client) (*jrpc2.ServerInfo, error))

// Common types used by all invocations.
var (
	cliType = reflect.TypeOf((*jrpc2.Client)(nil))
	errType = reflect.TypeOf((*error)(nil)).Elem()
	ctxType = reflect.TypeOf((*context.Context)(nil)).Elem()
)

// New reflectively constructs a function of type:
//
//     func(context.Context, *jrpc2.Client, X) (Y, error)
//
// that invokes the designated method via the client given, encoding the
// provided request and decoding the response automatically. This supports
// construction of client wrappers that have a more natural function
// signature. The caller should assert the expected type on the return value.
//
// As a special case, either X or Y may be nil, in which case the returned
// function will omit the request argument (X == nil) or result (Y == nil) and
// have the signature:
//
//     func(context.Context, *jrpc2.Client) (Y, error)
//     func(context.Context, *jrpc2.Client, X) error
//
// New will panic if both X and Y are nil.
//
// Example:
//    cli := jrpc2.NewClient(ch, nil)
//
//    type Req struct{ A, B int }
//
//    // Suppose Math.Add is a method taking *Req to int.
//    F := caller.New("Math.Add", caller.Options{
//       Params: (*Req)(nil),
//       Result: int(0),
//    }).(func(context.Context, *jrpc2.Client, *Req) (int, error))
//
//    n, err := F(ctx, cli, &Req{A: 7, B: 3})
//    if err != nil {
//       log.Fatal(err)
//    }
//    fmt.Println(n)
//
func New(method string, opts Options) interface{} {
	X := opts.Params
	Y := opts.Result

	// Establish the argument types for the function.
	reqType := reflect.TypeOf(X)
	if opts.Variadic {
		reqType = reflect.SliceOf(reqType)
	}
	argTypes := []reflect.Type{ctxType, cliType}
	if reqType != nil {
		argTypes = append(argTypes, reqType)
	}

	// Establish the return types for the function.
	rspType := reflect.TypeOf(Y)
	outTypes := []reflect.Type{rspType}
	if rspType == nil {
		outTypes[0] = errType
	} else {
		outTypes = append(outTypes, errType)
	}

	if reqType == nil && rspType == nil {
		panic("no parameters and no result")
	}

	funType := reflect.FuncOf(argTypes, outTypes, opts.Variadic)

	// Map reflections to a concrete request argument (or nil).
	var param func(v []reflect.Value) interface{}
	if reqType == nil {
		// If there is no request type, don't populate a request argument.
		param = func([]reflect.Value) interface{} { return nil }
	} else if reqType.Kind() == reflect.Slice {
		// Callers passing slice typed arguments will expect nil to behave like an
		// empty slice, but the JSON encoder renders them as "null".  Therefore,
		// for slice typed parameters catch the nil case and convert it silently
		// into an empty slice of the correct type.
		param = func(v []reflect.Value) interface{} {
			if v[2].IsNil() {
				return reflect.MakeSlice(reqType, 0, 0).Interface()
			}
			return v[2].Interface()
		}
	} else {
		// The default condition is we have one request argument.
		param = func(v []reflect.Value) interface{} { return v[2].Interface() }
	}

	// We need to construct a pointer to the base type for unmarshaling, but
	// remember whether the caller wants the pointer or the base value.
	wantPtr := rspType != nil && rspType.Kind() == reflect.Ptr
	if wantPtr {
		rspType = rspType.Elem()
	}

	// Decode results into the response type.
	var result func(*jrpc2.Response, error) []reflect.Value
	if rspType == nil {
		// There is no result type; just return the error value, or nil.
		result = func(_ *jrpc2.Response, err error) []reflect.Value {
			return []reflect.Value{errToValue(err)}
		}
	} else if wantPtr {
		// The caller wants a pointer to the result.
		result = func(rsp *jrpc2.Response, err error) []reflect.Value {
			rv := reflect.New(rspType)
			if err == nil {
				err = rsp.UnmarshalResult(rv.Interface())
			}
			return []reflect.Value{rv, errToValue(err)}
		}
	} else {
		// The caller wants the bare result type.
		result = func(rsp *jrpc2.Response, err error) []reflect.Value {
			rv := reflect.New(rspType)
			if err == nil {
				err = rsp.UnmarshalResult(rv.Interface())
			}
			return []reflect.Value{rv.Elem(), errToValue(err)}
		}
	}

	call := opts.caller()
	return reflect.MakeFunc(funType, func(args []reflect.Value) []reflect.Value {
		ctx := args[0].Interface().(context.Context)
		cli := args[1].Interface().(*jrpc2.Client)

		return result(call(cli, ctx, method, param(args)))
	}).Interface()
}

func errToValue(err error) reflect.Value {
	if err == nil {
		return reflect.Zero(errType)
	}
	return reflect.ValueOf(err).Convert(errType)
}

// Options control how the caller generated by New is constructed.
type Options struct {
	// The type of the request parameters, or nil if request parameters should
	// be omitted.
	Params interface{}

	// The type of the result value, which must not be nil.
	Result interface{}

	// If true, the constructed function will be variadic in its request
	// parameters, i.e.,
	//
	//    func(context.Context, *jrpc2.Client, ...X) (Y, error)
	//
	// instead of
	//
	//    func(context.Context, *jrpc2.Client, X) (Y, error)
	Variadic bool

	// If true, the constructed function will send a notification instead of
	// issuing a call.
	Notify bool
}

func (o Options) caller() func(*jrpc2.Client, context.Context, string, interface{}) (*jrpc2.Response, error) {
	if o.Notify {
		return func(cli *jrpc2.Client, ctx context.Context, method string, params interface{}) (*jrpc2.Response, error) {
			return nil, cli.Notify(ctx, method, params)
		}
	}
	return (*jrpc2.Client).Call
}
