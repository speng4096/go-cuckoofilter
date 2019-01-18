package cuckoofilter

import (
	"fmt"
	"math/rand"
	"time"
)

const maxCuckooCount = 500
const slotSize = 4

var ErrFull = fmt.Errorf("移位次数超过限制:%d", maxCuckooCount)
var ErrExist = fmt.Errorf("相同记录已存在")
var ErrNotExist = fmt.Errorf("未找到记录")

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

type CuckooFilter struct {
	table Table
}

func NewCuckooFilter(table Table) *CuckooFilter {
	return &CuckooFilter{
		table: table,
	}
}

func (c *CuckooFilter) Count() uint {
	return c.table.Count()
}

func (c *CuckooFilter) insert(i uint, fp byte) (bool, error) {
	bucket, err := c.table.Bucket(i)
	if err != nil {
		return false, err
	}
	if slot, ok := bucket.IndexEmpty(); ok {
		if err := c.table.SetSlot(i, slot, fp); err != nil {
			return false, err
		}
		c.table.IncrCount()
		return true, nil
	}
	return false, nil
}

func (c *CuckooFilter) info(data []byte) (uint, uint, byte, uint, error) {
	fp := fingerprint(data)
	num := c.table.BucketNum()
	i1 := hashRaw(data, num)
	i2 := hashAlt(i1, fp, num)
	return i1, i2, fp, num, nil
}

func (c *CuckooFilter) InsertUnique(data []byte) error {
	i1, i2, fp, num, err := c.info(data)
	if err != nil {
		return err
	}
	// 不重复插入
	if ok, err := c.lookup(fp, i1, i2); err != nil {
		return err
	} else if ok {
		return ErrExist
	}
	// 寻找空槽位放入fp
	if ok, err := c.insert(i1, fp); err != nil {
		return err
	} else if ok {
		return nil
	}
	if ok, err := c.insert(i2, fp); err != nil {
		return err
	} else if ok {
		return nil
	}
	// 通过移位放入fp
	var i uint
	if rand.Intn(2) == 0 {
		i = i1
	} else {
		i = i2
	}
	for k := 0; k < maxCuckooCount; k++ {
		j := uint(rand.Intn(slotSize))
		// 备份原fp
		rFp, err := c.table.Slot(i, j)
		if err != nil {
			return err
		}
		// 放入当前fp
		if err := c.table.SetSlot(i, j, fp); err != nil {
			return err
		}
		// 将原fp放入它的备用位置
		i := hashAlt(i, rFp, num)
		if ok, err := c.insert(i, rFp); err != nil {
			return err
		} else if ok {
			return nil
		}
		// 备用位置已满
		fp = rFp
	}
	return ErrFull
}

func (c *CuckooFilter) lookup(fp byte, i1 uint, i2 uint) (bool, error) {
	bucket, err := c.table.Bucket(i1)
	if err != nil {
		return false, err
	}
	if _, ok := bucket.IndexByte(fp); ok {
		return true, nil
	}

	bucket, err = c.table.Bucket(i2)
	if err != nil {
		return false, err
	}
	_, ok := bucket.IndexByte(fp)
	return ok, nil
}

func (c *CuckooFilter) Lookup(data []byte) (bool, error) {
	i1, i2, fp, _, err := c.info(data)
	if err != nil {
		return false, err
	}
	return c.lookup(fp, i1, i2)
}

func (c *CuckooFilter) Delete(data []byte) error {
	i1, i2, fp, _, err := c.info(data)
	if err != nil {
		return err
	}
	bucket, err := c.table.Bucket(i1)
	if err != nil {
		return err
	}
	if slot, ok := bucket.IndexByte(fp); ok {
		if err := c.table.SetSlot(i1, slot, 0); err != nil {
			return err
		}
		// 指纹只存在一个位置, 不用再校验备用位置
		c.table.DecrCount()
		return nil
	}
	// 校验备用位置
	bucket, err = c.table.Bucket(i2)
	if err != nil {
		return err
	}
	if slot, ok := bucket.IndexByte(fp); ok {
		if err := c.table.SetSlot(i2, slot, 0); err != nil {
			return err
		}
		c.table.DecrCount()
		return nil
	}
	// 注意, 表满时, data是Insert成功的, 不过它他踢走了另一个之前插入的元素
	return ErrNotExist
}

func init() {
	rand.Seed(time.Now().Unix())
}
