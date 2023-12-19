package web

import "net/http"

type RouterGroup interface {
	NewGroup(prefix string, OPT ...GroupOPTIONS) RouterGroup    // 建立新的Group
	Route(method string, pattern string, handlefunc HandleFunc) // 增加路由
	Use(middlewares ...Middleware)                              // 增加中间件
}

// Group 分组实现
type Group struct {
	Prefix string        // 前缀
	Mdls   []Middleware  // for 中间件
	Parent RouterGroup   // for 嵌套
	Engine *ServerEngine // all groups share one ServerEngine
}

var _ RouterGroup = &Group{}

type GroupOPTIONS func(*Group)

// GroupWithMiddleware 新增中间件
func GroupWithMiddleware(mdls ...Middleware) GroupOPTIONS {
	return func(g *Group) {
		g.Mdls = append(g.Mdls, mdls...)
	}
}

// NewGroup 用一个group 产生新的group，用于实现嵌套
func (g *Group) NewGroup(prefix string, OPT ...GroupOPTIONS) RouterGroup {
	NewG := &Group{
		Prefix: g.Prefix + prefix,
		Mdls:   make([]Middleware, 0), // 不继承上一级group的mdls,会在serverHTTP时进行累加
		Parent: g,
		Engine: g.Engine,
	}

	// 将新Group加入到ServerEngine整体的groups 列表中
	g.Engine.Groups = append(g.Engine.Groups, NewG)

	for _, O := range OPT {
		O(NewG)
	}
	return NewG
}

func (g *Group) Use(middlewares ...Middleware) {
	g.Mdls = append(g.Mdls, middlewares...)
}

func (g *Group) Route(method string, pattern string, handlefunc HandleFunc) {
	pattern = g.Prefix + pattern
	g.Engine.Route(method, pattern, handlefunc)
}

// Get 便捷方法：注册Get路由
func (g *Group) Get(pattern string, handlefunc HandleFunc) {
	g.Route(http.MethodGet, pattern, handlefunc)
}

// Post 便捷方法：注册Post路由
func (g *Group) Post(pattern string, handlefunc HandleFunc) {
	g.Route(http.MethodPost, pattern, handlefunc)
}
