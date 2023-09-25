package web

import (
	"fmt"
	"net/http"
	"reflect"
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
	}

	//初始化Router，循环注册路由示例testCase
	mockhandler := func(ctx *Context) {}
	r := newRouter()
	for _, tr := range testRoutes {
		r.Route(tr.method, tr.path, mockhandler)
	}

	wantRouter := router{
		trees: map[string]*node{
			http.MethodGet: &node{
				path: "/",
				children: map[string]*node{
					"user": &node{
						path: "user",
						children: map[string]*node{
							"home": &node{
								path:       "home",
								handlefunc: mockhandler},
						},
						handlefunc: mockhandler},
					"order": &node{
						path: "order",
						children: map[string]*node{
							"detail": &node{
								path:       "detail",
								handlefunc: mockhandler},
						},
						starChild: &node{path: "*", handlefunc: mockhandler}},
				},
				starChild: &node{
					path:       "*",
					handlefunc: mockhandler,
					starChild: &node{
						path: "*", handlefunc: mockhandler},
					children: map[string]*node{
						"abc": &node{
							path: "abc", handlefunc: mockhandler,
							starChild: &node{
								path: "*", handlefunc: mockhandler,
							},
						},
					},
				},
				handlefunc: mockhandler,
			},
			http.MethodPost: &node{
				path: "/",
				children: map[string]*node{
					"order": &node{
						path: "order",
						children: map[string]*node{
							"create": &node{
								path:       "create",
								handlefunc: mockhandler},
						},
					},
					"login": &node{
						path:       "login",
						handlefunc: mockhandler},
				},
			},
		},
	}

	msg, ok := wantRouter.rootEqual(r)
	assert.True(t, ok, msg)
	// 非法用例
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

	// 多个 /
	assert.PanicsWithValue(t, "web: 非法路由。不允许使用 //a/b, /a//b 之类的路由, [/a//b]", func() {
		r.Route(http.MethodGet, "/a//b", mockhandler)
	})
	assert.PanicsWithValue(t, "web: 非法路由。不允许使用 //a/b, /a//b 之类的路由, [//a/b]", func() {
		r.Route(http.MethodGet, "//a/b", mockhandler)
	})
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
	}

	mockHandler := func(ctx *Context) {}

	testCases := []struct {
		name     string
		method   string
		path     string
		found    bool
		wantNode *node
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
			wantNode: &node{
				path:       "/",
				handlefunc: mockHandler,
			},
		},
		{
			name:   "user",
			method: http.MethodGet,
			path:   "/user",
			found:  true,
			wantNode: &node{
				path:       "user",
				handlefunc: mockHandler,
			},
		},
		{
			name:   "no handler",
			method: http.MethodPost,
			path:   "/order",
			found:  true,
			wantNode: &node{
				path: "order",
			},
		},
		{
			name:   "two layer",
			method: http.MethodPost,
			path:   "/order/create",
			found:  true,
			wantNode: &node{
				path:       "create",
				handlefunc: mockHandler,
			},
		},
		// 通配符匹配
		{
			// 命中/order/*
			name:   "star match",
			method: http.MethodPost,
			path:   "/order/delete",
			found:  true,
			wantNode: &node{
				path:       "*",
				handlefunc: mockHandler,
			},
		},
		{
			// 命中通配符在中间的
			// /user/*/home
			name:   "star in middle",
			method: http.MethodGet,
			path:   "/user/Tom/home",
			found:  true,
			wantNode: &node{
				path:       "home",
				handlefunc: mockHandler,
			},
		},
		{
			// 比 /order/* 多了一段
			name:     "overflow",
			method:   http.MethodPost,
			path:     "/order/delete/123",
			found:    false,
			wantNode: nil,
		},
	}
	r := newRouter()
	for _, tr := range testRoutes {
		r.Route(tr.method, tr.path, mockHandler)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			n, found := r.findRouter(tc.method, tc.path)
			assert.Equal(t, tc.found, found)
			if !found {
				return
			}
			assert.Equal(t, tc.wantNode.path, n.path)
			wantVal := reflect.ValueOf(tc.wantNode.handlefunc)
			nVal := reflect.ValueOf(n.handlefunc)
			assert.Equal(t, wantVal, nVal)
		})

	}
}
