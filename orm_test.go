package spellsql

import (
	"database/sql"
	"fmt"
	"strconv"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	gmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	sureName = "xuesongtao"
	sureAge  = int32(20)
)

// CREATE TABLE `man` (
// 	`id` int NOT NULL AUTO_INCREMENT,
// 	`name` varchar(10) NOT NULL,
// 	`age` int NOT NULL,
// 	`addr` varchar(50) DEFAULT NULL,
// 	`hobby` varchar(255) DEFAULT '',
// 	`ext` text,
// 	`nickname` varchar(30) DEFAULT '',
// 	PRIMARY KEY (`id`)
// )

type Man struct {
	Id       int32  `json:"id,omitempty" gorm:"id" db:"id"`
	Name     string `json:"name,omitempty" gorm:"name" db:"name"`
	Age      int32  `json:"age,omitempty" gorm:"age" db:"age"`
	Addr     string `json:"addr,omitempty" gorm:"addr" db:"addr"`
	NickName string `json:"nickname" gorm:"nickname" db:"nickname"`
}

type Student struct {
	Id        int32  `json:"id,omitempty" gorm:"id" db:"id"`
	UId       int32  `json:"u_id,omitempty" gorm:"u_id" db:"u_id"`
	ClassName string `json:"class_name,omitempty" gorm:"class_name" db:"class_name"`
	Nickname  string `json:"nickname,omitempty" gorm:"nickname" db:"nickname"`
	Name      string `json:"name,omitempty" gorm:"name" db:"name"`
}

type Tmp struct {
	Name string
}

var (
	db     *sql.DB
	dbErr  error
	gdb    *gorm.DB
	sqlxdb *sqlx.DB
)

func init() {
	// db=Db
	InitMyDb(1)
}

