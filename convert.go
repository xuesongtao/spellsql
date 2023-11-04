package spellsql

import (
	. "database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var errNilPtr = errors.New("destination pointer is nil") // embedded in descriptive error

type convFieldInfo struct {
	exclude   bool         // 标记是否排除
	offset    int          // 偏移量
	tagVal    string       // tag val
	kind      reflect.Kind // 字段类型
	marshal   marshalFn    // 序列化方法
	unmarshal unmarshalFn  // 反序列化方法
}

type ConvStructObj struct {
	tag           string
	srcRv, destRv reflect.Value
	descFieldMap  map[string]*convFieldInfo // key: tagVal
	srcFieldMap   map[string]*convFieldInfo // key: tagVal
}

// NewConvStruct 转换 struct, 将两个对象相同 tag 进行转换
// 字段取值默认按 defaultTableTag 来取值
func NewConvStruct(tagName ...string) *ConvStructObj {
	obj := &ConvStructObj{
		tag:          defaultTableTag,
		srcFieldMap:  make(map[string]*convFieldInfo),
		descFieldMap: make(map[string]*convFieldInfo),
	}
	if len(tagName) > 0 && tagName[0] != "" {
		obj.tag = tagName[0]
	}
	return obj
}

func (c *ConvStructObj) initCacheFieldMap(ry reflect.Type, f func(tagVal string, field *convFieldInfo)) error {
	if ry.Kind() != reflect.Struct {
		return errors.New("it must is struct, it is " + ry.String())
	}

	fieldNum := ry.NumField()
	for i := 0; i < fieldNum; i++ {
		ty := ry.Field(i)
		tagVal := parseTag2Col(ty.Tag.Get(c.tag))
		if tagVal == "" {
			continue
		}

		f(
			tagVal,
			&convFieldInfo{
				offset: i,
				tagVal: tagVal,
				kind:   ty.Type.Kind(),
			},
		)
	}
	return nil
}

// Init 初始化
func (c *ConvStructObj) Init(src, dest interface{}) error {
	c.srcRv = removeValuePtr(reflect.ValueOf(src))
	err := c.initCacheFieldMap(
		c.srcRv.Type(),
		func(tagVal string, field *convFieldInfo) {
			c.srcFieldMap[tagVal] = field
		},
	)
	if err != nil {
		return err
	}

	c.destRv = removeValuePtr(reflect.ValueOf(dest))
	err = c.initCacheFieldMap(
		c.destRv.Type(),
		func(tagVal string, field *convFieldInfo) {
			c.descFieldMap[tagVal] = field
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// Exclude 需要排除的转换的值
// 注: 需要晚于 Init 调用
func (c *ConvStructObj) Exclude(tagVals ...string) *ConvStructObj {
	for _, tagVal := range tagVals {
		if obj, ok := c.descFieldMap[tagVal]; ok {
			obj.exclude = true
		}
	}
	return c
}

// SrcMarshal 设置需要将 src marshal 转 dest
// 如: src: obj => dest: string
// 注: 需要晚于 Init 调用
func (c *ConvStructObj) SrcMarshal(fn marshalFn, tagVal ...string) *ConvStructObj {
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
func (c *ConvStructObj) SrcUnmarshal(fn unmarshalFn, tagVal ...string) *ConvStructObj {
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

// Convert 转换
func (c *ConvStructObj) Convert() error {
	errBuf := new(strings.Builder)
	for tagVal, destFieldInfo := range c.descFieldMap {
		if destFieldInfo.exclude {
			continue
		}

		srcFieldInfo, ok := c.srcFieldMap[tagVal]
		if !ok {
			continue
		}
		srcVal := c.srcRv.Field(srcFieldInfo.offset)
		if srcVal.IsZero() {
			continue
		}

		destVal := c.destRv.Field(destFieldInfo.offset)
		if srcFieldInfo.marshal != nil { // src: obj => dest: string
			if destFieldInfo.kind != reflect.String {
				return fmt.Errorf("dest %q must string", tagVal)
			}

			b, err := srcFieldInfo.marshal(srcVal.Interface())
			if err != nil {
				return fmt.Errorf("src %q, dest %q marshal is failed, err: %v", tagVal, tagVal, err)
			}
			destVal.SetString(string(b))
			continue
		}
		if srcFieldInfo.unmarshal != nil { // src: string => dest: obj
			if srcFieldInfo.kind != reflect.String {
				return fmt.Errorf("src %q must string", tagVal)
			}

			if err := srcFieldInfo.unmarshal([]byte(srcVal.String()), destVal.Addr().Interface()); err != nil {
				return fmt.Errorf("src %q, dest %q unmarshal is failed, err: %v", tagVal, tagVal, err)
			}
			continue
		}

		// normal
		kind := srcVal.Kind()
		if isOneField(kind) { // 单字段
			err := convertAssign(destVal.Addr().Interface(), srcVal.Interface())
			if err != nil {
				if errBuf.Len() > 0 {
					errBuf.WriteString("; ")
				}
				errBuf.WriteString(c.joinConvertErr(tagVal, tagVal, err))
			}
			continue
		}

		if kind == reflect.Ptr || kind == reflect.Struct { // struct
			isPtr := kind == reflect.Ptr
			destValType := destVal.Type()
			if isPtr {
				destValType = destValType.Elem()
			}

			tmpObj := reflect.New(destValType)
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
			continue
		}

		if kind == reflect.Slice || kind == reflect.Array { // slice
			l := srcVal.Len()
			sliceSrcValType := srcVal.Type().Elem()        // 取 slice 值的类型
			isPtr := sliceSrcValType.Kind() == reflect.Ptr // 注: 这里只处理 struct ptr
			if isPtr {
				sliceSrcValType = removeTypePtr(sliceSrcValType) // 去 ptr
			}
			sliceSrcValKind := sliceSrcValType.Kind()
			if isOneField(sliceSrcValKind) { // 单字段
				tmpSlice := reflect.MakeSlice(srcVal.Type(), 0, l)
				for i := 0; i < l; i++ {
					tmpSlice = reflect.Append(tmpSlice, srcVal.Index(i))
				}
				destVal.Set(tmpSlice)
			} else if sliceSrcValKind == reflect.Struct { // struct
				tmpSlice := reflect.MakeSlice(srcVal.Type(), 0, l)
				for i := 0; i < l; i++ {
					tmpObj := reflect.New(sliceSrcValType)
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
		return errors.New(errBuf.String())
	}
	return nil
}

func (c *ConvStructObj) joinConvertErr(destFieldName, srcFieldName string, err error) string {
	return fmt.Sprintf("src %q, dest %q convert is failed, err: %v", destFieldName, srcFieldName, err)
}

// convertAssign copies to dest the value in src, converting it if possible.
// An error is returned if the copy would result in loss of information.
// dest should be a pointer type. If rows is passed in, the rows will
// be used as the parent for any cursor values converted from a
func convertAssign(dest, src interface{}) error {
	// Common cases, without reflect.
	switch s := src.(type) {
	case string:
		switch d := dest.(type) {
		case *string:
			if d == nil {
				return errNilPtr
			}
			*d = s
			return nil
		case *[]byte:
			if d == nil {
				return errNilPtr
			}
			*d = []byte(s)
			return nil
		case *RawBytes:
			if d == nil {
				return errNilPtr
			}
			*d = append((*d)[:0], s...)
			return nil
		}
	case []byte:
		switch d := dest.(type) {
		case *string:
			if d == nil {
				return errNilPtr
			}
			*d = string(s)
			return nil
		case *interface{}:
			if d == nil {
				return errNilPtr
			}
			*d = cloneBytes(s)
			return nil
		case *[]byte:
			if d == nil {
				return errNilPtr
			}
			*d = cloneBytes(s)
			return nil
		case *RawBytes:
			if d == nil {
				return errNilPtr
			}
			*d = s
			return nil
		}
	case time.Time:
		switch d := dest.(type) {
		case *time.Time:
			*d = s
			return nil
		case *string:
			*d = s.Format(time.RFC3339Nano)
			return nil
		case *[]byte:
			if d == nil {
				return errNilPtr
			}
			*d = []byte(s.Format(time.RFC3339Nano))
			return nil
		case *RawBytes:
			if d == nil {
				return errNilPtr
			}
			*d = s.AppendFormat((*d)[:0], time.RFC3339Nano)
			return nil
		}
	case nil:
		switch d := dest.(type) {
		case *interface{}:
			if d == nil {
				return errNilPtr
			}
			*d = nil
			return nil
		case *[]byte:
			if d == nil {
				return errNilPtr
			}
			*d = nil
			return nil
		case *RawBytes:
			if d == nil {
				return errNilPtr
			}
			*d = nil
			return nil
		}
	}

	var sv reflect.Value

	switch d := dest.(type) {
	case *string:
		sv = reflect.ValueOf(src)
		switch sv.Kind() {
		case reflect.Bool,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			*d = asString(src)
			return nil
		}
	case *[]byte:
		sv = reflect.ValueOf(src)
		if b, ok := asBytes(nil, sv); ok {
			*d = b
			return nil
		}
	case *RawBytes:
		sv = reflect.ValueOf(src)
		if b, ok := asBytes([]byte(*d)[:0], sv); ok {
			*d = RawBytes(b)
			return nil
		}
	case *bool:
		bv, err := driver.Bool.ConvertValue(src)
		if err == nil {
			*d = bv.(bool)
		}
		return err
	case *interface{}:
		*d = src
		return nil
	}
	dpv := reflect.ValueOf(dest)
	if dpv.Kind() != reflect.Ptr {
		return errors.New("destination not a pointer")
	}
	if dpv.IsNil() {
		return errNilPtr
	}

	if !sv.IsValid() {
		sv = reflect.ValueOf(src)
	}

	dv := reflect.Indirect(dpv)
	if sv.IsValid() && sv.Type().AssignableTo(dv.Type()) {
		switch b := src.(type) {
		case []byte:
			dv.Set(reflect.ValueOf(cloneBytes(b)))
		default:
			dv.Set(sv)
		}
		return nil
	}

	if dv.Kind() == sv.Kind() && sv.Type().ConvertibleTo(dv.Type()) {
		dv.Set(sv.Convert(dv.Type()))
		return nil
	}

	// The following conversions use a string value as an intermediate representation
	// to convert between various numeric types.
	//
	// This also allows scanning into user defined types such as "type Int int64".
	// For symmetry, also check for string destination types.
	switch dv.Kind() {
	case reflect.Ptr:
		dv.Set(reflect.New(dv.Type().Elem()))
		return convertAssign(dv.Interface(), src)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		s := asString(src)
		i64, err := strconv.ParseInt(s, 10, dv.Type().Bits())
		if err != nil {
			err = strconvErr(err)
			return fmt.Errorf("converting type %T (%q) to a %s: %v", src, s, dv.Kind(), err)
		}
		dv.SetInt(i64)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		s := asString(src)
		u64, err := strconv.ParseUint(s, 10, dv.Type().Bits())
		if err != nil {
			err = strconvErr(err)
			return fmt.Errorf("converting type %T (%q) to a %s: %v", src, s, dv.Kind(), err)
		}
		dv.SetUint(u64)
		return nil
	case reflect.Float32, reflect.Float64:
		s := asString(src)
		f64, err := strconv.ParseFloat(s, dv.Type().Bits())
		if err != nil {
			err = strconvErr(err)
			return fmt.Errorf("converting type %T (%q) to a %s: %v", src, s, dv.Kind(), err)
		}
		dv.SetFloat(f64)
		return nil
	case reflect.String:
		switch v := src.(type) {
		case string:
			dv.SetString(v)
			return nil
		case []byte:
			dv.SetString(string(v))
			return nil
		}
	}

	return fmt.Errorf("unsupported convert, type %T into type %T", src, dest)
}

func strconvErr(err error) error {
	if ne, ok := err.(*strconv.NumError); ok {
		return ne.Err
	}
	return err
}

func cloneBytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	c := make([]byte, len(b))
	copy(c, b)
	return c
}

func asString(src interface{}) string {
	switch v := src.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	}
	rv := reflect.ValueOf(src)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(rv.Uint(), 10)
	case reflect.Float64:
		return strconv.FormatFloat(rv.Float(), 'g', -1, 64)
	case reflect.Float32:
		return strconv.FormatFloat(rv.Float(), 'g', -1, 32)
	case reflect.Bool:
		return strconv.FormatBool(rv.Bool())
	}
	return fmt.Sprintf("%v", src)
}

func asBytes(buf []byte, rv reflect.Value) (b []byte, ok bool) {
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.AppendInt(buf, rv.Int(), 10), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.AppendUint(buf, rv.Uint(), 10), true
	case reflect.Float32:
		return strconv.AppendFloat(buf, rv.Float(), 'g', -1, 32), true
	case reflect.Float64:
		return strconv.AppendFloat(buf, rv.Float(), 'g', -1, 64), true
	case reflect.Bool:
		return strconv.AppendBool(buf, rv.Bool()), true
	case reflect.String:
		s := rv.String()
		return append(buf, s...), true
	}
	return
}
