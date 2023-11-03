package spellsql

import (
	"encoding/json"
	"reflect"
	"testing"

	// "github.com/jinzhu/copier"
)

type TmpDest struct {
	Name          string     `json:"name,omitempty"`
	Age           int        `json:"age,omitempty"`
	Hobby         []string   `json:"hobby,omitempty"`
	NeedMarshal   string     `json:"need_marshal,omitempty"`
	NeedUnmarshal []*TmpNest `json:"need_unmarshal,omitempty"`
	Copy          []*TmpNest `json:"copy,omitempty"`
}

type TmpSrc struct {
	Name          string     `json:"name,omitempty"`
	Age           int64      `json:"age,omitempty"`
	Hobby         []string   `json:"hobby,omitempty"`
	Test          string     `json:"test,omitempty"`
	NeedMarshal   []*TmpNest `json:"need_marshal,omitempty"`
	NeedUnmarshal string     `json:"need_unmarshal,omitempty"`
	Copy          []*TmpNest `json:"copy,omitempty"`
}

type TmpNest struct {
	Name string `json:"name,omitempty"`
}

func TestConvert(t *testing.T) {
	testCases := []struct {
		desc string
		src  TmpSrc
		dest TmpDest
		ok   TmpDest
	}{
		{
			desc: "单字段",
			src: TmpSrc{
				Name: "name",
				Age:  10,
				Test: "test",
			},
			dest: TmpDest{},
			ok: TmpDest{
				Name: "name",
				Age:  10,
			},
		},
		{
			desc: "有切片字段",
			src: TmpSrc{
				Name:  "name",
				Age:   10,
				Hobby: []string{"打篮球", "跑步"},
				Test:  "test",
			},
			dest: TmpDest{},
			ok: TmpDest{
				Name:  "name",
				Age:   10,
				Hobby: []string{"打篮球", "跑步"},
			},
		},
		{
			desc: "需要marshal",
			src: TmpSrc{
				Name:        "name",
				Age:         10,
				Hobby:       []string{"打篮球", "跑步"},
				NeedMarshal: []*TmpNest{{Name: "需要 marshal 测试"}},
			},
			dest: TmpDest{},
			ok: TmpDest{
				Name:        "name",
				Age:         10,
				Hobby:       []string{"打篮球", "跑步"},
				NeedMarshal: "[{\"name\":\"需要 marshal 测试\"}]",
			},
		},
		{
			desc: "需要unmarshal",
			src: TmpSrc{
				Name:          "name",
				Age:           10,
				Hobby:         []string{"打篮球", "跑步"},
				NeedUnmarshal: "[{\"name\":\"需要 unmarshal 测试\"}]",
			},
			dest: TmpDest{},
			ok: TmpDest{
				Name:          "name",
				Age:           10,
				Hobby:         []string{"打篮球", "跑步"},
				NeedUnmarshal: []*TmpNest{{Name: "需要 unmarshal 测试"}},
			},
		},
		{
			desc: "copy-demo",
			src: TmpSrc{
				Name:  "name",
				Age:   10,
				Hobby: []string{"打篮球", "跑步"},
				Copy:  []*TmpNest{{Name: "copy"}},
			},
			dest: TmpDest{},
			ok: TmpDest{
				Name:  "name",
				Age:   10,
				Hobby: []string{"打篮球", "跑步"},
				Copy:  []*TmpNest{{Name: "copy"}},
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			obj := NewConvStruct()

			obj.Init(&tC.src, &tC.dest)
			if tC.desc == "需要marshal" {
				obj.SrcMarshal(json.Marshal, "need_marshal")
			}
			if tC.desc == "需要unmarshal" {
				obj.SrcUnmarshal(json.Unmarshal, "need_unmarshal")
			}

			err := obj.Convert()
			if err != nil {
				t.Fatal(err)
			}

			if tC.desc == "copy-demo" {
				tC.src.Age = 0
				tC.src.Copy[0].Name = "修改"
				for _, v := range tC.dest.Copy {
					t.Log(v)
				}
			}
			t.Logf("dst: %+v", tC.dest)
			if !reflect.DeepEqual(tC.dest, tC.ok) {
				t.Errorf("convert is failed, dest: %+v, ok: %+v", tC.dest, tC.ok)
			}
		})
	}
}

