package spellsql

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	gmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	sureName = "测试1"
	sureAge  = int32(20)
)

// CREATE TABLE `man` (
// 	`id` int NOT NULL AUTO_INCREMENT,
// 	`name` varchar(10) NOT NULL,
// 	`age` int NOT NULL,
// 	`addr` varchar(50) DEFAULT NULL,
// 	`hobby` varchar(255) DEFAULT '',
// 	`json_txt` text CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
// 	`nickname` varchar(30) DEFAULT '',
// 	`xml_txt` text,
// 	`json1_txt` varchar(255),
// 	PRIMARY KEY (`id`)
//   )

type Man struct {
	Id       int32  `json:"id,omitempty" gorm:"id" db:"id"`
	Name     string `json:"name,omitempty" gorm:"name" db:"name"`
	Age      int32  `json:"age,omitempty" gorm:"age" db:"age"`
	Addr     string `json:"addr,omitempty" gorm:"addr" db:"addr"`
	NickName string `json:"nickname,omitempty" gorm:"nickname" db:"nickname"`
	JsonTxt  Tmp    `json:"json_txt,omitempty"`
	XmlTxt   Tmp    `json:"xml_txt,omitempty"`
	Json1Txt Tmp    `json:"json1_txt,omitempty"`
}

type ManCopy struct {
	Id       int32  `json:"id,omitempty" gorm:"id" db:"id"`
	Name     string `json:"name,omitempty" gorm:"name" db:"name"`
	Age      int32  `json:"age,omitempty" gorm:"age" db:"age"`
	Addr     string `json:"addr,omitempty" gorm:"addr" db:"addr"`
	NickName string `json:"nickname,omitempty" gorm:"nickname" db:"nickname"`
}

type Student struct {
	Id        int32  `json:"id,omitempty" gorm:"id" db:"id"`
	UId       int32  `json:"u_id,omitempty" gorm:"u_id" db:"u_id"`
	ClassName string `json:"class_name,omitempty" gorm:"class_name" db:"class_name"`
	Nickname  string `json:"nickname,omitempty" gorm:"nickname" db:"nickname"`
	Name      string `json:"name,omitempty" gorm:"name" db:"name"`
}

