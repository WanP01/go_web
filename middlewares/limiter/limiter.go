package limiter

import (
	"github.com/juju/ratelimit"
	"go_web/web"
	"net/http"
	"strings"
	"time"
)

// LimiterIface 令牌桶管理接口
type LimiterIface interface {
	Key(c *web.Context) string                          // 全局令牌桶map 中 接口访问地址（key ： url） 与令牌桶关联
	GetBucket(key string) (*ratelimit.Bucket, bool)     // 全局获取令牌桶
	AddBuckets(rules ...LimiterBucketRule) LimiterIface // 全局新增不同令牌桶
}

// LimiterBucketRule 令牌桶的规则属性
type LimiterBucketRule struct {
	Key          string        // 令牌桶对应的key（对应url）
	FillInterval time.Duration //放令牌的间隔时间
	Capacity     int64         //令牌桶容量
	Quantum      int64         // 每次到达间隔事件后的放入令牌桶的具体数量（默认1次1个，但也可指定）
}

// Limiter 全局令牌桶的集中管理
type Limiter struct {
	LimiterBuckets map[string]*ratelimit.Bucket
}

type MethodLimiter struct {
	*Limiter // 全局令牌桶
}

// Key 相同限流规则的接口的访问地址切割出相同部分作为调用特定令牌桶的 key
func (m *MethodLimiter) Key(c *web.Context) string {
	uri := c.R.RequestURI
	index := strings.Index(uri, "?") // xxxxx/xxx?***** 要的是问号前面的uri
	if index == -1 {
		return uri
	}
	return uri[:index]
}

func (m *MethodLimiter) GetBucket(key string) (*ratelimit.Bucket, bool) {
	bucket, ok := m.LimiterBuckets[key]
	return bucket, ok
}

func (m *MethodLimiter) AddBuckets(rules ...LimiterBucketRule) LimiterIface {
	for _, rule := range rules {
		if _, ok := m.LimiterBuckets[rule.Key]; !ok {
			bucket := ratelimit.NewBucketWithQuantum(rule.FillInterval, rule.Capacity, rule.Quantum)
			m.LimiterBuckets[rule.Key] = bucket
		}
	}
	return m
}

// Build 令牌桶中间件
func (m *MethodLimiter) Build() web.Middleware {
	return func(next web.HandleFunc) web.HandleFunc {
		return func(ctx *web.Context) {
			key := m.Key(ctx)
			if bucket, ok := m.LimiterBuckets[key]; ok {
				// 获取可以拿到1枚令牌时，扣除的令牌数（1枚），没有令牌时返回0
				count := bucket.TakeAvailable(1)
				if count == 0 { // 没有令牌，响应直接回复服务器繁忙即可
					ctx.RespData = []byte("Serve is busy")
					ctx.RespStatusCode = http.StatusTooManyRequests
					return
				}
			}

			next(ctx)
		}
	}
}
