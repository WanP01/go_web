package accesslog

import (
	"encoding/json"
	"go_web/web"
	"log"
)

// 中间件构造器（不同中间件的共有结构，内容随中间件变化）
type middlewarebuilder struct { // build套皮
	logFunc func(ctx *web.Context) // 日志中间件逻辑处理函数本体
}

// 用于构造自定义的middlewarebuilder
func NewLogFuncBuilder(lf func(ctx *web.Context)) *middlewarebuilder {
	return &middlewarebuilder{
		logFunc: lf,
	}
}

// 返回默认预设的的middlewarebuilder
func WithDefaultLogFunc() *middlewarebuilder {
	return NewLogFuncBuilder(func(ctx *web.Context) {
		l := AccessLog{
			Host:       ctx.R.Host,
			Route:      ctx.MatchRoute, //如果到路由前中途发生panic，此处无法获取相关值，因为需要到路由匹配那一步才有值
			HTTPMethod: ctx.R.Method,
			Path:       ctx.R.URL.Path,
		}
		val, _ := json.Marshal(l)
		log.Println(string(val))
	})
}

// 用于构造AOP中间件的级联引用
func (m *middlewarebuilder) Build() web.Middleware {
	return func(next web.HandleFunc) web.HandleFunc {
		return func(ctx *web.Context) {
			// 无defer调用下一个中间件之前做的事
			// 调用defer 实际是在next之后完成，Panic也可以完成记录
			defer func() {
				m.logFunc(ctx)
			}()
			next(ctx)
			// 调用下一个中间件之后做的事
		}
	}
}

type AccessLog struct {
	Host       string `json:"host"`        //客户端域名
	Route      string `json:"routet"`      // 匹配到的路由整体
	HTTPMethod string `json:"http_method"` // HTTP 方法 GET/POST等
	Path       string `json:"path"`        // url路径
}
