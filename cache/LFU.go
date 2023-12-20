package cache

import (
	"container/list"
	"strconv"
)

type LFUCache struct {
	maxBytes   int64 // 最大内存
	nBytes     int64 // 当前已使用内存
	minFreq    int64
	keyToEntry map[string]*list.Element
	freqToList map[string]*list.List
	// 回调函数
	OnEvicted func(key string, value Value)
}

type LFUEntry struct {
	key   string
	value Value
	freq  int64
}

func NewLFUCache(maxBytes int64, OnEvicted func(key string, value Value)) *LFUCache {
	return &LFUCache{
		maxBytes:   maxBytes,
		nBytes:     0,
		minFreq:    1,
		keyToEntry: make(map[string]*list.Element),
		freqToList: make(map[string]*list.List),
		OnEvicted:  OnEvicted,
	}
}

func (c *LFUCache) Get(key string) (value Value, ok bool) {
	entry := c.GetEntry(key)
	if entry != nil {
		return entry.value, true
	}
	return nil, false
}

func (c *LFUCache) GetEntry(key string) *LFUEntry {
	ele, ok := c.keyToEntry[key]
	if !ok {
		return nil
	} else {
		entry := ele.Value.(*LFUEntry)
		oldFreq := strconv.FormatInt(entry.freq, 10)
		oldList := c.freqToList[oldFreq]
		oldList.Remove(ele)
		if oldList.Len() == 0 {
			delete(c.freqToList, oldFreq)
			if c.minFreq == entry.freq {
				c.minFreq++
			}
		}
		entry.freq++
		newFreq := strconv.FormatInt(entry.freq, 10)
		newList, ok := c.freqToList[newFreq]
		if !ok {
			c.freqToList[newFreq] = list.New()
			newList = c.freqToList[newFreq]
		}
		c.keyToEntry[key] = newList.PushFront(entry)
		return entry
	}
}

func (c *LFUCache) Set(key string, value Value) {
	entry := c.GetEntry(key)
	if entry != nil {
		c.nBytes += int64(value.Len()) - int64(entry.value.Len())
		entry.value = value
		return
	} else {
		entry = &LFUEntry{
			key:   key,
			value: value,
			freq:  1,
		}
		length := int64(len(key) + value.Len() + 8)
		for c.maxBytes > 0 && c.nBytes+length > c.maxBytes {
			c.RemoveOldest()
		}
		c.minFreq = 1
		newList, ok := c.freqToList["1"]
		if !ok {
			c.freqToList["1"] = list.New()
			newList = c.freqToList["1"]
		}
		ele := newList.PushFront(entry)
		c.keyToEntry[key] = ele
		c.nBytes += length
	}
}

func (c *LFUCache) Del(key string) {
	ele, ok := c.keyToEntry[key]
	if !ok {
		return
	}
	if ele != nil {
		entry := ele.Value.(*LFUEntry)
		curFreq := strconv.FormatInt(entry.freq, 10)
		curList := c.freqToList[curFreq]
		curList.Remove(ele)
		if curList.Len() == 0 {
			delete(c.freqToList, curFreq)
			if entry.freq == c.minFreq {
				c.minFreq = int64(^uint(0) >> 1)
				for k, _ := range c.freqToList {
					kf, _ := strconv.ParseInt(k, 10, 64)
					if kf < c.minFreq {
						c.minFreq = kf
					}
				}
				if c.minFreq == int64(^uint(0)>>1) {
					c.minFreq = 0
				}
			}
		}
		delete(c.keyToEntry, entry.key)
		c.nBytes -= int64(len(entry.key) + entry.value.Len() + 8)
		if c.OnEvicted != nil {
			c.OnEvicted(entry.key, entry.value)
		}
	}
}

func (c *LFUCache) RemoveOldest() {
	oldFreq := strconv.FormatInt(c.minFreq, 10)
	oldList := c.freqToList[oldFreq]
	ele := oldList.Back()
	if ele != nil {
		oldList.Remove(ele)
		if oldList.Len() == 0 {
			delete(c.freqToList, oldFreq)
		}
		entry := ele.Value.(*LFUEntry)
		delete(c.keyToEntry, entry.key)
		c.nBytes -= int64(len(entry.key) + entry.value.Len() + 8)
		if c.OnEvicted != nil {
			c.OnEvicted(entry.key, entry.value)
		}
	}
}
func (c *LFUCache) Len() int {
	return len(c.keyToEntry)
}
