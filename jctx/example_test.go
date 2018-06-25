package jctx_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/herenow/jrpc2/jctx"
)

func ExampleEncode_basic() {
	ctx := context.Background()
	enc, err := jctx.Encode(ctx, json.RawMessage(`[1,2,3]`))
	if err != nil {
		log.Fatalln("Encode:", err)
	}
	fmt.Println(string(enc))
	// Output:
	// {"jctx":"1","payload":[1,2,3]}
}

func ExampleEncode_deadline() {
	deadline := time.Date(2018, 6, 9, 20, 45, 33, 1, time.UTC)

	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	enc, err := jctx.Encode(ctx, json.RawMessage(`{"A":"#1"}`))
	if err != nil {
		log.Fatalln("Encode:", err)
	}
	fmt.Println(pretty(enc))
	// Output:
	// {
	//   "jctx": "1",
	//   "deadline": "2018-06-09T20:45:33.000000001Z",
	//   "payload": {
	//     "A": "#1"
	//   }
	// }
}

func ExampleDecode() {
	const input = `{"jctx":"1","deadline":"2018-06-09T20:45:33.000000001Z","payload":["a", "b", "c"]}`

	ctx, param, err := jctx.Decode(context.Background(), json.RawMessage(input))
	if err != nil {
		log.Fatalln("Decode:", err)
	}
	dl, ok := ctx.Deadline()

	fmt.Println("params:", string(param))
	fmt.Println("deadline:", ok, dl)
	// Output:
	// params: ["a", "b", "c"]
	// deadline: true 2018-06-09 20:45:33.000000001 +0000 UTC
}

func ExampleWithMetadata() {
	type Auth struct {
		User  string `json:"user"`
		Token []byte `json:"token"`
	}
	ctx, err := jctx.WithMetadata(context.Background(), &Auth{
		User:  "Jon Snow",
		Token: []byte("28EF40F5-77C9-4744-B5BD-3ADCD1C15141"),
	})
	if err != nil {
		log.Fatalln("WithMetadata:", err)
	}

	enc, err := jctx.Encode(ctx, nil)
	if err != nil {
		log.Fatal("Encode:", err)
	}
	fmt.Println(pretty(enc))
	// Output:
	// {
	//   "jctx": "1",
	//   "meta": {
	//     "user": "Jon Snow",
	//     "token": "MjhFRjQwRjUtNzdDOS00NzQ0LUI1QkQtM0FEQ0QxQzE1MTQx"
	//   }
	// }
}

func ExampleUnmarshalMetadata() {
	// Setup for the example...
	const meta = `{"user":"Jon Snow","token":"MjhFRjQwRjUtNzdDOS00NzQ0LUI1QkQtM0FEQ0QxQzE1MTQx"}`
	ctx, err := jctx.WithMetadata(context.Background(), json.RawMessage(meta))
	if err != nil {
		log.Fatalln("Setup:", err)
	}

	// Demonstrates how to decode the value back.
	var auth struct {
		User  string `json:"user"`
		Token []byte `json:"token"`
	}
	if err := jctx.UnmarshalMetadata(ctx, &auth); err != nil {
		log.Fatalln("UnmarshalMetadata:", err)
	}
	fmt.Println("user:", auth.User)
	fmt.Println("token:", string(auth.Token))
	// Output:
	// user: Jon Snow
	// token: 28EF40F5-77C9-4744-B5BD-3ADCD1C15141
}

func pretty(v []byte) string {
	var buf bytes.Buffer
	if err := json.Indent(&buf, v, "", "  "); err != nil {
		log.Fatal(err)
	}
	return buf.String()
}
