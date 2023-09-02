package web

import "net/http"

//基于go map基本数据结构结构实现路由分发功能
type HandlerBasedonMap struct {
	//路由器的“method+path” 匹配 相应的处理函数 handlefunc
	handlefuncs map[string]func(c *Context)
}

//实现ServeHttp才能实现handler的接口
// type Handler interface {
// 	ServeHTTP(ResponseWriter, *Request)
// }
//handler——>call ServeHTTP() 主要负责路由分发（基于map实现）
func (h *HandlerBasedonMap) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := h.KeyGen(r.Method, r.URL.Path)
	//处理逻辑是否存在对应key：value
	if Handlefunc, OK := h.handlefuncs[key]; OK {
		Handlefunc(NewContext(w, r))
	} else {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("该页面未找到"))
	}

}

//封装路由map key生成功能
func (h *HandlerBasedonMap) KeyGen(method string, path string) string {
	return method + "#" + path //分隔符主要是避免key相同
}
