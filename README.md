#### 1. 介绍
- 通过 `sync.Pool`, `strings.Builder` 等实现的较高性能 sql 拼接工具(对比测试传统通过 `fmt` 进行拼接)
- 自动打印 sql 最终的 log
- 非法字符会自动转移


#### 2. 使用
- 可以参考 `getsqlstr_test.go` 里的测试方法
- 目前支持占位符 `?, ?d, ?v`


#### 3. 其他
- 欢迎大佬们指正, 希望大佬给 **start**