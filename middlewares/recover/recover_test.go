package recover

import (
	"go_web/web"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareBuilder_Build(t *testing.T) {

	mdlOPT := web.ServeWithMiddleware((&MiddlewareBuilder{
		StatusCode: 500,
		ErrMsg:     "你 Panic 了",
		LogFunc: func(ctx *web.Context) {
			log.Println(ctx.R.URL.Path)
		},
	}).Build())

	s := web.NewServerEngine("test", mdlOPT)
	s.Get("/user", func(ctx *web.Context) {
		ctx.RespData = []byte("hello, world")
	})

	s.Get("/panic", func(ctx *web.Context) {
		panic("闲着没事 panic")
	})

	//s.Start(":8081")
	res, _ := http.NewRequest(http.MethodGet, "http://localhost:8081/panic", nil)

	//模拟客户端访问
	resp := httptest.NewRecorder()
	s.ServeHTTP(resp, res)
}
