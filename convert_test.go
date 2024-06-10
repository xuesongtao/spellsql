package spellsql

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/jinzhu/copier"
)

type TmpDest struct {
	Name          string     `json:"name,omitempty"`
	Age           int        `json:"age,omitempty"`
	Hobby         []string   `json:"hobby,omitempty"`
	CopyPtr       *TmpNest   `json:"copy_ptr,omitempty"`
	Copy          TmpNest    `json:"copy,omitempty"`
	NeedMarshal   string     `json:"need_marshal,omitempty"`
	NeedUnmarshal []*TmpNest `json:"need_unmarshal,omitempty"`
	CopySlicePtr  []*TmpNest `json:"copy_slice_ptr,omitempty"`
	CopySlice     []TmpNest  `json:"copy_slice,omitempty"`
}

type TmpSrc struct {
	Test          string     `json:"test,omitempty"`
	Name          string     `json:"name,omitempty"`
	Age           int64      `json:"age,omitempty"`
	Hobby         []string   `json:"hobby,omitempty"`
	CopyPtr       *TmpNest   `json:"copy_ptr,omitempty"`
	Copy          TmpNest    `json:"copy,omitempty"`
	NeedMarshal   []*TmpNest `json:"need_marshal,omitempty"`
	NeedUnmarshal string     `json:"need_unmarshal,omitempty"`
	CopySlicePtr  []*TmpNest `json:"copy_slice_ptr,omitempty"`
	CopySlice     []TmpNest  `json:"copy_slice,omitempty"`
}

type TmpNest struct {
	Name    string `json:"name,omitempty"`
	NextPtr *Tmp   `json:"next_ptr,omitempty"`
	Next    Tmp    `json:"next,omitempty"`
}

type Tmp struct {
	Name string `json:"name,omitempty"`
}

