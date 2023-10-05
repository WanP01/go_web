package prometheus

import (
	"go_web/web"
	"net/http"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestPrometheus(t *testing.T) {

	prom := &MiddlewareBuilder{
		Subsystem: "web",
		Name:      "http_request",
		Help:      "这是测试例子",
		ConstLabels: map[string]string{
			"instance_id": "1234567",
		},
	}
	promMDL := prom.Build()
	proOPT := web.ServeWithMiddleware(promMDL)
	s := web.NewServerEngine("test_prometheus", proOPT)
	s.Get("/hello", func(ctx *web.Context) {
		ctx.W.Write([]byte("hello, world"))
	})

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		// 一般来说，在实际中我们都会单独准备一个端口给这种监控
		http.ListenAndServe(":2112", nil)
	}()
	s.Start(":8081")
}
