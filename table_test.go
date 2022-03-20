package spellsql

import (
	"database/sql"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

type Student struct {
	Id    int32  `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Age   int32  `json:"age,omitempty"`
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
	tab := NewTable(db, "student")
	tab.initFileMap()
	t.Log(tab.filedMap)
}

func TestInsert(t *testing.T) {
	s := Student{
		Name:  "xue",
		Age:   18,
		Addr:  "成都市",
		Hobby: []int{1, 2, 3},
	}
	rows, err := NewTable(db, "", "db").Insert(s)
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
		Name:  "xuesongtao",
		Age:   20,
		Addr:  "测试",
	}
	rows, err := NewTable(db).Update(s).Where("id=?", 1).Exec()
	if err != nil {
		t.Log(err)
		return 
	}
	t.Log(rows.LastInsertId())
}

func TestFind(t *testing.T) {
	var name string
	NewTable(db, "student").Select("name").Where("id=?", 2).Find(&name)
	t.Log(name)
}
