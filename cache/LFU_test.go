package cache

import (
	"reflect"
	"testing"
)

type ntring string

func (d ntring) Len() int {
	return len(d)
}

// 测试加入和查询
func TestLFUCache(t *testing.T) {
	c := NewLFUCache(int64(20), nil)
	c.Set("key1", ntring("1234"))
	if v, ok := c.Get("key1"); !ok || string(v.(ntring)) != "1234" {
		t.Fatalf("keyToEntry hit key1=1234 failed")
	}
	if _, ok := c.Get("key2"); ok {
		t.Fatalf("keyToEntry miss key2 failed")
	}
}

// 测试容量满后是否会移除最老的节点
func TestLFURemoveoldest(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "k3"
	v1, v2, v3 := "value1", "value2", "v3"
	cap := len(k1+k2+v1+v2) + 16

	lfu := NewLFUCache(int64(cap), nil)
	lfu.Set(k1, ntring(v1))
	lfu.Get(k1)
	lfu.Set(k2, ntring(v2)) // k2 淘汰
	lfu.Set(k3, ntring(v3))

	// k1 2 // k3 1 - k2 1
	if _, ok := lfu.Get("key2"); ok || lfu.Len() != 2 {
		t.Fatalf("Removeoldest key2 failed")
	}
	if _, ok := lfu.Get("key1"); !ok || lfu.Len() != 2 {
		t.Fatalf(" key1 remain failed")
	}
}

// 测试会回调函数
func TestLFUOnEvicted(t *testing.T) {
	keys := make([]string, 0)
	// 回调函数：记录被删除的数据的key
	callback := func(key string, value Value) {
		keys = append(keys, key)
	}
	lru := NewLFUCache(int64(40), callback)
	lru.Set("key1", ntring("123456"))
	lru.Get("key1")
	lru.Set("k2", ntring("k2"))
	lru.Set("k3", ntring("k3"))
	lru.Set("k4", ntring("k4"))

	expect := []string{"k2", "k3"}

	if !reflect.DeepEqual(expect, keys) {
		t.Fatalf("Call OnEvicted failed, expect keys equals to %s", expect)
	}
}
