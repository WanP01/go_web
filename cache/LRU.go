package cache

import "container/list"

type LRUCache struct {
	maxBytes   int64 // 最大内存
	nbytes     int64 // 当前已使用内存
	ll         *list.List
	keyToEntry map[string]*list.Element
	// 回调函数
	OnEvicted func(key string, value Value)
}

// LRUEntry 数据记录节点
type LRUEntry struct {
	key   string
	value Value
}

// Value 用于记录数值大小
type Value interface {
	Len() int
}

func NewLRU(maxBytes int64, onEvicted func(string, Value)) *LRUCache {
	return &LRUCache{
		maxBytes:   maxBytes,
		nbytes:     0,
		ll:         list.New(),
		keyToEntry: make(map[string]*list.Element),
		OnEvicted:  onEvicted,
	}
}

func (c *LRUCache) Get(key string) (value Value, ok bool) {
	if ele := c.GetEntry(key); ele != nil {
		return ele.value, true
	}
	return nil, false
}

func (c *LRUCache) GetEntry(key string) *LRUEntry {
	ele, ok := c.keyToEntry[key]
	if !ok {
		return nil
	}
	c.ll.MoveToFront(ele)
	return ele.Value.(*LRUEntry)
}

func (c *LRUCache) Set(key string, value Value) {
	entry := c.GetEntry(key)
	if entry != nil {
		c.nbytes += int64(value.Len()) - int64(entry.value.Len())
		entry.value = value
		return
	} else {
		entry = &LRUEntry{
			key:   key,
			value: value,
		}
		ele := c.ll.PushFront(entry)
		c.keyToEntry[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	// 设计为 c.maxBytes <= 0 意味着不淘汰内存
	for c.maxBytes > 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

func (c *LRUCache) Del(key string) {
	ele, ok := c.keyToEntry[key]
	if !ok {
		return
	} else {
		if ele != nil {
			c.ll.Remove(ele)
			delete(c.keyToEntry, key)
			entry := ele.Value.(*LRUEntry)
			c.nbytes -= int64(len(key) + entry.value.Len())
			if c.OnEvicted != nil {
				c.OnEvicted(key, entry.value)
			}
		}
	}
}

func (c *LRUCache) RemoveOldest() {
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*LRUEntry)
		delete(c.keyToEntry, kv.key)
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Len 数据总数量
func (c *LRUCache) Len() int {
	return c.ll.Len()
}
