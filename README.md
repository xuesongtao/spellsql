# spellsql

* `spellsql` 拼接器:
    > 1.使用 `sync.Pool`, `strings.Builder` 等提高 `sql` 拼接工具的性能  
    > 2.💯覆盖使用场景  
    > 3.支持 可控打印 `sql` 最终的 `log`; 非法字符自动转义; 支持格式化 `sql` 等  

* 为了解决满足性能和释放双手添加了 `orm` 功能, 支持: `mysql`, `pg`
    > 1.新增/更新: 支持通过 `struct` 解析值进行操作; 支持对字段进行 **序列化** 操作; 支持设置**别名, 设置默认值**等  
    > 2.删除: 支持通过 `struct` 解析值进行  
    > 3.查询: 支持单表/多表查询; 支持对结果进行回调处理; 查询性能接近原生; 支持对结果映射到 `struct/map/slice/单字段`等

在公司技术选型中，大多数 ORM 框架比较重，且性能与重量成正比。为了追求极致性能（接近原生 `database/sql`）和开发效率，我们开发了 spellsql：

1.  提供了灵活且安全的 SQL 拼接工具。
2.  在此基础上封装了轻量级的 ORM 功能，满足大部分业务场景的需求。

## 安装

```bash
go get -u gitee.com/xuesongtao/spellsql/v2
```

## 快速开始

### 1. 占位符使用

spellsql 提供了三种占位符来满足不同的 SQL 拼接需求：

- **`?`**: 直接根据参数类型自动填充。

  ```go
  // 自动推导类型
  sql := NewSql("SELECT * FROM user WHERE name = ? AND age = ?", "test", 20).GetSqlStr()
  // => SELECT * FROM user WHERE name = "test" AND age = 20

  // 支持切片展开
  sql := NewSql("SELECT * FROM user WHERE id IN (?)", []int{1, 2, 3}).GetSqlStr()
  // => SELECT * FROM user WHERE id IN (1,2,3)
  ```

- **`?d`**: 将数字型字符串转为数字，其他类型转义为 0。常用于表名或明确的数字字段。

  ```go
  sql := NewSql("SELECT * FROM user WHERE id = ?d", "123").GetSqlStr()
  // => SELECT * FROM user WHERE id = 123
  ```

- **`?v`**: 原样输出字符串（不加引号），适用于表名、列名或子查询。
  ```go
  // 危险！请确保参数完全可控
  sql := NewSql("SELECT * FROM ?v WHERE id = ?d", "my_table", "100").GetSqlStr()
  // => SELECT * FROM my_table WHERE id = 100
  ```
  > ⚠️ **注意**: `?v` 不会进行转义处理，请勿直接用于外部用户输入，以避免 SQL 注入风险。

### 2. 基础 CRUD (SQL 构建器)

#### 插入 (Insert)

```go
s := NewSql("INSERT INTO sys_user (username, password)")
s.SetInsertValues("xuesongtao", "123456")
s.SetInsertValues("admin", "654321")
s.GetSqlStr()
// Output:
// INSERT INTO sys_user (username, password) VALUES ("xuesongtao", "123456"), ("admin", "654321");
```

#### 查询 (Select)

```go
s := NewSql("SELECT * FROM user u LEFT JOIN role r ON u.id = r.user_id")
s.SetWhere("u.age > ?", 18)
s.SetOrWhere("u.status = ?", 1)
s.GetTotalSqlStr() // 获取统计 SQL
s.SetLimit(0, 10)
s.GetSqlStr()
// Output:
// SELECT COUNT(*) FROM user u LEFT JOIN role r ON u.id = r.user_id WHERE u.age > 18 OR u.status = 1;
// SELECT * FROM user u LEFT JOIN role r ON u.id = r.user_id WHERE u.age > 18 OR u.status = 1 LIMIT 0, 10;
```

#### 更新 (Update)

```go
s := NewSql("UPDATE sys_user SET")
s.SetUpdateValue("login_count", 1)
s.SetUpdateValueArgs("last_login = ?", time.Now())
s.SetWhereArgs("id = ?", 123)
s.GetSqlStr()
// Output:
// UPDATE sys_user SET login_count = 1, last_login = "2023-10-27 10:00:00" WHERE id = 123;
```

