package spellsql

import (
	"database/sql"
	"fmt"
	"testing"

	// _ "github.com/go-sql-driver/mysql"
	// gmysql "gorm.io/driver/mysql"
	// "gorm.io/gorm"
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
	db    *sql.DB
	dbErr error
	// gdb   *gorm.DB
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

// func init() {
// 	gdb, dbErr = gorm.Open(gmysql.Open("root:12345678@tcp(127.0.0.1:3306)/mystudy"), &gorm.Config{})
// 	if dbErr != nil {
// 		panic(dbErr)
// 	}
// }

func TestParseTable(t *testing.T) {
	var m Man
	c, v, e := NewTable(db).getHandleTableCol2Val(m, false, "man")
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
}

func TestGetCol(t *testing.T) {
	for i := 0; i < 1; i++ {
		tab := NewTable(db, "man")
		tab.initCacheCol2InfoMap()
		t.Logf("%+v", tab.cacheCol2InfoMap)
		for k, v := range tab.cacheCol2InfoMap {
			fmt.Println(k, v)
		}
	}
}

func TestInsert(t *testing.T) {
	m := Man{
		Name: "xue1234",
		Age:  18,
		Addr: "成都市",
	}
	rows, err := NewTable(db).Insert(m)
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(rows.LastInsertId())
}

func TestDelete(t *testing.T) {
	m := Man{
		Id: 9,
	}
	rows, err := NewTable(db).Delete(m).Exec()
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(rows.LastInsertId())
}

func TestDelete1(t *testing.T) {
	rows, err := NewTable(db, "man").Delete().Where("id=?", 11).Exec()
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(rows.LastInsertId())
}

func TestUpdate(t *testing.T) {
	m := Man{
		Name: "xuesongtao",
		Age:  20,
		Addr: "测试",
	}
	rows, err := NewTable(db).Update(m).Where("id=?", 7).Exec()
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(rows.LastInsertId())
}

func TestFindOne(t *testing.T) {
	var m Man
	err := NewTable(db, "man").Select("name,age").Where("id=?", 1).FindOne(&m)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", m)
}

func TestFindOne1(t *testing.T) {
	var (
		name string
		age  int
	)
	err := NewTable(db, "man").Select("name,age").Where("id=?", 1).FindOne(&name, &age)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v %d", name, age)
}

func TestFindForJoin(t *testing.T) {
	var m []Man
	sqlStr := GetSqlStr("SELECT m.name,m.age,s.nickname FROM man m JOIN student s ON m.id=s.u_id")
	NewTable(db).Raw(sqlStr).FindAll(&m)
	t.Log(m)
}

// FindOne 性能对比
//  go test -benchmem -run=^$ -bench ^BenchmarkFindOne gitee.com/xuesongtao/spellsql -v -count=5

// func BenchmarkFindOneGorm(b *testing.B) {
// 	for i := 0; i < b.N; i++ {
// 		var m Man
// 		gdb.Table("man").Find(&m, "id=?", 2)
// 	}

// 	// BenchmarkFindOneGorm-8        19682             61327 ns/op            3684 B/op         60 allocs/op
// 	// BenchmarkFindOneGorm-8        19852             60416 ns/op            3684 B/op         60 allocs/op
// 	// BenchmarkFindOneGorm-8        19795             60345 ns/op            3684 B/op         60 allocs/op
// }

func BenchmarkFindOneOrmQueryRowScan(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var m Man
		_ = NewTable(db, "man").IsPrintSql(false).Select("name,age,addr").Where("id=?", 2).QueryRowScan(&m.Id, &m.Age, &m.Addr)
	}

	// BenchmarkFindOneOrmQueryRowScan-8          33057             35859 ns/op            1232 B/op         31 allocs/op
	// BenchmarkFindOneOrmQueryRowScan-8          33205             35904 ns/op            1232 B/op         31 allocs/op
	// BenchmarkFindOneOrmQueryRowScan-8          33292             35981 ns/op            1232 B/op         31 allocs/op
}

