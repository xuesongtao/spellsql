# spellsql [![Open Source Love](https://badges.frapsoft.com/os/v1/open-source.svg?v=103)](https://gitee.com/xuesongtao/spellsql) 


#### 1. 介绍 
* 通过 `sync.Pool`,  `strings.Builder` 等实现的高性能 sql 拼接工具
* 具有: 可控打印 sql 最终的 log, 非法字符自动转义, 支持格式化 sql等
* 支持轻量级 `orm`, 性能方面接近原生(即: database/sql)
* 安装:  

```  
go get -u gitee.com/xuesongtao/spellsql
```

#### 2. 占位符
* 目前支持占位符 `?, ?d, ?v`, 说明如下:

##### 2.1 占位符 ?
* 直接根据 args 中类型来自动推动 arg 的类型, 使用如下:
1. 第一种用法: 根据 args 中类型来自动推动 arg 的类型  

```    
如: NewCacheSql("SELECT username, password FROM sys_user WHERE username = ? AND password = ?", "test", 123).GetSqlStr()
=> SELECT username, password FROM sys_user WHERE username = "test" AND password = 123
```

2. 第二种用法: 当 arg 为 []int, 暂时支持 []int, []int32, []int64  
  
```  
如: NewCacheSql("SELECT username, password FROM sys_user WHERE id IN (?)", []int{1, 2, 3}).GetSqlStr()
=> SELECT username, password FROM sys_user WHERE id IN (1,2,3)
```

##### 2.2 占位符 ?d
* 只会把数字型的字符串转为数字型, 如果是字母的话会被转义为 **0**, 如: `"123" => 123`; `[]string{"1", "2", "3"} => 1,2,3`, 如下:
第一种用法: 当 arg 为字符串时, 又想不加双引号就用这个  

```  
如: NewCacheSql("SELECT username, password FROM sys_user WHERE id = ?d", "123").GetSqlStr()
=> SELECT username, password FROM sys_user WHERE id = 123
```

第二种用法: 当 arg 为 []string, 又想把解析后的单个元素不加引号  

```  
如: NewCacheSql("SELECT username, password FROM sys_user WHERE id IN (?d)", []string{"1", "2", "3"}).GetSqlStr()
=> SELECT username, password FROM sys_user WHERE id IN (1,2,3)
```

##### 2.3 占位符为: ?v
* 这样会让字符串类型不加引号, 原样输出, 如: "test" => test;
第一种用法: 当 arg 为字符串时, 又想不加双引号就用这个, 注: 只支持 arg 为字符串类型  

```  
如: NewCacheSql("SELECT username, password FROM ?v WHERE id = ?d", "sys_user", "123").GetSqlStr()
=> SELECT username, password FROM sys_user WHERE id = 123
```

第二种用法: 子查询  

```  
如: NewCacheSql("SELECT u.username, u.password FROM sys_user su LEFT JOIN user u ON su.id = u.id WHERE u.id IN (?v)", FmtSqlStr("SELECT id FROM user WHERE name=?", "test").GetSqlStr()
=> SELECT u.username, u.password FROM sys_user su LEFT JOIN user u ON su.id = u.id WHERE u.id IN (SELECT id FROM user WHERE name="test");
```

*  **注:** 由于这种不会进行转义处理, 所有这种不推荐用于请求输入(外部非法输入)的内容, 会出现 **SQL 注入风险**; 当我们明确知道参数是干什么的可以使用会简化我们代码, 这里就不进行演示.

#### 3. spellsql 使用
* 可以参考 `getsqlstr_test.go` 里的测试方法
##### 3.1 新增  

```  
s := NewCacheSql("INSERT INTO sys_user (username, password, name)")
s.SetInsertValues("xuesongtao", "123456", "阿桃")
s.SetInsertValues("xuesongtao", "123456", "阿桃")
s.GetSqlStr()

// Output:
// INSERT INTO sys_user (username, password, name) VALUES ("test", 123456, "阿涛"), ("xuesongtao", "123456", "阿桃"), ("xuesongtao", "123456", "阿桃");
```

##### 3.2 删除  

```  
s := NewCacheSql("DELETE FROM sys_user WHERE id = ?", 123)
if true {
    s.SetWhere("name", "test")
}
s.GetSqlStr()
// Output:
// DELETE FROM sys_user WHERE id = 123 AND name = "test";
```

##### 3.3 查询  

