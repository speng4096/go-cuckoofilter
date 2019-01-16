package cuckoofilter

import (
	"github.com/dgryski/go-metro"
)

// 注意, 数据的fingerprint不能为0, 因为0为槽的默认值
func fingerprint(data []byte) byte {
	return byte(metro.Hash64(data, 2018)%255 + 1)
}

// Hash根据原始数据计算, 并小于buckets数量
func hashRaw(data []byte, numBuckets uint) uint {
	return uint(metro.Hash64(data, 2019)) % numBuckets
}

// HashAlt根据fingerprint与hash(不管是hash1还是hash2)计算, 并小于buckets数量
// 由于使用异或运算, 传入hash为hash1时, 计算得到hash2, 传入hash2时, 计算得到hash1
// numBuckets必须为2的幂, 这样只有fpHash尾部n位参与异或运算(n等于numBuckets的二进制长度)
// 否则会错位: a != hashAlt(hashAlt(a, ...), ...)
func hashAlt(hash uint, fp byte, numBuckets uint) uint {
	fpHash := uint(metro.Hash64([]byte{fp}, 2020))
	return (hash ^ fpHash) % numBuckets
}

func nextPow2(n uint64) uint {
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	n++
	return uint(n)
}
