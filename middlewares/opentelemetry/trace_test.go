package opentelemetry

import (
	"go_web/web"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
)

func TestTraceMiddleware(t *testing.T) {
	// zipkin注册为registered global trace provider
	initZipkin(t)
	//获取registered global trace provider,产生tracer实例
	tracer := otel.GetTracerProvider().Tracer("test_tracer")
	//注册trace middleware
	TracerMiddleware := NewTraceMiddlewareBuilder(tracer).Build()
	OPT := web.ServeWithMiddleware(TracerMiddleware)

	s := web.NewsdkHttpServer("test_serve", OPT)
	s.Get("/user", func(ctx *web.Context) {
		c, span := tracer.Start(ctx.R.Context(), "first_layer")
		defer span.End()

		c, second := tracer.Start(c, "second_layer")
		time.Sleep(time.Second)
		c, third1 := tracer.Start(c, "third_layer_1")
		time.Sleep(100 * time.Millisecond)
		third1.End()
		c, third2 := tracer.Start(c, "third_layer_1")
		time.Sleep(300 * time.Millisecond)
		third2.End()
		second.End()
		ctx.RespStatusCode = 200
		ctx.RespData = []byte("hello, world")
	})

	res, _ := http.NewRequest(http.MethodGet, "http://localhost:8081/user/123", nil)

	//模拟客户端访问
	resp := httptest.NewRecorder()
	s.ServeHTTP(resp, res)

	// //模拟客户端访问
	// go func() {
	// 	// time.Sleep(10 * time.Second)
	// 	s.Start(":8081")
	// }()

	// time.Sleep(1 * time.Second)
	// http.DefaultClient.Do(res)
}
