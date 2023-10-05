package errhandle

import (
	"bytes"
	"go_web/web"
	"html/template"
	"net/http"
	"testing"
)

func TestNewMiddlewareBuilder(t *testing.T) {
	//注册错误页面
	page := `
<html>
	<h1>404 NOT FOUND</h1>
</html>
`
	tpl, err := template.New("404").Parse(page)
	if err != nil {
		t.Fatal(err)
	}
	buffer := &bytes.Buffer{}
	err = tpl.Execute(buffer, nil)
	if err != nil {
		t.Fatal(err)
	}
	//生成中间件组件
	mdlOPT := web.ServeWithMiddleware(NewMiddlewareBuilder().
		RegisterError(404, buffer.Bytes()).Build())
	//注册中间件
	s := web.NewServerEngine("test", mdlOPT)

	s.Get("/hello", func(ctx *web.Context) {
		ctx.RespData = []byte("hello, world")
		ctx.RespStatusCode = http.StatusNotFound //模拟未找到的情况 404
	})

	s.Start(":8081")
}
