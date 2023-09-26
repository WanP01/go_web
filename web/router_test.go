package web

import (
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoute(t *testing.T) {
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
	r := newRouter()
	for _, tr := range testRoutes {
		r.Route(tr.method, tr.path, mockhandler)
	}

	wantRouter := router{
		trees: map[string]*node{
			http.MethodGet: {
				path: "/", handlefunc: mockhandler,
				children: map[string]*node{
					"user": {
						path: "user",
						children: map[string]*node{
							"home": {
								path: "home", typ: nodetypeStatic,
								handlefunc: mockhandler},
						},
						handlefunc: mockhandler},
					"order": {
						path: "order",
						children: map[string]*node{
							"detail": {
								path: "detail", typ: nodetypeStatic,
								handlefunc: mockhandler},
						},
						starChild: &node{path: "*", handlefunc: mockhandler, typ: nodetypeStar}},
					"param": {
						path: "param", typ: nodetypeStatic,
						paramChild: &node{
							path: ":id", handlefunc: mockhandler, typ: nodetypeParam, paramName: "id",
							children: map[string]*node{
								"detail": {
									path: "detail", handlefunc: mockhandler, typ: nodetypeStatic,
									paramChild: &node{path: ":username", handlefunc: mockhandler, typ: nodetypeParam, paramName: "username"},
								},
							},
							starChild: &node{path: "*", handlefunc: mockhandler, typ: nodetypeStar},
						}},
				},
				starChild: &node{
					path: "*", typ: nodetypeStar,
					handlefunc: mockhandler,
					starChild: &node{
						path: "*", handlefunc: mockhandler, typ: nodetypeStar},
					children: map[string]*node{
						"abc": {
							path: "abc", handlefunc: mockhandler, typ: nodetypeStatic,
							starChild: &node{
								path: "*", handlefunc: mockhandler, typ: nodetypeStar,
							},
						},
					},
				},
			},
			http.MethodPost: {
				path: "/",
				children: map[string]*node{
					"order": {
						path: "order", typ: nodetypeStatic,
						children: map[string]*node{
							"create": {
								path: "create", typ: nodetypeStatic,
								handlefunc: mockhandler},
						},
					},
					"login": {
						path: "login", typ: nodetypeStatic,
						handlefunc: mockhandler},
				},
			},
			http.MethodDelete: {
				path: "/",
				children: map[string]*node{
					"reg": {
						path: "reg", regExpr: regexp.MustCompile(".*"), typ: nodetypeStatic,
						regexpChild: &node{path: ":id(.*)", paramName: "id", handlefunc: mockhandler, typ: nodetypeRegexp},
					},
				},
				regExpr: regexp.MustCompile("^.+$"),
				regexpChild: &node{typ: nodetypeRegexp, path: ":name(^.+$)", paramName: "name",
					children: map[string]*node{"abc": {path: "abc", typ: nodetypeStatic, handlefunc: mockhandler}},
				},
			},
		},
	}

	msg, ok := wantRouter.rootEqual(r)
	assert.True(t, ok, msg)

	// 非法用例 **************************************************
	r = newRouter()
	// r.Route(http.MethodGet, "a/b/c", mockhandler)
	// 空字符串
	assert.PanicsWithValue(t, "web:路由是空字符串", func() {
		r.Route(http.MethodGet, "", mockhandler)
	})

	// 前导没有 /
	assert.PanicsWithValue(t, "web: 路由必须以 / 开头", func() {
		r.Route(http.MethodGet, "a/b/c", mockhandler)
	})

	// 后缀有 /
	assert.PanicsWithValue(t, "web: 路由不能以 / 结尾", func() {
		r.Route(http.MethodGet, "/a/b/c/", mockhandler)
	})

	// 根节点重复注册
	r.Route(http.MethodGet, "/", mockhandler)
	assert.PanicsWithValue(t, "web: 路由冲突[/]", func() {
		r.Route(http.MethodGet, "/", mockhandler)
	})
	// 普通节点重复注册
	r.Route(http.MethodGet, "/a/b/c", mockhandler)
	assert.PanicsWithValue(t, "web: 与现已有的路由冲突[/a/b/c]", func() {
		r.Route(http.MethodGet, "/a/b/c", mockhandler)
	})
	// 参数节点重复注册
	r = newRouter()
	r.Route(http.MethodGet, "/a/:id", mockhandler)
	assert.PanicsWithValue(t, "web: 与现已有的路由冲突[/a/:id]", func() {
		r.Route(http.MethodGet, "/a/:id", mockhandler)
	})
	// 通配符节点重复注册
	r = newRouter()
	r.Route(http.MethodGet, "/a/*", mockhandler)
	assert.PanicsWithValue(t, "web: 与现已有的路由冲突[/a/*]", func() {
		r.Route(http.MethodGet, "/a/*", mockhandler)
	})
	// 正则符节点重复注册
	r = newRouter()
	r.Route(http.MethodGet, "/a/:id(.*)", mockhandler)
	assert.PanicsWithValue(t, "web: 与现已有的路由冲突[/a/:id(.*)]", func() {
		r.Route(http.MethodGet, "/a/:id(.*)", mockhandler)
	})

	// 多个 /
	assert.PanicsWithValue(t, "web: 非法路由。不允许使用 //a/b, /a//b 之类的路由, [/a//b]", func() {
		r.Route(http.MethodGet, "/a//b", mockhandler)
	})
	assert.PanicsWithValue(t, "web: 非法路由。不允许使用 //a/b, /a//b 之类的路由, [//a/b]", func() {
		r.Route(http.MethodGet, "//a/b", mockhandler)
	})

	// 同时注册通配符路由和参数路由
	r = newRouter()
	assert.PanicsWithValue(t, "web: 非法路由，已有通配符路由。不允许同时注册通配符路由和参数路由 [:id]", func() {
		r.Route(http.MethodGet, "/a/*", mockhandler)
		r.Route(http.MethodGet, "/a/:id", mockhandler)
	})
	r = newRouter()
	assert.PanicsWithValue(t, "web: 非法路由，已有参数路由。不允许同时注册参数路由和通配符路由 [*]", func() {
		r.Route(http.MethodGet, "/a/b/:id", mockhandler)
		r.Route(http.MethodGet, "/a/b/*", mockhandler)
	})
	r = newRouter()
	assert.PanicsWithValue(t, "web: 非法路由，已有通配符路由。不允许同时注册通配符路由和参数路由 [:id]", func() {
		r.Route(http.MethodGet, "/*", mockhandler)
		r.Route(http.MethodGet, "/:id", mockhandler)
	})
	r = newRouter()
	assert.PanicsWithValue(t, "web: 非法路由，已有参数路由。不允许同时注册参数路由和通配符路由 [*]", func() {
		r.Route(http.MethodGet, "/:id", mockhandler)
		r.Route(http.MethodGet, "/*", mockhandler)
	})
	// 同时注册通配符路由，参数路由，正则路由
	r = newRouter()
	assert.PanicsWithValue(t, "web: 非法路由，已有通配符路由。不允许同时注册通配符路由和正则路由 [:id(.*)]", func() {
		r.Route(http.MethodGet, "/a/b/*", mockhandler)
		r.Route(http.MethodGet, "/a/b/:id(.*)", mockhandler)
	})
	r = newRouter()
	assert.PanicsWithValue(t, "web: 非法路由，已有参数路由。不允许同时注册参数路由和正则路由 [:id(.*)]", func() {
		r.Route(http.MethodGet, "/a/b/:id", mockhandler)
		r.Route(http.MethodGet, "/a/b/:id(.*)", mockhandler)
	})
	r = newRouter()
	assert.PanicsWithValue(t, "web: 非法路由，已有正则路由。不允许同时注册正则路由和通配符路由 [*]", func() {
		r.Route(http.MethodGet, "/a/b/:id(.*)", mockhandler)
		r.Route(http.MethodGet, "/a/b/*", mockhandler)
	})
	r = newRouter()
	assert.PanicsWithValue(t, "web: 非法路由，已有正则路由。不允许同时注册正则路由和参数路由 [:id]", func() {
		r.Route(http.MethodGet, "/a/b/:id(.*)", mockhandler)
		r.Route(http.MethodGet, "/a/b/:id", mockhandler)
	})

	// 参数冲突
	assert.PanicsWithValue(t, "web: 路由冲突，参数路由冲突，已有 :id, 新注册 :name", func() {
		r.Route(http.MethodGet, "/a/b/c/:id", mockhandler)
		r.Route(http.MethodGet, "/a/b/c/:name", mockhandler)
	})
	// ****************************************************************************
}

func (r *router) rootEqual(w *router) (msg string, ok bool) {
	for k, v := range r.trees {
		dst, ok := w.trees[k]
		if !ok {
			return fmt.Sprint("router trees is not equal:http method is not Found"), false
		}
		msg, ok := v.nodeEqual(dst)
		if !ok {
			return k + "-" + msg, ok
		}
	}
	return "", true
}

func (n *node) nodeEqual(w *node) (msg string, ok bool) {
	if w == nil {
		return "目标节点为 nil", false
	}
	//比较typ
	if n.typ != w.typ { //
		return fmt.Sprintf("%s 与 %s节点 typ 不相等", n.path, w.path), false
	}
	//比较path
	if n.path != w.path {
		return fmt.Sprintf("%s 与 %s节点 path 不相等", n.path, w.path), false
	}
	// 比较handlefunc
	nf := reflect.ValueOf(n.handlefunc)
	wf := reflect.ValueOf(w.handlefunc)
	if nf != wf {
		return fmt.Sprintf("%s 与 %s节点 handlefunc %s and %s 不相等 ", n.path, w.path, nf.Type().String(), wf.Type().String()), false
	}
	//比较starChild
	if n.starChild != nil {
		msg, ok := n.starChild.nodeEqual(w.starChild)
		if !ok {
			return fmt.Sprintf("%s 与 %s节点 starChild 不相等,%s", n.path, w.path, msg), false
		}
	} else {
		if w.starChild != nil {
			return fmt.Sprintf("%s 与 %s节点 starChild 不相等,%s", n.path, w.path, msg), false
		}
	}
	// 比较regexpChild
	if n.regexpChild != nil {
		msg, ok := n.regexpChild.nodeEqual(w.regexpChild)
		if !ok {
			return fmt.Sprintf("%s 与 %s节点 regexpChild 不相等,%s", n.path, w.path, msg), false
		}
	} else {
		if w.regexpChild != nil {
			return fmt.Sprintf("%s 与 %s节点 regexpChild 不相等,%s", n.path, w.path, msg), false
		}
	}
	//比较paramName
	if n.paramName != w.paramName {
		return fmt.Sprintf("%s 与 %s节点 paramName 不相等,%s", n.path, w.path, msg), false
	}
	// 比较paramChild
	if n.paramChild != nil {
		msg, ok := n.paramChild.nodeEqual(w.paramChild)
		if !ok {
			return fmt.Sprintf("%s 与 %s节点 paramChild 不相等,%s", n.path, w.path, msg), false
		}
	} else {
		if w.paramChild != nil {
			return fmt.Sprintf("%s 与 %s节点 paramChild 不相等,%s", n.path, w.path, msg), false
		}
	}
	//比较children
	if len(n.children) != len(w.children) {
		return fmt.Sprintf("%s and %s子节点长度不等", n.path, w.path), false
	}

	for k, v := range n.children {
		dst, ok := w.children[k]
		if !ok {
			return fmt.Sprintf("%s 目标节点缺少子节点 %s", n.path, k), false
		}
		msg, ok := v.nodeEqual(dst)
		if !ok {
			return n.path + "-" + msg, ok
		}
	}
	return "", true
}

func TestFindRoute(t *testing.T) {
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
		{
			// 未命中 /:id([0-9]+)/home
			name:   "not :id([0-9]+)",
			method: http.MethodDelete,
			path:   "/abc/home",
			found:  false,
			mi:     nil,
		},
	}
	r := newRouter()
	for _, tr := range testRoutes {
		r.Route(tr.method, tr.path, mockHandler)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mi, found := r.findRouter(tc.method, tc.path)
			assert.Equal(t, tc.found, found)
			if !found {
				return
			}
			assert.Equal(t, tc.mi.pathParams, mi.pathParams)
			wantVal := reflect.ValueOf(tc.mi.n.handlefunc)
			nVal := reflect.ValueOf(mi.n.handlefunc)
			assert.Equal(t, wantVal, nVal)
		})

	}
}
