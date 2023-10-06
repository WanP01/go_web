package web

import (
	"net/http"
	"testing"
)

// go test -v -run="none" -bench=. -benchtime="30s" -benchmem
func BenchmarkRouterAdd(b *testing.B) {
	b.StopTimer()
	b.StartTimer()
	//测试用例
	testRoutes := []struct {
		method string
		path   string
	}{
		{
			method: http.MethodGet,
			path:   "/",
		},
		{
			method: http.MethodGet,
			path:   "/user",
		},
		{
			method: http.MethodGet,
			path:   "/user/home",
		},
		{
			method: http.MethodGet,
			path:   "/order/detail",
		},
		{
			method: http.MethodPost,
			path:   "/order/create",
		},
		{
			method: http.MethodPost,
			path:   "/login",
		},
		// 通配符测试用例
		{
			method: http.MethodGet,
			path:   "/order/*",
		},
		{
			method: http.MethodGet,
			path:   "/*",
		},
		{
			method: http.MethodGet,
			path:   "/*/*",
		},
		{
			method: http.MethodGet,
			path:   "/*/abc",
		},
		{
			method: http.MethodGet,
			path:   "/*/abc/*",
		},
		// 参数路由
		{
			method: http.MethodGet,
			path:   "/param/:id",
		},
		{
			method: http.MethodGet,
			path:   "/param/:id/detail",
		},
		{
			method: http.MethodGet,
			path:   "/param/:id/*",
		},
		{
			method: http.MethodGet,
			path:   "/param/:id/detail/:username",
		},
		// 正则路由
		{
			method: http.MethodDelete,
			path:   "/reg/:id(.*)",
		},
		{
			method: http.MethodDelete,
			path:   "/:name(^.+$)/abc",
		},
	}

	//初始化Router，循环注册路由示例testCase
	mockhandler := func(ctx *Context) {}

	for i := 0; i <= b.N; i++ {
		r := newRouter()
		for _, tr := range testRoutes {
			r.Route(tr.method, tr.path, mockhandler)
		}
	}
}

