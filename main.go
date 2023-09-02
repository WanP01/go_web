package main

import (
	"go_web/web"
)

func main() {
	//创建新的Server
	server := web.NewsdkHttpServer("test_server")

	//绑定handlefunc
	server.Route("GET", "/hello", web.Sign)

	//监听端口 & 创建默认路由DefaultServerMux(负责分配路由到指定函数)
	server.Start(":8081")

}
