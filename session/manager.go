package session

import "go_web/web"

type Manager struct {
	Store
	Propagator
	SessCtxKey string
}

// GetSession 将会尝试从 ctx 中拿到 Session，
// 如果成功了，那么它会将 Session 实例缓存到 ctx 的 UserValues 里面
func (m *Manager) GetSession(ctx *web.Context) (Session, error) {
	if ctx.UserValues == nil {
		ctx.UserValues = make(map[string]any, 1)
	}
	//获取session储存字段名对应的session实例，比如 "_Session"(因为UserValues也可能被用户用来存储其他事务的数据)
	val, ok := ctx.UserValues[m.SessCtxKey]
	if ok {
		return val.(Session), nil //interface 断言成 Session 实例
	}
	id, err := m.Extract(ctx.R) //获取对应Session Id
	if err != nil {
		return nil, err
	}
	sess, err := m.Get(ctx.R.Context(), id) //通过session id 获取对应 Session 实例
	if err != nil {
		return nil, err
	}
	//ctx 保存这次的session实例 sess，以便后续复用
	ctx.UserValues[m.SessCtxKey] = sess
	return sess, nil
}

// InitSession 初始化一个 session，并且注入到 http response 里面
func (m *Manager) InitSession(ctx *web.Context, id string) (Session, error) {
	sess, err := m.Generate(ctx.R.Context(), id) //调用m.store.Generate()，内部实现ctx的保存
	if err != nil {
		return nil, err
	}
	if err = m.Inject(id, ctx.W); err != nil { //将产生的Session id 注入到Http header中
		return nil, err
	}
	if ctx.UserValues == nil {
		ctx.UserValues = make(map[string]any, 1)
	}
	ctx.UserValues[m.SessCtxKey] = sess
	return sess, nil
}

// RefreshSession 刷新 Session
// 更新ctx 缓存
func (m *Manager) RefreshSession(ctx *web.Context) (Session, error) {
	sess, err := m.GetSession(ctx)
	if err != nil {
		return nil, err
	}
	// 刷新存储的过期时间
	err = m.Refresh(ctx.R.Context(), sess.ID())
	if err != nil {
		return nil, err
	}
	// 重新注入 HTTP 里面
	if err = m.Inject(sess.ID(), ctx.W); err != nil {
		return nil, err
	}
	sess, err = m.GetSession(ctx)
	if err != nil {
		return nil, err
	}
	ctx.UserValues[m.SessCtxKey] = sess
	return sess, err
}

// RemoveSession 删除 Session
func (m *Manager) RemoveSession(ctx *web.Context) error {
	sess, err := m.GetSession(ctx)
	if err != nil {
		return err
	}
	err = m.Store.Remove(ctx.R.Context(), sess.ID())
	if err != nil {
		return err
	}
	delete(ctx.UserValues, m.SessCtxKey)
	return m.Propagator.Remove(ctx.W)
}
