package memory

import (
	"context"
	"errors"
	"go_web/session"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
)

type Store struct {
	// 利用一个内存缓存来帮助我们管理过期时间
	cache      *cache.Cache
	expiration time.Duration
	RWM        sync.RWMutex //防止并发session
}

type CacheTimeOPTION func(s *Store)

// NewStore 创建一个 Store 的实例
// 实际上，这里也可以考虑使用 Option 设计模式，允许用户控制过期检查的间隔
func NewStore(expiration time.Duration, timeOPT ...CacheTimeOPTION) *Store {
	s := &Store{
		cache:      cache.New(expiration, time.Second), //每隔一秒间隔检查过期时间是否到期
		expiration: expiration,                         //保存过期时间
	}
	for _, opt := range timeOPT {
		opt(s)
	}
	return s
}

func WithCacheTimeOPTION(expiration time.Duration, t time.Duration) CacheTimeOPTION {
	return func(s *Store) {
		s.cache = cache.New(expiration, t)
	}
}

// 新增缓存session 实例
func (s *Store) Generate(ctx context.Context, id string) (session.Session, error) {
	sess := &memorySession{
		id:         id,
		data:       make(map[string]string),
		expiration: s.expiration,
	}
	s.RWM.Lock()
	defer s.RWM.Unlock()
	s.cache.Set(sess.ID(), sess, s.expiration)
	return sess, nil
}

// 更新过期时间
func (s *Store) Refresh(ctx context.Context, id string) error {

	sess, err := s.Get(ctx, id) // cache缓存内取出对应id的session
	if err != nil {
		return nil
	}
	s.RWM.Lock()
	defer s.RWM.Unlock()
	s.cache.Set(sess.ID(), sess, s.expiration) // 变更cache缓存内session对应的过期时间
	return nil
}

// 删除缓存实例
func (s *Store) Remove(ctx context.Context, id string) error {
	s.RWM.Lock()
	defer s.RWM.Unlock()
	s.cache.Delete(id) //缓存内删除session
	return nil
}

// 获取缓存实例
func (s *Store) Get(ctx context.Context, id string) (session.Session, error) {
	s.RWM.RLock()
	defer s.RWM.RUnlock()
	sess, ok := s.cache.Get(id) // cache缓存内取出对应id的session
	if !ok {
		return nil, errors.New("session not found")
	}
	return sess.(*memorySession), nil
}

type memorySession struct {
	id         string
	data       map[string]string
	expiration time.Duration
}

func (m *memorySession) Get(ctx context.Context, key string) (string, error) {
	val, ok := m.data[key]
	if !ok {
		return "", errors.New("找不到这个 key")
	}
	return val, nil
}

func (m *memorySession) Set(ctx context.Context, key string, val string) error {
	m.data[key] = val
	return nil
}

func (m *memorySession) ID() string {
	return m.id
}
