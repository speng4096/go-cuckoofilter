package cuckoofilter

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"path"
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
			t.Errorf("Insert失败, data=%s > %s", data, errInsert)
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

func TestCuckooFilter_MemTable(t *testing.T) {
	const capacity = 1 << 22
	table := NewMemTable(capacity)
	defer table.Close()
	testCuckooFilter(table, capacity, t)
}

func TestCuckooFilter_MMAPTable(t *testing.T) {
	const capacity = 1 << 22
	tempFile := path.Join(os.TempDir(), fmt.Sprintf("cuckoo-%d.mmap", time.Now().UnixNano()))
	t.Logf("TempFile: %s", tempFile)
	table, err := NewMMAPTable(tempFile, capacity)
	defer table.Close()
	if err != nil {
		t.Fatalf("创建MMAPTable失败 > %s", err)
	}
	testCuckooFilter(table, capacity, t)
}

func benchmarkCuckooFilter(table Table, capacity uint, b *testing.B) {
	cf := NewCuckooFilter(table)
	rand.Seed(time.Now().UnixNano())
	successCount := 0
	insertedCount := 0
	existCount := 0
	errCount := 0
	const blockCount = 8
	start := time.Now().UnixNano()
	for i := 0; i < blockCount; i++ { // 槽位数量
		blockStart := time.Now().UnixNano()
		for k := uint(0); k < capacity/blockCount; k++ { // 1GB
			insertedCount++
			data := []byte(fmt.Sprintf("%d:%d", i, k))
			err := cf.InsertUnique(data)
			if err == nil {
				successCount++
			} else if err == ErrExist {
				existCount++
			} else {
				errCount++
			}
		}
		t := float64(time.Now().UnixNano()-start) / 1e9
		tBlock := float64(time.Now().UnixNano()-blockStart) / 1e9
		log.Printf(
			"Benchmark InsertUnique block=%.2f%%-%.2f%%, capacity=%d, t=%.1f(%.1f), inserted=%d(%.2f/s), success=%d(%.2f%%), exist=%d(%.2f%%), err=%d(%.2f%%), load=%.2f%%",
			100*float64(i)/blockCount, 100*float64(i+1)/blockCount, capacity, t, tBlock, insertedCount, float64(capacity/blockCount)/tBlock, successCount, float64(successCount)*100/float64(insertedCount),
			existCount, float64(existCount)*100/float64(insertedCount), errCount, float64(errCount)*100/float64(insertedCount), float64(100*successCount)/float64(capacity),
		)
	}
}

func BenchmarkCuckooFilter_MEMTable(b *testing.B) {
	const capacity = 1 << 30 // 1GB
	table := NewMemTable(capacity)
	defer table.Close()
	benchmarkCuckooFilter(table, capacity, b)
}

func BenchmarkCuckooFilter_MMAPTable(b *testing.B) {
	const capacity = 1 << 30 // 1GB
	tempFile := path.Join(os.TempDir(), fmt.Sprintf("cuckoo-%d.mmap", time.Now().UnixNano()))
	b.Logf("TempFile: %s", tempFile)
	table, err := NewMMAPTable(tempFile, capacity)
	defer table.Close()
	if err != nil {
		b.Fatalf("创建MMAPTable失败 > %s", err)
	}
	benchmarkCuckooFilter(table, capacity, b)
}
