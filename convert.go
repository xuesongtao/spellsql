package spellsql

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"gitee.com/xuesongtao/spellsql/v2/internal"
	"gitee.com/xuesongtao/spellsql/v2/utils"
)

type convFieldInfo struct {
	exclude   bool          // 标记是否排除
	offset    int           // 偏移量
	field     string        // tag val, 如果 tag 为空, 则为字段名
	ry        reflect.Type  // 字段类型
	tv        reflect.Value // 值
	marshal   MarshalFn     // 序列化方法
	unmarshal UnmarshalFn   // 反序列化方法
}

func (c *convFieldInfo) getKind() reflect.Kind {
	return c.ry.Kind()
}

type ConvStructObj struct {
	deep bool // 是否深拷贝
	tag  string
	// srcRv, destRv reflect.Value
	destFieldMap map[string]*convFieldInfo // key: tagVal
	srcFieldMap  map[string]*convFieldInfo // key: tagVal
}

// NewConvStruct 转换 struct, 将两个对象相同 tag 进行转换, 所有内容进行深拷贝
// 字段取值默认按 defaultTableTag 来取值
func NewConvStruct(tagName ...string) *ConvStructObj {
	obj := &ConvStructObj{
		deep:         true,
		tag:          internal.DefaultTableTag,
		srcFieldMap:  make(map[string]*convFieldInfo),
		destFieldMap: make(map[string]*convFieldInfo),
	}
	if len(tagName) > 0 && tagName[0] != "" {
		obj.tag = tagName[0]
	}
	return obj
}

func (c *ConvStructObj) initFieldMap(tv reflect.Value, f func(tagVal string, field *convFieldInfo)) error {
	tv = utils.RemoveValuePtr(tv)
	ty := tv.Type()
	if tv.Kind() != reflect.Struct {
		return errors.New("it must is struct, it is " + ty.String())
	}

	fieldNum := ty.NumField()
	for i := 0; i < fieldNum; i++ {
		ty := ty.Field(i)
		field := ty.Tag.Get(c.tag)
		if field == "" && utils.IsExported(ty.Name) {
			field = ty.Name
		}
		field = utils.ParseTag2Col(field)
		if field == "" {
			continue
		}

		f(
			field,
			&convFieldInfo{
				offset: i,
				field:  field,
				ry:     ty.Type,
				tv:     tv.Field(i),
			},
		)
	}
	return nil
}

