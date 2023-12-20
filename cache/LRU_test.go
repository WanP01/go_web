package cache

import (
	"reflect"
	"testing"
)

type String string

func (d String) Len() int {
	return len(d)
}

// 测试加入和查询
func TestLRUCache(t *testing.T) {
	c := NewLRU(int64(10), nil)
	c.Set("key1", String("1234"))
	if v, ok := c.Get("key1"); !ok || string(v.(String)) != "1234" {
		t.Fatalf("keyToEntry hit key1=1234 failed")
	}
	if _, ok := c.Get("key2"); ok {
		t.Fatalf("keyToEntry miss key2 failed")
	}
}

// 测试容量满后是否会移除最老的节点
func TestRemoveoldest(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "k3"
	v1, v2, v3 := "value1", "value2", "v3"
	cap := len(k1 + k2 + v1 + v2)

	lru := NewLRU(int64(cap), nil)
	lru.Set(k1, String(v1))
	lru.Set(k2, String(v2))
	lru.Set(k3, String(v3))

	if _, ok := lru.Get("key1"); ok || lru.Len() != 2 {
		t.Fatalf("Removeoldest key1 failed")
	}
	if _, ok := lru.Get("key2"); !ok || lru.Len() != 2 {
		t.Fatalf(" key2 remain failed")
	}
}

// 测试会回调函数
func TestOnEvicted(t *testing.T) {
	keys := make([]string, 0)
	// 回调函数：记录被删除的数据的key
	callback := func(key string, value Value) {
		keys = append(keys, key)
	}
	lru := NewLRU(int64(10), callback)
	lru.Set("key1", String("123456"))
	lru.Set("k2", String("k2"))
	lru.Set("k3", String("k3"))
	lru.Set("k4", String("k4"))

	expect := []string{"key1", "k2"}

	if !reflect.DeepEqual(expect, keys) {
		t.Fatalf("Call OnEvicted failed, expect keys equals to %s", expect)
	}
}
