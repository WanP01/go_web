package redis

import (
	"context"
	"errors"
	"fmt"
	"go_web/session"
	"time"

	"github.com/redis/go-redis/v9"
)

var errSessionNotExist = errors.New("redis-session: session 不存在")

type StoreOption func(store *Store)

type Store struct {
	prefix     string
	client     redis.Cmdable
	expiration time.Duration
}

func NewStore(client redis.Cmdable, opts ...StoreOption) *Store {
	res := &Store{
		client:     client,
		prefix:     "session",
		expiration: time.Minute * 15,
	}
	for _, opt := range opts {
		opt(res)
	}
	return res
}

func key(prefix, id string) string {
	return fmt.Sprintf("%s_%s", prefix, id)
}

// key:map[Field:Value]=> key:s.Key,Field:"_session_id",value:id
// 这里只示范session{"_session_id"：id}这种数据结构，更多字段可以采用HMSET
func (s *Store) Generate(ctx context.Context, id string) (session.Session, error) {
	key := key(s.prefix, id)
	_, err := s.client.HSet(ctx, key, "_session_id", id).Result()
	if err != nil {
		return nil, err
	}
	_, err = s.client.Expire(ctx, id, s.expiration).Result()
	if err != nil {
		return nil, err
	}
	return &Session{
		id:     id,
		key:    key,
		client: s.client,
	}, nil
}

// 更新过期时间
func (s *Store) Refresh(ctx context.Context, id string) error {
	key := key(s.prefix, id)
	ok, err := s.client.Expire(ctx, key, s.expiration).Result()
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("session:id 对应的session不存在")
	}
	return nil
}

func (s *Store) Remove(ctx context.Context, id string) error {
	key := key(s.prefix, id)
	cnt, err := s.client.Del(ctx, key).Result()
	if err != nil {
		return err
	}
	//id 对应的session不存在
	if cnt != 1 {
		//return errors.New("session:id 对应的session不存在")
		return nil
	}
	return nil
}

func (s *Store) Get(ctx context.Context, id string) (session.Session, error) {
	key := key(s.prefix, id)
	cnt, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	//id 对应的session不存在,返回-1
	if cnt < 0 {
		return nil, errors.New("session:id 对应的session不存在")
	}
	return &Session{
		id:     id,
		key:    key,
		client: s.client,
	}, nil
}

type Session struct {
	id     string
	key    string
	client redis.Cmdable
}

func (s *Session) Get(ctx context.Context, key string) (string, error) {
	// s.key 指 key:map(Filed:value)中的key，Filed指key
	val, err := s.client.HGet(ctx, s.key, key).Result()
	return val, err
}

func (s *Session) Set(ctx context.Context, key string, val string) error {
	const lua = `
if redis.call("exists", KEYS[1])
then
	return redis.call("hset", KEYS[1], ARGV[1], ARGV[2])
else
	return -1
end
`
	res, err := s.client.Eval(ctx, lua, []string{s.key}, key, val).Int()
	if err != nil {
		return err
	}
	if res < 0 {
		return errSessionNotExist
	}
	return nil
}

func (s *Session) ID() string {
	return s.id
}
