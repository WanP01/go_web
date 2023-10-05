package accesslog

import (
	"encoding/json"
	"fmt"
	"go_web/web"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestServeWithMiddleware(t *testing.T) {

	// 更换定制版的accesslog logFunc
	type NewAccessLog struct {
		Defa AccessLog
		Tim  time.Time //访问时间
	}

	// 实例化accesslog  logFunc
	l := NewLogFuncBuilder(func(ctx *web.Context) {
		l := NewAccessLog{
			Defa: AccessLog{
				Host:       ctx.R.Host,
				Route:      ctx.MatchRoute, //如果到路由前中途发生panic，此处无法获取相关值，因为需要到路由匹配那一步才有值
				HTTPMethod: ctx.R.Method,
				Path:       ctx.R.URL.Path,
			},
			Tim: time.Now(),
		}
		val, _ := json.Marshal(l)
		log.Println(string(val))
	})

	AcesslogMiddleware := l.Build()

	mdlsOpt := web.ServeWithMiddleware(AcesslogMiddleware)

	s := web.NewServerEngine("test_mdls", mdlsOpt)
	fmt.Printf("s.Mdls: %v\n", s.Mdls)
	s.Route(http.MethodGet, "/user/:id", func(c *web.Context) {
		fmt.Printf("c: %v\n", c.R.Host)
		fmt.Printf("c: %v\n", c.MatchRoute)
		fmt.Printf("c: %v\n", c.R.Method)
		fmt.Printf("c: %v\n", c.R.URL.Path)
		// time.Sleep(5 * time.Hour)
	})
	res, _ := http.NewRequest(http.MethodGet, "http://localhost:8081/user/123", nil)

	//模拟客户端访问
	resp := httptest.NewRecorder()
	s.ServeHTTP(resp, res)

	//模拟客户端访问
	go func() {
		// time.Sleep(10 * time.Second)
		s.Start(":8081")
	}()

	time.Sleep(1 * time.Second)
	http.DefaultClient.Do(res)

}