func BenchmarkRouterFinder(b *testing.B) {
	testRoutes := []struct {
		method string
		path   string
	}{
		{
			method: http.MethodGet,
			path:   "/",
		},
		{
			method: http.MethodGet,
			path:   "/user",
		},
		{
			method: http.MethodPost,
			path:   "/order/create",
		},
		//通配符注册
		{
			method: http.MethodGet,
			path:   "/user/*/home",
		},
		{
			method: http.MethodPost,
			path:   "/order/*",
		},
		// 参数路由
		{
			method: http.MethodGet,
			path:   "/param/:id",
		},
		{
			method: http.MethodGet,
			path:   "/param/:id/detail",
		},
		{
			method: http.MethodGet,
			path:   "/param/:id/*",
		},
		{
			method: http.MethodGet,
			path:   "/param/:id/detail/:username",
		},
		{
			method: http.MethodGet,
			path:   "/param/:id/again/:id",
		},
		// 正则
		{
			method: http.MethodDelete,
			path:   "/reg/:id(.*)",
		},
		{
			method: http.MethodDelete,
			path:   "/:id([0-9]+)/home",
		},
		//优先级
		{
			method: http.MethodGet,
			path:   "/k1/k2/k3",
		},
		{
			method: http.MethodGet,
			path:   "/k1/k2/:id",
		},
		{
			method: http.MethodGet,
			path:   "/k1/k2/:id(.*)",
		},
		//回溯算法
		{
			method: http.MethodGet,
			path:   "/k1/:reg(.*)/k3",
		},
		{
			method: http.MethodGet,
			path:   "/k1/:parm/k2",
		},
		{
			method: http.MethodGet,
			path:   "/k1/*/k1",
		},
		{
			method: http.MethodGet,
			path:   "/k1/*",
		},
	}

	mockHandler := func(ctx *Context) {}

	testCases := []struct {
		name   string
		method string
		path   string
		found  bool
		mi     *matchInfo
	}{
		{
			name:   "method not found",
			method: http.MethodHead,
		},
		{
			name:   "path not found",
			method: http.MethodGet,
			path:   "/abc",
		},
		{
			name:   "root",
			method: http.MethodGet,
			path:   "/",
			found:  true,
			mi: &matchInfo{
				n: &node{
					path:       "/",
					handlefunc: mockHandler,
				}},
		},
		{
			name:   "user",
			method: http.MethodGet,
			path:   "/user",
			found:  true,
			mi: &matchInfo{
				n: &node{
					path:       "user",
					handlefunc: mockHandler,
				}},
		},
		{
			name:   "no handler",
			method: http.MethodPost,
			path:   "/order",
			found:  true,
			mi: &matchInfo{
				n: &node{
					path: "order",
				}},
		},
		{
			name:   "two layer",
			method: http.MethodPost,
			path:   "/order/create",
			found:  true,
			mi: &matchInfo{
				n: &node{
					path:       "create",
					handlefunc: mockHandler,
				}},
		},
		// 通配符匹配
		{
			// 命中/order/*
			name:   "star match",
			method: http.MethodPost,
			path:   "/order/delete",
			found:  true,
			mi: &matchInfo{
				n: &node{
					path:       "*",
					handlefunc: mockHandler,
				}},
		},
		{
			// 命中通配符在中间的
			// /user/*/home
			name:   "star in middle",
			method: http.MethodGet,
			path:   "/user/Tom/home",
			found:  true,
			mi: &matchInfo{
				n: &node{
					path:       "home",
					handlefunc: mockHandler,
				}},
		},
		{
			// 比 /order/* 多了一段
			name:   "overflow",
			method: http.MethodPost,
			path:   "/order/delete/123",
			found:  true,
			mi: &matchInfo{
				n: &node{
					path:       "*",
					handlefunc: mockHandler,
				}},
		},
		// 参数匹配
		{
			// 命中 /param/:id
			name:   ":id",
			method: http.MethodGet,
			path:   "/param/123",
			found:  true,
			mi: &matchInfo{
				n: &node{
					path:       ":id",
					handlefunc: mockHandler,
				},
				pathParams: map[string]string{"id": "123"},
			},
		},
		{
			// 命中 /param/:id/*
			name:   ":id*",
			method: http.MethodGet,
			path:   "/param/123/abc",
			found:  true,
			mi: &matchInfo{
				n: &node{
					path:       "*",
					handlefunc: mockHandler,
				},
				pathParams: map[string]string{"id": "123"},
			},
		},

		{
			// 命中 /param/:id/detail
			name:   ":id*detail",
			method: http.MethodGet,
			path:   "/param/123/detail",
			found:  true,
			mi: &matchInfo{
				n: &node{
					path:       "detail",
					handlefunc: mockHandler,
				},
				pathParams: map[string]string{"id": "123"},
			},
		},
		{
			// 命中 /param/:id/detail/:username
			name:   ":id*detail:username",
			method: http.MethodGet,
			path:   "/param/123/detail/liming",
			found:  true,
			mi: &matchInfo{
				n: &node{
					path:       ":username",
					handlefunc: mockHandler,
				},
				pathParams: map[string]string{"id": "123", "username": "liming"},
			},
		},
		{
			// 命中 /param/:id/again/:id
			name:   ":id*again:id",
			method: http.MethodGet,
			path:   "/param/123/again/liming",
			found:  true,
			mi: &matchInfo{
				n: &node{
					path:       ":id",
					handlefunc: mockHandler,
				},
				pathParams: map[string]string{"id": "liming"},
			},
		},
		//正则匹配
		{
			// 命中 /reg/:id(.*)
			name:   ":id(.*)",
			method: http.MethodDelete,
			path:   "/reg/123",
			found:  true,
			mi: &matchInfo{
				n: &node{
					path:       ":id(.*)",
					handlefunc: mockHandler,
				},
				pathParams: map[string]string{"id": "123"},
			},
		},
		{
			// 命中 /:id([0-9]+)/home
			name:   ":id([0-9]+)",
			method: http.MethodDelete,
			path:   "/123/home",
			found:  true,
			mi: &matchInfo{
				n: &node{
					path:       "home",
					handlefunc: mockHandler,
				},
				pathParams: map[string]string{"id": "123"},
			},
		},
		// {
		// 	// 未命中 /:id([0-9]+)/home
		// 	name:   "not :id([0-9]+)",
		// 	method: http.MethodDelete,
		// 	path:   "/abc/home",
		// 	found:  false,
		// 	mi:     nil,
		// },
		//回溯算法 与 优先级
		// "/k1/k2/k3"
		//"/k1/k2/:id(.*)"
		//"/k1/k2/:id"
		// "/k1/:reg(.*)/k3"
		//"/k1/:parm/k2"
		// "/k1/*/k1"
		// "/k1/*"

		//优先级
		{
			// 命中 "/k1/k2/k3"
			name:   "/k1/k2/k3",
			method: http.MethodGet,
			path:   "k1/k2/k3",
			found:  true,
			mi: &matchInfo{
				n: &node{
					path:       "k3",
					handlefunc: mockHandler,
				},
			},
		},
		{ //"/k1/k2/:id(.*)" 和 "/k1/k2/:id" 竞争，优先正则
			// 命中  "/k1/k2/:id(.*)"
			name:   "/k1/k2/:id(.*)",
			method: http.MethodGet,
			path:   "k1/k2/123",
			found:  true,
			mi: &matchInfo{
				n: &node{
					path:       ":id(.*)",
					handlefunc: mockHandler,
				},
				pathParams: map[string]string{"id": "123"},
			},
		},
		//回溯算法
		{
			// 命中  "/k1/:reg(.*)/k3"
			name:   "/k1/:reg(.*)/k3",
			method: http.MethodGet,
			path:   "k1/123/k3",
			found:  true,
			mi: &matchInfo{
				n: &node{
					path:       "k3",
					handlefunc: mockHandler,
				},
				pathParams: map[string]string{"reg": "123"},
			},
		},
		{
			// 命中  "/k1/:parm/k2"
			name:   "/k1/:parm/k2",
			method: http.MethodGet,
			path:   "k1/123/k2",
			found:  true,
			mi: &matchInfo{
				n: &node{
					path:       "k2",
					handlefunc: mockHandler,
				},
				pathParams: map[string]string{"parm": "123"},
			},
		},
		{
			// 命中  "/k1/*/k1"
			name:   "/k1/*/k1",
			method: http.MethodGet,
			path:   "k1/123/k1",
			found:  true,
			mi: &matchInfo{
				n: &node{
					path:       "k1",
					handlefunc: mockHandler,
				},
			},
		},
		{
			// 命中  "/k1/*"
			name:   "/k1/*",
			method: http.MethodGet,
			path:   "k1/k123/k0",
			found:  true,
			mi: &matchInfo{
				n: &node{
					path:       "*",
					handlefunc: mockHandler,
				},
			},
		},
	}

	b.StopTimer()
	r := newRouter()
	for _, tr := range testRoutes {
		r.Route(tr.method, tr.path, mockHandler)
	}
	b.StartTimer()
	//测试用例

	for i := 0; i <= b.N; i++ {
		r := newRouter()
		for _, tc := range testCases {
			_, _ = r.findRouter(tc.method, tc.path)
		}
	}
}
