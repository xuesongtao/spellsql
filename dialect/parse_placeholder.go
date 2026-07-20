package dialect

import (
	"reflect"
	"strings"

	"gitee.com/xuesongtao/spellsql/v2/internal"
	"gitee.com/xuesongtao/spellsql/v2/utils"
)

type arg struct {
	del bool
	val interface{}
}

type ParsePlaceholder struct {
	dbType    DbType
	buf       *strings.Builder
	waitParse string
	args      []interface{}
}

// NewParsePlaceholder 创建一个占位符解析器
// dbType: 数据库类型
// sqlStr: 待解析的 sql 语句
// args: 占位符对应的参数
// 支持的占位符有:
// ?: 常规占位符, 会根据数据库类型替换为对应数据库的占位符, 例如 mysql 为 ?, pg 为 $1, $2, ...
// ?d: (特殊占位符)数字占位符, 会替换成数字参数, arg 支持 string/[]string
// ?v: (特殊占位符)原样输出占位符, 会替换为原样参数, arg 支持 string
func NewParsePlaceholder(dt DbType, sqlStr string, args ...interface{}) *ParsePlaceholder {
	obj := &ParsePlaceholder{
		dbType:    dt,
		waitParse: sqlStr,
		args:      args,
		buf:       &strings.Builder{},
	}
	return obj.replace()
}

func (p *ParsePlaceholder) loopWaitParse(f func(curIndex, argIndex, sqlSqlLastIndex int) int) {
	p.buf.Reset()

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
		// 使用过滤后的 args
		i = f(i, argIndex, sqlLen-1)
	}
}

func (p *ParsePlaceholder) toNum(v string) string {
	tmpBuf := internal.GetTmpBuf()
	defer internal.PutTmpBuf(tmpBuf)

	for i := 0; i < len(v); i++ {
		b := v[i]
		if b >= '0' && b <= '9' {
			tmpBuf.WriteByte(b)
		}
	}
	if tmpBuf.Len() > 0 {
		return tmpBuf.String()
	}
	return "0"
}

// replace 将 ?d, ?v 进行替换为对应的值
func (p *ParsePlaceholder) replace() *ParsePlaceholder {
	tmpArgs := make([]arg, len(p.args))
	for i, v := range p.args {
		tmpArgs[i] = arg{val: v}
	}

	// 需要将 ?d, ?v 进行替换为对应的值, 这两个占位符只会出现在 string 类型的参数中
	p.loopWaitParse(
		func(curIndex, argIndex, sqlSqlLastIndex int) int {
			if curIndex < sqlSqlLastIndex {
				switch v := tmpArgs[argIndex].val.(type) {
				case string:
					// 如果占位符?在最后一位时, 就不往下执行了防止 panic
					// 判断下如果为 ?d 字符的话, 这里不需要加引号
					// 如果包含字母的话, 就转为 0, 防止数字型注入
					if p.waitParse[curIndex+1] == 'd' {
						p.buf.WriteString(p.toNum(v))
						curIndex++
						tmpArgs[argIndex].del = true
						return curIndex
					} else if p.waitParse[curIndex+1] == 'v' { // 原样输出
						p.buf.WriteString(v)
						curIndex++
						tmpArgs[argIndex].del = true
						return curIndex
					}
				case []string:
					// 判断下是否加引号
					if p.waitParse[curIndex+1] == 'd' {
						lastIndex := len(v) - 1
						for i1 := 0; i1 <= lastIndex; i1++ {
							p.buf.WriteString(p.toNum(v[i1]))
							if i1 < lastIndex {
								p.buf.WriteString(", ")
							}
						}
						curIndex++
						tmpArgs[argIndex].del = true
						return curIndex
					}
				}
			}
			p.buf.WriteByte(p.waitParse[curIndex]) // 直接输出 ?
			return curIndex
		})

	p.waitParse = p.buf.String()
	p.args = make([]interface{}, 0)
	for _, v := range tmpArgs {
		if v.del {
			continue
		}
		p.args = append(p.args, v.val)
	}
	return p
}

// Parse 将占位符进行解析, 将占位符替换为对应的值
func (p *ParsePlaceholder) Parse() *ParsePlaceholder {
	gd := GetDialect(p.dbType)
	p.loopWaitParse(
		func(curIndex, argIndex, sqlSqlLastIndex int) int {
			switch val := p.args[argIndex].(type) {
			case string:
				if val == internal.NULL || val == internal.DEFAULT {
					p.buf.WriteString(val)
				} else {
					p.buf.WriteString(WarpValue(gd, internal.EscapeOfHasNum(val, gd.GetValueEscapeMap())))
				}
			case []string:
				lastIndex := len(val) - 1
				for i1 := 0; i1 <= lastIndex; i1++ {
					p.buf.WriteString(gd.GetWarpValueStrSymbol())
					p.buf.WriteString(internal.EscapeOfHasNum(val[i1], gd.GetValueEscapeMap()))
					p.buf.WriteString(gd.GetWarpValueStrSymbol())
					if i1 < lastIndex {
						p.buf.WriteString(", ")
					}
				}
			case []byte:
				p.buf.WriteString(WarpValue(gd, internal.EscapeOfHasNum(string(val), gd.GetValueEscapeMap())))
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
					p.buf.WriteString(WarpValue(gd, internal.EscapeOfHasNum(reflectValue.String(), gd.GetValueEscapeMap())))
				default:
					p.buf.WriteString("undefined")
				}
			}
			return curIndex
		},
	)
	return p
}

// Replace 将占位符 "?" 替换为对应的数据库占位符, 例如 mysql 为 ?, pg 为 $1, $2, ...
func (p *ParsePlaceholder) Replace() *ParsePlaceholder {
	p.loopWaitParse(
		func(curIndex, argIndex, lastIndex int) int {
			switch p.dbType {
			case Postgres:
				p.buf.WriteString("$")
				p.buf.WriteString(utils.Int2Str(int64(argIndex + 1)))
			default:
				p.buf.WriteString("?")
			}
			return curIndex
		},
	)
	return p
}

// Result 获取最终的 sql 语句
func (p *ParsePlaceholder) Result() string {
	return p.buf.String()
}

// Args 获取最终的参数列表
func (p *ParsePlaceholder) Args() []interface{} {
	return p.args
}
