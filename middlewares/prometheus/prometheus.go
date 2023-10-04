package prometheus

import (
	"go_web/web"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type MiddlewareBuilder struct {
	Name        string
	Subsystem   string
	ConstLabels map[string]string
	Help        string
}

//这种代码参数过多且很丑，应当用option模式修改Struct的值
// func NewPremotheusBuilder(Name string, Subsystem string, Labels map[string]string, Help string) *MiddlewareBuilder {
// 	return &MiddlewareBuilder{
// 		Name:        Name,
// 		Subsystem:   Subsystem,
// 		ConstLabels: Labels,
// 		Help:        Help,
// 	}
// }

//创建新的MiddlewareBuilder
//方法1 直接 struct 填写
//方法2 有默认值的情况下应当采用Option模式

type OPTIONS func(m *MiddlewareBuilder)

func NewPremotheusBuilder(OPT ...OPTIONS) *MiddlewareBuilder {
	promBuiler := &MiddlewareBuilder{
		Name:        "prometheus",
		Subsystem:   "default",
		Help:        "this is a default Prometheus Middleware",
		ConstLabels: nil,
	}

	for _, opt := range OPT {
		opt(promBuiler)
	}
	//其他约束条件

	return promBuiler
}

func WithName(Name string) OPTIONS {
	return func(m *MiddlewareBuilder) {
		m.Name = Name
	}
}

func WithSubsystem(Subsystem string) OPTIONS {
	return func(m *MiddlewareBuilder) {
		m.Subsystem = Subsystem
	}
}

func WithHelp(Help string) OPTIONS {
	return func(m *MiddlewareBuilder) {
		m.Help = Help
	}
}

func WithConstLabels(ConstLabels map[string]string) OPTIONS {
	return func(m *MiddlewareBuilder) {
		m.ConstLabels = ConstLabels
	}
}

// MiddlewareBuilder 已经产生的情况下调用Build 生成 中间件组件
func (m *MiddlewareBuilder) Build() web.Middleware {
	summartVec := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name:        m.Name,
		Subsystem:   m.Subsystem,
		Help:        m.Help,
		ConstLabels: m.ConstLabels,
	}, []string{"pattern", "method", "status"}) // 配置自定义Summary Vector
	prometheus.MustRegister(summartVec) // 注册Summary Vector
	return func(next web.HandleFunc) web.HandleFunc {
		return func(ctx *web.Context) {
			startTime := time.Now()
			next(ctx)
			DurTime := time.Now().Sub(startTime)
			go report(DurTime, ctx, summartVec)
		}
	}
}

//通过promhttp.handle()暴露端口出去

// prometheus会自动发送数据，需要在发送前填好metric
func report(dur time.Duration, ctx *web.Context, vec prometheus.ObserverVec) {
	status := ctx.RespStatusCode
	pattern := ctx.MatchRoute
	method := ctx.R.Method
	if pattern == "" {
		pattern = "unkown"
	}
	//填充label和value
	vec.WithLabelValues(pattern, method, strconv.Itoa(status)).Observe(float64(dur.Microseconds()))
}
