package test

import (
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"math"
	"testing"

	"gitee.com/xuesongtao/spellsql/v2"
	"gitee.com/xuesongtao/spellsql/v2/dialect"
	// _ "github.com/lib/pq"
)

const (
	sureName = "xue1234"
	sureAge  = int32(18)
	sureAddr = "成都市"
)

// CREATE TABLE "public"."man" (
// 	"id" SERIAL PRIMARY KEY,
// 	"name" varchar(10) COLLATE "pg_catalog"."default" NOT NULL,
// 	"age" int4 NOT NULL,
// 	"addr" varchar(50) COLLATE "pg_catalog"."default",
// 	"hobby" varchar(255) COLLATE "pg_catalog"."default",
// 	"json_txt" text COLLATE "pg_catalog"."default",
// 	"nickname" varchar(30) COLLATE "pg_catalog"."default",
// 	"xml_txt" text COLLATE "pg_catalog"."default",
// 	"json1_txt" varchar(255) COLLATE "pg_catalog"."default",
// );

//   ALTER TABLE "public"."man"
// 	OWNER TO "postgres";

var (
	pgDb *sql.DB
)

func init() {
	var err error
	pgDb, err = sql.Open("postgres", "host=localhost port=5432 user=postgres password=123456 dbname=postgres sslmode=disable")
	if err != nil {
		panic(err)
	}
	err = pgDb.Ping()
	if err != nil {
		panic(err)
	}
	pgDb.SetMaxOpenConns(1)
	pgDb.SetMaxIdleConns(1)

	// 初始化 pg tmer
	spellsql.GlobalDbType(dialect.Postgres)
	pgDb.ExecContext(context.Background(), "TRUNCATE TABLE man RESTART IDENTITY")
}

func InitPgTestMain(t *testing.T, size ...int) {
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
		_, err := spellsql.NewTable(pgDb, "man").Insert(prepareMan).Exec()
		if err != nil {
			t.Fatal("prepare data failed:", err)
		}
	}
}

func TestLocalPg(t *testing.T) {
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

	tableObj := spellsql.NewTable(pgDb, "man")
	tableObj.SetMarshalFn(json.Marshal, "json_txt", "json1_txt")
	tableObj.SetMarshalFn(xml.Marshal, "xml_txt")
	res, err := tableObj.Insert(m).DbType(dialect.Postgres).Exec()
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

func TestInsertForPg(t *testing.T) {
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

		tableObj := spellsql.NewTable(pgDb, "man")
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

		tableObj := spellsql.NewTable(pgDb, "man")
		tableObj.SetMarshalFn(json.Marshal, "json_txt", "json1_txt")
		tableObj.SetMarshalFn(xml.Marshal, "xml_txt")
		var mm []interface{}
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

func TestDeleteForPg(t *testing.T) {
	m := Man{
		Id: 9,
	}
	_, err := spellsql.NewTable(pgDb).Delete(m).Exec()
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateForPg(t *testing.T) {
	m := Man{
		Name: "xue12",
		Age:  20,
		Addr: "测试",
		JsonTxt: Tmp{
			Name: "json",
			Data: "test update json marshal",
		},
	}

	tableObj := spellsql.NewTable(pgDb, "man")
	tableObj.SetMarshalFn(json.Marshal, "json_txt")
	_, err := tableObj.Update(m, "id=?", 2).Exec()
	if err != nil {
		t.Fatal(err)
	}
}

func TestRawForPg(t *testing.T) {
	InitPgTestMain(t)
	var m Man
	sqlObj := spellsql.NewCacheSql("SELECT name,age FROM man WHERE id=1")
	err := spellsql.NewTable(pgDb).Raw(sqlObj).FindOne(&m)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", m)
	if !Equal(m.Name, sureName) || !Equal(m.Age, sureAge) {
		t.Error(NoEqErr)
	}
}

func TestFindOneForPg(t *testing.T) {
	InitPgTestMain(t)
	var m Man
	tableObj := spellsql.NewTable(pgDb)
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

func TestFindAllForPg(t *testing.T) {
	InitPgTestMain(t, 20)
	t.Run("ummarshal", func(t *testing.T) {
		var m []Man
		var err error
		tableObj := spellsql.NewTable(pgDb)
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
		tableObj := spellsql.NewTable(pgDb).Select("name").From("man")
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
