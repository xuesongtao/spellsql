package spellsql

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// IndexForBF 查找, 通过 BF 算法来获取匹配的 index
// isFont2End 是否从主串前向后遍历查找
// 如果匹配的内容靠前建议 isFont2End=true, 反之 false
// TODO 暂不支持中文
func IndexForBF(isFont2End bool, s, substr string) int {
	substrLen := len(substr)
	sLen := len(s)
	switch {
	case sLen == 0 || substrLen == 0:
		return 0
	case substrLen > sLen:
		return -1
	}

	if isFont2End {
		for i := 0; i <= sLen-substrLen; i++ {
			for j := 0; j < substrLen; j++ {
				mainStr := s[i+j]
				sonStr := substr[j]
				if mainStr != sonStr {
					break
				}
				// 如果 j 为最后一个值的话说明全匹配
				if j == substrLen-1 {
					return i
				}
			}
		}
		return -1
	}

	for i := sLen - 1; i >= 0; i-- {
		for j := substrLen - 1; j >= 0; j-- {
			mainStr := s[i]
			sonStr := substr[j]
			if mainStr != sonStr {
				break
			}
			// 如果 j 为最后一个值的话说明全匹配
			if j == 0 {
				return i
			}

			// 如果匹配到最开头的字符时 i=0, 如果 i--, i 为负数, s[i] 会 panic
			if i > 0 {
				i--
			}
		}
	}
	return -1
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
		index := IndexForBF(true, s, split)
		if index < 0 {
			// 这里需要处理最后一个字符
			saveFunc(s)
			break
		}
		saveFunc(s[:index])
		s = s[index+1:]

		// 这样可以防止最后一位为 split 字符, 到时就会出现一个空
		if s == "" {
			break
		}
	}
	buf := new(strings.Builder)
	buf.Grow(strLen / 2)
	lastIndex := len(sortSlice) - 1
	for index, val := range sortSlice {
		v := distinctMap[val]
		if index < lastIndex {
			buf.WriteString(v + split)
		} else {
			buf.WriteString(v)
		}
	}
	return buf.String()
}

// DistinctIds 去重
func DistinctIds(ids []string) []string {
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

// removeValuePtr 移除多指针
func removeValuePtr(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}

// removeTypePtr 移除多指针
func removeTypePtr(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// isExported 是可导出
func isExported(fieldName string) bool {
	if fieldName == "" {
		return false
	}
	first := fieldName[0]
	return first >= 'A' && first <= 'Z'
}
