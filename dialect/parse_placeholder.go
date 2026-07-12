package dialect

import (
	"reflect"
	"strings"

	"gitee.com/xuesongtao/spellsql/v2/internal"
	"gitee.com/xuesongtao/spellsql/v2/utils"
)

type ParsePlaceholder struct {
	dbType    DbType
	buf       *strings.Builder
	waitParse string
	args      []interface{}
}

func NewParsePlaceholder(dt DbType, sqlStr string, args ...interface{}) *ParsePlaceholder {
	obj := &ParsePlaceholder{
		dbType:    dt,
		waitParse: sqlStr,
		args:      args,
		buf:       &strings.Builder{},
	}
	return obj
}

func (p *ParsePlaceholder) loop(f func(curIndex, argIndex, lastIndex int) int) {
	argLen := len(p.args)
	if argLen == 0 {
		p.buf.WriteString(p.waitParse)
		return
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
		i = f(i, argIndex, sqlLen-1)
	}
}

func (p *ParsePlaceholder) Parse() *ParsePlaceholder {
	p.buf.Reset()
	gd := GetDialect(p.dbType)
	p.loop(
		func(i, argIndex, lastIndex int) int {
			switch val := p.args[argIndex].(type) {
			case string:
				if i < lastIndex {
					// 如果占位符?在最后一位时, 就不往下执行了防止 panic
					// 判断下如果为 ?d 字符的话, 这里不需要加引号
					// 如果包含字母的话, 就转为 0, 防止数字型注入
					if p.waitParse[i+1] == 'd' {
						p.buf.WriteString(internal.EscapeOfHasNum(val, true, gd.GetValueEscapeMap()))
						i++
						return i
					} else if p.waitParse[i+1] == 'v' { // 原样输出
						p.buf.WriteString(val)
						i++
						return i
					}
				}

				if val == internal.NULL {
					p.buf.WriteString(internal.NULL)
				} else {
					p.buf.WriteString(WarpValue(gd, internal.EscapeOfHasNum(val, false, gd.GetValueEscapeMap())))
				}
			case []string:
				lastIndex := len(val) - 1
				// 判断下是否加引号
				isAdd := true
				// 这里必须小于最后一个最后一值才行
				if i < lastIndex {
					if p.waitParse[i+1] == 'd' {
						isAdd = false
						i++
					}
					for i1 := 0; i1 <= lastIndex; i1++ {
						if isAdd {
							p.buf.WriteString(gd.GetWarpValueStrSymbol())
						}
						p.buf.WriteString(internal.EscapeOfHasNum(val[i1], !isAdd, gd.GetValueEscapeMap()))
						if isAdd {
							p.buf.WriteString(gd.GetWarpValueStrSymbol())
						}
						if i1 < lastIndex {
							p.buf.WriteString(", ")
						}
					}
				} else {
					// 最后一个占位符
					for i1 := 0; i1 <= lastIndex; i1++ {
						p.buf.WriteString(WarpValue(gd, internal.EscapeOfHasNum(val[i1], false, gd.GetValueEscapeMap())))
						if i1 < lastIndex {
							p.buf.WriteString(", ")
						}
					}
				}
			case []byte:
				p.buf.WriteString(WarpValue(gd, internal.EscapeOfHasNum(string(val), false, gd.GetValueEscapeMap())))
			case int:
				p.buf.WriteString(utils.Int2Str(int64(val)))
			case int32:
				p.buf.WriteString(utils.Int2Str(int64(val)))
			case uint:
				p.buf.WriteString(utils.UInt2Str(uint64(val)))
			case uint32:
				p.buf.WriteString(utils.UInt2Str(uint64(val)))
			case []int:
				lastIndex := len(val) - 1
				for i1 := 0; i1 <= lastIndex; i1++ {
					p.buf.WriteString(utils.Int2Str(int64(val[i1])))
					if i1 < lastIndex {
						p.buf.WriteString(", ")
					}
				}
			case []int32:
				lastIndex := len(val) - 1
				for i1 := 0; i1 <= lastIndex; i1++ {
					p.buf.WriteString(utils.Int2Str(int64(val[i1])))
					if i1 < lastIndex {
						p.buf.WriteString(", ")
					}
				}
			default:
				// slow path
				reflectValue := reflect.ValueOf(val)
				switch reflectValue.Kind() {
				case reflect.Slice, reflect.Array: // 这里不会有 []string, 不需要处理符号, 所以直接处理即可
					lastIndex := reflectValue.Len() - 1
					for i1 := 0; i1 <= lastIndex; i1++ {
						p.buf.WriteString(utils.Str(reflectValue.Index(i1).Interface()))
						if i1 < lastIndex {
							p.buf.WriteString(", ")
						}
					}
				case reflect.Float32, reflect.Float64:
					p.buf.WriteString(utils.Str(reflectValue.Float()))
				case reflect.Int8, reflect.Int16, reflect.Int, reflect.Int32, reflect.Int64:
					p.buf.WriteString(utils.Str(reflectValue.Int()))
				case reflect.Uint8, reflect.Uint16, reflect.Uint, reflect.Uint32, reflect.Uint64:
					p.buf.WriteString(utils.Str(reflectValue.Uint()))
				case reflect.String:
					p.buf.WriteString(WarpValue(gd, internal.EscapeOfHasNum(reflectValue.String(), false, gd.GetValueEscapeMap())))
				default:
					p.buf.WriteString("undefined")
				}
			}
			return i
		},
	)
	return p
}

// Replace 将占位符替换为对应的数据库占位符, 例如 mysql 为 ?, pg 为 $1, $2, ...
func (p *ParsePlaceholder) Replace() *ParsePlaceholder {
	p.buf.Reset()
	// 需要将 ?, ?d, ?v 进行替换为对应数据库的占位符
	p.loop(
		func(curIndex, argIndex, lastIndex int) int {
			hasSuffix := false
			if curIndex < lastIndex {
				next := p.waitParse[curIndex+1]
				if next == 'd' || next == 'v' {
					hasSuffix = true
				}
			}
			switch p.dbType {
			case Postgres:
				p.buf.WriteString("$")
				p.buf.WriteString(utils.Int2Str(int64(argIndex + 1)))
			default:
				p.buf.WriteString("?")
			}

			if hasSuffix {
				curIndex++
			}
			return curIndex
		},
	)
	return p
}

func (p *ParsePlaceholder) Result() string {
	return p.buf.String()
}
