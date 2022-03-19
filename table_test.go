package spellsql

import (
	"database/sql"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

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
	rows, err := db.Query(GetSqlStr("SHOW COLUMNS FROM student"))
	if err != nil {
		// glog.Errorf("mysql query is failed, err: %v, sqlStr: %v", err, sqlStr)
		return
	}
	defer rows.Close()

	filedNames := make([]string, 0, 10)
	for rows.Next() {
		var filedName string
		if err = rows.Scan(&filedName); err != nil {
			t.Log(err)
			continue
		}
		filedNames = append(filedNames, filedName)
	}
	t.Log(filedNames)
}