```  
s := NewCacheSql("SELECT * FROM user u LEFT JOIN role r ON u.id = r.user_id")
s.SetOrWhere("u.name", "xue")
s.SetOrWhereArgs("(r.id IN (?d))", []string{"1", "2"})
s.SetWhere("u.age", ">", 20)
s.SetWhereArgs("u.addr = ?", "南部")
s.GetTotalSqlStr()
s.SetLimit(1, 10)
s.GetSqlStr()

// Output:
// sqlTotalStr: SELECT COUNT(*) FROM user u LEFT JOIN role r ON u.id = r.user_id WHERE u.name = "xue" OR (r.id IN (1,2)) AND u.age > 20 AND u.addr = "南部";
// sqlStr: SELECT * FROM user u LEFT JOIN role r ON u.id = r.user_id WHERE u.name = "xue" OR (r.id IN (1,2)) AND u.age > 20 AND u.addr = "南部" LIMIT 0, 10;
```

##### 3.4 修改  

```  
s := NewCacheSql("UPDATE sys_user SET")
idsStr := []string{"1", "2", "3", "4", "5"}
s.SetUpdateValue("name", "xue")
s.SetUpdateValueArgs("age = ?, score = ?", 18, 90.5)
s.SetWhereArgs("id IN (?d) AND name = ?", idsStr, "tao")
s.GetSqlStr()

// Output:
// UPDATE sys_user SET name = "xue", age = 18, score = 90.50 WHERE id IN (1,2,3,4,5) AND name = "tao";
```

#### 3.5 追加  

```  
s := NewCacheSql("INSERT INTO sys_user (username, password, age)")
s.SetInsertValuesArgs("?, ?, ?d", "xuesongtao", "123", "20")
s.Append("ON DUPLICATE KEY UPDATE username=VALUES(username)")
s.GetSqlStr()

// Output:
// INSERT INTO sys_user (username, password, age) VALUES ("xuesongtao", "123", 20) ON DUPLICATE KEY UPDATE username=VALUES(username);
```

##### 3.6 复用
*  1.  `NewCacheSql()` 获取的对象在调用 `GetSqlStr()` 后会重置并放入内存池, 是不能对结果进行再进行 `GetSqlStr()`, 当然你是可以对结果作为 `NewCacheSql()` 的入参进行使用以此达到复用, 这样代码看起来不是多优雅, 分页处理案例如下:  

```  
sqlObj := NewCacheSql("SELECT * FROM user_info WHERE status = 1")
handleFn := func(obj *SqlStrObj, page, size int32) {
    // 业务代码
    fmt.Println(obj.SetLimit(page, size).SetPrintLog(false).GetSqlStr())
}

// 每次同步大小
var (
    totalNum int32 = 30
    page int32 = 1
    size int32 = 10
    totalPage int32 = int32(math.Ceil(float64(totalNum / size)))
)

sqlStr := sqlObj.SetPrintLog(false).GetSqlStr("", "")
for page <= totalPage {
    handleFn(NewCacheSql(sqlStr), page, size)
    page++
}

// Output:
// SELECT * FROM user_info WHERE u_status = 1 LIMIT 0, 10;
// SELECT * FROM user_info WHERE u_status = 1 LIMIT 10, 10;
// SELECT * FROM user_info WHERE u_status = 1 LIMIT 20, 10;
```

*  `NewSql()` 的产生的对象不会放入内存池, 可以进行多次调用 `GetSqlStr()`, 对应上面的示例可以使用 `NewSql()` 再调用 `Clone()` 进行处理, 如下:  

```  
sqlObj := NewSql("SELECT u_name, phone, account_id FROM user_info WHERE u_status = 1")
handleFn := func(obj *SqlStrObj, page, size int32) {
    // 业务代码
    fmt.Println(obj.SetLimit(page, size).SetPrintLog(false).GetSqlStr())
}

// 每次同步大小
var (
    totalNum int32 = 30
    page int32 = 1
    size int32 = 10
    totalPage int32 = int32(math.Ceil(float64(totalNum / size)))
)

for page <= totalPage {
    handleFn(sqlObj.Clone(), page, size)
    page++
}

// Output:
// SELECT * FROM user_info WHERE u_status = 1 LIMIT 0, 10;
// SELECT * FROM user_info WHERE u_status = 1 LIMIT 10, 10;
// SELECT * FROM user_info WHERE u_status = 1 LIMIT 20, 10;
```

#### 4 orm 功能介绍
*  `spellsql_orm` 能够高效的处理单表 `CURD`. 在查询方面的性能接近原生, 其中做几个性能对比: **原生** >= **spellsql_orm** > **gorm** (orm_test.go 里有测试数据), 可以在 `dev` 分支上测试
* 支持自定义 `tag`, 默认 `json`  

