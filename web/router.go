package web

import (
	"fmt"
	"regexp"
	"strings"
)

// 抽象

// HandleFunc 处理函数抽象
type HandleFunc func(ctx *Context)

// Routable 将原有ServerEngine的route注册功能包装进Handler接口，避免Server直接调用HandlerBasedonMap的内部结构
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

// Route addRoute 注册路由。
// method 是 HTTP 方法
// 静态匹配
// - path 必须以 / 开始并且结尾不能有 /，中间也不允许有连续的 /
// - 已经注册了的路由，无法被覆盖。例如 /user/home 注册两次，会冲突
// - 不支持大小写忽略
// 通配符匹配
// X 不支持 a/b/c & a/*/* 两个路由同时注册下, a/b/d 匹配（即不回溯匹配）
// - a/*/c 不支持 a/b1/b2/c ,但 a/b/* 支持 a/b/c1/c2 , 末尾支持多段匹配
// 参数匹配
// - 不能在同一个位置注册不同的参数路由，例如 /user/:id 和 /user/:name 冲突
// - 同名路径参数，在路由匹配的时候，值会被覆盖。例如 /user/:id/abc/:id，那么 /user/123/abc/456 最终 id = 456
// 正则匹配
// - 相同正则路由的情况下获得路由不确定（map遍历的不确定性）
// 整体
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
	root, ok := ro.trees[method] //先确认路由树是否有对应httpMethod
	if !ok {
		return nil, false
	}
	if pattern == "/" { // 根节点单独处理
		return &matchInfo{n: root}, true
	}
	mi := matchInfo{}
	segs := strings.Split(strings.Trim(pattern, "/"), "/") //去除首尾的"/", 例如"/user/post/"=>"user/post"=>[user,post]
	res, ok := root.BackSearchChild(segs, 0, &mi)
	if !ok { //没找到对应节点
		return nil, false
	}
	mi.n = res
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
// 这是不回溯匹配（已修改为回溯匹配）
type node struct {
	typ   nodetype //路由类型(必填)
	path  string   //path URL路径(必填)
	route string   //根节点到此节点的完整路由路径(必填)

	handlefunc HandleFunc //命中路由后的处理函数

	children  map[string]*node // 静态匹配(子path到子节点的映射)
	starChild *node            // 通配符匹配 *

	paramChild  *node            // 路径参数匹配 :param(一般只设一个就行，没有太多的必要)
	regexpChild map[string]*node // 正则匹配 :paramname(regExpr)
	regExpr     *regexp.Regexp   // 正则路由表达式  regExpr
	paramName   string           // 参数名称 => 正则和参数匹配都可以使用
}

// BackSearchChild 按照 静态，正则，参数，通配符的优先级递归遍历 route找到第一个符合的节点，通过回溯记录参数和正则匹配的信息
func (n *node) BackSearchChild(segs []string, cnt int, mi *matchInfo) (*node, bool) {

	if cnt == len(segs) {
		return n, true
	}
	//这一轮的对比URL pattern
	pattern := segs[cnt]
	// 先进入静态路由匹配
	if n.children != nil {
		if cn, ok := n.children[pattern]; ok {
			res, ok := cn.BackSearchChild(segs, cnt+1, mi)
			if ok {
				return res, true
			}
		}
	}
	// 正则匹配可能有多个
	if n.regexpChild != nil {
		for exprs, regn := range n.regexpChild { //遍历n.regexpChild看是否能匹配上
			ismatch := regexp.MustCompile(exprs).MatchString(pattern)
			if ismatch { //匹配上了，加入nodelist(可能的节点列表)
				mi.addValue(n.regexpChild[exprs].paramName, pattern)
				res, ok := regn.BackSearchChild(segs, cnt+1, mi)
				if ok {
					return res, true
				}
				mi.RemoveValue(n.regexpChild[exprs].paramName)
			} // 没有匹配上就跳过
		}
	}
	//参数匹配只有一个（:id和:name没有明显区别）
	if n.paramChild != nil {
		mi.addValue(n.paramChild.paramName, pattern)
		res, ok := n.paramChild.BackSearchChild(segs, cnt+1, mi)
		if ok {
			return res, true
		}
		mi.RemoveValue(n.paramChild.paramName)
	}
	//通配符匹配（注意末尾*可以匹配多段）
	if n.starChild != nil {
		res, ok := n.starChild.BackSearchChild(segs, cnt+1, mi)
		if ok {
			return res, true
		}
		// 末尾 * 通配符的情况下实现匹配
		if (n.starChild.typ == nodetypeStar) && (n.starChild.handlefunc != nil) {
			return n.starChild, true
		}
	}
	return nil, false
}

