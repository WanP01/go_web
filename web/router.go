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

// addRoute 注册路由。
// method 是 HTTP 方法
// path 必须以 / 开始并且结尾不能有 /，中间也不允许有连续的 /
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
	segs := strings.Split(pattern[1:], "/") //去除第一个“/”，根节点已经特殊处理了
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

// findRoute 查找对应的节点
// 注意，返回的 node 内部 HandleFunc 不为 nil 才算是注册了路由，findout只管查找结点，不负责确认handlefunc 是否存在，即不区分中间节点和末尾节点
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

// node 代表路由树的节点
// 路由树的匹配顺序是：
// 1. 静态完全匹配
// 2. 通配符匹配
// **不支持** a/b/c & a/*/* 两个路由同时注册下, a/b/d 匹配（即无法回溯）
// **不支持** a/* 与 a/b/c 匹配
type node struct {
	path       string           // path URL路径
	children   map[string]*node //子path到子节点的映射
	handlefunc HandleFunc       //命中路由后的处理函数
	starChild  *node            // 通配符匹配 *
	paramChild *node            // 路径参数匹配 :id
}

// childof 查找并返回子节点
func (n *node) Childof(pattern string) (*node, bool) {
	if n.children == nil { //无子节点
		// if n.starChild != nil {
		// 	return n.starChild,true
		// }
		// return nil, false
		return n.starChild, (n.starChild != nil) //有通配符就返回通配符，无通配符就返回失败
	}
	res, ok := n.children[pattern]
	if !ok { //子节点未找到对应path 的node
		return n.starChild, (n.starChild != nil) //有通配符就返回通配符，无通配符就返回失败
	}
	return res, ok // 找到对应node即返回
}

// childOrCreate 查找子节点，如果子节点不存在就创建一个
// 并且将子节点放回去了 children 中
func (n *node) ChildOrCreate(pattern string) *node {
	if pattern == "*" {
		if n.starChild == nil {
			n.starChild = &node{path: "*"}
		}
		return n.starChild
	}
	if n.children == nil { //无子节点
		n.children = make(map[string]*node)
	}
	root, ok := n.children[pattern]
	if !ok {
		n.children[pattern] = &node{path: pattern}
		return n.children[pattern]
	}
	return root
}