```  
type Man struct {
    Id int32 `json:"id,omitempty"`
    Name string `json:"name,omitempty"`
    Age int32 `json:"age,omitempty"`
    Addr string `json:"addr,omitempty"`
}

```
##### 4.1 新增  

```  
m := Man{
    Name: "xue1234",
    Age: 18,
    Addr: "成都市",
}

// 1
rows, _ = InsertForObj(db, "man", m)
t.Log(rows.LastInsertId())

// 3
sqlObj := NewCacheSql("INSERT INTO man (name,age,addr) VALUES (?, ?, ?)", m.Name, m.Age, m.Addr)
rows, _ = ExecForSql(db, sqlObj)
t.Log(rows.LastInsertId())
```

##### 4.2 删除  

```  
m := Man{
    Id: 9,
}

// 1
rows, _ := NewTable(db).Delete(m).Exec()
t.Log(rows.LastInsertId())

// 2
rows, _ = DeleteWhere(db, "man", "id=?", 9)
t.Log(rows.LastInsertId())

// 3
sqlObj := NewCacheSql("DELETE FROM man WHERE id=?", 9)
rows, _ = ExecForSql(db, sqlObj)
t.Log(rows.LastInsertId())
```

##### 4.3 修改  

```  
m := Man{
    Name: "xue12",
    Age: 20,
    Addr: "测试",
}

// 1
rows, _ := NewTable(db).Update(m, "id=?", 7).Exec()
t.Log(rows.LastInsertId())

// 2
sqlObj := NewCacheSql("UPDATE man SET name=?,age=?,addr=? WHERE id=?", m.Name, m.Age, m.Addr, 7)
rows, _ = ExecForSql(db, sqlObj)
t.Log(rows.LastInsertId())
```

##### 4.4 查询
###### 4.4.1 单查询  

```  
var m Man
// 1
_ = NewTable(db, "man").Select("name,age").Where("id=?", 1).FindOne(&m)
t.Log(m)

// 2
_ = NewTable(db).SelectAuto("name,age", "man").Where("id=?", 1).FindOne(&m)
t.Log(m)

// 3
_ = FindOne(db, NewCacheSql("SELECT name,age FROM man WHERE id=?", 1), &m)
t.Log(m)

// 4, 对查询结果进行内容修改
_ = FindOneFn(db, NewCacheSql("SELECT name,age FROM man WHERE id=?", 1), &m, func(_row interface{}) error {
    v := _row.(*Man)
    v.Name = "被修改了哦"
    v.Age = 100000
    return nil
})
t.Log(m)

// 5
_ = FindWhere(db, "man", &m, "id=?", 1)
t.Log(m)

// 6
var b map[string]string
_ = FindWhere(db, "man", &b, "id=?", 1)
t.Log(b)
```

* 查询结果支持: `struct`, `map`, `单字段`
* 数据库返回的 `NULL` 类型, 不需要处理, `orm` 会自行处理, 如果传入空类型值会报错(如: sql.NullString)

###### 4.4.2 多条记录查询

```  
var m []*Man
err := NewTable(db, "man").Select("id,name,age,addr").Where("id>?", 1).FindAll(&m, func(_row interface{}) error {
    v := _row.(*Man)
    if v.Id == 5 {
        v.Name = "test"
    }
    fmt.Println(v.Id, v.Name, v.Age)
    return nil
})
if err != nil {
    t.Fatal(err)
}
t.Logf("%+v", m)
```

* 查询结果支持的切片类型: `struct`, `map`, `单字段`
* 数据库返回的 `NULL` 类型, 不需要处理, `orm` 会自行处理, 如果传入空类型值会报错(如: sql.NullString)

###### 4.4.3 别名查询

```  
type Tmp struct {
    Name1 string `json:"name_1,omitempty"`
    Age1 int32 `json:"age_1,omitempty"`
}

var m Tmp
err := NewTable(db).
TagAlias(map[string]string{"name_1": "name", "age_1": "age"}).
Select("name,age").
From("man").
FindWhere(&m, "id=?", 1)
if err != nil {
    t.Fatal(err)
}
```

###### 4.4.3 其他
* 使用可以参考 `orm_test.go` 和 `example_orm_test.go`
* 在连表查询时, 如果两个表的列名相同查询结果会出现错误, 我们可以通过根据别名来区分, 或者直接调用 `Query` 来自行对结果进行处理(注: 调用 `Query` 时需要处理 `Null` 类型)

#### 其他

* 欢迎大佬们指正, 希望大佬给 **star**，[to gitee](https://gitee.com/xuesongtao/spellsql)