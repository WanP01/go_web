package web

import (
	// "log"
	// "net"

	"net/http"
)

// 创建Sever顶层设计
type Server interface {
	Routable   //路由方法
	Start()    // 启动初始化
	Shutdown() // 关闭收尾
}

// 创建Server实例
type sdkHttpServer struct {
	Name   string //用于log日志名字
	Router Router //用于路由分发的结构体
}

// 实现Server路由绑定功能,底层调用http.HandleFunc
// HandleBasedonMap 封装成接口过后，路由注册功能可由Router 接口实现，此处仅为转发
func (s *sdkHttpServer) Route(method string, pattern string, handlefunc HandleFunc) {
	s.Router.Route(method, pattern, handlefunc)
}

// 便捷方法：注册Get路由
func (s *sdkHttpServer) Get(pattern string, handlefunc HandleFunc) {
	s.Route(http.MethodGet, pattern, handlefunc)
}

// 便捷方法：注册Post路由
func (s *sdkHttpServer) Post(pattern string, handlefunc HandleFunc) {
	s.Route(http.MethodPost, pattern, handlefunc)
}

// 需要实现Server初始化功能，调用http.ListenAndServer()
// 实现自己的Router路由分发器s.Router
func (s *sdkHttpServer) Start(addr string) error {
	return http.ListenAndServe(addr, s.Router)
}

// 实现Server关闭功能，pass
func (s *sdkHttpServer) Shutdown() {

}

// 实现Server创建功能
func NewsdkHttpServer(name string) *sdkHttpServer {
	return &sdkHttpServer{
		Name: name,
		// Router: NewHandlerBasedonMap(), // 用于map实现的路由树 V1
		Router: newRouter(),
	}
}
