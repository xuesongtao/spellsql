package test

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"math"
	"testing"

	"gitee.com/xuesongtao/spellsql/v2"
	"gitee.com/xuesongtao/spellsql/v2/dialect"
	_ "github.com/glebarez/go-sqlite"
)

// 测试表
var sqliteDb *sql.DB

// CREATE TABLE man (
// 	"id" INTEGER PRIMARY KEY AUTOINCREMENT,
// 	"name" varchar(10) NOT NULL,
// 	"age" int NOT NULL,
// 	"addr" varchar(50) DEFAULT NULL,
// 	"hobby" varchar(255) DEFAULT '',
// 	"json_txt" text,
// 	"nickname" varchar(30) DEFAULT '',
// 	"xml_txt" text,
// 	"json1_txt" varchar(255) DEFAULT NULL
// );

func init() {
	var err error
	sqliteDb, err = sql.Open("sqlite", "file:sqlite.db?cache=shared&mode=rwc")
	if err != nil {
		panic(err)
	}

	// 关键：设置连接池参数（SQLite有特殊限制）
	// 默认 sql.Open 会保留无限连接，但 SQLite 并发写容易锁库
	sqliteDb.SetMaxOpenConns(1) // 强烈建议设为1，避免 "database is locked" 错误
	sqliteDb.SetMaxIdleConns(1) // 保持一个空闲连接足矣

	// 测试连通性
	if err := sqliteDb.Ping(); err != nil {
		panic(err)
	}
	sqliteDb.Exec("DELETE FROM man")
	sqliteDb.Exec("DELETE FROM sqlite_sequence WHERE name = 'man';")
}

func InitTestMainForSqlite(t *testing.T, size ...int) {
	defaultSize := 1
	if len(size) > 0 {
		defaultSize = size[0]
	}
	for i := 0; i < defaultSize; i++ {
		prepareMan := Man{
			Name:    sureName,
			Age:     sureAge,
			Addr:    sureAddr,
			JsonTxt: Tmp{Name: "json", Data: "test json marshal"},
			// XmlTxt:   Tmp{Name: "xml", Data: "test xml marshal"},
			Json1Txt: Tmp{Name: "json1", Data: "test json1 marshal"},
		}

		// 强制插入 ID 为 1 的数据（假设表已 TRUNCATE）
		// 或者使用 InsertsIg (Insert Ignore) 防止冲突
		_, err := spellsql.NewTable(sqliteDb, "man").DbType(dialect.SQLite).Insert(prepareMan).Exec()
		if err != nil {
			t.Fatal("prepare data failed:", err)
		}
	}
}

func TestLocalForSqlite(t *testing.T) {
	m := Man{
		Name:  "xue1234",
		Age:   18,
		Addr:  "成都市",
		Hobby: "打篮球",
		JsonTxt: Tmp{
			Name: "json",
			Data: "test json marshal",
		},
		XmlTxt: Tmp{
			Name: "xml",
			Data: "test xml marshal",
		},
		Json1Txt: Tmp{
			Name: "json1",
			Data: "test json1 marshal",
		},
	}

	tableObj := spellsql.NewTable(sqliteDb, "man").DbType(dialect.SQLite)
	tableObj.SetMarshalFn(json.Marshal, "json_txt", "json1_txt")
	tableObj.SetMarshalFn(xml.Marshal, "xml_txt")
	res, err := tableObj.Insert(m).DbType(dialect.SQLite).Exec()
	if err != nil {
		t.Fatal(err)
	}
	r, err := res.RowsAffected()
	if err != nil {
		t.Fatal(err)
	}
	if r == 0 {
		t.Error("insert is failed")
	}
}

func TestInsertForSqlite(t *testing.T) {
	t.Run("insert", func(t *testing.T) {
		m := Man{
			Name:  "xue1234",
			Age:   18,
			Addr:  "成都市",
			Hobby: "打篮球",
			JsonTxt: Tmp{
				Name: "json",
				Data: "test json marshal",
			},
			XmlTxt: Tmp{
				Name: "xml",
				Data: "test xml marshal",
			},
			Json1Txt: Tmp{
				Name: "json1",
				Data: "test json1 marshal",
			},
		}

		tableObj := spellsql.NewTable(sqliteDb, "man").DbType(dialect.SQLite)
		tableObj.SetMarshalFn(json.Marshal, "json_txt", "json1_txt")
		tableObj.SetMarshalFn(xml.Marshal, "xml_txt")
		res, err := tableObj.Insert(m).Exec()
		if err != nil {
			t.Fatal(err)
		}
		r, err := res.RowsAffected()
		if err != nil {
			t.Fatal(err)
		}
		if r == 0 {
			t.Error("insert is failed")
		}
	})

	t.Run("insert many", func(t *testing.T) {
		m := Man{
			Name:  "xue1234",
			Age:   18,
			Addr:  "成都市",
			Hobby: "打篮球",
			JsonTxt: Tmp{
				Name: "json",
				Data: "test json marshal",
			},
			XmlTxt: Tmp{
				Name: "xml",
				Data: "test xml marshal",
			},
			Json1Txt: Tmp{
				Name: "json1",
				Data: "test json1 marshal",
			},
		}

		tableObj := spellsql.NewTable(sqliteDb, "man").DbType(dialect.SQLite)
		tableObj.SetMarshalFn(json.Marshal, "json_txt", "json1_txt")
		tableObj.SetMarshalFn(xml.Marshal, "xml_txt")
		var mm []any
		size := 3
		for i := 0; i < size; i++ {
			tmp := m
			tmp.Name += "_" + fmt.Sprint(i)
			if i == 1 {
				tmp.Hobby = ""
				tmp.Age = 1
			}
			mm = append(mm, tmp)
		}
		res, err := tableObj.Insert(mm...).Exec()
		if err != nil {
			t.Fatal(err)
		}
		r, err := res.RowsAffected()
		if err != nil {
			t.Fatal(err)
		}
		if r == 0 || r != int64(size) {
			t.Error("insert is failed")
		}
	})
}