func InitMyDb(...uint8) {
	db, dbErr = sql.Open("mysql", "root:12345678@tcp(127.0.0.1:3306)/mystudy")
	if dbErr != nil {
		panic(dbErr)
	}
	dbErr = db.Ping()
	if dbErr != nil {
		panic(dbErr)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
}

func init() {
	dbsn := "root:12345678@tcp(127.0.0.1:3306)/mystudy"
	gdb, dbErr = gorm.Open(gmysql.Open(dbsn), &gorm.Config{})
	if dbErr != nil {
		panic(dbErr)
	}

	// 也可以使用MustConnect连接不成功就panic
	sqlxdb, dbErr = sqlx.Connect("mysql", dbsn)
	if dbErr != nil {
		fmt.Printf("connect DB failed, err:%v\n", dbErr)
		return
	}
	// 设置最大连接数
	db.SetMaxOpenConns(20)
	// 设置最大闲置数
	db.SetMaxIdleConns(10)
}

func TestParseTable(t *testing.T) {
	m := Man{
		Id:       1,
		Name:     "xuesongtao",
		Age:      20,
		Addr:     "四川成都",
		NickName: "a-tao",
	}
	c, v, e := NewTable(db).getHandleTableCol2Val(m, false, "man")
	t.Log(c, v, e)

	c, v, e = NewTable(db).getHandleTableCol2Val(m, false, "man")
	t.Log(c, v, e)
}

func TestGetNullType(t *testing.T) {
	// CREATE TABLE test_col (
	// 	id INT auto_increment PRIMARY KEY,
	// 	l_tinyint TINYINT,
	// 	l_int int,
	// 	l_long LONG,
	// 	l_float FLOAT,
	// 	l_dec DECIMAL,
	// 	l_char CHAR(10),
	// 	l_varchar VARCHAR(10),
	// 	l_text LONGTEXT
	// ) COMMENT '测试字段';

	type TestColInfo struct {
		Id       int32  `json:"id,omitempty"`
		LTinyint int8   `json:"l_tinyint,omitempty"`
		LInt     int32  `json:"l_int,omitempty"`
		LLong    string `json:"l_long,omitempty"`
		LFloat   string `json:"l_float,omitempty"`
		LDec     string `json:"l_dec,omitempty"`
		LChar    string `json:"l_char,omitempty"`
		LVarchar string `json:"l_varchar,omitempty"`
		LText    string `json:"l_text,omitempty"`
	}
	var data TestColInfo
	err := FindWhere(db, "test_col", &data, "id=1")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(data)

	for i := 0; i < 1; i++ {
		tab := NewTable(db, "test_col")
		tab.initCacheCol2InfoMap()
		for k, v := range tab.cacheCol2InfoMap {
			fmt.Printf("k: %v, v: %+v\n", k, v)
		}
	}
}

func TestInsert(t *testing.T) {
	m := Man{
		Name: "xue1234",
		Age:  18,
		Addr: "成都市",
	}

	t.Run("insert for obj", func(t *testing.T) {
		_, err := InsertForObj(db, "man", m)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("insert for sql", func(t *testing.T) {
		sqlObj := NewCacheSql("INSERT INTO man (name,age,addr) VALUES (?, ?, ?)", m.Name, m.Age, m.Addr)
		_, err := ExecForSql(db, sqlObj)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestDelete(t *testing.T) {
	m := Man{
		Id: 9,
	}
	t.Run("delete for obj", func(t *testing.T) {
		_, err := NewTable(db).Delete(m).Exec()
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("delete for where", func(t *testing.T) {
		_, err := DeleteWhere(db, "man", "id=?", 9)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("delete for sql", func(t *testing.T) {
		sqlObj := NewCacheSql("DELETE FROM man WHERE id=?", 9)
		_, err := ExecForSql(db, sqlObj)
		if err != nil {
			t.Fatal(err)
		}
	})

}

func TestUpdate(t *testing.T) {
	m := Man{
		Name: "xue12",
		Age:  20,
		Addr: "测试",
	}

	t.Run("update for obj", func(t *testing.T) {
		_, err := UpdateForObj(db, "man", m, "id=?", 7)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("update for sql", func(t *testing.T) {
		sqlObj := NewCacheSql("UPDATE man SET name=?,age=?,addr=? WHERE id=?", m.Name, m.Age, m.Addr, 7)
		_, err := ExecForSql(db, sqlObj)
		if err != nil {
			t.Fatal(err)
		}
	})

}

func TestFindOne(t *testing.T) {
	t.Log("find one test start")
	t.Run("select 2 struct", func(t *testing.T) {
		var m Man
		err := NewTable(db, "man").Select("name,age").Where("id=?", 1).FindOne(&m)
		if err != nil {
			t.Fatal(err)
		}

		if !equal(m.Name, sureName) || !equal(m.Age, sureAge) {
			t.Error(noEqErr)
		}
	})

	t.Run("selectAuto 2 struct", func(t *testing.T) {
		var m Man
		err := SelectFindOne(db, m, "man", FmtSqlStr("id=?", 1), &m)
		if err != nil {
			t.Fatal(err)
		}
		if !equal(m.Name, sureName) || !equal(m.Age, sureAge) {
			t.Error(noEqErr)
		}
	})

	t.Run("findOne for sql", func(t *testing.T) {
		var m Man
		err := FindOne(db, NewCacheSql("SELECT name,age FROM man WHERE id=?", 1), &m)
		if err != nil {
			t.Fatal(err)
		}
		if !equal(m.Name, sureName) || !equal(m.Age, sureAge) {
			t.Error(noEqErr)
		}
	})

	t.Run("findOne one field", func(t *testing.T) {
		var name string
		err := NewTable(db, "man").Select("name").FindWhere(&name, "id=?", 1)
		if err != nil {
			t.Fatal(err)
		}

		if !equal(name, sureName) {
			t.Error(noEqErr)
		}
	})

	t.Run("findOne many field", func(t *testing.T) {
		var (
			name string
			age  int32
		)
		err := NewTable(db, "man").Select("name,age").Where("id=?", 1).FindOne(&name, &age)
		if err != nil {
			t.Fatal(err)
		}
		if !equal(name, sureName) || !equal(age, sureAge) {
			t.Error(noEqErr)
		}
	})

	t.Run("findOne 2 map", func(t *testing.T) {
		var b map[string]string
		err := NewTable(db, "man").Select("name,age").Where("id=?", 1).FindOne(&b)
		if err != nil {
			t.Fatal(err)
		}
		if !equal(b["name"], sureName) || !equal(b["age"], fmt.Sprintf("%d", sureAge)) {
			t.Error(noEqErr)
		}
	})

	t.Run("findOne selectCallBack 2 struct", func(t *testing.T) {
		var m Man
		tmpName := "被修改了哦"
		tmpAge := int32(1000)
		err := FindOneFn(db, NewCacheSql("SELECT name,age FROM man WHERE id=?", 1), &m, func(_row interface{}) error {
			v := _row.(*Man)
			v.Name = tmpName
			v.Age = tmpAge
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if !equal(m.Name, tmpName) || !equal(m.Age, tmpAge) {
			t.Error(noEqErr)
		}
	})

	t.Run("findOne selectCallBack 2 one field", func(t *testing.T) {
		var (
			name string
			tmp  = "被修改了哦"
		)
		err := FindOneFn(db, NewCacheSql("SELECT name FROM man WHERE id=?", 1), &name, func(_row interface{}) error {
			v := _row.(*string)
			*v = tmp
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if !equal(name, tmp) {
			t.Error(noEqErr)
		}
	})

	t.Run("findOne selectCallBack map", func(t *testing.T) {
		var (
			tmp = "被修改了哦"
		)
		var b map[string]string
		err := NewTable(db).SelectAuto(Man{}).Where("id=1").FindOneFn(&b, func(_row interface{}) error {
			v := _row.(map[string]string)
			v["name"] = tmp
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if !equal(b["name"], tmp) {
			t.Error(noEqErr)
		}
	})

	t.Log("find one test end")
}

func TestFindWhere(t *testing.T) {
	t.Log("find where test start")
	t.Run("findWhere 2 one field", func(t *testing.T) {
		var name string
		err := NewTable(db, "man").Select("name").FindWhere(&name, "id=?", 1)
		if err != nil {
			t.Fatal(err)
		}
		if !equal(name, sureName) {
			t.Error(noEqErr)
		}
	})

	t.Run("findWhere 2 struct", func(t *testing.T) {
		var m Man
		err := FindWhere(db, "man", &m, "id=?", 1)
		if err != nil {
			t.Fatal(err)
		}
		if !equal(m.Name, sureName) || !equal(m.Age, sureAge) {
			t.Error(noEqErr)
		}
	})

	t.Run("findWhere 2 struct slice", func(t *testing.T) {
		var m []Man
		err := FindWhere(db, "man", &m, "id>0")
		if err != nil {
			t.Fatal(err)
		}
		if len(m) < 1 {
			t.Error("select res is no ok")
			return
		}

		// 按 id=1 判断
		if !equal(m[0].Name, sureName) || !equal(m[0].Age, sureAge) {
			t.Error(noEqErr)
		}
	})

	t.Run("findWhere 2 map", func(t *testing.T) {
		var b map[string]string
		err := FindWhere(db, "man", &b, "id=?", 1)
		if err != nil {
			t.Fatal(err)
		}
		if !equal(b["name"], sureName) || !equal(b["age"], fmt.Sprintf("%d", sureAge)) {
			t.Error(noEqErr)
		}
	})

	t.Run("findWhere 2 map slice", func(t *testing.T) {
		var b []map[string]string
		err := NewTable(db, "man").Select("name,age").FindWhere(&b, "id>0")
		if err != nil {
			t.Fatal(err)
		}
		if len(b) < 1 {
			t.Error("select res is no ok")
			return
		}

		// 按 id=1 判断
		if !equal(b[0]["name"], sureName) || !equal(b[0]["age"], fmt.Sprintf("%d", sureAge)) {
			t.Error(noEqErr)
		}
	})
	t.Log("find where test end")
}

func TestFindForJoin(t *testing.T) {
	// 连表查询时, 如果两个表有相同名字查询结果会出现错误, 推荐使用别名来区分/使用Query 对结果我们自己进行处理
	t.Run("find simple join", func(t *testing.T) {
		var m []Man
		sqlStr := NewCacheSql("SELECT m.name,m.age FROM man m JOIN student s ON m.id=s.u_id WHERE m.id=1")
		err := NewTable(db).Raw(sqlStr).FindAll(&m)
		if err != nil {
			t.Fatal(err)
		}
		if len(m) < 1 {
			t.Error("select res is no ok")
			return
		}
		if !equal(m[0].Name, sureName) || !equal(m[0].Age, sureAge) {
			t.Error(noEqErr)
		}
	})

	t.Run("find all join", func(t *testing.T) {
		sqlStr := NewCacheSql("SELECT m.name,m.age FROM man m JOIN student s ON m.id=s.u_id WHERE m.id=1")
		rows, err := NewTable(db).Raw(sqlStr).Query()
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()

		var m []Man
		for rows.Next() {
			var v Man
			err = rows.Scan(&v.Name, &v.Age)
			if err != nil {
				t.Fatal(err)
			}
			m = append(m, v)
		}

		if len(m) < 1 {
			t.Error("select res is no ok")
			return
		}

		if !equal(m[0].Name, sureName) || !equal(m[0].Age, sureAge) {
			t.Error(noEqErr)
		}
	})
}

func TestCount(t *testing.T) {
	var (
		total1, total2, total3 int32
	)
	err := NewTable(db, "man").SelectCount().FindWhere(&total1, "id>?", 1)
	if err != nil {
		t.Fatal(err)
	}
	err = Count(db, "man", &total2, "id>1")
	if err != nil {
		t.Fatal(err)
	}
	err = NewTable(db, "man").SelectAll().Where("id>?", 1).Count(&total3)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("total: ", total2)
	if total1 != total2 || total2 != total3 {
		t.Error(noEqErr)
	}
}

// FindOne 性能对比, 以下是在 mac11 m1 上测试
// go test -benchmem -run=^$ -bench ^BenchmarkFindOne gitee.com/xuesongtao/spellsql -v -count=5

func BenchmarkFindOneGorm(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var m Man
		gdb.Table("man").Find(&m, "id=?", 1)
	}

	// BenchmarkFindOneGorm-8        19682             61327 ns/op            3684 B/op         60 allocs/op
	// BenchmarkFindOneGorm-8        19852             60416 ns/op            3684 B/op         60 allocs/op
	// BenchmarkFindOneGorm-8        19795             60345 ns/op            3684 B/op         60 allocs/op
}

func BenchmarkFindOneOrmQueryRowScan(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var m Man
		_ = NewTable(db, "man").IsPrintSql(false).Select("name,age,addr").Where("id=?", 1).QueryRowScan(&m.Id, &m.Age, &m.Addr)
	}

	// BenchmarkFindOneOrmQueryRowScan-8          33057             35859 ns/op            1232 B/op         31 allocs/op
	// BenchmarkFindOneOrmQueryRowScan-8          33205             35904 ns/op            1232 B/op         31 allocs/op
	// BenchmarkFindOneOrmQueryRowScan-8          33292             35981 ns/op            1232 B/op         31 allocs/op
}

func BenchmarkFindOneQueryRowScan(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var m Man
		sqlStr := FmtSqlStr("SELECT name,age,addr FROM man WHERE id=?", 1)
		_ = db.QueryRow(sqlStr).Scan(&m.Id, &m.Age, &m.Addr)
	}

	// BenchmarkFindOneQueryRowScan-8             33396             35710 ns/op            1160 B/op         29 allocs/op
	// BenchmarkFindOneQueryRowScan-8             33398             36411 ns/op            1160 B/op         29 allocs/op
	// BenchmarkFindOneQueryRowScan-8             32521             36563 ns/op            1160 B/op         29 allocs/op
}

func BenchmarkFindOneOrm(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var m Man
		_ = NewTable(db, "man").IsPrintSql(false).Select("name,age,addr").Where("id=?", 1).FindOne(&m)
	}

	// BenchmarkFindOneOrm-8                      31897             37022 ns/op            1633 B/op         39 allocs/op
	// BenchmarkFindOneOrm-8                      32440             36693 ns/op            1633 B/op         39 allocs/op
	// BenchmarkFindOneOrm-8                      32326             36890 ns/op            1633 B/op         39 allocs/op
}

func TestFindAll(t *testing.T) {
	t.Log("find all test start")
	t.Run("findAll 2 struct ptr slice", func(t *testing.T) {
		var m []*Man
		err := NewTable(db, "man").Select("id,name,age,addr").Where("id>?", 0).FindAll(&m)
		if err != nil {
			t.Fatal(err)
		}
		if len(m) < 1 {
			t.Error("select res is no ok")
			return
		}

		if !equal(m[0].Name, sureName) || !equal(m[0].Age, sureAge) {
			t.Error(noEqErr)
		}
	})

	t.Run("findAll 2 map slice", func(t *testing.T) {
		var m []map[string]string
		err := NewTable(db, "man").Select("id,name,age,addr").Where("id>?", 0).FindAll(&m)
		if err != nil {
			t.Fatal(err)
		}
		if len(m) < 1 {
			t.Error("select res is no ok")
			return
		}

		age, _ := strconv.Atoi(m[0]["age"])
		if !equal(m[0]["name"], sureName) || !equal(int32(age), sureAge) {
			t.Error(noEqErr)
		}
	})

	t.Run("query", func(t *testing.T) {
		rows, err := NewTable(db).SelectAuto(Man{}).Where("id>?", 0).Query()
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()

		var m []Man
		for rows.Next() {
			var (
				id, age        int
				name, nickname string
				addr           sql.NullString
			)
			err = rows.Scan(&id, &name, &age, &addr, &nickname)
			if err != nil {
				t.Log(err)
			}
			m = append(m, Man{
				Name:     name,
				Age:      int32(age),
			})
			// t.Log(id, name, age, addr.String, nickname)
		}
		if !equal(m[0].Name, sureName) || !equal(m[0].Age, sureAge) {
			t.Error(noEqErr)
		}
	})

	t.Run("findAll oneField slice", func(t *testing.T) {
		var names []string
		err := NewTable(db, "man").Select("name").Where("id>?", 0).FindAll(&names)
		if err != nil {
			t.Fatal(err)
		}
		if len(names) < 1 {
			t.Error("select res is no ok")
			return
		}
		if !equal(names[0], sureName) {
			t.Error(noEqErr)
		}
	})

	t.Run("findAll selectCallBack struct slice", func(t *testing.T) {
		var m []*Man
		tmp := "被修改了"
		err := NewTable(db, "man").Select("id,name,age,addr").Where("id>?", 0).FindAll(&m, func(_row interface{}) error {
			v := _row.(*Man)
			if v.Id == 1 {
				v.Name = tmp

			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(m) < 1 {
			t.Error("select res is no ok")
			return
		}

		if !equal(m[0].Name, tmp) || !equal(m[0].Age, sureAge) || !equal(m[0].Addr, "") {
			t.Error(noEqErr)
		}
	})

	t.Run("findAll selectCallBack map slice", func(t *testing.T) {
		var b []map[string]string
		tmp := "被修改了"
		err := NewTable(db).SelectAuto(Man{}).Where("id>0").FindAll(&b, func(_row interface{}) error {
			v := _row.(map[string]string)
			if v["id"] == "1" {
				v["name"] = tmp
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(b) < 1 {
			t.Error("select res is no ok")
			return
		}
		age, _ := strconv.Atoi(b[0]["age"])
		if !equal(b[0]["name"], tmp) || !equal(int32(age), sureAge) {
			t.Error(noEqErr)
		}
	})

	t.Log("find all test end")
}

func TestSqlxSelect(t *testing.T) {
	var m []*Man
	sqlStr := FmtSqlStr("SELECT id,name,age,addr FROM man WHERE id>? LIMIT ?, ?", 1, 0, 10)
	err := sqlxdb.Select(&m, sqlStr)
	if err != nil { // 没有处理 NULL
		t.Fatal(err)
	}
	t.Log(m)
	for _, v := range m {
		t.Log(v)
	}
}

// 以下是在 mac11 m1 上测试
// go test -benchmem -run=^$ -bench ^BenchmarkFindAll gitee.com/xuesongtao/spellsql -v -count=5

func BenchmarkFindAllGorm(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var m []*Man
		// sqlStr := FmtSqlStr("SELECT * FROM man WHERE id>?", 1)
		gdb.Table("man").Limit(10).Find(&m, "id>?", 1)
		// b.Log(m)
	}

	// BenchmarkFindAllGorm-8             15201             78782 ns/op            6815 B/op        167 allocs/op
	// BenchmarkFindAllGorm-8             15229             79158 ns/op            6815 B/op        167 allocs/op
	// BenchmarkFindAllGorm-8             15264             78660 ns/op            6815 B/op        167 allocs/op
}

func BenchmarkFindAllSqlx(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var m []*Man
		sqlStr := FmtSqlStr("SELECT name,age,addr FROM man WHERE id>? LIMIT ?, ?", 1, 0, 10)
		sqlxdb.Select(&m, sqlStr)
		// b.Log(m)
	}

	// 说明: sqlx 不能自动处理 null 这里的查询结果不全
	// BenchmarkFindAllSqlx-8             25459             46214 ns/op            2049 B/op         63 allocs/op
	// BenchmarkFindAllSqlx-8             26474             45306 ns/op            2049 B/op         63 allocs/op
	// BenchmarkFindAllSqlx-8             26432             45002 ns/op            2049 B/op         63 allocs/op
}

func BenchmarkFindAllQuery(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sqlStr := FmtSqlStr("SELECT name,age,addr FROM man WHERE id>? LIMIT ?, ?", 1, 0, 10)
		rows, err := db.Query(sqlStr)
		if err != nil {
			return
		}

		var res []*Man
		for rows.Next() {
			var info Man
			var addr sql.NullString
			err = rows.Scan(&info.Name, &info.Age, &addr)
			if err != nil {
				continue
			}
			info.Addr = addr.String
			res = append(res, &info)
		}
		rows.Close()
	}

	// BenchmarkFindAll2-8        27165             42027 ns/op            1448 B/op         50 allocs/op
	// BenchmarkFindAll2-8        27633             43206 ns/op            1448 B/op         50 allocs/op
	// BenchmarkFindAll2-8        26761             43401 ns/op            1448 B/op         50 allocs/op
}

func BenchmarkFindAllOrm(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var m []*Man
		_ = NewTable(db, "man").IsPrintSql(false).Select("name,age,addr").Where("id>?", 1).Limit(0, 10).FindAll(&m)
	}

	// BenchmarkFindAllOrm-8              26319             45615 ns/op            2288 B/op         74 allocs/op
	// BenchmarkFindAllOrm-8              26319             45538 ns/op            2288 B/op         74 allocs/op
	// BenchmarkFindAllOrm-8              26275             45809 ns/op            2288 B/op         74 allocs/op
}
