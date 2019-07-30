# go-cuckoofilter
CuckooFilter用于快速判断一个元素是否属于某个集合, 以少量错误率换取很高的空间利用率.

本项目将CuckooFilter存储部分接口化, 实现了两种存储方式(Table接口):

+ MemTable, 内存存储, 对应Golang中的[]byte类型
+ MMAPTable, 文件存储, 使用MMAP快速读写文件



## 下载

```bash
go get -u github.com/spencer404/go-cuckoofilter
```



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



## 主要操作

### 在内存中创建过滤器

```go
table := cuckoofilter.NewMemTable(capacity)
cf := cuckoofilter.NewCuckooFilter(table)
```

Capacity等于占用内存/文件的大小(字节), 项目默认配置fingerprint占用空间为1Byte. 

因此, `1 Capacity` == `1 Byte空间占用` == `最多容纳1个元素`.

业务中具体Capacity的数值, 理应在预计元素数量, 性能, 错误率之间寻找点.

但如果你像我一样懒的话, 直接取`预计元素数量 * 2`.

如果无法估计元素数量, 可以使用多层的过滤器, 当空间利用率过高时, 再增加一个新的过滤器.



### 在文件中创建过滤器

```go
table, err := NewMMAPTable(filename, capacity)
cf := cuckoofilter.NewCuckooFilter(table)
```

注意, 若程序意外退出, 未被刷入文件的数据将会丢失.

### 插入元素

```go
err := cf.InsertUnique(a)
```

其返回值err可能为:

+ `ErrExist`, 元素a已存在(可能是Lookup误报)
+ `nil`, 元素a已被插入到哈希表的空位中(哈希表接近饱和时, 可能会挪走旧元素以腾出空位, 引起Lookup误报)
+ `ErrFull`, 多次尝试后, 仍未能在哈希表中找到空位, 此时哈希表已接近饱和
+ 其他情况, 通常是底层存储出现了异常

### 是否存在

```go
ok, err := cf.Lookup(a)
```

+ 当`err!=nil`, 通常是底层存储出现了异常
+ 当`ok==true`时, 可能是误报: 存在已插入的元素b与a具有相同的哈希
+ 当`ok==false`时, 可能是误报: InsertUnique(b)时, 由于哈希表已接近饱和, a给b挪出空位后, a无处安放, 被移出了哈希表(其实可增加Rollback流程,使得`ok==false`是确定的)

### 删除元素

```go
err := cf.Delete(a)
```

+ 当`err!=nil`, 通常是底层存储出现了异常

+ 当`err==ErrNotExist`, 未找到元素a, 无法删除. 可能是误报, 原因见`是否存在 > 当ok==false` 

+ 注意, 只删除已插入过的元素, 若删除未曾插入的元素, 会使Lookup操作出现更多的误报.

  假设元素k未曾插入到过滤器中,  a已被插入到过滤器中, a与k具有相同的哈希, 那么Delete(k)时会返回成功, 但实际上把a给删掉了.



## 性能测试

### 测试平台

+ Intel Xeon L5640 @ 2.27GHz
+ Ubuntu 16.04
+ 普通SSD

+ 测试方法为插入Capacity条不重复的数据

### MemTable:InsertUnique

+ Capacity: 1073741824Byte (=1GB)
+ FingerprintSize: 1Byte
+ SlotSize: 4Byte

| 区间        | 已用时(区间用时) | 已写入(区间速率)     | OK(nil) | ErrExist | ErrFull | 空间利用率 |
| ----------- | ---------------- | -------------------- | ------- | -------- | ------- | ---------- |
| 0%-12.5%    | 83.1(83.1)       | 134217728(1613886/s) | 99.80%  | 0.20%    | 0.00%   | 12.48%     |
| 12.5%-25.0% | 168.3(85.2)      | 268435456(1575325/s) | 99.61%  | 0.39%    | 0.00%   | 24.90%     |
| 25.0%-37.5% | 253.5(85.2)      | 402653184(1575325/s) | 99.41%  | 0.59%    | 0.00%   | 37.28%     |
| 37.5%-50.0% | 339.8(86.3)      | 536870912(1555246/s) | 99.22%  | 0.78%    | 0.00%   | 49.61%     |
| 50.0%-62.5% | 431.5(91.7)      | 671088640(1463661/s) | 99.02%  | 0.97%    | 0.01%   | 61.89%     |
| 62.5%-75.0% | 586.3(154.8)     | 805306368(867040/s)  | 98.69%  | 1.17%    | 0.14%   | 74.02%     |
| 75.0%-87.5% | 1210.1(623.8)    | 939524096(215161/s)  | 97.51%  | 1.36%    | 1.13%   | 85.32%     |
| 87.5%-100%  | 3510.9(2300.8)   | 1073741824(58335/s)  | 93.69%  | 1.54%    | 4.77%   | 93.69%     |

### MMAPTable:InsertUnique

+ Capacity: 1073741824Byte (=1GB)
+ FingerprintSize: 1Byte
+ SlotSize: 4Byte

| 区间        | 已用时(区间用时) | 已写入(区间速率)    | OK(nil) | ErrExist | ErrFull | 空间利用率 |
| ----------- | ---------------- | ------------------- | ------- | -------- | ------- | ---------- |
| 0%-12.5%    | 247.5(247.5)     | 134217728(542292/s) | 99.80%  | 0.20%    | 0.00%   | 12.48%     |
| 12.5%-25.0% | 557.0(309.5)     | 268435456(433668/s) | 99.61%  | 0.39%    | 0.00%   | 24.90%     |
| 25.0%-37.5% | 790.5(233.5)     | 402653184(574830/s) | 99.41%  | 0.59%    | 0.00%   | 37.28%     |
| 37.5%-50.0% | 1084.8(294.3)    | 536870912(456097/s) | 99.22%  | 0.78%    | 0.00%   | 49.61%     |
| 50.0%-62.5% | 1535.3(450.6)    | 671088640(297894/s) | 99.02%  | 0.97%    | 0.01%   | 61.89%     |
| 62.5%-75.0% | 2134.0(598.6)    | 805306368(224202/s) | 98.69%  | 1.17%    | 0.14%   | 74.02%     |
| 75.0%-87.5% | 3770.3(1636.3)   | 939524096(82024/s)  | 97.51%  | 1.36%    | 1.13%   | 85.32%     |
| 87.5%-100%  | 8909.4(5139.1)   | 1073741824(26117/s) | 93.69%  | 1.54%    | 4.77%   | 93.69%     |



## 参考资料

+ [cmu:Cuckoo Filter: Practically Better Than Bloom](https://www.cs.cmu.edu/~binfan/papers/conext14_cuckoofilter.pdf)
+ [github:seiflotfy/cuckoofilter](https://github.com/seiflotfy/cuckoofilter)
+ [github:edsrzf/mmap-go](https://github.com/edsrzf/mmap-go)