// func TestCopyerConvert(t *testing.T) {
// 	testCases := []struct {
// 		desc string
// 		src  TmpSrc
// 		dest TmpDest
// 		ok   TmpDest
// 	}{
// 		{
// 			desc: "单字段",
// 			src: TmpSrc{
// 				Name: "name",
// 				Age:  10,
// 				Test: "test",
// 			},
// 			dest: TmpDest{},
// 			ok: TmpDest{
// 				Name: "name",
// 				Age:  10,
// 			},
// 		},
// 		{
// 			desc: "有切片字段",
// 			src: TmpSrc{
// 				Name:  "name",
// 				Age:   10,
// 				Hobby: []string{"打篮球", "跑步"},
// 				Test:  "test",
// 			},
// 			dest: TmpDest{},
// 			ok: TmpDest{
// 				Name:  "name",
// 				Age:   10,
// 				Hobby: []string{"打篮球", "跑步"},
// 			},
// 		},
// 		{
// 			desc: "需要marshal",
// 			src: TmpSrc{
// 				Name:        "name",
// 				Age:         10,
// 				Hobby:       []string{"打篮球", "跑步"},
// 				NeedMarshal: []*TmpNest{{Name: "需要 marshal 测试"}},
// 			},
// 			dest: TmpDest{},
// 			ok: TmpDest{
// 				Name:        "name",
// 				Age:         10,
// 				Hobby:       []string{"打篮球", "跑步"},
// 				NeedMarshal: "[{\"name\":\"需要 marshal 测试\"}]",
// 			},
// 		},
// 		{
// 			desc: "需要unmarshal",
// 			src: TmpSrc{
// 				Name:          "name",
// 				Age:           10,
// 				Hobby:         []string{"打篮球", "跑步"},
// 				NeedUnmarshal: "[{\"name\":\"需要 unmarshal 测试\"}]",
// 			},
// 			dest: TmpDest{},
// 			ok: TmpDest{
// 				Name:          "name",
// 				Age:           10,
// 				Hobby:         []string{"打篮球", "跑步"},
// 				NeedUnmarshal: []*TmpNest{{Name: "需要 unmarshal 测试"}},
// 			},
// 		},
// 		{
// 			desc: "copy-demo",
// 			src: TmpSrc{
// 				Name:  "name",
// 				Age:   10,
// 				Hobby: []string{"打篮球", "跑步"},
// 				Copy:  []*TmpNest{{Name: "copy"}},
// 			},
// 			dest: TmpDest{},
// 			ok: TmpDest{
// 				Name:  "name",
// 				Age:   10,
// 				Hobby: []string{"打篮球", "跑步"},
// 				Copy:  []*TmpNest{{Name: "copy"}},
// 			},
// 		},
// 	}

// 	for _, tC := range testCases {
// 		t.Run(tC.desc, func(t *testing.T) {
// 			err := copier.Copy(&tC.dest, &tC.src)
// 			if err != nil {
// 				t.Error("err:", err)
// 			}
// 			t.Logf("dst: %+v", tC.dest)
// 			if !reflect.DeepEqual(tC.dest, tC.ok) {
// 				t.Errorf("convert is failed, dest: %+v, ok: %+v", tC.dest, tC.ok)
// 			}
// 		})
// 	}
// }

