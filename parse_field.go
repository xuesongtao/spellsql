package spellsql

import (
	"reflect"
	"strings"
)

type ParsePlaceholder struct {
	buf       *strings.Builder
	waitParse string
	args      []interface{}
}

func NewParsePlaceholder(sqlStr string, args ...interface{}) *ParsePlaceholder {
	obj := &ParsePlaceholder{
		buf:       &strings.Builder{},
		waitParse: sqlStr,
		args:      args,
	}
	return obj
}

func (p *ParsePlaceholder) Parse(tabMeter TableMetaer) *ParsePlaceholder {
	argLen := len(p.args)
	if argLen == 0 {
		p.buf.WriteString(p.waitParse)
		return p
	}

	sqlLen := len(p.waitParse)
	argIndex := -1
	for i := 0; i < sqlLen; i++ {
		v := p.waitParse[i]
		if v != '?' {
			p.buf.WriteByte(v)
			continue
		}
		argIndex++

		// 如果参数不够的话就不进行处理
		if argIndex > argLen-1 {
			p.buf.WriteByte(v)
			continue
		}

		switch val := p.args[argIndex].(type) {
		case string:
			if i < sqlLen-1 {
				// 如果占位符?在最后一位时, 就不往下执行了防止 panic
				// 判断下如果为 ?d 字符的话, 这里不需要加引号
				// 如果包含字母的话, 就转为 0, 防止数字型注入
				if p.waitParse[i+1] == 'd' {
					p.buf.WriteString(toEscape(val, true, tabMeter.GetValueEscapeMap()))
					i++
					continue
				} else if p.waitParse[i+1] == 'v' { // 原样输出
					p.buf.WriteString(val)
					i++
					continue
				}
			}

			if val == NULL {
				p.buf.WriteString(NULL)
			} else {
				p.buf.WriteByte(tabMeter.GetParcelFieldSymbol())
				p.buf.WriteString(toEscape(val, false, tabMeter.GetValueEscapeMap()))
				p.buf.WriteByte(tabMeter.GetParcelFieldSymbol())
			}
		case []string:
			lastIndex := len(val) - 1
			// 判断下是否加引号
			isAdd := true
			// 这里必须小于最后一个最后一值才行
			if i < sqlLen-1 {
				if p.waitParse[i+1] == 'd' {
					isAdd = false
					i++
				}
				for i1 := 0; i1 <= lastIndex; i1++ {
					if isAdd {
						p.buf.WriteByte(tabMeter.GetParcelFieldSymbol())
					}
					p.buf.WriteString(toEscape(val[i1], !isAdd, tabMeter.GetValueEscapeMap()))
					if isAdd {
						p.buf.WriteByte(tabMeter.GetParcelFieldSymbol())
					}
					if i1 < lastIndex {
						p.buf.WriteByte(',')
					}
				}
			} else {
				// 最后一个占位符
				for i1 := 0; i1 <= lastIndex; i1++ {
					p.buf.WriteByte(tabMeter.GetParcelFieldSymbol())
					p.buf.WriteString(toEscape(val[i1], false, tabMeter.GetValueEscapeMap()))
					p.buf.WriteByte(tabMeter.GetParcelFieldSymbol())
					if i1 < lastIndex {
						p.buf.WriteByte(',')
					}
				}
			}
		case []byte:
			p.buf.WriteByte(tabMeter.GetParcelFieldSymbol())
			p.buf.WriteString(toEscape(string(val), false, tabMeter.GetValueEscapeMap()))
			p.buf.WriteByte(tabMeter.GetParcelFieldSymbol())
		case int:
			p.buf.WriteString(Int2Str(int64(val)))
		case int32:
			p.buf.WriteString(Int2Str(int64(val)))
		case uint:
			p.buf.WriteString(UInt2Str(uint64(val)))
		case uint32:
			p.buf.WriteString(UInt2Str(uint64(val)))
		case []int:
			lastIndex := len(val) - 1
			for i1 := 0; i1 <= lastIndex; i1++ {
				p.buf.WriteString(Int2Str(int64(val[i1])))
				if i1 < lastIndex {
					p.buf.WriteByte(',')
				}
			}
		case []int32:
			lastIndex := len(val) - 1
			for i1 := 0; i1 <= lastIndex; i1++ {
				p.buf.WriteString(Int2Str(int64(val[i1])))
				if i1 < lastIndex {
					p.buf.WriteByte(',')
				}
			}
		default:
			// slow path
			reflectValue := reflect.ValueOf(val)
			switch reflectValue.Kind() {
			case reflect.Slice, reflect.Array: // 这里不会有 []string, 不需要处理符号, 所以直接处理即可
				lastIndex := reflectValue.Len() - 1
				for i1 := 0; i1 <= lastIndex; i1++ {
					p.buf.WriteString(Str(reflectValue.Index(i1).Interface()))
					if i1 < lastIndex {
						p.buf.WriteByte(',')
					}
				}
			case reflect.Float32, reflect.Float64:
				p.buf.WriteString(Str(reflectValue.Float()))
			case reflect.Int8, reflect.Int16, reflect.Int, reflect.Int32, reflect.Int64:
				p.buf.WriteString(Str(reflectValue.Int()))
			case reflect.Uint8, reflect.Uint16, reflect.Uint, reflect.Uint32, reflect.Uint64:
				p.buf.WriteString(Str(reflectValue.Uint()))
			case reflect.String:
				p.buf.WriteByte(tabMeter.GetParcelFieldSymbol())
				p.buf.WriteString(toEscape(reflectValue.String(), false, tabMeter.GetValueEscapeMap()))
				p.buf.WriteByte(tabMeter.GetParcelFieldSymbol())
			default:
				p.buf.WriteString("undefined")
			}
		}
	}
	return p
}

func (p *ParsePlaceholder) Result() string {
	return p.buf.String()
}

func Parse(sqlStr string, args ...interface{}) *strings.Builder {
	return NewParsePlaceholder(sqlStr, args...).Parse(getTmerFn()).buf
}
