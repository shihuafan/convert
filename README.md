# convert
通过go/ast生成两个相似的struct的convert函数
## 1、 下载&安装
```shell
go install github.com/shihuafan/convert@latest
```
## 2、 使用
```
example
├── a
│   └── a.go
├── b
│   └── b.go
└── convert.go
```
a.go
```go
type A struct {
    Name string
    Age  *int8
    High *int
}
```
b.go
```go
type B struct {
    Name *string
    Age  int
    High int
}
```
创建convert.go文件, 添加`go:generate convert -from=a.A -to=b.B`。
并确保两个struct所在的包已经被当前包import。

convert.go
```go
package main

import (
    "convert/example/a"
    "convert/example/b"
)

//go:generate convert -src=a.A -tgt=b.B
func ConvertAToB(a *a.A) *b.B {
    return nil
}
```
执行`go generate`，会看到函数已经被替换

convert.go
```go
package main

import (
    "convert/example/a"
    "convert/example/b"
)

//go:generate convert -src=a.A -tgt=b.B
func ConvertAToB(a *a.A) *b.B {
    b := &b.B{}
    if a.Age != nil {
        b.Age = int(*a.Age)
    }
    if a.High != nil {
        b.High = *a.High
    }
    b.Name = &a.Name
    return b
}
```
## 3、 Why not json or reflect
在大多数情况下，两个struct并不完全相同，而不论是json还是反射，都是在运行时执行，在代码阶段是无法具体知道有哪些字段是会被修改的，这样会带来很大的不确定性。
因此，使用了代码生成，在convert之后由开发者进行一定的检查，并对一些无法支持的字段进行补充。