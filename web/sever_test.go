package web

import (
	"html/template"
	"net/http"
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

	// localhost:8081/form/123?username=xiaoming
	server.Post("/form/:id", func(ctx *Context) {
		id, _ := ctx.PathValue("id").ToInt64()
		username, _ := ctx.QueryValue("username").ToString()
		age, _ := ctx.FormValue("age").ToInt64()
		header, _ := ctx.HeaderJson("Content-Type").ToString()
		err := ctx.BindJSON(NewsignUpReq())
		ctx.setHeader("test", "setheader")
		ctx.SetCookie(&http.Cookie{
			Name:  "cookie",
			Value: "setcookie123",
		})
		ctx.OKRequestJson(struct {
			id     int64
			un     string
			age    int64
			header string
			err    error
		}{
			id,
			username,
			age,
			header,
			err,
		})
	})

	server.Start(":8081")
}

func TestServerWithRenderEngine(t *testing.T) {
	// 新建模板引擎
	tpl, err := template.ParseGlob("testdata/tpls/*.gohtml")
	if err != nil {
		t.Fatal(err)
	}
	s := NewsdkHttpServer("test", ServeWithTemplateEngine(&GOTemplateEngine{T: tpl}))
	s.Get("/login", func(ctx *Context) {
		er := ctx.Render("login.gohtml", nil)
		if er != nil {
			t.Fatal(er)
		}
	})
	err = s.Start(":8081")
	if err != nil {
		t.Fatal(err)
	}
}