func TestJsonConvert(t *testing.T) {
	testCases := []struct {
		desc string
		src  TmpSrc
		dest TmpDest
		ok   TmpDest
	}{
		{
			desc: "单字段",
			src: TmpSrc{
				Name: "name",
				Age:  10,
				Test: "test",
			},
			dest: TmpDest{},
			ok: TmpDest{
				Name: "name",
				Age:  10,
			},
		},
		{
			desc: "有切片字段",
			src: TmpSrc{
				Name:  "name",
				Age:   10,
				Hobby: []string{"打篮球", "跑步"},
				Test:  "test",
			},
			dest: TmpDest{},
			ok: TmpDest{
				Name:  "name",
				Age:   10,
				Hobby: []string{"打篮球", "跑步"},
			},
		},
		{
			desc: "需要marshal",
			src: TmpSrc{
				Name:        "name",
				Age:         10,
				Hobby:       []string{"打篮球", "跑步"},
				NeedMarshal: []*TmpNest{{Name: "需要 marshal 测试"}},
			},
			dest: TmpDest{},
			ok: TmpDest{
				Name:        "name",
				Age:         10,
				Hobby:       []string{"打篮球", "跑步"},
				NeedMarshal: "[{\"name\":\"需要 marshal 测试\"}]",
			},
		},
		{
			desc: "需要unmarshal",
			src: TmpSrc{
				Name:          "name",
				Age:           10,
				Hobby:         []string{"打篮球", "跑步"},
				NeedUnmarshal: "[{\"name\":\"需要 unmarshal 测试\"}]",
			},
			dest: TmpDest{},
			ok: TmpDest{
				Name:          "name",
				Age:           10,
				Hobby:         []string{"打篮球", "跑步"},
				NeedUnmarshal: []*TmpNest{{Name: "需要 unmarshal 测试"}},
			},
		},
		{
			desc: "copy-demo",
			src: TmpSrc{
				Name:  "name",
				Age:   10,
				Hobby: []string{"打篮球", "跑步"},
				Copy:  []*TmpNest{{Name: "copy"}},
			},
			dest: TmpDest{},
			ok: TmpDest{
				Name:  "name",
				Age:   10,
				Hobby: []string{"打篮球", "跑步"},
				Copy:  []*TmpNest{{Name: "copy"}},
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			b, _ := json.Marshal(&tC.src)
			if err := json.Unmarshal(b, &tC.dest); err != nil {
				t.Error("json unmarshal err:", err)
			}
			t.Logf("dst: %+v", tC.dest)
			if !reflect.DeepEqual(tC.dest, tC.ok) {
				t.Errorf("convert is failed, dest: %+v, ok: %+v", tC.dest, tC.ok)
			}
		})
	}
}

func BenchmarkConvert(b *testing.B) {
	src := TmpSrc{
		Name:        "name",
		Age:         10,
		Hobby:       []string{"打篮球", "跑步"},
		NeedMarshal: []*TmpNest{{Name: "需要 marshal 测试"}},
	}
	dest := TmpDest{}
	for i := 0; i < b.N; i++ {
		obj := NewConvStruct()
		obj.Init(&src, &dest)
		_ = obj.Convert()
	}
}

func BenchmarkJsonConvert(b *testing.B) {
	src := TmpSrc{
		Name:        "name",
		Age:         10,
		Hobby:       []string{"打篮球", "跑步"},
		NeedMarshal: []*TmpNest{{Name: "需要 marshal 测试"}},
	}
	dest := TmpDest{}
	for i := 0; i < b.N; i++ {
		b, _ := json.Marshal(&src)
		json.Unmarshal(b, &dest)
	}
}
// func BenchmarkCopyerConvert(b *testing.B) {
// 	src := TmpSrc{
// 		Name:        "name",
// 		Age:         10,
// 		Hobby:       []string{"打篮球", "跑步"},
// 		NeedMarshal: []*TmpNest{{Name: "需要 marshal 测试"}},
// 	}
// 	dest := TmpDest{}
// 	for i := 0; i < b.N; i++ {
// 		copier.Copy(&dest, &src)
// 	}
// }
