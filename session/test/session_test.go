package test

import (
	"fmt"
	"go_web/session"
	"go_web/session/cookie"
	"go_web/session/memory"
	"go_web/web"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSession(t *testing.T) {

	m := session.Manager{
		SessCtxKey: "_sess",
		Store:      memory.NewStore(30 * time.Minute),
		Propagator: cookie.NewPropagator("sessid",
			cookie.WithCookieOption(func(c *http.Cookie) {
				c.HttpOnly = true
			})),
	}

	// session 以 middleware形式嵌入
	mdlOPT := web.ServeWithMiddleware(func(next web.HandleFunc) web.HandleFunc {
		return func(ctx *web.Context) {
			// 执行校验
			if ctx.R.URL.Path != "/login" {
				sess, err := m.GetSession(ctx)
				// 不管发生了什么错误，对于用户我们都是返回未授权
				if err != nil {
					ctx.RespStatusCode = http.StatusUnauthorized //实际返回redict最好
					return
				}
				_ = m.Refresh(ctx.R.Context(), sess.ID())
			}
			next(ctx)
		}
	})

	//初始化服务器
	s := web.NewServerEngine("test", mdlOPT)

	s.Get("/login", func(ctx *web.Context) {
		// 前面就是你登录的时候一大堆的登录校验
		id := uuid.New()
		sess, err := m.InitSession(ctx, id.String())
		if err != nil {
			ctx.RespStatusCode = http.StatusInternalServerError
			return
		}
		// 然后根据自己的需要设置
		err = sess.Set(ctx.R.Context(), "mykey", "some value")
		if err != nil {
			ctx.RespStatusCode = http.StatusInternalServerError
			return
		}
	})

	s.Get("/hello", func(ctx *web.Context) {
		sess, err := m.GetSession(ctx)
		if err != nil {
			ctx.RespStatusCode = http.StatusInternalServerError
			return
		}
		val, err := sess.Get(ctx.R.Context(), "mykey")
		ctx.RespData = []byte(val)
		fmt.Printf("%v", ctx.UserValues)
	})

	s.Get("/logout", func(ctx *web.Context) {
		_ = m.RemoveSession(ctx)
		fmt.Printf("%v", ctx.UserValues)
	})

	s.Start(":8081")
}