func TestConvertUnsafe(t *testing.T) {
	testCases := []struct {
		desc string
		src  TmpSrc
		dest TmpDest
		ok   TmpDest
	}{{
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
			Hobby: []string{"", "跑步"},
		},
	}, {
		desc: "嵌套",
		src: TmpSrc{
			Name:  "name",
			Age:   10,
			Hobby: []string{"打篮球", "跑步"},
			Test:  "test",
			CopyPtr: &TmpNest{
				Name: "test",
				NextPtr: &Tmp{
					Name: "test1",
				},
				Next: Tmp{
					Name: "test",
				},
			},
		},
		dest: TmpDest{},
		ok: TmpDest{
			Name:  "name",
			Age:   10,
			Hobby: []string{"打篮球", "跑步"},
			CopyPtr: &TmpNest{
				Name: "test",
				NextPtr: &Tmp{
					Name: "test1",
				},
				Next: Tmp{
					Name: "test",
				},
			},
		}},
	}

	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			obj := NewConvStruct()
			obj.Init(&tC.src, &tC.dest)

			err := obj.ConvertUnsafe()
			if err != nil {
				t.Error(err)
			}

			if tC.desc == "有切片字段" {
				tC.src.Hobby[0] = ""
			}
			CheckObj(t, tC.dest, tC.ok)
		})
	}
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
			desc: "单字段",
			src: TmpSrc{
				Name: "name",
				Age:  10,
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
				NeedMarshal: "[{\"name\":\"需要 marshal 测试\",\"next\":{}}]",
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
			desc: "嵌套值类型结构体",
			src: TmpSrc{
				Name:  "name",
				Age:   10,
				Hobby: []string{"打篮球", "跑步"},
				Copy:  TmpNest{Name: "值类型结构体"},
			},
			dest: TmpDest{},
			ok: TmpDest{
				Name:  "name",
				Age:   10,
				Hobby: []string{"打篮球", "跑步"},
				Copy:  TmpNest{Name: "值类型结构体"},
			},
		},
		{
			desc: "嵌套指针类型结构体",
			src: TmpSrc{
				Name:    "name",
				Age:     10,
				Hobby:   []string{"打篮球", "跑步"},
				Copy:    TmpNest{Name: "值类型结构体"},
				CopyPtr: &TmpNest{Name: "针类型结构体"},
			},
			dest: TmpDest{},
			ok: TmpDest{
				Name:    "name",
				Age:     10,
				Hobby:   []string{"打篮球", "跑步"},
				Copy:    TmpNest{Name: "值类型结构体"},
				CopyPtr: &TmpNest{Name: "针类型结构体"},
			},
		},
		{
			desc: "嵌套指针类型结构体",
			src: TmpSrc{
				Name:  "name",
				Age:   10,
				Hobby: []string{"打篮球", "跑步"},
				CopyPtr: &TmpNest{
					Name: "针类型结构体",
					NextPtr: &Tmp{
						Name: "NextPtr",
					},
					Next: Tmp{
						Name: "Next",
					},
				},
			},
			dest: TmpDest{},
			ok: TmpDest{
				Name:  "name",
				Age:   10,
				Hobby: []string{"打篮球", "跑步"},
				CopyPtr: &TmpNest{
					Name: "针类型结构体",
					NextPtr: &Tmp{
						Name: "NextPtr",
					},
					Next: Tmp{
						Name: "Next",
					},
				},
			},
		},
		{
			desc: "copy-slice-ptr-demo",
			src: TmpSrc{
				Name:         "name",
				Age:          10,
				Hobby:        []string{"打篮球", "跑步"},
				CopySlicePtr: []*TmpNest{{Name: "copy"}},
			},
			dest: TmpDest{},
			ok: TmpDest{
				Name:         "name",
				Age:          10,
				Hobby:        []string{"打篮球", "跑步"},
				CopySlicePtr: []*TmpNest{{Name: "copy"}},
			},
		},
		{
			desc: "copy-slice-demo",
			src: TmpSrc{
				Name:         "name",
				Age:          10,
				Hobby:        []string{"打篮球", "跑步"},
				CopySlicePtr: []*TmpNest{{Name: "copy-ptr"}},
				CopySlice:    []TmpNest{{Name: "copy"}},
			},
			dest: TmpDest{},
			ok: TmpDest{
				Name:         "name",
				Age:          10,
				Hobby:        []string{"打篮球", "跑步"},
				CopySlicePtr: []*TmpNest{{Name: "copy-ptr"}},
				CopySlice:    []TmpNest{{Name: "copy"}},
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
				t.Error(err)
			}

			// 修改 src 值
			if tC.desc == "单字段" {
				tC.src.Name = "test"
				tC.src.Age = 0
			}
			if tC.desc == "有切片字段" {
				tC.src.Hobby[0] = ""
			}
			if tC.desc == "copy-ptr-demo" {
				tC.src.Age = 0
				tC.src.CopySlicePtr[0].Name = "修改"
			}
			if tC.desc == "copy-demo" {
				tC.src.Name = "修改"
				tC.src.Age = 0
				tC.src.CopySlicePtr[0].Name = "修改"
				tC.src.CopySlice[0].Name = "修改"
			}
			CheckObj(t, tC.dest, tC.ok)
		})
	}
}

func CheckObj(t *testing.T, dest, src interface{}) {
	destBytes, _ := json.Marshal(dest)
	srcBytes, _ := json.Marshal(src)
	if bytes.Equal(destBytes, srcBytes) {
		return
	}
	t.Errorf("dest: %v, src: %v", string(destBytes), string(srcBytes))
}