func TestDeleteForSqlite(t *testing.T) {
	m := Man{
		Id: 9,
	}
	_, err := spellsql.NewTable(sqliteDb).DbType(dialect.SQLite).Delete(m).Exec()
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateForSqlite(t *testing.T) {
	m := Man{
		Name: "xue12",
		Age:  20,
		Addr: "测试",
		JsonTxt: Tmp{
			Name: "json",
			Data: "test update json marshal",
		},
	}

	tableObj := spellsql.NewTable(sqliteDb, "man").DbType(dialect.SQLite)
	tableObj.SetMarshalFn(json.Marshal, "json_txt")
	_, err := tableObj.Update(m, "id=?", 2).Exec()
	if err != nil {
		t.Fatal(err)
	}
}

func TestInitManForSqlite(t *testing.T) {
	InitTestMainForSqlite(t)
}

func TestRawForSqlite(t *testing.T) {
	InitTestMainForSqlite(t)
	var m Man
	sqlObj := spellsql.NewCacheSql("SELECT name,age FROM man WHERE id=1")
	err := spellsql.NewTable(sqliteDb).DbType(dialect.SQLite).Raw(sqlObj).FindOne(&m)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", m)
	if !Equal(m.Name, sureName) || !Equal(m.Age, sureAge) {
		t.Error(NoEqErr)
	}
}

func TestFindOneForSqlite(t *testing.T) {
	InitTestMainForSqlite(t)
	var m Man
	tableObj := spellsql.NewTable(sqliteDb).DbType(dialect.SQLite)
	tableObj.SetUnmarshalFn(json.Unmarshal, "json_txt", "json1_txt")
	tableObj.SetUnmarshalFn(xml.Unmarshal, "xml_txt")
	err := tableObj.SelectAuto(Man{}).Where("id=1").FindOneFn(&m)
	if err != nil {
		t.Fatal(err)
	}

	jsonTxt := Tmp{
		Name: "json",
		Data: "test json marshal",
	}
	// xmlTxt := Tmp{
	// 	Name: "xml",
	// 	Data: "test xml marshal",
	// }
	json1Txt := Tmp{
		Name: "json1",
		Data: "test json1 marshal",
	}
	t.Logf("%+v", m)
	if !Equal(m.Name, sureName) || !Equal(m.Age, sureAge) || !StructValEqual(m.JsonTxt, jsonTxt) || !StructValEqual(m.Json1Txt, json1Txt) {
		t.Error(NoEqErr)
	}
}

func TestFindAllForSqlite(t *testing.T) {
	InitTestMainForSqlite(t, 20)
	t.Run("ummarshal", func(t *testing.T) {
		var m []Man
		var err error
		tableObj := spellsql.NewTable(sqliteDb).DbType(dialect.SQLite)
		tableObj.SetUnmarshalFn(json.Unmarshal, "json_txt", "json1_txt")
		tableObj.SetUnmarshalFn(xml.Unmarshal, "xml_txt")
		err = tableObj.SelectAuto(Man{}).Limit(1, 10).FindWhere(&m, "id>0")
		if err != nil {
			t.Fatal(err)
		}
		if len(m) == 0 {
			t.Error("res is null")
			return
		}

		jsonTxt := Tmp{
			Name: "json",
			Data: "test json marshal",
		}
		// xmlTxt := Tmp{
		// 	Name: "xml",
		// 	Data: "test xml marshal",
		// }
		json1Txt := Tmp{
			Name: "json1",
			Data: "test json1 marshal",
		}
		t.Logf("%+v", m)
		first := m[0]
		if !Equal(first.Name, sureName) || !Equal(first.Age, sureAge) || !StructValEqual(first.JsonTxt, jsonTxt) || !StructValEqual(first.Json1Txt, json1Txt) {
			t.Error(NoEqErr)
		}
	})

	t.Run("findAll page", func(t *testing.T) {
		size := 5
		tableObj := spellsql.NewTable(sqliteDb).DbType(dialect.SQLite).Select("name").From("man")
		var total int
		_ = tableObj.Count(&total)
		if total == 0 {
			return
		}

		totalPage := math.Ceil(float64(total) / float64(size))
		var names []string
		for page := int32(1); page <= int32(totalPage); page++ {
			var tmp []string
			newTableObj := tableObj.Clone()
			err := newTableObj.Clone().OrderBy("id ASC").Limit(page, int32(size)).FindAll(&tmp)
			if err != nil {
				t.Fatal(err)
			}
			names = append(names, tmp...)
		}
		// t.Logf("%+v", names)
		if !Equal(len(names), total) {
			t.Error(NoEqErr)
		}
	})
}
