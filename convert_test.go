package spellsql

import (
	"encoding/json"
	"reflect"
	"testing"
)

type TmpDest struct {
	Name          string     `json:"name,omitempty"`
	Age           int        `json:"age,omitempty"`
	Hobby         []string   `json:"hobby,omitempty"`
	NeedMarshal   string     `json:"need_marshal,omitempty"`
	NeedUnmarshal []*TmpNest `json:"need_unmarshal,omitempty"`
}

type TmpSrc struct {
	Name          string     `json:"name,omitempty"`
	Age           int64      `json:"age,omitempty"`
	Hobby         []string   `json:"hobby,omitempty"`
	Test          string     `json:"test,omitempty"`
	NeedMarshal   []*TmpNest `json:"need_marshal,omitempty"`
	NeedUnmarshal string     `json:"need_unmarshal,omitempty"`
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
			// tC.src.Name = "testname"
			if !reflect.DeepEqual(tC.dest, tC.ok) {
				t.Errorf("convert is failed, dest: %+v, ok: %+v", tC.dest, tC.ok)
			}
		})
	}
}
