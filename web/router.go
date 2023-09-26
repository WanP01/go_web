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
	mi, ok := ro.findRouter(ctx.R.Method, ctx.R.URL.Path)
	if !ok || mi.n.handlefunc == nil {
		ctx.W.WriteHeader(http.StatusNotFound)
		ctx.W.Write([]byte("你找的页面未发现"))
		return
	}
	ctx.pathParams = mi.pathParams
	mi.n.handlefunc(ctx)
}

// addRoute 注册路由。
// method 是 HTTP 方法
// - path 必须以 / 开始并且结尾不能有 /，中间也不允许有连续的 /
// - 已经注册了的路由，无法被覆盖。例如 /user/home 注册两次，会冲突
// - 不能在同一个位置注册不同的参数路由，例如 /user/:id 和 /user/:name 冲突
// - 不能在同一个位置同时注册通配符路由和参数路由，例如 /user/:id 和 /user/* 冲突
// - 同名路径参数，在路由匹配的时候，值会被覆盖。例如 /user/:id/abc/:id，那么 /user/123/abc/456 最终 id = 456
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
	// paramChild 参数匹配要多一步，需要保存匹配到的参数

	if root.handlefunc != nil {
		//已经有注册了
		panic(fmt.Sprintf("web: 与现已有的路由冲突[%s]", pattern))
	}
	root.handlefunc = handlefunc
}

// findRoute 查找对应的节点
// 注意，返回的 node 内部 HandleFunc 不为 nil 才算是注册了路由，findout只管查找结点，不负责确认handlefunc 是否存在，即不区分中间节点和末尾节点
func (ro *router) findRouter(method string, pattern string) (*matchInfo, bool) {
	root, ok := ro.trees[method]
	if !ok {
		return nil, false
	}
	if pattern == "/" {
		return &matchInfo{n: root}, true
	}
	// 客户查询路由时格式可能多种多样
	mi := matchInfo{}
	segs := strings.Split(strings.Trim(pattern, "/"), "/") //去除首尾的"/", 例如"/user/post/"=>"user/post"=>[user,post]
	for _, seg := range segs {
		var isParamMatch bool
		root, ok, isParamMatch = root.Childof(seg)
		if !ok {
			return nil, false
		}
		if isParamMatch {
			mi.addValue(root.path[1:], seg)
		}
	}
	mi.n = root
	return &mi, true
}

// node 代表路由树的节点
// 路由树的匹配顺序是：
// 1. 静态完全匹配
// 2. 路径参数匹配：形式 :param_name
// 3. 通配符匹配：*
// **不支持** a/b/c & a/*/* 两个路由同时注册下, a/b/d 匹配（即不回溯匹配）
// **不支持** a/* 与 a/b/c 匹配
type node struct {
	path       string           // path URL路径
	children   map[string]*node //子path到子节点的映射
	handlefunc HandleFunc       //命中路由后的处理函数
	starChild  *node            // 通配符匹配 *
	paramChild *node            // 路径参数匹配 :id
}

// childof 查找并返回子节点 *node
// first bool 返回确认是否找到对应节点
// second bool 返回确认是否是 paramChild,触发参数保存操作
func (n *node) Childof(pattern string) (node *node, isFound bool, isParamMatch bool) {
	if n.children == nil { //无子节点
		if n.paramChild != nil { //参数匹配符合
			return n.paramChild, true, true
		}
		return n.starChild, (n.starChild != nil), false //有通配符就返回通配符，无通配符就返回失败
	}
	res, ok := n.children[pattern]
	if !ok { //子节点未找到对应path 的node
		if n.paramChild != nil { //参数匹配符合
			return n.paramChild, true, true
		}
		return n.starChild, (n.starChild != nil), false //有通配符就返回通配符，无通配符就返回失败
	}
	return res, ok, false // 找到对应node即返回
}

// childOrCreate 查找子节点，如果子节点不存在就创建一个,并且将子节点放回去了 children 中
// 不允许同时有参数匹配和通配符匹配,user/:username && user/* 不能同时存在
func (n *node) ChildOrCreate(pattern string) *node {
	//先确认参数匹配 paramChild
	if pattern[0] == ':' {
		if n.starChild != nil {
			panic(fmt.Sprintf("web: 非法路由，已有通配符路由。不允许同时注册通配符路由和参数路由 [%s]", pattern))
		}
		if n.paramChild != nil {
			if n.paramChild.path != pattern {
				panic(fmt.Sprintf("web: 路由冲突，参数路由冲突，已有 %s, 新注册 %s", n.paramChild.path, pattern))
			}
		} else {
			n.paramChild = &node{path: pattern}
		}
		return n.paramChild
	}
	//再确认通配符 starChild
	if pattern == "*" {
		if n.paramChild != nil {
			panic(fmt.Sprintf("web: 非法路由，已有路径参数路由。不允许同时注册通配符路由和参数路由 [%s]", pattern))
		}
		if n.starChild == nil {
			n.starChild = &node{path: "*"}
		}
		return n.starChild
	}
	// 最后确认是否是子节点
	if n.children == nil { //无子节点
		n.children = make(map[string]*node)
	}
	root, ok := n.children[pattern]
	if !ok { //如果没有找到，那么会创建一个新的节点node
		n.children[pattern] = &node{path: pattern}
		return n.children[pattern]
	}
	//Children 找到了就直接返回
	return root
}

// findrouter路由匹配 返回的结果
type matchInfo struct {
	n          *node             //返回找到的路由节点
	pathParams map[string]string //返回中间记录的参数匹配结果
}

func (m *matchInfo) addValue(pattern string, seg string) {
	if m.pathParams == nil {
		//支持不同参数，可能不止一段，即 user/:id/:username 多段参数匹配
		m.pathParams = map[string]string{}
	}
	m.pathParams[pattern] = seg // 相同命名参数仅保留最后的匹配数字
}