#### 删除 (Delete)

```go
s := NewSql("DELETE FROM sys_user")
s.SetWhere("status = ?", -1)
s.GetSqlStr()
// Output:
// DELETE FROM sys_user WHERE status = -1;
```

### 3. ORM 功能 (对象关系映射)

spellsql 提供了一个轻量级的 ORM 模块 `spellsql_orm`，用于将数据库记录映射到 Go 结构体。

#### 定义结构体

```go
type User struct {
    Id      int32  `json:"id,omitempty"`
    Name    string `json:"name,omitempty"`
    Age     int32  `json:"age,omitempty"`
    Address string `json:"address,omitempty"`
}
```

#### 插入数据

```go
// 方式一：自动解析结构体
user := User{Name: "test", Age: 20}
rows, err := InsertForObj(db, "user_table", user)

// 方式二：使用构建器
sqlObj := NewSql("INSERT INTO user_table (name, age) VALUES (?, ?)", user.Name, user.Age)
rows, err := ExecForSql(db, sqlObj)
```

#### 查询数据

**单条查询:**

```go
var user User
// 自动映射到结构体
err := NewTable(db, "user_table").
    Select("id, name, age").
    Where("id = ?", 1).
    FindOne(&user)

// 如果字段名与数据库不一致，可以使用 TagAlias
```

**多条查询:**

```go
var users []*User
err := NewTable(db, "user_table").
    Where("age > ?", 18).
    FindAll(&users, func(row interface{}) error {
        u := row.(*User)
        // 可选：在此处修改查询结果（闭包回调）
        return nil
    })
```

**高级查询 (原SQL映射):**

```go
var userMap map[string]interface{}
sqlObj := NewSql("SELECT name, age FROM user WHERE id = ?", 1)
err := FindOne(db, sqlObj, &userMap)
```

#### 更新与删除

```go
// 更新
user := User{Id: 1, Name: "updated_name"}
_ = NewTable(db).Update(user, "id=?", 1).Exec()

// 删除
_ = NewTable(db).Delete(User{Id: 1}).Exec()
```

## 项目结构

该项目结构清晰，主要分为以下几个核心模块：

- **`builder/`**: SQL 语法构建核心(推荐使用进行sql拼接)。
  - 包含 `Insert`, `Delete`, `Update`, `Select` 和 `Where` 的构建逻辑。
  - 使用 `Builder` 模式将参数安全地拼接成 SQL 字符串。
- **`dialect/`**: 数据库方言适配器。
  - 定义了 `Dialect` 接口，支持 MySQL 和 PostgreSQL。
  - 负责处理特定数据库的语法差异（如占位符、转义字符、LIMIT 语法）。
- **`internal/`**: 内部工具包。
  - **Cache**: 使用 LRU 算法缓存表结构信息，提高反射性能。
  - **Scan**: 高效处理数据库返回的 `NULL` 类型（`sql.NullString`, `sql.NullInt64` 等）。
  - **Escape**: SQL 字符转义处理。
- **`utils/`**: 通用工具函数。
  - 包含字符串处理、切片去重、类型转换等辅助函数。
- **`orm_*`**: 对象关系映射层。
  - 提供了 `NewTable`, `Insert`, `Update`, `Delete`, `Select` 等高级 API。
  - 自动处理 `struct` 到 `table` 的映射，支持自定义 Tag 和序列化。
- **`spellsql_*`**: 轻量级 ORM 模块，封装了常用的 CRUD 操作。

## 其他功能

### 1. Search After (深分页/游标查询)

- `searchafter` 模块，专门用于处理深分页场景（类似 ES Search After），支持基于上一次查询结果的下一页拉取，避免大数据量下的 offset 性能衰减。

### 2. 模型转换

- `convert` 模块，提供了结构体相互转换(业务场景: po 与 vo 层对象转换)，方便在不同层之间传递数据。

## 致谢

* 使用可以参考 `orm_test.go` 和 `example_orm_test.go`
* 在连表查询时, 如果两个表的列名相同查询结果会出现错误, 我们可以通过根据别名来区分, 或者直接调用 `Query` 来自行对结果进行处理(注: 调用 `Query` 时需要处理 `Null` 类型)
