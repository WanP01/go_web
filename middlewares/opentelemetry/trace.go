package opentelemetry

import (
	"go_web/web"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const defaultInstrumentationName = "go_web/middlewares/opentelementry"

type Middlewarebuilder struct {
	Tracer trace.Tracer
}

func NewTraceMiddlewareBuilder(t trace.Tracer) *Middlewarebuilder {
	return &Middlewarebuilder{
		Tracer: t,
	}
}

func (m *Middlewarebuilder) Build() web.Middleware {
	if m.Tracer == nil {
		//GetTracerProvider返回注册的全局跟踪提供程序。如果没有注册，则返回NoopTracerProvider的实例。
		m.Tracer = otel.GetTracerProvider().Tracer(defaultInstrumentationName)
	}
	return func(next web.HandleFunc) web.HandleFunc {
		return func(ctx *web.Context) { // tracer 初始化
			//获取ctx.request结构内的Context
			reqCtx := ctx.R.Context()

			//将reqCtx与客户端的trace.Ctx相关联,从trace.ctx拿到对应的traceID，SpanID等
			reqCtx = otel.GetTextMapPropagator().Extract(reqCtx, propagation.HeaderCarrier(ctx.R.Header))

			// 建立子span
			// Start创建一个跨度和上下文。包含新创建的跨度的上下文。
			// 如果上下文。在' ctx '中提供的上下文包含一个Span，那么新创建的Span将是该Span的子跨度，否则它将是根跨度。
			// 此行为可以通过提供' WithNewRoot() '作为Span选项来覆盖，即使' ctx '包含Span，也会导致新创建的Span成为根Span。
			// 当创建Span时，建议使用' WithAttributes() ' SpanOption提供所有已知的Span属性，因为采样器只能访问创建Span时提供的属性。
			// 任何创建的Span也必须结束。这是用户的责任。如果没有结束span，这个API的实现可能会泄漏内存或其他资源。
			reqCtx, span := m.Tracer.Start(reqCtx, "middleware_0", trace.WithAttributes())

			// span.End 执行之后，就意味着 span 本身已经确定无疑了，将不能再变化了
			defer span.End()

			//设置span记录的tag数据
			span.SetAttributes(attribute.String("http.method", ctx.R.Method))
			span.SetAttributes(attribute.String("peer.hostname", ctx.R.Host))
			span.SetAttributes(attribute.String("http.url", ctx.R.URL.String()))
			span.SetAttributes(attribute.String("http.scheme", ctx.R.URL.Scheme))
			span.SetAttributes(attribute.String("span.kind", "server"))
			span.SetAttributes(attribute.String("component", "web"))
			span.SetAttributes(attribute.String("peer.address", ctx.R.RemoteAddr))
			span.SetAttributes(attribute.String("http.proto", ctx.R.Proto))

			//将trace经手的ctx 传回 ctx.R.context 中,继续链路调用
			ctx.R = ctx.R.WithContext(reqCtx)

			//中间件调用下一层
			next(ctx)

			// 使用命中的路由来作为 span 的名字（只有到达路由之后才能填充这一层数据）
			if ctx.MatchRoute != "" {
				span.SetName(ctx.MatchRoute)
			}

			// 怎么拿到响应的状态呢？ctx.W是接口，未提供相关获取响应码的方法，http.ResponseWriter接口对应的实例http.response私有，不对外开放
			// 只能context新建字段保存相关信息
			span.SetAttributes(attribute.Int("http.status", ctx.RespStatusCode))
		}
	}
}
