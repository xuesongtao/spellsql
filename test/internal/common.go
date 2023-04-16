package internal

import (
	"fmt"
	"reflect"
)

const (
	NoEqErr = "src, dest is not eq"
)


type Man struct {
	Id       int32  `json:"id,omitempty" gorm:"id" db:"id"`
	Name     string `json:"name,omitempty" gorm:"name" db:"name"`       // 姓名
	SName    string `json:"s_name,omitempty" gorm:"s_name" db:"s_name"` // 学名
	Age      int32  `json:"age,omitempty" gorm:"age" db:"age"`
	Addr     string `json:"addr,omitempty" gorm:"addr" db:"addr"`
	Hobby    string `json:"hobby,omitempty"`
	NickName string `json:"nickname,omitempty" gorm:"nickname" db:"nickname"`
	JsonTxt  Tmp    `json:"json_txt,omitempty"`
	XmlTxt   Tmp    `json:"xml_txt,omitempty"`
	Json1Txt Tmp    `json:"json1_txt,omitempty"`
}

type Tmp struct {
	Name string `json:"name,omitempty" xml:"name"`
	Data string `json:"data,omitempty" xml:"data"`
}

func StructValEqual(dest, src interface{}) bool {
	destVal := reflect.ValueOf(dest)
	srcVal := reflect.ValueOf(src)
	if destVal.NumField() != srcVal.NumField() {
		fmt.Printf("dest: %v\n", dest)
		fmt.Printf("src: %v\n", src)
		return false
	}
	for i := 0; i < destVal.NumField(); i++ {
		if ok := Equal(destVal.Field(i).Interface(), srcVal.Field(i).Interface()); !ok {
			return false
		}
	}
	return true
}

func Equal(dest, src interface{}) bool {
	ok := reflect.DeepEqual(dest, src)
	if !ok {
		fmt.Printf("dest: %v\n", dest)
		fmt.Printf("src: %v\n", src)
	}
	return ok
}
