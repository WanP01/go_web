package web

import (
	"fmt"
	"net/http"
	"strings"
)

// 抽象

// 处理函数抽象
type HandleFunc func(ctx *Context)

// 包装一下HandlerBasedonMap,当前过于依赖HandlerBasedonMap的struct，设计应当依赖于接口
type Router interface {
	http.Handler // 组合原有serverHttp接口方法，用于实现：“路由的分发功能”——DefaultServerMux
	Routable     // 组合原有sdkHttpServer的route注册功能，用于实现：“路由的注册&查询功能”——HandleFunc
}

// 将原有sdkHttpServer的route注册功能包装进Handler接口，避免Server直接调用HandlerBasedonMap的内部结构
type Routable interface {
	Route(method string, pattern string, handlefunc func(c *Context)) //路由注册功能
	// Route 注册一个路由
	// method 是 HTTP 方法
	// 我们并不采取这种设计方案（多路由处理函数）
	// Route(method string, path string, handlers... HandleFunc)
}

// 用于支持路由树的操作
type router struct {
	trees map[string]*node // http method => 路由树根节点
}

// 路由初始化
func newRouter() *router {
	return &router{
		trees: map[string]*node{},
	}
}

// 路由树分发匹配路径功能
func (ro *router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := NewContext(w, r)
	root, ok := ro.findRouter(ctx.R.Method, ctx.R.URL.Path)
	if !ok || root.handlefunc == nil {
		ctx.W.WriteHeader(http.StatusNotFound)
		ctx.W.Write([]byte("你找的页面未发现"))
		return
	}
	root.handlefunc(ctx)
}

// 路由树注册功能
func (ro *router) Route(method string, pattern string, handlefunc func(c *Context)) {
	//用户注册路由时可以针对格式提要求
	if pattern == "" {
		panic("web:路由是空字符串")
	}
	if pattern[0] != '/' {
		panic("web: 路由必须以 / 开头")
	}
	if pattern != "/" && pattern[len(pattern)-1] == '/' {
		panic("web: 路由不能以 / 结尾")
	}
	//判断是否已经注册method的方法树，方法树root应当是"/"的node
	root, ok := ro.trees[method]
	if !ok { //全新的Method方法
		//创建默认的'/'node节点
		root = &node{path: "/"}
		ro.trees[method] = root
	}
	if pattern == "/" {
		if root.handlefunc != nil {
			panic("web: 路由冲突[/]")
		}
		root.handlefunc = handlefunc
		return
	}
	segs := strings.Split(pattern[1:], "/")
	// segs := strings.Split(strings.Trim(pattern, "/"), "/") //不采用，因为会去除首尾的所有的"/", 错误路由例如"//user/post/"=>"user/post" 无法识别
	for _, seg := range segs {
		if seg == "" {
			panic(fmt.Sprintf("web: 非法路由。不允许使用 //a/b, /a//b 之类的路由, [%s]", pattern))
		}
		root = root.ChildOrCreate(seg)
	}
	if root.handlefunc != nil {
		//已经有注册了
		panic(fmt.Sprintf("web: 与现已有的路由冲突[%s]", pattern))
	}
	root.handlefunc = handlefunc
}

// 额外抽象的寻找路由的函数
func (ro *router) findRouter(method string, pattern string) (*node, bool) {
	root, ok := ro.trees[method]
	if !ok {
		return nil, false
	}
	if pattern == "/" {
		return root, true
	}
	// 客户查询路由时格式可能多种多样
	segs := strings.Split(strings.Trim(pattern, "/"), "/") //去除首尾的"/", 例如"/user/post/"=>"user/post"=>[user,post]
	for _, seg := range segs {
		root, ok = root.Childof(seg)
		if !ok {
			return nil, false
		}
	}
	return root, true
}

// 判断路由子节点
func (n *node) Childof(pattern string) (*node, bool) {
	if n.children == nil {
		return nil, false
	}
	res, ok := n.children[pattern]
	return res, ok
}

// 判断并创造对应路由子节点
func (n *node) ChildOrCreate(pattern string) *node {
	if n.children == nil {
		n.children = make(map[string]*node)
	}
	root, ok := n.children[pattern]
	if !ok {
		n.children[pattern] = &node{path: pattern}
		return n.children[pattern]
	}
	return root
}

// 路由树实现子节点
type node struct {
	path       string           // path URL路径
	children   map[string]*node //子path到子节点的映射
	handlefunc HandleFunc       //命中路由后的处理函数
}

// node 两种形态：
//1. 最后的节点（没有子节点）
// type node struct {
// 	path       lastURL
// 	children   nil
// 	handlefunc  HandleFunc
// }
//2. 中间的节点(可能有或没有处理函数)
// type node struct {
// 	path       MiddleURL
// 	children   map[string]*node
// 	handlefunc  handleFunc
// }
