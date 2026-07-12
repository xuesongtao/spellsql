package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"gitee.com/xuesongtao/spellsql/v2/internal"
)

// IndexForBF 查找, 通过 BF 算法来获取匹配的 index
// isFont2End 是否从主串前向后遍历查找
// 如果匹配的内容靠前建议 isFont2End=true, 反之 false
// Deprecated 推荐用 Index
func IndexForBF(isFont2End bool, s, substr string) int {
	return Index(s, substr, isFont2End)
}

// Index
// 默认从前向后查找, 如果匹配的内容靠后建议 isFont2End=false
func Index(s, substr string, isFont2End ...bool) int {
	substrLen := len(substr)
	sLen := len(s)
	switch {
	case sLen == 0 || substrLen == 0:
		return 0
	case substrLen > sLen:
		return -1
	}
	defaultIsFont2End := true
	if len(isFont2End) > 0 {
		defaultIsFont2End = isFont2End[0]
	}
	if defaultIsFont2End {
		return strings.Index(s, substr)
	}
	return strings.LastIndex(s, substr)
}

// Str 将内容转为 string
func Str(src interface{}) string {
	if src == nil {
		return ""
	}

	switch value := src.(type) {
	case int:
		return strconv.Itoa(value)
	case int8:
		return strconv.Itoa(int(value))
	case int16:
		return strconv.Itoa(int(value))
	case int32:
		return strconv.Itoa(int(value))
	case int64:
		return strconv.FormatInt(value, 10)
	case uint:
		return strconv.FormatUint(uint64(value), 10)
	case uint8:
		return strconv.FormatUint(uint64(value), 10)
	case uint16:
		return strconv.FormatUint(uint64(value), 10)
	case uint32:
		return strconv.FormatUint(uint64(value), 10)
	case uint64:
		return strconv.FormatUint(value, 10)
	case float32:
		return strconv.FormatFloat(float64(value), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(value)
	case string:
		return value
	case []byte:
		return string(value)
	default:
		return fmt.Sprintf("%v", value)
	}
}

// Int64 将数字型类型转为 int64
func Int64(num interface{}) int64 {
	switch v := num.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case int16:
		return int64(v)
	case int8:
		return int64(v)
	}
	return 0
}

// DistinctIdsStr 将输入拼接 id 参数按照指定字符进行去重, 如:
// DistinctIdsStr("12345,123,20,123,20,15", ",")
// => 12345,123,20,15
func DistinctIdsStr(s string, split string) string {
	strLen := len(s)
	if strLen == 0 {
		return s
	}

	distinctMap := make(map[string]string, strLen/2)
	sortSlice := make([]string, 0, strLen/2) // 用于保证输出顺序
	saveFunc := func(val string) {
		val = strings.Trim(val, " ")
		if _, ok := distinctMap[val]; !ok {
			distinctMap[val] = val
			sortSlice = append(sortSlice, val)
		}
	}

	for {
		index := Index(s, split)
		if index < 0 {
			// 这里需要处理最后一个字符
			saveFunc(s)
			break
		}
		saveFunc(s[:index])
		s = s[index+1:]

		// 这样可以防止最后一位为 split 字符, 到时就会出现一个空
		if Null(s) {
			break
		}
	}
	buf := internal.GetTmpBuf(strLen / 2)
	defer internal.PutTmpBuf(buf)
	lastIndex := len(sortSlice) - 1
	for index, val := range sortSlice {
		v := distinctMap[val]
		if index < lastIndex {
			buf.WriteString(v)
			buf.WriteString(split)
		} else {
			buf.WriteString(v)
		}
	}
	return buf.String()
}

// DistinctIds 去重
func DistinctIds(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	tmp := make(map[string]struct{}, len(ids))
	res := make([]string, 0, len(ids))

	for _, id := range ids {
		if _, ok := tmp[id]; !ok {
			tmp[id] = struct{}{}
			res = append(res, id)
		}
	}
	return res
}

// RemoveValuePtr 移除多指针
func RemoveValuePtr(v reflect.Value) reflect.Value {
	last := v
	for v.Kind() == reflect.Ptr {
		// 如果最外层是未初始化的指针类型, 就不要再处理了, 直接返回未初始的类型就可以了, 防止 panic Zero Value
		if v.IsNil() {
			v = last
			break
		}
		v = v.Elem()
		last = v
	}
	return v
}

// removeTypePtr 移除多指针
func RemoveTypePtr(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// isExported 是可导出
func IsExported(fieldName string) bool {
	if Null(fieldName) {
		return false
	}
	first := fieldName[0]
	return first >= 'A' && first <= 'Z'
}

func Null(val string) bool {
	return val == ""
}

// isOneField 是否为单字段
func IsOneField(kind reflect.Kind) bool {
	// 将常用的类型放在前面
	switch kind {
	case reflect.String,
		reflect.Int64, reflect.Int32, reflect.Int, reflect.Int16, reflect.Int8,
		reflect.Uint64, reflect.Uint32, reflect.Uint, reflect.Uint16, reflect.Uint8,
		reflect.Float32, reflect.Float64,
		reflect.Bool:
		return true
	}
	return false
}

// parseTag2Col 解析 tag 中表的列名
func ParseTag2Col(tag string) (column string) {
	// 因为 tag 中有可能出现多个值, 需要处理下
	tmpIndex := Index(tag, ",")
	if tmpIndex > -1 {
		column = tag[:tmpIndex]
	} else {
		column = tag
	}
	return
}

// GetOffset 根据分页获取 offset
// page 从 1 开始
// 注: page, size 只支持 int 系列类型
func GetOffset(page, size interface{}) (int64, int64) {
	pageInt64, sizeInt64 := Int64(page), Int64(size)
	if pageInt64 <= 0 {
		pageInt64 = 1
	}
	if sizeInt64 <= 0 {
		sizeInt64 = 10
	}
	return sizeInt64, (pageInt64 - 1) * sizeInt64
}

func Int2Str(i int64) string {
	return strconv.FormatInt(i, 10)
}

func UInt2Str(i uint64) string {
	return strconv.FormatUint(i, 10)
}

func MarshalNoEscape(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(buf.Bytes()), nil
}