type Tmp struct {
	Name string `json:"name,omitempty" xml:"name"`
	Data string `json:"data,omitempty" xml:"data"`
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
		Name:     "测试",
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
	// DROP TABLE IF EXISTS test_col;
	// CREATE TABLE `test_col` (
	// 	`id` int NOT NULL AUTO_INCREMENT,
	// 	`id1` varchar(10) NOT NULL,
	// 	`l_tinyint` tinyint DEFAULT NULL,
	// 	`l_int` int DEFAULT NULL,
	// 	`l_long` mediumtext,
	// 	`l_float` float DEFAULT NULL,
	// 	`l_dec` decimal(10, 0) DEFAULT NULL,
	// 	`l_char` char(10) DEFAULT NULL,
	// 	`l_varchar` varchar(10) DEFAULT NULL,
	// 	`l_text` longtext,
	// 	`l_tint` tinyint unsigned NOT NULL,
	// 	`l_bint` bigint unsigned NOT NULL,
	// 	`l_tfloat` decimal(65, 0) unsigned NOT NULL,
	// 	PRIMARY KEY (`id`, `id1`),
	// 	KEY `a` (`l_int`)
	//   )

	type TestColInfo struct {
		Id       int32  `json:"id,omitempty"`
		Id1      string `json:"id1,omitempty"`
		LTinyint int8   `json:"l_tinyint,omitempty"`
		LInt     int32  `json:"l_int,omitempty"`
		LLong    string `json:"l_long,omitempty"`
		LFloat   string `json:"l_float,omitempty"`
		LDec     string `json:"l_dec,omitempty"`
		LChar    string `json:"l_char,omitempty"`
		LVarchar string `json:"l_varchar,omitempty"`
		LText    string `json:"l_text,omitempty"`
		LTint    int8   `json:"l_tint,omitempty"`
		LBint    int64  `json:"l_bint,omitempty"`
		LTfloat  string `json:"l_tfloat,omitempty"`
	}
	var data TestColInfo
	err := FindWhere(db, "test_col", &data, "id=1")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", data)

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

	t.Run("insert for obj", func(t *testing.T) {
		r, err := InsertForObj(db, "man", m)
		if err != nil {
			t.Fatal(err)
		}
		if rr, _ := r.RowsAffected(); rr == 0 {
			t.Error("inset failed")
		}
	})

	t.Run("insert for sql", func(t *testing.T) {
		sqlObj := NewCacheSql("INSERT INTO man (name,age,addr) VALUES (?, ?, ?)", m.Name, m.Age, m.Addr)
		r, err := ExecForSql(db, sqlObj)
		if err != nil {
			t.Fatal(err)
		}

		if rr, _ := r.RowsAffected(); rr == 0 {
			t.Error("inset failed")
		}
	})

	t.Run("insert marshal", func(t *testing.T) {
		tableObj := NewTable(db, "man")
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

	t.Run("insert exclude", func(t *testing.T) {
		tableObj := NewTable(db, "man").Exclude("addr")
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

	t.Run("insert tag alias", func(t *testing.T) {
		type Tmp struct {
			Id    int32  `json:"id,omitempty"`
			Name1 string `json:"name,omitempty"`
			Age1  int32  `json:"age_1,omitempty"`
			Addr1 string `json:"addr_1,omitempty"`
		}
		m := Tmp{
			Name1: "xue1234",
			Age1:  18,
			Addr1: "成都市",
		}
		tableObj := NewTable(db, "man").TagAlias(map[string]string{"age_1": "age", "addr_1": "addr"})
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

	t.Run("update marshal", func(t *testing.T) {
		tableObj := NewTable(db, "man")
		tableObj.SetMarshalFn(json.Marshal, "json_txt", "json1_txt")
		_, err := tableObj.Update(m, "id=?", 7).Exec()
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("update alias", func(t *testing.T) {
		type Tmp struct {
			Id    int32  `json:"id,omitempty"`
			Name1 string `json:"name,omitempty"`
			Age1  int32  `json:"age_1,omitempty"`
			Addr1 string `json:"addr_1,omitempty"`
		}
		m := Tmp{
			Name1: "xue1234",
			Age1:  18,
			Addr1: "成都市",
		}
		tableObj := NewTable(db, "man").TagAlias(map[string]string{"age_1": "age", "addr_1": "addr"})
		_, err := tableObj.Update(m, "id=?", 7).Exec()
		if err != nil {
			t.Fatal(err)
		}
	})
}

// find 单元测试: go test -run ^TestFind

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

	t.Run("findOne unmarshal", func(t *testing.T) {
		var m Man
		tableObj := NewTable(db)
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
		xmlTxt := Tmp{
			Name: "xml",
			Data: "test xml marshal",
		}
		json1Txt := Tmp{
			Name: "json1",
			Data: "test json1 marshal",
		}
		t.Logf("%+v", m)
		if !equal(m.Name, sureName) || !equal(m.Age, sureAge) || !structValEqual(m.JsonTxt, jsonTxt) || !structValEqual(m.XmlTxt, xmlTxt) || !structValEqual(m.Json1Txt, json1Txt) {
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

	t.Run("findOneIgnoreRes", func(t *testing.T) {
		var (
			id2InfoMap = make(map[int32]Man)
			tmp        = "被修改了哦"
			m          Man
		)
		err := NewTable(db).SelectAuto(m).Where("id>0").Limit(0, 10).FindOneIgnoreResult(&m, func(_row interface{}) error {
			v := _row.(*Man)
			if v.Id == 1 {
				v.Name = tmp
			}
			id2InfoMap[v.Id] = *v
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		// for id, info := range id2InfoMap {
		// 	t.Logf("%d>%+v", id, info)
		// }
		id1Info := id2InfoMap[1]
		if !equal(id1Info.Name, tmp) || !equal(id1Info.Age, sureAge) {
			t.Error(noEqErr)
		}
	})

	t.Run("findOne alias", func(t *testing.T) {
		type Tmp struct {
			Name1 string `json:"name_1,omitempty"`
			Age1  int32  `json:"age_1,omitempty"`
		}
		var m Tmp
		err := NewTable(db).
			TagAlias(map[string]string{"name_1": "name", "age_1": "age"}).
			Select("name,age").
			From("man").
			Where("id=?", 1).
			FindOne(&m)
		if err != nil {
			t.Fatal(err)
		}
		if !equal(m.Name1, sureName) || !equal(m.Age1, sureAge) {
			t.Error(noEqErr)
		}
	})

	t.Log("find one test end")
}

func TestTmp(t *testing.T) {
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

	t.Run("findWhere unmarshal", func(t *testing.T) {
		var m Man
		tableObj := NewTable(db)
		tableObj.SetUnmarshalFn(json.Unmarshal, "json_txt", "json1_txt")
		tableObj.SetUnmarshalFn(xml.Unmarshal, "xml_txt")
		err := tableObj.SelectAuto(Man{}).FindWhere(&m, "id=?", 1)
		if err != nil {
			t.Fatal(err)
		}

		jsonTxt := Tmp{
			Name: "json",
			Data: "test json marshal",
		}
		xmlTxt := Tmp{
			Name: "xml",
			Data: "test xml marshal",
		}
		json1Txt := Tmp{
			Name: "json1",
			Data: "test json1 marshal",
		}
		// t.Logf("%+v", m)
		if !equal(m.Name, sureName) || !equal(m.Age, sureAge) || !structValEqual(m.JsonTxt, jsonTxt) || !structValEqual(m.XmlTxt, xmlTxt) || !structValEqual(m.Json1Txt, json1Txt) {
			t.Error(noEqErr)
		}
	})

	t.Run("findWhere alias", func(t *testing.T) {
		type Tmp struct {
			Name1 string `json:"name_1,omitempty"`
			Age1  int32  `json:"age_1,omitempty"`
		}
		var m Tmp
		err := NewTable(db).
			TagAlias(map[string]string{"name_1": "name", "age_1": "age"}).
			Select("name,age").
			From("man").
			FindWhere(&m, "id=?", 1)
		if err != nil {
			t.Fatal(err)
		}
		if !equal(m.Name1, sureName) || !equal(m.Age1, sureAge) {
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

// FindOne 性能对比, 以下是在 mac11 pro m1 上测试
// go test -benchmem -run=^$ -bench ^BenchmarkFindOne gitee.com/xuesongtao/spellsql -v -count=5

func BenchmarkFindOneGorm(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var m ManCopy
		gdb.Table("man").Find(&m, "id=?", 1)
	}

	// BenchmarkFindOneGorm-8                     16958             66604 ns/op            4364 B/op         78 allocs/op
	// BenchmarkFindOneGorm-8                     18019             66307 ns/op            4365 B/op         78 allocs/op
	// BenchmarkFindOneGorm-8                     17989             66318 ns/op            4365 B/op         78 allocs/op
	// BenchmarkFindOneGorm-8                     18040             66146 ns/op            4365 B/op         78 allocs/op
	// BenchmarkFindOneGorm-8                     18103             66284 ns/op            4365 B/op         78 allocs/op
}

func BenchmarkFindOneOrmQueryRowScan(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var m Man
		_ = NewTable(db, "man").IsPrintSql(false).Select("name,age,addr").Where("id=?", 1).QueryRowScan(&m.Id, &m.Age, &m.Addr)
	}

	// BenchmarkFindOneOrmQueryRowScan-8          30986             38634 ns/op            1645 B/op         38 allocs/op
	// BenchmarkFindOneOrmQueryRowScan-8          30747             38706 ns/op            1645 B/op         38 allocs/op
	// BenchmarkFindOneOrmQueryRowScan-8          30957             38568 ns/op            1645 B/op         38 allocs/op
	// BenchmarkFindOneOrmQueryRowScan-8          30950             38644 ns/op            1645 B/op         38 allocs/op
	// BenchmarkFindOneOrmQueryRowScan-8          30837             38797 ns/op            1645 B/op         38 allocs/op
}

func BenchmarkFindOneQueryRowScan(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var m Man
		sqlStr := FmtSqlStr("SELECT name,age,addr FROM man WHERE id=?", 1)
		_ = db.QueryRow(sqlStr).Scan(&m.Id, &m.Age, &m.Addr)
	}

	// BenchmarkFindOneQueryRowScan-8             32281             37144 ns/op            1187 B/op         29 allocs/op
	// BenchmarkFindOneQueryRowScan-8             32155             37214 ns/op            1187 B/op         29 allocs/op
	// BenchmarkFindOneQueryRowScan-8             32061             37085 ns/op            1187 B/op         29 allocs/op
	// BenchmarkFindOneQueryRowScan-8             32131             37132 ns/op            1187 B/op         29 allocs/op
	// BenchmarkFindOneQueryRowScan-8             32160             37183 ns/op            1187 B/op         29 allocs/op
}

func BenchmarkFindOneOrm(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var m Man
		_ = NewTable(db, "man").IsPrintSql(false).Select("name,age,addr").Where("id=?", 1).FindOne(&m)
	}

	// BenchmarkFindOneOrm-8                      31676             38230 ns/op            1329 B/op         32 allocs/op
	// BenchmarkFindOneOrm-8                      31736             37573 ns/op            1329 B/op         32 allocs/op
	// BenchmarkFindOneOrm-8                      31701             37737 ns/op            1329 B/op         32 allocs/op
	// BenchmarkFindOneOrm-8                      31719             37599 ns/op            1329 B/op         32 allocs/op
	// BenchmarkFindOneOrm-8                      31854             37576 ns/op            1329 B/op         32 allocs/op
}

func BenchmarkFindOneOrmForRaw(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var m Man
		_ = NewTable(db).IsPrintSql(false).Raw(NewCacheSql("SELECT name,age,addr FROM man WHERE id=?", 1)).FindOne(&m)
	}

	// BenchmarkFindOneOrmForRaw-8                31778             37649 ns/op            1337 B/op         33 allocs/op
	// BenchmarkFindOneOrmForRaw-8                31771             37633 ns/op            1337 B/op         33 allocs/op
	// BenchmarkFindOneOrmForRaw-8                31602             37587 ns/op            1337 B/op         33 allocs/op
	// BenchmarkFindOneOrmForRaw-8                31701             38329 ns/op            1337 B/op         33 allocs/op
	// BenchmarkFindOneOrmForRaw-8                31494             37600 ns/op            1337 B/op         33 allocs/op
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
				Name: name,
				Age:  int32(age),
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

	t.Run("findWhere unmarshal", func(t *testing.T) {
		var m []Man
		var err error
		tableObj := NewTable(db)
		tableObj.SetUnmarshalFn(json.Unmarshal, "json_txt", "json1_txt")
		tableObj.SetUnmarshalFn(xml.Unmarshal, "xml_txt")
		err = tableObj.SelectAuto(Man{}).FindWhere(&m, "id>0")
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
		xmlTxt := Tmp{
			Name: "xml",
			Data: "test xml marshal",
		}
		json1Txt := Tmp{
			Name: "json1",
			Data: "test json1 marshal",
		}
		// t.Logf("%+v", m)
		first := m[0]
		if !equal(first.Name, sureName) || !equal(first.Age, sureAge) || !structValEqual(first.JsonTxt, jsonTxt) || !structValEqual(first.XmlTxt, xmlTxt) || !structValEqual(first.Json1Txt, json1Txt) {
			t.Error(noEqErr)
		}
	})

	t.Run("findAll alias", func(t *testing.T) {
		type Tmp struct {
			Name1 string `json:"name_1,omitempty"`
			Age1  int32  `json:"age_1,omitempty"`
		}
		var m []Tmp
		err := NewTable(db).
			TagAlias(map[string]string{"name_1": "name", "age_1": "age"}).
			Select("name,age").
			From("man").
			Limit(0, 10).
			FindAll(&m)
		if err != nil {
			t.Fatal(err)
		}
		if !equal(m[0].Name1, sureName) || !equal(m[0].Age1, sureAge) {
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

// 以下是在 mac11 pro m1 上测试
// go test -benchmem -run=^$ -bench ^BenchmarkFindAll gitee.com/xuesongtao/spellsql -v -count=5

func BenchmarkFindAllGorm(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var m []*ManCopy
		// sqlStr := FmtSqlStr("SELECT * FROM man WHERE id>?", 1)
		gdb.Table("man").Limit(10).Find(&m, "id>?", 1)
		// b.Log(m)
	}

	// BenchmarkFindAllGorm-8             11581             92114 ns/op            8962 B/op        273 allocs/op
	// BenchmarkFindAllGorm-8             12896             91718 ns/op            8962 B/op        273 allocs/op
	// BenchmarkFindAllGorm-8             12811             91760 ns/op            8961 B/op        273 allocs/op
	// BenchmarkFindAllGorm-8             13072             91517 ns/op            8962 B/op        273 allocs/op
	// BenchmarkFindAllGorm-8             13081             91715 ns/op            8962 B/op        273 allocs/op
}

func BenchmarkFindAllSqlx(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var m []*ManCopy
		sqlStr := FmtSqlStr("SELECT name,age,addr FROM man WHERE id>? LIMIT ?, ?", 1, 0, 10)
		sqlxdb.Select(&m, sqlStr)
		// b.Log(m)
	}

	// 说明: sqlx 不能自动处理 null 这里的查询结果不全
	// BenchmarkFindAllSqlx-8             23478             51939 ns/op            2057 B/op         64 allocs/op
	// BenchmarkFindAllSqlx-8             22462             51812 ns/op            2057 B/op         64 allocs/op
	// BenchmarkFindAllSqlx-8             23031             51906 ns/op            2057 B/op         64 allocs/op
	// BenchmarkFindAllSqlx-8             23037             51854 ns/op            2057 B/op         64 allocs/op
	// BenchmarkFindAllSqlx-8             23042             51755 ns/op            2057 B/op         64 allocs/op
}

func BenchmarkFindAllQuery(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sqlStr := FmtSqlStr("SELECT name,age,addr FROM man WHERE id>? LIMIT ?, ?", 1, 0, 10)
		rows, err := db.Query(sqlStr)
		if err != nil {
			return
		}

		var res []*ManCopy
		for rows.Next() {
			var info ManCopy
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

	// BenchmarkFindAllQuery-8            23402             51492 ns/op            2769 B/op         99 allocs/op
	// BenchmarkFindAllQuery-8            23434             51465 ns/op            2769 B/op         99 allocs/op
	// BenchmarkFindAllQuery-8            22744             52913 ns/op            2769 B/op         99 allocs/op
	// BenchmarkFindAllQuery-8            23089             51675 ns/op            2769 B/op         99 allocs/op
	// BenchmarkFindAllQuery-8            23162             51550 ns/op            2768 B/op         99 allocs/op
}

func BenchmarkFindAllOrm(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var m []*ManCopy
		_ = NewTable(db).IsPrintSql(false).Select("name,age,addr").From("man").Where("id>?", 1).Limit(0, 10).FindAll(&m)
	}

	// BenchmarkFindAllOrm-8              21327             57296 ns/op            3235 B/op        115 allocs/op
	// BenchmarkFindAllOrm-8              21588             55743 ns/op            3235 B/op        115 allocs/op
	// BenchmarkFindAllOrm-8              21538             55733 ns/op            3235 B/op        115 allocs/op
	// BenchmarkFindAllOrm-8              21524             56888 ns/op            3235 B/op        115 allocs/op
	// BenchmarkFindAllOrm-8              21366             55844 ns/op            3235 B/op        115 allocs/op
}

func BenchmarkFindAllOrmForRawHaveTableName(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var m []*ManCopy
		_ = NewTable(db, "man").IsPrintSql(false).Raw(NewCacheSql("SELECT name,age,addr FROM man WHERE id>? LIMIT ?, ?", 1, 0, 10)).FindAll(&m)
	}

	// BenchmarkFindAllOrmForRawHaveTableName-8           21619             55702 ns/op            3259 B/op        113 allocs/op
	// BenchmarkFindAllOrmForRawHaveTableName-8           21435             55492 ns/op            3259 B/op        113 allocs/op
	// BenchmarkFindAllOrmForRawHaveTableName-8           21566             55808 ns/op            3259 B/op        113 allocs/op
	// BenchmarkFindAllOrmForRawHaveTableName-8           21469             55339 ns/op            3259 B/op        113 allocs/op
	// BenchmarkFindAllOrmForRawHaveTableName-8           21606             56016 ns/op            3259 B/op        113 allocs/op
}

func BenchmarkFindAllOrmForRawHaveNoTableName(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var m []*ManCopy
		_ = NewTable(db).IsPrintSql(false).Raw(NewCacheSql("SELECT name,age,addr FROM man WHERE id>? LIMIT ?, ?", 1, 0, 10)).FindAll(&m)
	}

	// BenchmarkFindAllOrmForRawHaveNoTableName-8         20707             57362 ns/op            3580 B/op        133 allocs/op
	// BenchmarkFindAllOrmForRawHaveNoTableName-8         20935             57278 ns/op            3580 B/op        133 allocs/op
	// BenchmarkFindAllOrmForRawHaveNoTableName-8         20716             57452 ns/op            3580 B/op        133 allocs/op
	// BenchmarkFindAllOrmForRawHaveNoTableName-8         20769             61357 ns/op            3580 B/op        133 allocs/op
	// BenchmarkFindAllOrmForRawHaveNoTableName-8         20109             58092 ns/op            3580 B/op        133 allocs/op
}
