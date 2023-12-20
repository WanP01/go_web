package cache

import (
	"fmt"
	"log"
	"testing"
)

// map 模拟耗时的数据库。
var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func TestGroupGet(t *testing.T) {
	// 调用回调函数（getter）获取缓存数据的次数
	loadCounts := make(map[string]int, len(db))
	testCache := NewGroup("test", 2<<10, GetterFunc(func(key string) ([]byte, error) {
		log.Println("[SlowDB] search key", key)
		if v, ok := db[key]; ok {
			if _, ok := loadCounts[key]; !ok {
				loadCounts[key] = 0
			}
			loadCounts[key] += 1
			return []byte(v), nil
		}
		return nil, fmt.Errorf("%s not exist", key)
	}))

	for k, v := range db {
		// 第一次缓存未命中，从数据库map拿取数据，并存入缓存
		if view, err := testCache.Get(k); err != nil || view.String() != v {
			t.Fatal("failed to get value of Tom")
		} // load from callback function

		// 第二次看缓存是否命中
		if _, err := testCache.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss", k)
		} // cache hit
	}

	if view, err := testCache.Get("unknown"); err == nil {
		t.Fatalf("the value of unknow should be empty, but %s got", view)
	}
}
