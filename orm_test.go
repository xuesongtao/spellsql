package spellsql

import (
	"database/sql"
	"fmt"
	"testing"

	// _ "github.com/go-sql-driver/mysql"
	// gmysql "gorm.io/driver/mysql"
	// "gorm.io/gorm"
)

type Man struct {
	Id   int32  `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Age  int32  `json:"age,omitempty"`
	Addr string `json:"addr,omitempty"`
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

func InitMyDb(...uint8)  {
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

func TestGetCol(t *testing.T) {
	for i := 0; i < 1; i++ {
		tab := NewTable(db, "man")
		tab.initCol2InfoMap()
		t.Logf("%+v", tab.col2InfoMap)
		for k, v := range tab.col2InfoMap {
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
		Id: 1,
	}
	rows, err := NewTable(db).Delete(m).Exec()
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(rows.LastInsertId())
}

func TestDelete1(t *testing.T) {
	rows, err := NewTable(db, "man").Delete().Where("id=?", 1).Exec()
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
	rows, err := NewTable(db).Update(m).Where("id=?", 1).Exec()
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(rows.LastInsertId())
}

func TestFindOne(t *testing.T) {
	var m Man
	err := NewTable(db, "man").Select("*").Where("id=?", 1).FindOne(&m)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", m)
}

func TestFindForJoin(t *testing.T) {
	var m []Man
	sqlStr := GetSqlStr("SELECT m.name,m.age FROM man m JOIN student s ON m.id=s.u_id")
	NewTable(db).Raw(sqlStr).FindAll(&m)
	t.Log(m)
}

func BenchmarkFindOne(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var m Man
		_ = NewTable(db, "man").PrintSql(false).Select("name,age,addr").Where("id=?", 1).FindOne(&m)
	}

	// BenchmarkFindOne-8         32341             35948 ns/op            1576 B/op         39 allocs/op
	// BenchmarkFindOne-8         32796             36229 ns/op            1576 B/op         39 allocs/op
	// BenchmarkFindOne-8         32755             36180 ns/op            1576 B/op         39 allocs/op
}

// func BenchmarkFindOne1(b *testing.B) {
// 	for i := 0; i < b.N; i++ {
// 		var m Man
// 		gdb.Table("man").Find(&m, "id=?", 2)
// 	}

// 	// BenchmarkFindOne1-8        19682             61327 ns/op            3684 B/op         60 allocs/op
// 	// BenchmarkFindOne1-8        19852             60416 ns/op            3684 B/op         60 allocs/op
// 	// BenchmarkFindOne1-8        19795             60345 ns/op            3684 B/op         60 allocs/op
// }

func BenchmarkFindOne2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var m Man
		_ = NewTable(db, "man").PrintSql(false).Select("name,age,addr").Where("id=?", 2).QueryRowScan(&m.Id, &m.Age, &m.Addr)
	}

	// BenchmarkFindOne2-8        33466             35516 ns/op            1233 B/op         31 allocs/op
	// BenchmarkFindOne2-8        33404             35501 ns/op            1233 B/op         31 allocs/op
	// BenchmarkFindOne2-8        33453             35496 ns/op            1233 B/op         31 allocs/op
}

func BenchmarkFindOne3(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var m Man
		sqlStr := FmtSqlStr("SELECT name,age,addr FROM man WHERE id=?", 2)
		_ = db.QueryRow(sqlStr).Scan(&m.Id, &m.Age, &m.Addr)
	}

	// BenchmarkFindOne3-8        33596             35717 ns/op            1160 B/op         29 allocs/op
	// BenchmarkFindOne3-8        33660             35226 ns/op            1161 B/op         29 allocs/op
	// BenchmarkFindOne3-8        33541             35269 ns/op            1160 B/op         29 allocs/op
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

func BenchmarkFindAll(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var m []*Man
		_ = NewTable(db, "man").PrintSql(false).Select("*").Where("id>?", 1).FindAll(&m)
	}

	// BenchmarkFindAll-8         26055             43635 ns/op            3313 B/op         92 allocs/op
	// BenchmarkFindAll-8         25959             44419 ns/op            3313 B/op         92 allocs/op
	// BenchmarkFindAll-8         25070             44121 ns/op            3313 B/op         92 allocs/op
}

// func BenchmarkFindAll1(b *testing.B) {
// 	for i := 0; i < b.N; i++ {
// 		var m []Man
// 		// sqlStr := FmtSqlStr("SELECT * FROM man WHERE id>?", 1)
// 		gdb.Table("man").Find(&m, "id>?", 1)
// 		// b.Log(m)
// 	}

// 	// BenchmarkFindAll1-8        16104             77294 ns/op            5366 B/op         94 allocs/op
// 	// BenchmarkFindAll1-8        16206             72038 ns/op            5365 B/op         94 allocs/op
// 	// BenchmarkFindAll1-8        15954             71622 ns/op            5366 B/op         94 allocs/op
// }

func BenchmarkFindAll2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sqlStr := FmtSqlStr("SELECT name,age,addr FROM man WHERE id>?", 1)
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
