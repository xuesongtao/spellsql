package utils

import (
	"errors"
	"reflect"

	"gitee.com/xuesongtao/spellsql/v2/internal"
)

type ParseStructFieldRet struct {
	Offset int           // 偏移量
	Field  string        // tag val, 如果 tag 为空, 则为字段名
	Ty     reflect.Type  // 字段类型
	Tv     reflect.Value // 值
}

// ParseStruct 解析 struct 的字段, 返回字段信息
func ParseStructField(tv reflect.Value, tag ...string) ([]*ParseStructFieldRet, error) {
	defaultTag := internal.DefaultTableTag
	if len(tag) > 0 && tag[0] != "" {
		defaultTag = tag[0]
	}
	tv = RemoveValuePtr(tv)
	ty := tv.Type()
	if tv.Kind() != reflect.Struct {
		return nil, errors.New("it must is struct, it is " + ty.String())
	}

	res := make([]*ParseStructFieldRet, 0)
	fieldNum := ty.NumField()
	for i := 0; i < fieldNum; i++ {
		ty := ty.Field(i)
		field := ty.Tag.Get(defaultTag)
		if field == "" && IsExported(ty.Name) {
			field = ty.Name
		}
		field = ParseTag2Col(field)
		if field == "" {
			continue
		}

		res = append(res, &ParseStructFieldRet{
			Offset: i,
			Field:  field,
			Ty:     ty.Type,
			Tv:     tv.Field(i),
		})
	}
	return res, nil
}