func BenchmarkFindOneQueryRowScan(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var m Man
		sqlStr := FmtSqlStr("SELECT name,age,addr FROM man WHERE id=?", 2)
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
	var m []*Man
	err := NewTable(db, "man").Select("id,name,age,addr").Where("id>?", 1).FindAll(&m)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", m)
	for _, v := range m {
		fmt.Println(v)
	}
}

func TestFindAll1(t *testing.T) {
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
	for _, v := range m {
		fmt.Println(v)
	}
}

func TestFindAll2(t *testing.T) {
	var names []string
	fn := func(_row interface{}) error {
		n := _row.(string)
		fmt.Println(n)
		return nil
	}
	err := NewTable(db, "man").Select("addr").Where("id>?", 1).FindAll(&names, fn)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(len(names), names)
}

// func BenchmarkFindAllGorm(b *testing.B) {
// 	for i := 0; i < b.N; i++ {
// 		var m []*Man
// 		// sqlStr := FmtSqlStr("SELECT * FROM man WHERE id>?", 1)
// 		gdb.Table("man").Find(&m, "id>?", 1)
// 		// b.Log(m)
// 	}

// 	// BenchmarkFindAllGorm-8             15712             75883 ns/op            6118 B/op        127 allocs/op
// 	// BenchmarkFindAllGorm-8             15595             77351 ns/op            6118 B/op        127 allocs/op
// 	// BenchmarkFindAllGorm-8             15823             75588 ns/op            6118 B/op        127 allocs/op
// }

func BenchmarkFindAllQuery(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sqlStr := FmtSqlStr("SELECT name,age,addr FROM man WHERE id>? LIMIT ?, ?", 1, 0, 10)
		rows, err := db.Query(sqlStr)
		if err != nil {
			return
		}
		defer rows.Close()

		res := make([]*Man, 0, 10)
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
	}

	// BenchmarkFindAll2-8        27165             42027 ns/op            1448 B/op         50 allocs/op
	// BenchmarkFindAll2-8        27633             43206 ns/op            1448 B/op         50 allocs/op
	// BenchmarkFindAll2-8        26761             43401 ns/op            1448 B/op         50 allocs/op
}

func BenchmarkFindAllOrm(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m := make([]*Man, 0, 10)
		_ = NewTable(db, "man").IsPrintSql(false).Select("name,age,addr").Where("id>?", 1).Limit(0, 10).FindAll(&m)
	}

	// BenchmarkFindAllOrm-8              26319             45615 ns/op            2288 B/op         74 allocs/op
	// BenchmarkFindAllOrm-8              26319             45538 ns/op            2288 B/op         74 allocs/op
	// BenchmarkFindAllOrm-8              26275             45809 ns/op            2288 B/op         74 allocs/op
}

func TestFindWhereForOneFiled(t *testing.T) {
	var name string
	err := NewTable(db, "man").Select("name").FindWhere(&name, "id=?", 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", name)
}

func TestFindWhereForStruct(t *testing.T) {
	var m Man
	err := NewTable(db).FindWhere(&m, "id=?", 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", m)
}

func TestFindWhereForSliceStruct(t *testing.T) {
	var m []Man
	err := NewTable(db).FindWhere(&m, "id>?", 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", m)

	var m1 []*Man
	err = NewTable(db).FindWhere(&m1, "id>?", 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", m1)
	for _, v := range m1 {
		t.Logf("%+v", v)
	}
}

func TestFindWhere(t *testing.T) {
	var m []Man
	err := FindWhere(db, "man", &m, "id>1")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", m)
}

func TestFindWhereCount(t *testing.T) {
	var total int32
	NewTable(db, "man").SelectCount().FindWhere(&total, "id>?", 1)
	t.Log(total)

	Count(db, "man", &total, "id>1")
	t.Log(total)
}

func TestSelectFindWhere(t *testing.T) {
	var m Man
	SelectFindWhere(db, "name", "man", &m, "id=?", 1)
	t.Log(m)
}
