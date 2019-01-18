package cuckoofilter

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"testing"
	"time"
)

func TestMMAPTable(t *testing.T) {
	tempFile := path.Join(os.TempDir(), fmt.Sprintf("cuckoo-%d.mmap", time.Now().UnixNano()))
	t.Logf("TempFile: %s", tempFile)
	fp, err := os.Create(tempFile)
	if err != nil {
		t.Fatalf("打开文件失败: %s", tempFile)
	}
	bucket0 := [4]byte{100, 101, 102, 103}
	_, err = fp.Write(bucket0[:])
	if err != nil {
		t.Fatalf("写入测试数据失败 > %s", err)
	}
	err = fp.Close()
	if err != nil {
		t.Fatalf("关闭测试文件失败 > %s", err)
	}

	table, err := NewMMAPTable(tempFile, 1<<20)
	defer table.Close()
	if err != nil {
		t.Fatalf("创建MMAPTable失败 > %s", err)
	}
	for i, v := range bucket0 {
		b, err := table.Slot(0, uint(i))
		if err != nil {
			t.Errorf("GetSlot失败 > %s", err)
		}
		if b != v {
			t.Errorf("GetSlot与存档数据不一致, %d!=%d", b, v)
		}
	}
	bucket0 = [4]byte{1, 2, 3, 4}
	for i, v := range bucket0 {
		if err := table.SetSlot(0, uint(i), v); err != nil {
			t.Errorf("SetSlot失败 > %s", err)
		}
	}
	for i, v := range bucket0 {
		b, err := table.Slot(0, uint(i))
		if err != nil {
			t.Errorf("GetSlot失败 > %s", err)
		}
		if b != v {
			t.Errorf("GetSlot不匹配, %d!=%d", b, v)
		}
	}
	b, err := table.Bucket(0)
	if err != nil {
		t.Errorf("GetBucker失败 > %s", err)
	}
	if *b != bucket0 {
		t.Errorf("GetBucker不匹配, %v", *b)
	}
}

func TestMemTable_Decode(t *testing.T) {
	data := Bucket{1, 2, 3, 4}
	buf := bytes.NewReader(data[:])
	table, err := NewMemTableFromReader(buf)
	defer table.Close()
	if err != nil {
		t.Errorf("恢复MemTable失败 > %s", err)
	}
	b, err := table.Bucket(0)
	if err != nil {
		t.Errorf("GetBucket失败 > %s", err)
	}
	if *b != data {
		t.Errorf("GetBucket不匹配, %v!=%v", *b, data)
	}
}

func TestMemTable_Encode(t *testing.T) {
	data := Bucket{1, 2, 3, 4}
	table := NewMemTable(uint(len(data)))
	defer table.Close()
	for i, v := range data {
		if err := table.SetSlot(0, uint(i), v); err != nil {
			t.Errorf("SetSlot失败 > %s", err)
		}
	}

	buf := bytes.NewBuffer([]byte{})
	if err := table.Encode(buf); err != nil {
		t.Errorf("Encode失败 > %s", err)
	}
	b := buf.Bytes()
	for i, v := range data {
		if b[i] != v {
			t.Errorf("Encode不匹配, %v!=%v", b, data)
			break
		}
	}
}

func TestMMAPTable_Decode(t *testing.T) {
	data := Bucket{1, 2, 3, 4}
	buf := bytes.NewReader(data[:])
	table, err := NewMemTableFromReader(buf)
	defer table.Close()
	if err != nil {
		t.Errorf("恢复MemTable失败 > %s", err)
	}
	b, err := table.Bucket(0)
	if err != nil {
		t.Errorf("GetBucket失败 > %s", err)
	}
	if *b != data {
		t.Errorf("GetBucket不匹配, %v!=%v", *b, data)
	}
}

func TestMMAPTable_Encode(t *testing.T) {
	data := Bucket{1, 2, 3, 4}
	tempFile := path.Join(os.TempDir(), fmt.Sprintf("cuckoo-%d.mmap", time.Now().UnixNano()))
	t.Logf("TempFile: %s", tempFile)
	table, err := NewMMAPTable(tempFile, 1<<20)
	defer table.Close()
	if err != nil {
		t.Fatalf("创建MMAPTable失败 > %s", err)
	}

	for i, v := range data {
		if err := table.SetSlot(0, uint(i), v); err != nil {
			t.Errorf("SetSlot失败 > %s", err)
		}
	}

	buf := bytes.NewBuffer([]byte{})
	if err := table.Encode(buf); err != nil {
		t.Errorf("Encode失败 > %s", err)
	}
	b := buf.Bytes()
	for i, v := range data {
		if b[i] != v {
			t.Errorf("Encode不匹配, %v!=%v", b, data)
			break
		}
	}
}
