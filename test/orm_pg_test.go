package internal

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"testing"

	"gitee.com/xuesongtao/spellsql"
	"gitee.com/xuesongtao/spellsql/test/internal"
	_ "github.com/lib/pq"
)

const (
	sureName = "xue1234"
	sureAge  = int32(18)
)

// CREATE TABLE "public"."man" (
// 	"id" int4 NOT NULL DEFAULT nextval('man_id_seq'::regclass),
// 	"name" varchar(10) COLLATE "pg_catalog"."default" NOT NULL,
// 	"age" int4 NOT NULL,
// 	"addr" varchar(50) COLLATE "pg_catalog"."default",
// 	"hobby" varchar(255) COLLATE "pg_catalog"."default",
// 	"json_txt" text COLLATE "pg_catalog"."default",
// 	"nickname" varchar(30) COLLATE "pg_catalog"."default",
// 	"xml_txt" text COLLATE "pg_catalog"."default",
// 	"json1_txt" varchar(255) COLLATE "pg_catalog"."default",
// 	CONSTRAINT "man_pkey" PRIMARY KEY ("id")
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
}

func TestInsertForPg(t *testing.T) {
	m := internal.Man{
		Name:  "xue1234",
		Age:   18,
		Addr:  "成都市",
		Hobby: "打篮球",
		JsonTxt: internal.Tmp{
			Name: "json",
			Data: "test json marshal",
		},
		XmlTxt: internal.Tmp{
			Name: "xml",
			Data: "test xml marshal",
		},
		Json1Txt: internal.Tmp{
			Name: "json1",
			Data: "test json1 marshal",
		},
	}

	tableObj := spellsql.NewTable(pgDb, "man").Tmer(spellsql.Pg("man"))
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
}

func TestDeleteForPg(t *testing.T) {
	m := internal.Man{
		Id: 9,
	}
	_, err := spellsql.NewTable(pgDb).Tmer(spellsql.Pg("man")).Delete(m).Exec()
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateForPg(t *testing.T) {
	m := internal.Man{
		Name: "xue12",
		Age:  20,
		Addr: "测试",
		JsonTxt: internal.Tmp{
			Name: "json",
			Data: "test update json marshal",
		},
	}

	tableObj := spellsql.NewTable(pgDb, "man").Tmer(spellsql.Pg("man"))
	tableObj.SetMarshalFn(json.Marshal, "json_txt")
	_, err := tableObj.Update(m, "id=?", 2).Exec()
	if err != nil {
		t.Fatal(err)
	}
}

func TestFindOneForPg(t *testing.T) {
	var m internal.Man
	tableObj := spellsql.NewTable(pgDb).Tmer(spellsql.Pg("man"))
	tableObj.SetUnmarshalFn(json.Unmarshal, "json_txt", "json1_txt")
	tableObj.SetUnmarshalFn(xml.Unmarshal, "xml_txt")
	err := tableObj.SelectAuto(internal.Man{}).Where("id=1").FindOneFn(&m)
	if err != nil {
		t.Fatal(err)
	}

	jsonTxt := internal.Tmp{
		Name: "json",
		Data: "test json marshal",
	}
	xmlTxt := internal.Tmp{
		Name: "xml",
		Data: "test xml marshal",
	}
	json1Txt := internal.Tmp{
		Name: "json1",
		Data: "test json1 marshal",
	}
	t.Logf("%+v", m)
	if !internal.Equal(m.Name, sureName) || !internal.Equal(m.Age, sureAge) || !internal.StructValEqual(m.JsonTxt, jsonTxt) || !internal.StructValEqual(m.XmlTxt, xmlTxt) || !internal.StructValEqual(m.Json1Txt, json1Txt) {
		t.Error(internal.NoEqErr)
	}
}

func TestFindAllForPg(t *testing.T) {
	var m []internal.Man
	var err error
	tableObj := spellsql.NewTable(pgDb).Tmer(spellsql.Pg("man"))
	tableObj.SetUnmarshalFn(json.Unmarshal, "json_txt", "json1_txt")
	tableObj.SetUnmarshalFn(xml.Unmarshal, "xml_txt")
	err = tableObj.SelectAuto(internal.Man{}).FindWhere(&m, "id>0")
	if err != nil {
		t.Fatal(err)
	}
	if len(m) == 0 {
		t.Error("res is null")
		return
	}

	jsonTxt := internal.Tmp{
		Name: "json",
		Data: "test json marshal",
	}
	xmlTxt := internal.Tmp{
		Name: "xml",
		Data: "test xml marshal",
	}
	json1Txt := internal.Tmp{
		Name: "json1",
		Data: "test json1 marshal",
	}
	t.Logf("%+v", m)
	first := m[0]
	if !internal.Equal(first.Name, sureName) || !internal.Equal(first.Age, sureAge) || !internal.StructValEqual(first.JsonTxt, jsonTxt) || !internal.StructValEqual(first.XmlTxt, xmlTxt) || !internal.StructValEqual(first.Json1Txt, json1Txt) {
		t.Error(internal.NoEqErr)
	}
}
