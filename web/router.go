package web

import (
	"fmt"
	"regexp"
	"strings"
)

// 抽象

// 处理函数抽象
type HandleFunc func(ctx *Context)

// // 包装一下HandlerBasedonMap,当前过于依赖HandlerBasedonMap的struct，设计应当依赖于接口
// type Router interface {
// 	http.Handler // 组合原有serverHttp接口方法，用于实现：“路由的分发功能”——DefaultServerMux
// 	Routable     // 组合原有sdkHttpServer的route注册功能，用于实现：“路由的注册&查询功能”——HandleFunc
// }

// 将原有sdkHttpServer的route注册功能包装进Handler接口，避免Server直接调用HandlerBasedonMap的内部结构
type Routable interface {
	Route(method string, pattern string, handlefunc HandleFunc)  //路由注册功能
	findRouter(method string, pattern string) (*matchInfo, bool) // 路由查找功能
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

// 确保router实现Routable接口
var _ Routable = &router{}

// addRoute 注册路由。
// method 是 HTTP 方法
// 静态匹配
// - path 必须以 / 开始并且结尾不能有 /，中间也不允许有连续的 /
// - 已经注册了的路由，无法被覆盖。例如 /user/home 注册两次，会冲突
// - 不支持大小写忽略
// 通配符匹配
// - 不支持 a/b/c & a/*/* 两个路由同时注册下, a/b/d 匹配（即不回溯匹配）
// - a/*/c 不支持 a/b1/b2/c ,但 a/b/* 支持 a/b/c1/c2 , 末尾支持多段匹配
// 参数匹配
// - 不能在同一个位置注册不同的参数路由，例如 /user/:id 和 /user/:name 冲突
// - 同名路径参数，在路由匹配的时候，值会被覆盖。例如 /user/:id/abc/:id，那么 /user/123/abc/456 最终 id = 456
// 正则匹配
// -不支持重复路由（？）
// 整体
// - 正则，参数，通配符 不能注册在同一节点
// - 正则，参数，通配符 不支持重复注册
// - 不支持并发实现注册（即服务器启动后的注册新路由）（？）
func (ro *router) Route(method string, pattern string, handlefunc HandleFunc) {
	//禁止非正常格式路由
	if pattern == "" {
		panic("web:路由是空字符串")
	}
	if pattern[0] != '/' {
		panic("web: 路由必须以 / 开头")
	}
	if pattern != "/" && pattern[len(pattern)-1] == '/' {
		panic("web: 路由不能以 / 结尾")
	}

	//根节点：判断是否已经注册method的方法树，方法树root应当是"/"的node
	root, ok := ro.trees[method]
	if !ok { //全新的Method方法
		//创建默认的'/'node节点
		root = &node{path: "/", route: "/"}
		ro.trees[method] = root
	}
	if pattern == "/" {
		if root.handlefunc != nil {
			panic("web: 路由冲突[/]")
		}
		root.handlefunc = handlefunc
		return
	}
	routepath := ""
	//非根节点：静态/正则/参数/通配符
	segs := strings.Split(pattern[1:], "/") //去除第一个“/”，根节点已经特殊处理了
	// segs := strings.Split(strings.Trim(pattern, "/"), "/") //不采用，因为会去除首尾的所有的"/", 错误路由例如"//user/post/"=>"user/post" 无法识别
	for _, seg := range segs {
		if seg == "" {
			panic(fmt.Sprintf("web: 非法路由。不允许使用 //a/b, /a//b 之类的路由, [%s]", pattern))
		}
		root = root.ChildOrCreate(seg)
		routepath += "/"
		routepath += seg
		root.route = routepath
	}

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

	mi := matchInfo{}
	segs := strings.Split(strings.Trim(pattern, "/"), "/") //去除首尾的"/", 例如"/user/post/"=>"user/post"=>[user,post]
	for _, seg := range segs {
		var n *node
		var isFound bool
		n, isFound = root.Childof(seg)
		if !isFound {
			if root.typ == nodetypeStar {
				mi.n = root
				return &mi, true
			}
			return nil, false
		}
		if n.paramName != "" {
			mi.addValue(n.paramName, seg)
		}
		root = n
	}
	mi.n = root
	return &mi, true
}

// 路由类型
type nodetype int

const (
	//静态路由
	nodetypeStatic = iota
	//正则路由
	nodetypeRegexp
	//参数路由
	nodetypeParam
	//通配符路由
	nodetypeStar
)

// node 代表路由树的节点
// 路由树的匹配顺序是：
// 1. 静态完全匹配
// 2. 正则匹配，形式 :param_name(reg_expr)
// 3. 路径参数匹配：形式 :param_name
// 4. 通配符匹配：*
// 这是不回溯匹配
type node struct {
	typ   nodetype //路由类型(必填)
	path  string   //path URL路径(必填)
	route string   //根节点到此节点的完整路由路径(必填)

	handlefunc HandleFunc //命中路由后的处理函数

	children  map[string]*node // 静态匹配(子path到子节点的映射)
	starChild *node            // 通配符匹配 *

	paramChild  *node          // 路径参数匹配 :param
	regexpChild *node          // 正则匹配 :paramname(regExpr)
	regExpr     *regexp.Regexp // 正则路由表达式  regExpr
	paramName   string         // 参数名称 => 正则和参数匹配都可以使用
}

// childof 查找并返回子节点 *node
// first bool 返回确认是否找到对应节点
func (n *node) Childof(pattern string) (node *node, isFound bool) {
	if n.children == nil { //无子节点
		return n.childOfNonStatic(pattern)
	}
	res, ok := n.children[pattern]
	if !ok { //子节点未找到对应path 的node
		return n.childOfNonStatic(pattern)
	}
	return res, ok // 找到对应node即返回
}

// childOfNonStatic 从非静态匹配的子节点里面查找
func (n *node) childOfNonStatic(pattern string) (*node, bool) {
	if n.regexpChild != nil {
		ismatch := n.regExpr.MatchString(pattern)
		if ismatch { // 正则匹配符合
			return n.regexpChild, true
		}
	}
	if n.paramChild != nil { //参数匹配符合
		return n.paramChild, true
	}
	return n.starChild, (n.starChild != nil) //有通配符就返回通配符，无通配符就返回失败
}

// childOrCreate 查找子节点，
// 首先会判断 path 是不是通配符路径
// 其次判断 path 是不是正则和参数路径，即以 : 开头的路径
// 最后会从 children 里面查找，
// 如果没有找到，那么会创建一个新的节点，并且保存在 node 里面
func (n *node) ChildOrCreate(pattern string) *node {
	//先确认是参数匹配还是正则匹配
	if pattern[0] == ':' {
		param_Name, expr, isReg := parseFaram(pattern)
		if isReg { //正则匹配 regexpChild
			return n.childOrCreateReg(pattern, expr, param_Name)
		} else { //参数匹配 paramChild
			return n.childOrCreateParam(pattern, param_Name)
		}
	}
	//再确认通配符 starChild
	if pattern == "*" {
		if n.paramChild != nil {
			panic(fmt.Sprintf("web: 非法路由，已有参数路由。不允许同时注册参数路由和通配符路由 [%s]", pattern))
		}
		if n.regexpChild != nil {
			panic(fmt.Sprintf("web: 非法路由，已有正则路由。不允许同时注册正则路由和通配符路由 [%s]", pattern))
		}
		if n.starChild == nil {
			n.starChild = &node{path: "*", typ: nodetypeStar}
		}
		return n.starChild
	}
	// 最后确认是否是子节点 children
	if n.children == nil { //无子节点
		n.children = make(map[string]*node)
	}
	root, ok := n.children[pattern]
	if !ok { //如果没有找到，那么会创建一个新的节点node
		n.children[pattern] = &node{path: pattern, typ: nodetypeStatic}
		return n.children[pattern]
	}
	//Children 找到了就直接返回
	return root
}

func (n *node) childOrCreateReg(pattern string, expr *regexp.Regexp, param_Name string) *node {
	if n.paramChild != nil {
		panic(fmt.Sprintf("web: 非法路由，已有参数路由。不允许同时注册参数路由和正则路由 [%s]", pattern))
	}
	if n.starChild != nil {
		panic(fmt.Sprintf("web: 非法路由，已有通配符路由。不允许同时注册通配符路由和正则路由 [%s]", pattern))
	}
	if n.regexpChild != nil {
		if n.regexpChild.path != pattern || n.regExpr.String() != expr.String() {
			panic(fmt.Sprintf("web: 路由冲突，正则路由冲突，已有 %s, 新注册 %s", n.regexpChild.path, pattern))
		}
	} else {
		n.regExpr = expr
		n.regexpChild = &node{path: pattern, typ: nodetypeRegexp, paramName: param_Name}
	}
	return n.regexpChild
}

func (n *node) childOrCreateParam(pattern string, param_Name string) *node {
	if n.regexpChild != nil {
		panic(fmt.Sprintf("web: 非法路由，已有正则路由。不允许同时注册正则路由和参数路由 [%s]", pattern))
	}
	if n.starChild != nil {
		panic(fmt.Sprintf("web: 非法路由，已有通配符路由。不允许同时注册通配符路由和参数路由 [%s]", pattern))
	}
	if n.paramChild != nil {
		if n.paramChild.path != pattern {
			panic(fmt.Sprintf("web: 路由冲突，参数路由冲突，已有 %s, 新注册 %s", n.paramChild.path, pattern))
		}
	} else {
		n.paramChild = &node{path: pattern, typ: nodetypeParam, paramName: param_Name}
	}
	return n.paramChild
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

// string 匹配的路径名（也是参数的存储的变量名）
// *regexp.Regexp 指编译好的regexp表达式
// bool 确认是否为正则匹配
func parseFaram(pattern string) (string, *regexp.Regexp, bool) {
	//去除:
	param := pattern[1:]                    // 参数:id & 正则 :name(Expr)
	params := strings.SplitN(param, "(", 2) //最多分成2块，因为可能正则里会嵌套（）:username((?=[123]).*)
	if len(params) == 2 {                   //正则
		if strings.HasSuffix(params[1], ")") { // (?=[123]).*)
			expr := strings.TrimRight(params[1], ")") //(?=[123]).*
			regexpr := regexp.MustCompile(expr)
			return params[0], regexpr, true
		}
	}
	return param, nil, false // 参数
}
