package cuckoofilter

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func testCuckooFilter(table Table, count int, t *testing.T) {
	var inserted [][]byte
	var successCount uint = 0
	cf := NewCuckooFilter(table)
	rand.Seed(time.Now().UnixNano())
	for i := uint(0); i < uint(count); i++ {
		data := []byte(fmt.Sprintf("%d:%d", i, rand.Uint64()))
		errInsert := cf.InsertUnique(data)
		ok, errLookup := cf.Lookup(data)
		if errInsert == nil || errInsert == ErrFull {
			inserted = append(inserted, data)
		}
		if errInsert == nil {
			successCount++
		}
		if errLookup != nil {
			t.Errorf("Lookup失败, data=%s > %s", data, errLookup)
		} else if errInsert == ErrExist && !ok {
			t.Errorf("Insert报告记录重复(误报), Lookup却失败, data=%s", data)
		} else if errInsert == nil && !ok {
			t.Errorf("Insert成功, Lookup却失败, data=%s", data)
		} else if errInsert != nil && errInsert != ErrExist && errInsert != ErrFull {
			t.Errorf("Insert败, data=%s > %s", data, errInsert)
		}
	}
	if successCount != cf.Count() {
		t.Errorf("Insert成功数量不相等, successCount=%d, cf.Count()=%d", successCount, cf.Count())
	}
	for _, data := range inserted {
		_ = cf.Delete(data)
	}
	if cf.Count() != 0 {
		t.Errorf("Delete所有后, cf.Count()=%d!=0", cf.Count())
	}
}

func TestMemTable(t *testing.T) {
	const capacity = 1 << 21
	table := NewMemTable(capacity)
	count := capacity
	testCuckooFilter(table, int(count), t)
}