// Childof （已废弃） children of 层序遍历查找所有满足的子节点 *node
// bool 返回确认是否找到对应节点
func (n *node) Childof(pattern string) ([]*node, bool) {
	nodeList := make([]*node, 0)
	if n.children != nil { //静态匹配
		res, ok := n.children[pattern]
		if ok { //子节点未找到对应path 的node
			nodeList = append(nodeList, res)
		}
	}
	if n.regexpChild != nil { //正则匹配
		for exprs, regn := range n.regexpChild { //遍历n.regexpChild看是否能匹配上
			ismatch := regexp.MustCompile(exprs).MatchString(pattern)
			if ismatch { //匹配上了，加入nodelist(可能的节点列表)
				nodeList = append(nodeList, regn)
			} // 没有匹配上就跳过
		}
	}
	if n.paramChild != nil { // 参数匹配

		nodeList = append(nodeList, n.paramChild)

	}
	if n.starChild != nil { //通配符匹配
		nodeList = append(nodeList, n.starChild)
	}
	return nodeList, (len(nodeList) != 0) // 找到对应node即返回
}

// ChildOrCreate childOrCreate 查找子节点，
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
		if n.starChild == nil {
			n.starChild = &node{path: "*", typ: nodetypeStar}
		}
		return n.starChild
	}
	// 最后确认是否是子节点 children
	if n.children == nil { //无子节点
		n.children = make(map[string]*node)
		n.children[pattern] = &node{path: pattern, typ: nodetypeStatic}
		return n.children[pattern]
	}
	//有子节点搜索一下
	root, ok := n.children[pattern]
	if !ok { //如果没有找到，那么会创建一个新的节点node
		n.children[pattern] = &node{path: pattern, typ: nodetypeStatic}
		return n.children[pattern]
	}
	//Children 找到了就直接返回
	return root
}

func (n *node) childOrCreateReg(pattern string, expr *regexp.Regexp, param_Name string) *node {
	if n.regexpChild == nil {
		n.regexpChild = make(map[string]*node) //初始化,直接新建子node
		n.regexpChild[expr.String()] = &node{path: pattern, typ: nodetypeRegexp, paramName: param_Name, regExpr: expr}
	}
	if n.regexpChild != nil {
		regn, ok := n.regexpChild[expr.String()] //用对应这个子node的expr的string作为map[key]
		if !ok {                                 // 现有正则子节点内没有这种表达式，新增子node即可
			n.regexpChild[expr.String()] = &node{path: pattern, typ: nodetypeRegexp, paramName: param_Name, regExpr: expr}

		} else if regn.path != pattern || regn.regExpr.String() != expr.String() { //现有正则子节点内存在这种表达式，对比是否一致
			panic(fmt.Sprintf("web: 路由冲突，正则路由冲突，已有 %s, 新注册 %s", regn.path, pattern))
		}
	}
	//Children 找到了就直接返回
	return n.regexpChild[expr.String()]
}

func (n *node) childOrCreateParam(pattern string, param_Name string) *node {
	if n.paramChild == nil {
		n.paramChild = &node{path: pattern, typ: nodetypeParam, paramName: param_Name}
	}
	if n.paramChild != nil && n.paramChild.path != pattern {
		panic(fmt.Sprintf("web: 路由冲突，参数路由冲突，已有 %s, 新注册 %s", n.paramChild.path, pattern))
	}
	//Children 找到了就直接返回
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

func (m *matchInfo) RemoveValue(pattern string) {
	delete(m.pathParams, pattern)
	if len(m.pathParams) == 0 {
		//如果m.pathParams没有数值了，可以消去初始化，方便后面判nil
		m.pathParams = nil
	}

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
