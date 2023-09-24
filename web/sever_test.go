package web

import (
	"testing"
)

func TestSever(t *testing.T) {

	server := NewsdkHttpServer("testSever")

	server.Get("/", func(ctx *Context) {
		ctx.W.Write([]byte("hello,this is index_get"))
	})
	server.Post("/hello", func(ctx *Context) {
		ctx.W.Write([]byte("hello,this is hello_post"))
	})

	server.Start(":8081")
}
