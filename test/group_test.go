package test

import (
	"fmt"
	"go_web/web"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGroup(t *testing.T) {
	mdl1 := func(next web.HandleFunc) web.HandleFunc {
		return func(ctx *web.Context) {
			println("这是Server级别的middleware: 0-/-mdl1")
			next(ctx)
		}
	}
	mdl2 := func(next web.HandleFunc) web.HandleFunc {
		return func(ctx *web.Context) {
			println("这是Group级别的middleware: 1-/a-mdl2")
			next(ctx)
		}
	}
	mdl3 := func(next web.HandleFunc) web.HandleFunc {
		return func(ctx *web.Context) {
			println("这是Group/Group级别的middleware: 2-/a/b-mdl3")
			next(ctx)
		}
	}

	s := web.NewServerEngine("test_group_middleware")
	s.Use(mdl1)

	v1 := s.NewGroup("/a")
	v1.Use(mdl2)

	v2 := v1.NewGroup("/b")
	v2.Use(mdl3)

	fmt.Println("Server / middleware Test_____________________________________________")
	Request1, _ := http.NewRequest(http.MethodGet, "http://localhost:8081/", nil)
	Response1 := httptest.NewRecorder()
	s.ServeHTTP(Response1, Request1)

	fmt.Println("GroupV1 /a middleware Test_____________________________________________")
	Request2, _ := http.NewRequest(http.MethodGet, "http://localhost:8081/a", nil)
	Response2 := httptest.NewRecorder()
	s.ServeHTTP(Response2, Request2)

	fmt.Println("GroupV2 /a/b middleware Test_____________________________________________")
	Request3, _ := http.NewRequest(http.MethodGet, "http://localhost:8081/a/b", nil)
	Response3 := httptest.NewRecorder()
	s.ServeHTTP(Response3, Request3)

}
