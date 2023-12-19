package web

import (
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// Server 创建Sever顶层设计
type Server interface {
	http.Handler             // ServeHTTP(ResponseWriter, *Request)
	Routable                 //路由方法
	Start(addr string) error // 启动初始化
	Shutdown()               // 关闭收尾
}

// ServerEngine 创建Server实例
type ServerEngine struct {
	Name      string         //用于log日志名字
	Router    Routable       //用于路由分发的结构体
	Mdls      []Middleware   // 注册的路由
	tplEngine TemplateEngine // 模板引擎
	Groups    []*Group       //所有Group统一管理
	*Group                   // 初始节点group
}

// 确保 ServerEngine 肯定实现了 Server 接口
var _ Server = &ServerEngine{}

// ——————————————————————————————————————OPTION————————————————————————————————————————

// HttpSeverOPT 修改Server内部的字段的函数
type HttpSeverOPT func(s *ServerEngine)

// NewServerEngine 实现Server创建功能
func NewServerEngine(name string, OPT ...HttpSeverOPT) *ServerEngine {
	SHS := &ServerEngine{
		Name: name,
		// Router: NewHandlerBasedonMap(), // 用于map实现的路由树 V1
		Router:    newRouter(),
		Mdls:      make([]Middleware, 0),
		tplEngine: nil,
		Group:     &Group{},
		Groups:    []*Group{},
	}

	// 构建 ServerEngine 与 Group 的关联
	SHS.Group.Engine = SHS
	SHS.Groups = append(SHS.Groups, SHS.Group)

	//原地修改SHS字段值
	for _, opt := range OPT {
		opt(SHS)
	}
	return SHS
}

//———————————————————————————————————————Build——————————————————————————————————————————

// ServeWithMiddleware 更改Server中间件mdls字段的OPTION模式
func ServeWithMiddleware(mdles ...Middleware) HttpSeverOPT {
	return func(s *ServerEngine) {
		s.Mdls = mdles
	}
}

// ServeWithTemplateEngine 更改Server tplEngine 字段的OPTION模式
func ServeWithTemplateEngine(engine TemplateEngine) HttpSeverOPT {
	return func(s *ServerEngine) {
		s.tplEngine = engine
	}
}

//————————————————————————————————————————Method————————————————————————————————————————

// Use 增加Middlewares
func (s *ServerEngine) Use(middlewares ...Middleware) {
	s.Mdls = append(s.Mdls, middlewares...)
}

// Server路由树分发匹配路径功能
func (s *ServerEngine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := NewContext(w, r, s.tplEngine)
	// 最后一个应该是 HTTPServer 执行路由匹配，执行用户代码
	root := s.serve
	// 从后往前组装中间件
	// 先组装group级别的Middlewares（记得是倒叙组装）
	var middleGroup []*Group
	for _, g := range s.Groups {
		if strings.HasPrefix(ctx.R.URL.Path, g.Prefix) {
			middleGroup = append(middleGroup, g)
		}
	}
	//按照前缀长短排序// 比如 /a,/a/b, 先组装/a/b，再组装/a
	sort.Slice(middleGroup, func(i, j int) bool {
		return len(middleGroup[i].Prefix) > len(middleGroup[j].Prefix)
	})
	for _, g := range middleGroup {
		for i := len(g.Mdls) - 1; i >= 0; i-- {
			root = g.Mdls[i](root)
		}
	}

	// 再组装系统级别的Middlewares
	for i := len(s.Mdls) - 1; i >= 0; i-- {
		root = s.Mdls[i](root)
	}

	//新增组装的最后一项（ctx.Resdata 和 ctx.RespStatusCode需要写回ctx.W）
	m := func(next HandleFunc) HandleFunc {
		return func(c *Context) {
			next(ctx)
			//在最后写回客户响应时刷新进c.W, flashResp 是最后一个步骤
			s.flashResp(ctx)
		}
	}
	//在最后写回客户响应时刷新进c.W
	root = m(root)
	//执行组装好的middleware Chain
	root(ctx)

}

func (s *ServerEngine) flashResp(c *Context) {
	if c.RespStatusCode > 0 {
		c.W.WriteHeader(c.RespStatusCode)
	}
	c.W.Header().Set("Content-Length", strconv.Itoa(len(c.RespData)))
	_, err := c.W.Write(c.RespData)
	if err != nil {
		log.Fatalln("回写响应失败", err)
	}
}

func (s *ServerEngine) serve(ctx *Context) {
	mi, ok := s.findRouter(ctx.R.Method, ctx.R.URL.Path)
	if !ok || mi.n == nil || mi.n.handlefunc == nil {
		ctx.RespStatusCode = http.StatusNotFound
		return
	}
	ctx.PathParams = mi.pathParams
	ctx.MatchRoute = mi.n.route
	mi.n.handlefunc(ctx)
}

// Route 实现Server路由绑定功能,底层调用http.HandleFunc
func (s *ServerEngine) Route(method string, pattern string, handlefunc HandleFunc) {
	s.Router.Route(method, pattern, handlefunc)
}
func (s *ServerEngine) findRouter(method string, pattern string) (*matchInfo, bool) {
	return s.Router.findRouter(method, pattern)
}

// Get 便捷方法：注册Get路由
func (s *ServerEngine) Get(pattern string, handlefunc HandleFunc) {
	s.Route(http.MethodGet, pattern, handlefunc)
}

// Post 便捷方法：注册Post路由
func (s *ServerEngine) Post(pattern string, handlefunc HandleFunc) {
	s.Route(http.MethodPost, pattern, handlefunc)
}

//需要实现Server初始化功能，调用http.ListenAndServer()

// Start 实现自己的Router路由分发器s.Router
func (s *ServerEngine) Start(addr string) error {
	return http.ListenAndServe(addr, s)
}

// Shutdown 实现Server关闭功能，pass
func (s *ServerEngine) Shutdown() {

}
