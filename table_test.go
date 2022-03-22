package spellsql

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

type Student struct {
	Id    int32  `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Age   string `json:"age,omitempty"`
	Addr  string `json:"addr,omitempty"`
	Hobby []int  `json:"hobby"`
}

var db *sql.DB
var dbErr error

func init() {
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

func TestGetCol(t *testing.T) {
	for i := 0; i < 10; i++ {
		tab := NewTable(db, "student")
		tab.initCol2InfoMap()
		t.Logf("%+v", tab.col2InfoMap)
	}
}

func TestInsert(t *testing.T) {
	s := Student{
		Name: "xue",
		Age:  "18",
		// Addr:  "成都市",
		Hobby: []int{1, 2, 3},
	}
	rows, err := NewTable(db).Insert(s)
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(rows.LastInsertId())
}

func TestDelete(t *testing.T) {
	s := Student{
		Id: 1,
	}
	rows, err := NewTable(db).Delete(s).Exec()
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(rows.LastInsertId())
}

func TestDelete1(t *testing.T) {
	rows, err := NewTable(db, "student").Delete().Where("id=?", 1).Exec()
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(rows.LastInsertId())
}

func TestUpdate(t *testing.T) {
	s := Student{
		Name: "xuesongtao",
		Age:  "20",
		Addr: "测试",
	}
	rows, err := NewTable(db).Update(s).Where("id=?", 1).Exec()
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(rows.LastInsertId())
}

func TestFindOne(t *testing.T) {
	// var name string
	// NewTable(db, "student").Select("name").Where("id=?", 2).Find(&name)
	// t.Log(name)

	var stu Student
	err := NewTable(db, "student").Select("*").Where("id=?", 21).FindOne(&stu)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", stu)
}

func BenchmarkFindOne(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var stu Student
		_ = NewTable(db, "student").PrintSql(false).Select("name,age,addr").Where("id=?", 2).FindOne(&stu)
	}

	// BenchmarkFindOne-8   	   30301	     37021 ns/op	    1920 B/op	      51 allocs/op
	// BenchmarkFindOne-8   	   30214	     36949 ns/op	    1760 B/op	      46 allocs/op
	// BenchmarkFindOne-8   	   32007	     37540 ns/op	    1760 B/op	      46 allocs/op
}

func BenchmarkFindOne1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var stu Student
		_ = NewTable(db, "student").PrintSql(false).Select("name,age,addr").Where("id=?", 2).QueryRowScan(&stu.Id, &stu.Age, &stu.Addr)
	}

	// BenchmarkFindOne1-8   	   33696	     35633 ns/op	    1268 B/op	      32 allocs/op
	// BenchmarkFindOne1-8   	   30796	     36616 ns/op	    1275 B/op	      31 allocs/op
	// BenchmarkFindOne1-8   	   33342	     36705 ns/op	    1275 B/op	      31 allocs/op
}

func BenchmarkFindOne2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var stu Student
		sqlStr := FmtSqlStr("SELECT name,age,addr FROM student WHERE id=?", 2)
		_ = db.QueryRow(sqlStr).Scan(&stu.Id, &stu.Age, &stu.Addr)
	}

	// BenchmarkFindOne2-8   	   32036	     36519 ns/op	    1219 B/op	      29 allocs/op
	// BenchmarkFindOne2-8   	   31311	     35898 ns/op	    1219 B/op	      29 allocs/op
	// BenchmarkFindOne2-8   	   31790	     35642 ns/op	    1219 B/op	      29 allocs/op
}

func TestFindAll(t *testing.T) {
	var stu []*Student
	err := NewTable(db, "student").Select("*").Where("id>?", 1).FindAll(&stu)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", stu)
	for _, v := range stu {
		fmt.Println(v)
	}
}

func BenchmarkFindAll(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var stu []*Student
		_ = NewTable(db, "student").PrintSql(false).Select("*").Where("id>?", 1).FindAll(&stu)
	}

	// BenchmarkFindAll-8   	   16917	     62777 ns/op	    9810 B/op	     408 allocs/op
	// BenchmarkFindAll-8   	   17708	     61489 ns/op	    9810 B/op	     408 allocs/op
	// BenchmarkFindAll-8   	   18070	     64316 ns/op	    9811 B/op	     408 allocs/op
}

func BenchmarkFindAll1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sqlStr :=  FmtSqlStr("SELECT name,age,addr FROM student WHERE id>?", 1)
		rows, err := db.Query(sqlStr)
		if err != nil {
			return
		}
		defer rows.Close()
		
		res := make([]*Student, 0, 10)
		for rows.Next() {
			var info Student
			var addr sql.NullString
			err = rows.Scan(&info.Name, &info.Age, &addr)
			if err != nil {
				continue
			}
			info.Addr = addr.String
			res = append(res, &info)
		}
	}

	// BenchmarkFindAll1-8   	   23088	     50797 ns/op	    5129 B/op	     178 allocs/op
	// BenchmarkFindAll1-8   	   22690	     51807 ns/op	    5129 B/op	     178 allocs/op
	// BenchmarkFindAll1-8   	   23044	     51137 ns/op	    5129 B/op	     178 allocs/op
}
