package session

import (
	"context"
	"net/http"
)

//接口定义

// session 结构本体的抽象
// curd 增删改查Session中某个字段的值
type Session interface {
	ID() string                                            //查询ID的快捷方式
	Get(ctx context.Context, key string) (string, error)   //查Session里的key-value
	Set(ctx context.Context, key string, val string) error //改Session里的key-value
	// 增和删放在了store里面，因为session结构一般是固定的，用户自己实现的，一般是整体新建和删除，不太会有删除Seesion其中一个字段的可能
}

// Store 管理 Session
// 从设计的角度来说，Generate 方法和 Refresh 在处理 Session 过期时间上有点关系
// 也就是说，如果 Generate 设计为接收一个 expiration 参数，
// 那么 Refresh 也应该接收一个 expiration 参数。
// 因为这意味着用户来管理过期时间
type Store interface {
	Generate(ctx context.Context, id string) (Session, error) //增：Generate 生成一个 session
	Remove(ctx context.Context, id string) error              //删：移除一个 session
	Get(ctx context.Context, id string) (Session, error)      //查：查找一个session
	Refresh(ctx context.Context, id string) error             //改：更新一个session
	// Refresh 这种设计是一直用同一个 id 的
	// 如果想支持 Refresh 换 ID，那么可以重新生成一个，并移除原有的
	// 又或者 Refresh(ctx context.Context, id string) (Session, error)
	// 其中返回的是一个新的 Session
}

type Propagator interface {
	Inject(id string, writer http.ResponseWriter) error //增： Inject 将 session id 注入到里面，Inject 必须是幂等的
	Extract(req *http.Request) (string, error)          //查：Extract 将 session id 从 http.Request 中提取出来，例如从 cookie 中将 session id 提取出来
	Remove(writer http.ResponseWriter) error            //删：Remove 将 session id 从 http.ResponseWriter 中删除，例如删除对应的 cookie里的session ID
	//session过期即删，断开连接就删，一个连接中途没有需改session id的必要
}