func TestCopyerConvert(t *testing.T) {
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
		// { // 浅拷贝
		// 	desc: "有切片字段",
		// 	src: TmpSrc{
		// 		Name:  "name",
		// 		Age:   10,
		// 		Hobby: []string{"打篮球", "跑步"},
		// 		Test:  "test",
		// 	},
		// 	dest: TmpDest{},
		// 	ok: TmpDest{
		// 		Name:  "name",
		// 		Age:   10,
		// 		Hobby: []string{"打篮球", "跑步"},
		// 	},
		// },
		// { // 不支持
		// 	desc: "需要marshal",
		// 	src: TmpSrc{
		// 		Name:        "name",
		// 		Age:         10,
		// 		Hobby:       []string{"打篮球", "跑步"},
		// 		NeedMarshal: []*TmpNest{{Name: "需要 marshal 测试"}},
		// 	},
		// 	dest: TmpDest{},
		// 	ok: TmpDest{
		// 		Name:        "name",
		// 		Age:         10,
		// 		Hobby:       []string{"打篮球", "跑步"},
		// 		NeedMarshal: "[{\"name\":\"需要 marshal 测试\",\"next\":{}}]",
		// 	},
		// },
		// {
		// 	desc: "需要unmarshal",
		// 	src: TmpSrc{
		// 		Name:          "name",
		// 		Age:           10,
		// 		Hobby:         []string{"打篮球", "跑步"},
		// 		NeedUnmarshal: "[{\"name\":\"需要 unmarshal 测试\"}]",
		// 	},
		// 	dest: TmpDest{},
		// 	ok: TmpDest{
		// 		Name:          "name",
		// 		Age:           10,
		// 		Hobby:         []string{"打篮球", "跑步"},
		// 		NeedUnmarshal: []*TmpNest{{Name: "需要 unmarshal 测试"}},
		// 	},
		// },
		{
			desc: "嵌套值类型结构体",
			src: TmpSrc{
				Name:  "name",
				Age:   10,
				Hobby: []string{"打篮球", "跑步"},
				Copy:  TmpNest{Name: "值类型结构体"},
			},
			dest: TmpDest{},
			ok: TmpDest{
				Name:  "name",
				Age:   10,
				Hobby: []string{"打篮球", "跑步"},
				Copy:  TmpNest{Name: "值类型结构体"},
			},
		},
		{
			desc: "嵌套指针类型结构体",
			src: TmpSrc{
				Name:    "name",
				Age:     10,
				Hobby:   []string{"打篮球", "跑步"},
				Copy:    TmpNest{Name: "值类型结构体"},
				CopyPtr: &TmpNest{Name: "针类型结构体"},
			},
			dest: TmpDest{},
			ok: TmpDest{
				Name:    "name",
				Age:     10,
				Hobby:   []string{"打篮球", "跑步"},
				Copy:    TmpNest{Name: "值类型结构体"},
				CopyPtr: &TmpNest{Name: "针类型结构体"},
			},
		},
		{
			desc: "嵌套指针类型结构体",
			src: TmpSrc{
				Name:  "name",
				Age:   10,
				Hobby: []string{"打篮球", "跑步"},
				CopyPtr: &TmpNest{
					Name: "针类型结构体",
					NextPtr: &Tmp{
						Name: "NextPtr",
					},
					Next: Tmp{
						Name: "Next",
					},
				},
			},
			dest: TmpDest{},
			ok: TmpDest{
				Name:  "name",
				Age:   10,
				Hobby: []string{"打篮球", "跑步"},
				CopyPtr: &TmpNest{
					Name: "针类型结构体",
					NextPtr: &Tmp{
						Name: "NextPtr",
					},
					Next: Tmp{
						Name: "Next",
					},
				},
			},
		},
		{
			desc: "copy-slice-ptr-demo",
			src: TmpSrc{
				Name:         "name",
				Age:          10,
				Hobby:        []string{"打篮球", "跑步"},
				CopySlicePtr: []*TmpNest{{Name: "copy"}},
			},
			dest: TmpDest{},
			ok: TmpDest{
				Name:         "name",
				Age:          10,
				Hobby:        []string{"打篮球", "跑步"},
				CopySlicePtr: []*TmpNest{{Name: "copy"}},
			},
		},
		{
			desc: "copy-slice-demo",
			src: TmpSrc{
				Name:         "name",
				Age:          10,
				Hobby:        []string{"打篮球", "跑步"},
				CopySlicePtr: []*TmpNest{{Name: "copy-ptr"}},
				CopySlice:    []TmpNest{{Name: "copy"}},
			},
			dest: TmpDest{},
			ok: TmpDest{
				Name:         "name",
				Age:          10,
				Hobby:        []string{"打篮球", "跑步"},
				CopySlicePtr: []*TmpNest{{Name: "copy-ptr"}},
				CopySlice:    []TmpNest{{Name: "copy"}},
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {

			err := copier.Copy(&tC.dest, &tC.src) // 有浅拷贝情况
			if err != nil {
				t.Error(err)
			}
			// 修改 src 值
			if tC.desc == "单字段" {
				tC.src.Name = "test"
				tC.src.Age = 0
			}
			if tC.desc == "有切片字段" {
				tC.src.Hobby[0] = ""
			}
			if tC.desc == "copy-ptr-demo" {
				tC.src.Age = 0
				tC.src.CopySlicePtr[0].Name = "修改"
			}
			if tC.desc == "copy-demo" {
				tC.src.Name = "修改"
				tC.src.Age = 0
				tC.src.CopySlicePtr[0].Name = "修改"
				tC.src.CopySlice[0].Name = "修改"
			}
			CheckObj(t, tC.dest, tC.ok)
		})
	}
}

