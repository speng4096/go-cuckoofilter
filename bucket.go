package cuckoofilter

const slotSize = 4

type Bucket [slotSize]byte

func (b *Bucket) IndexByte(fp byte) (uint, bool) {
	for i, v := range b {
		if v == fp {
			return uint(i), true
		}
	}
	return 0, false
}

func (b *Bucket) IndexEmpty() (uint, bool) {
	for i, v := range b {
		if v == 0 {
			return uint(i), true
		}
	}
	return 0, false
}

type Table interface {
	// 获取指定索引位置的Bucket
	Bucket(index uint) (*Bucket, error)
	BucketNum() (uint, error)
	// 获取指定槽
	Slot(index uint, slot uint) (byte, error)
	// 设置槽
	SetSlot(index uint, slot uint, fingerprint byte) error
}

type MemTable struct {
	buckets  []Bucket
	capacity uint
}

func NewMemTable(capacity uint) *MemTable {
	capacity = nextPow2(uint64(capacity)) / slotSize
	buckets := make([]Bucket, capacity)
	for i := range buckets {
		buckets[i] = Bucket{}
	}
	return &MemTable{buckets, capacity}
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

func (t *MemTable) BucketNum() (uint, error) {
	return uint(t.capacity), nil
}
