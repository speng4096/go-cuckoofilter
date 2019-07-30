package cuckoofilter

import (
	"fmt"
	"github.com/edsrzf/mmap-go"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"os"
	"runtime"
)

type Table interface {
	// 获取指定索引位置的Bucket
	Bucket(index uint) (*Bucket, error)
	BucketNum() uint
	// 获取指定槽
	Slot(index uint, slot uint) (byte, error)
	// 设置槽
	SetSlot(index uint, slot uint, fingerprint byte) error
	// 编码, 可用于保存哈希表
	Encode(io.Writer) error
	// fingerprint数量
	IncrCount()
	DecrCount()
	Count() uint
	// 清空数据
	Truncate() error
	io.Closer
}

type MemTable struct {
	buckets   []Bucket
	bucketNum uint
	count     uint
}

func NewMemTable(capacity uint) *MemTable {
	capacity = nextPow2(uint64(capacity))
	bucketNum := capacity / slotSize
	if bucketNum == 0 {
		bucketNum = 1
	}
	t := &MemTable{bucketNum: bucketNum, count: 0}
	_ = t.Truncate()
	return t
}

func (t *MemTable) Truncate() error {
	t.buckets = make([]Bucket, t.bucketNum)
	for i := range t.buckets {
		t.buckets[i] = Bucket{}
	}
	t.count = 0
	return nil
}

func (t *MemTable) Bucket(index uint) (*Bucket, error) {
	return &t.buckets[index], nil
}

func (t *MemTable) Slot(index uint, offset uint) (byte, error) {
	return t.buckets[index][offset], nil
}

func (t *MemTable) SetSlot(index uint, offset uint, fingerprint byte) error {
	t.buckets[index][offset] = fingerprint
	return nil
}

func (t *MemTable) BucketNum() uint {
	return uint(t.bucketNum)
}

func (t *MemTable) IncrCount() {
	t.count++
}

func (t *MemTable) DecrCount() {
	t.count--
}

func (t *MemTable) Count() uint {
	return t.count
}

func (t *MemTable) Close() error {
	t.buckets = []Bucket{}
	runtime.GC()
	return nil
}

func (t *MemTable) Encode(writer io.Writer) error {
	for i := uint(0); i < t.bucketNum; i++ {
		if _, err := writer.Write(t.buckets[i][:]); err != nil {
			return errors.Wrap(err, "encode failed")
		}
	}
	return nil
}

type MMAPTable struct {
	m         mmap.MMap
	bucketNum uint
	count     uint
	file      *os.File
	capacity  uint
}

func NewMMAPTable(filename string, capacity uint) (*MMAPTable, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "open file failed")
	}

	capacity = nextPow2(uint64(capacity))
	if err := file.Truncate(int64(capacity)); err != nil {
		return nil, errors.Wrapf(err, "file.truncate(%d) failed", capacity)
	}
	m, err := mmap.Map(file, mmap.RDWR, 0)
	if err != nil {
		return nil, errors.Wrap(err, "mmap file failed")
	}

	bucketNum := capacity / slotSize
	if bucketNum == 0 {
		bucketNum = 1
	}
	return &MMAPTable{m, bucketNum, 0, file, capacity}, nil
}

func (f *MMAPTable) Truncate() error {
	if _, err := f.file.Seek(0, 0); err != nil {
		return errors.Wrap(err, "file.seek failed(0)")
	}
	if err := f.file.Truncate(0); err != nil {
		return errors.Wrap(err, "file.truncate(0) failed")
	}
	if err := f.file.Truncate(int64(f.capacity)); err != nil {
		return errors.Wrapf(err, "file.truncate(%d) failed", f.capacity)
	}
	f.count = 0
	return nil
}

func (f *MMAPTable) Bucket(index uint) (*Bucket, error) {
	var b Bucket
	s := f.m[index*slotSize : (index+1)*slotSize]
	copy(b[:], s)
	return &b, nil
}

func (f *MMAPTable) BucketNum() uint {
	return f.bucketNum
}

func (f *MMAPTable) Slot(index uint, slot uint) (byte, error) {
	return f.m[index*slotSize+slot], nil
}

func (f *MMAPTable) SetSlot(index uint, slot uint, fingerprint byte) error {
	a := index*slotSize + slot
	f.m[a] = fingerprint
	return nil
}

func (f *MMAPTable) IncrCount() {
	f.count++
}

func (f *MMAPTable) DecrCount() {
	f.count--
}

func (f *MMAPTable) Count() uint {
	return f.count
}

func (f *MMAPTable) Close() error {
	if err := f.file.Close(); err != nil {
		return errors.Wrap(err, "close file failed")
	}
	if err := f.m.Unmap(); err != nil {
		return errors.Wrap(err, "unmap failed")
	}
	return nil
}

func (f *MMAPTable) Encode(writer io.Writer) error {
	if _, err := writer.Write(f.m); err != nil {
		return errors.Wrap(err, "encode failed")
	}
	return nil
}

func NewMemTableFromReader(reader io.Reader) (*MemTable, error) {
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, errors.Wrap(err, "read failed")
	}

	capacity := len(bytes)
	if capacity%slotSize != 0 {
		return nil, fmt.Errorf("expected capacity to be multiuple of %d, got %d", slotSize, capacity)
	}

	bucketNum := uint(capacity) / slotSize
	if bucketNum == 0 {
		bucketNum = 1
	}
	count := uint(0) // fingerprint数量
	buckets := make([]Bucket, bucketNum)

	for i, b := range buckets {
		for j := range b {
			index := (i * len(b)) + j
			if bytes[index] != 0 {
				buckets[i][j] = bytes[index]
				count++
			}
		}
	}

	return &MemTable{
		buckets:   buckets,
		bucketNum: bucketNum,
		count:     count,
	}, nil
}
