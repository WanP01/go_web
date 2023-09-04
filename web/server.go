package web

import (
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
	Name    string  //用于log日志名字
	Handler Handler //用于路由分发
}

// 实现Server路由绑定功能,底层调用http.HandleFunc
// HandleBasedonMap 封装成接口过后，路由注册功能可由Handler 接口实现，此处仅为转发
func (s *sdkHttpServer) Route(method string, pattern string, handlefunc func(c *Context)) {
	s.Handler.Route(method, pattern, handlefunc)
}

// 需要实现Server初始化功能，调用http.ListenAndServer()
// 实现自己的handler路由分发器s.Handler
func (s *sdkHttpServer) Start(addr string) error {
	return http.ListenAndServe(addr, s.Handler)
}

// 实现Server关闭功能，pass
func (s *sdkHttpServer) Shutdown() {

}

// 实现Server创建功能
func NewsdkHttpServer(name string) *sdkHttpServer {
	return &sdkHttpServer{
		Name:    name,
		Handler: NewHandlerBasedonMap(),
	}
}
