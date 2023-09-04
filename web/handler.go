package web

import "net/http"

//——————————————————————————————————抽象层————————————————————————————————————————————————————————

//包装一下HandlerBasedonMap,当前过于依赖HandlerBasedonMap的struct，设计应当依赖于接口
type Handler interface {
	http.Handler // 组合原有serverHttp接口方法，用于实现：“路由的分发功能”——DefaultServerMux
	Routable     // 组合原有sdkHttpServer的route注册功能，用于实现：“路由的注册功能”——HandleFunc
}

//将原有sdkHttpServer的route注册功能包装进Handler接口，避免Server直接调用HandlerBasedonMap的内部结构
type Routable interface {
	Route(method string, pattern string, handlefunc func(c *Context)) //路由注册功能
}

//—————————————————————————————————————实现层——————————————————————————————————————————————————————

//基于go map基本数据结构结构实现路由分发和注册功能
type HandlerBasedonMap struct {
	//路由器的“method+path” 匹配 相应的处理函数 handlefunc
	handlefuncs map[string]func(c *Context)
}

//路由分发（基于map实现）：handler——>call ServeHTTP()
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

//路由注册功能
func (h *HandlerBasedonMap) Route(method string, pattern string, handlefunc func(c *Context)) {
	key := h.KeyGen(method, pattern) //对应路径生成key
	h.handlefuncs[key] = handlefunc  //注册对应路径的处理函数，本质代替了http.HandleFunc作用
}

//封装路由map key生成功能
func (h *HandlerBasedonMap) KeyGen(method string, path string) string {
	return method + "#" + path //分隔符主要是避免key相同
}

//HandlerBasedonMap生成函数
func NewHandlerBasedonMap() *HandlerBasedonMap {
	return &HandlerBasedonMap{handlefuncs: make(map[string]func(c *Context))}
}

//GO语言小技巧，用于确保HandleBasedonMap确实实现Handler接口（如有方法未实现，会报错）
var _ Handler = &HandlerBasedonMap{}