func TestJsonConvert(t *testing.T) {
	t.Skip()
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
				NeedMarshal: "[{\"name\":\"需要 marshal 测试\",\"next\":{}}]",
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
			desc: "嵌套值类型结构体",
			src: TmpSrc{
				Name:  "name",
				Age:   10,
				Hobby: []string{"打篮球", "跑步"},
				Copy:  TmpNest{Name: "值类型结构体"},
			},
			dest: TmpDest{},
			ok: TmpDest{
				Name:  "name",
				Age:   10,
				Hobby: []string{"打篮球", "跑步"},
				Copy:  TmpNest{Name: "值类型结构体"},
			},
		},
		{
			desc: "嵌套指针类型结构体",
			src: TmpSrc{
				Name:    "name",
				Age:     10,
				Hobby:   []string{"打篮球", "跑步"},
				Copy:    TmpNest{Name: "值类型结构体"},
				CopyPtr: &TmpNest{Name: "针类型结构体"},
			},
			dest: TmpDest{},
			ok: TmpDest{
				Name:    "name",
				Age:     10,
				Hobby:   []string{"打篮球", "跑步"},
				Copy:    TmpNest{Name: "值类型结构体"},
				CopyPtr: &TmpNest{Name: "针类型结构体"},
			},
		},
		{
			desc: "嵌套指针类型结构体",
			src: TmpSrc{
				Name:  "name",
				Age:   10,
				Hobby: []string{"打篮球", "跑步"},
				CopyPtr: &TmpNest{
					Name: "针类型结构体",
					NextPtr: &Tmp{
						Name: "NextPtr",
					},
					Next: Tmp{
						Name: "Next",
					},
				},
			},
			dest: TmpDest{},
			ok: TmpDest{
				Name:  "name",
				Age:   10,
				Hobby: []string{"打篮球", "跑步"},
				CopyPtr: &TmpNest{
					Name: "针类型结构体",
					NextPtr: &Tmp{
						Name: "NextPtr",
					},
					Next: Tmp{
						Name: "Next",
					},
				},
			},
		},
		{
			desc: "copy-slice-ptr-demo",
			src: TmpSrc{
				Name:         "name",
				Age:          10,
				Hobby:        []string{"打篮球", "跑步"},
				CopySlicePtr: []*TmpNest{{Name: "copy"}},
			},
			dest: TmpDest{},
			ok: TmpDest{
				Name:         "name",
				Age:          10,
				Hobby:        []string{"打篮球", "跑步"},
				CopySlicePtr: []*TmpNest{{Name: "copy"}},
			},
		},
		{
			desc: "copy-slice-demo",
			src: TmpSrc{
				Name:         "name",
				Age:          10,
				Hobby:        []string{"打篮球", "跑步"},
				CopySlicePtr: []*TmpNest{{Name: "copy-ptr"}},
				CopySlice:    []TmpNest{{Name: "copy"}},
			},
			dest: TmpDest{},
			ok: TmpDest{
				Name:         "name",
				Age:          10,
				Hobby:        []string{"打篮球", "跑步"},
				CopySlicePtr: []*TmpNest{{Name: "copy-ptr"}},
				CopySlice:    []TmpNest{{Name: "copy"}},
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			b, _ := json.Marshal(&tC.src)
			if err := json.Unmarshal(b, &tC.dest); err != nil {
				t.Error("json unmarshal err:", err)
			}
			CheckObj(t, tC.dest, tC.ok)
		})
	}
}

func BenchmarkConvert(b *testing.B) {
	src := TmpSrc{
		Name:  "name",
		Age:   10,
		Hobby: []string{"打篮球", "跑步"},
	}
	dest := TmpDest{}
	for i := 0; i < b.N; i++ {
		obj := NewConvStruct()
		obj.Init(&src, &dest)
		// _ = obj.Convert()
		_ = obj.ConvertUnsafe()
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
func BenchmarkCopyerConvert(b *testing.B) {
	src := TmpSrc{
		Name:        "name",
		Age:         10,
		Hobby:       []string{"打篮球", "跑步"},
		NeedMarshal: []*TmpNest{{Name: "需要 marshal 测试"}},
	}
	dest := TmpDest{}
	for i := 0; i < b.N; i++ {
		copier.Copy(&dest, &src)
	}
}
