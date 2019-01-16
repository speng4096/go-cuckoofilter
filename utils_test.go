package cuckoofilter

import (
	"bufio"
	"math/rand"
	"os"
	"testing"
)

func TestHash(t *testing.T) {
	fp, err := os.Open("/usr/share/dict/words")
	if err != nil {
		t.Fatal(err)
	}
	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		data := scanner.Bytes()
		fp := fingerprint(data)
		numBucket := nextPow2(uint64(rand.Intn(99999)) + 1024) // 太小容易碰撞
		hash1 := hashRaw(data, numBucket)
		hash2 := hashAlt(hash1, fp, numBucket)
		hash3 := hashAlt(hash2, fp, numBucket)
		hash4 := hashAlt(hash3, fp, numBucket)
		if hash1 == hash2 {
			t.Fatalf("hash1与hash2应不相等, %d!=%d", hash1, hash2)
		}
		if hash1 != hash3 {
			t.Fatalf("hash2应能通过hashAlt计算得到hash1, %d!=%d", hash1, hash3)
		}
		if hash4 != hash2 {
			t.Fatalf("hash2通过两次hashAlt计算应与自己相等, %d!=%d", hash4, hash2)
		}
	}
}
