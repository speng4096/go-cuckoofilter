# go-cuckoofilter
根据论文 ["Cuckoo Filter: Practically Better Than Bloom"](https://www.cs.cmu.edu/~binfan/papers/conext14_cuckoofilter.pdf) 编写.

## 示例

```go
package main

import (
	"fmt"
	"github.com/spencer404/go-cuckoofilter"
)

func main() {
	table := cuckoofilter.NewMemTable(1 << 30) // Size: 1GB
	cf := cuckoofilter.NewCuckooFilter(table)

	err := cf.InsertUnique([]byte("Hello"))
	fmt.Println(err) // <nil>

	ok, err := cf.Lookup([]byte("Hello"))
	fmt.Println(ok, err) // true <nil>

	n := cf.Count()
	fmt.Println(n) // 1

	err = cf.Delete([]byte("Hello"))
	fmt.Println(err) // <nil>

	ok, err = cf.Lookup([]byte("Hello"))
	fmt.Println(ok, err) // false <nil>

	n = cf.Count()
	fmt.Println(n) // 0
}
```