// Init 初始化
func (c *ConvStructObj) Init(src, dest interface{}) error {
	err := c.initFieldMap(
		reflect.ValueOf(src),
		func(tagVal string, field *convFieldInfo) {
			c.srcFieldMap[tagVal] = field
		},
	)
	if err != nil {
		return err
	}

	err = c.initFieldMap(
		reflect.ValueOf(dest),
		func(tagVal string, field *convFieldInfo) {
			c.destFieldMap[tagVal] = field
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// Exclude 需要排除 dest 的转换的值
// 注: 需要晚于 Init 调用
func (c *ConvStructObj) Exclude(tagVals ...string) *ConvStructObj {
	for _, tagVal := range tagVals {
		if obj, ok := c.destFieldMap[tagVal]; ok {
			obj.exclude = true
		}
	}
	return c
}

// SrcMarshal 设置需要将 src marshal 转 dest
// 如: src: obj => dest: string
// 注: 需要晚于 Init 调用
func (c *ConvStructObj) SrcMarshal(fn MarshalFn, tagVal ...string) *ConvStructObj {
	if c.srcFieldMap == nil {
		return c
	}

	for _, v := range tagVal {
		if c.srcFieldMap[v] != nil {
			c.srcFieldMap[v].marshal = fn
		}
	}
	return c
}

// SrcUnmarshal 设置需要 src unmarshal 转 dest
// 如: src: string => dest: obj
// 注: 需要晚于 Init 调用
func (c *ConvStructObj) SrcUnmarshal(fn UnmarshalFn, tagVal ...string) *ConvStructObj {
	if c.srcFieldMap == nil {
		return c
	}

	for _, v := range tagVal {
		if c.srcFieldMap[v] != nil {
			c.srcFieldMap[v].unmarshal = fn
		}
	}
	return c
}

// ConvertUnsafe 转换, 主要使用浅拷贝进行转换
// 相较于 Convert 减少了多余内存分配, 性能要好些
func (c *ConvStructObj) ConvertUnsafe() error {
	c.deep = false
	return c.Convert()
}

// Convert 转换, 所有内容进行深拷贝
// 说明: src 为零值的字段将不会进行转换
func (c *ConvStructObj) Convert() error {
	errBuf := new(strings.Builder)
	for tagVal, destFieldInfo := range c.destFieldMap {
		if destFieldInfo.exclude {
			continue
		}

		srcFieldInfo, ok := c.srcFieldMap[tagVal]
		if !ok {
			continue
		}

		// 取值
		srcVal := srcFieldInfo.tv
		if srcVal.IsZero() {
			continue
		}

		srcVal = utils.RemoveValuePtr(srcVal)
		srcKind := srcVal.Kind()
		destVal := destFieldInfo.tv
		if srcFieldInfo.marshal != nil { // src: obj => dest: string
			if destFieldInfo.getKind() != reflect.String {
				errBuf.WriteString(fmt.Sprintf("src %q is set marshal, but dest %q is not string;", tagVal, tagVal))
				continue
			}

			b, err := srcFieldInfo.marshal(srcVal.Interface())
			if err != nil {
				errBuf.WriteString(fmt.Sprintf("src %q, dest %q marshal is failed, err: %v;", tagVal, tagVal, err))
				continue
			}
			destVal.SetString(string(b))
		} else if srcFieldInfo.unmarshal != nil { // src: string => dest: obj
			if srcFieldInfo.getKind() != reflect.String {
				errBuf.WriteString(fmt.Sprintf("dest %q is set unmarshal, but src %q is not string;", tagVal, tagVal))
				continue
			}

			if err := srcFieldInfo.unmarshal([]byte(srcVal.String()), destVal.Addr().Interface()); err != nil {
				errBuf.WriteString(fmt.Sprintf("src %q, dest %q unmarshal is failed, err: %v;", tagVal, tagVal, err))
				continue
			}
		} else if utils.IsOneField(srcKind) || !c.deep { // src: 单字段 => dest: 单字段
			err := internal.ConvertAssign(destVal.Addr().Interface(), srcVal.Interface())
			if err != nil {
				errBuf.WriteString(c.joinConvertErr(tagVal, tagVal, err))
			}
		} else if srcKind == reflect.Struct { // src: struct => dest: struct
			destValType := destVal.Type()
			isPtr := destValType.Kind() == reflect.Ptr
			if isPtr {
				destValType = destValType.Elem()
			}

			tmpObj := reflect.New(destValType) // 临时对象
			convObj := NewConvStruct(c.tag)
			_ = convObj.Init(srcVal.Interface(), tmpObj.Interface())
			err := convObj.Convert()
			if err != nil {
				errBuf.WriteString(c.joinConvertErr(tagVal, tagVal, err))
				continue
			}

			if isPtr {
				destVal.Set(tmpObj)
			} else {
				destVal.Set(tmpObj.Elem())
			}
		} else if srcKind == reflect.Slice || srcKind == reflect.Array { // src: slice => dest: slice
			// 需要判断下 dest slice 的类型
			if destVal.Kind() != reflect.Slice && destVal.Kind() != reflect.Array {
				errBuf.WriteString(c.joinConvertErr(tagVal, tagVal, errors.New("dest is not a slice")))
				continue
			}
			l := srcVal.Len()
			sliceDstValType := destVal.Type().Elem()       // 取 slice 值的类型
			isPtr := sliceDstValType.Kind() == reflect.Ptr // 注: 这里只处理 struct ptr
			if isPtr {
				sliceDstValType = utils.RemoveTypePtr(sliceDstValType) // 去 ptr
			}
			sliceDstValKind := sliceDstValType.Kind()
			if utils.IsOneField(sliceDstValKind) { // 单字段
				tmpSlice := reflect.MakeSlice(destVal.Type(), 0, l)
				for i := 0; i < l; i++ {
					tmpSlice = reflect.Append(tmpSlice, srcVal.Index(i))
				}
				destVal.Set(tmpSlice)
			} else if sliceDstValKind == reflect.Struct { // struct
				tmpSlice := reflect.MakeSlice(destVal.Type(), 0, l)
				for i := 0; i < l; i++ {
					tmpObj := reflect.New(sliceDstValType)
					convObj := NewConvStruct(c.tag)
					_ = convObj.Init(srcVal.Index(i).Interface(), tmpObj.Interface())
					err := convObj.Convert()
					if err != nil {
						errBuf.WriteString(c.joinConvertErr(tagVal, tagVal, err))
						continue
					}
					if !isPtr {
						tmpSlice = reflect.Append(tmpSlice, tmpObj.Elem())
						continue
					}
					tmpSlice = reflect.Append(tmpSlice, tmpObj)
				}
				destVal.Set(tmpSlice)
			}
		}
	}

	if errBuf.Len() > 0 {
		return errors.New(strings.TrimSuffix(errBuf.String(), ";"))
	}
	return nil
}

func (c *ConvStructObj) joinConvertErr(destFieldName, srcFieldName string, err error) string {
	return fmt.Sprintf("src %q, dest %q convert is failed, err: %v;", destFieldName, srcFieldName, err)
}
