package cache

import (
	"fmt"
	"log"
	"sync"
	"testing"
)

var sdb = map[string]string{
	"key1": "1",
	"key2": "2",
	"key3": "3",
}

func TestSingleFlightGroup_Do(t *testing.T) {
	loadCount := map[string]int{}
	var wg sync.WaitGroup
	testGroup := NewGroup("test", 2<<10, GetterFunc(func(key string) ([]byte, error) {
		if v, ok := sdb[key]; ok {
			if _, ok := loadCount[key]; !ok {
				loadCount[key] = 0
			}
			loadCount[key] += 1
			return []byte(v), nil
		}
		return nil, fmt.Errorf("%s not exist\n", key)
	}))

	for i := 0; i <= 10; i++ {
		wg.Add(1)
		goID := i
		go func() {
			defer wg.Done()
			Bval, err := testGroup.Get("key1")
			if err != nil || Bval.String() != sdb["key1"] {
				t.Fatalf("cache hit fail")
			}
			log.Printf("%v get success", goID)
			return
		}()
	}
	wg.Wait()
	fmt.Printf("Getter load: %v\n", loadCount["key1"])
	if loadCount["key1"] != 1 {
		t.Fatalf("singleFlight Fail")
	}
}
