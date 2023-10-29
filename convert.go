package spellsql

import (
	. "database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

var errNilPtr = errors.New("destination pointer is nil") // embedded in descriptive error

type convFieldInfo struct {
	offset    int // 偏移量
	tagVal    string
	ty        reflect.Type
	marshal   marshalFn   // 序列化方法
	unmarshal unmarshalFn // 反序列化方法
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
		tag: defaultTableTag,
	}
	if len(tagName) > 0 && tagName[0] != "" {
		obj.tag = tagName[0]
	}
	return obj
}

func (c *ConvStructObj) initSrc(ry reflect.Type) error {
	c.srcFieldMap = make(map[string]*convFieldInfo)
	err := c.initCacheFieldMap(ry, func(tagVal string, field *convFieldInfo) {
		c.srcFieldMap[tagVal] = field
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *ConvStructObj) initDest(ry reflect.Type) error {
	c.descFieldMap = make(map[string]*convFieldInfo)
	err := c.initCacheFieldMap(ry, func(tagVal string, field *convFieldInfo) {
		c.descFieldMap[tagVal] = field
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *ConvStructObj) initCacheFieldMap(ry reflect.Type, f func(tagVal string, field *convFieldInfo)) error {
	if ry.Kind() != reflect.Struct {
		return errors.New("val must is struct")
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
				ty:     ty.Type,
			},
		)
	}
	return nil
}

// Init 初始化
func (c *ConvStructObj) Init(src, dest interface{}) error {
	c.srcRv = removeValuePtr(reflect.ValueOf(src))
	c.destRv = removeValuePtr(reflect.ValueOf(dest))
	if err := c.initSrc(c.srcRv.Type()); err != nil {
		return err
	}
	if err := c.initDest(c.destRv.Type()); err != nil {
		return err
	}
	return nil
}

// SrcMarshal 设置需要将 src marshal 转 dest
// 如: src: obj => dest: string
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
	for tagVal, destFieldInfo := range c.descFieldMap {
		srcFieldInfo, ok := c.srcFieldMap[tagVal]
		if !ok {
			continue
		}
		srcVal := c.srcRv.Field(srcFieldInfo.offset)
		if srcVal.IsZero() {
			continue
		}

		destVal := c.destRv.Field(destFieldInfo.offset)
		if srcFieldInfo.marshal != nil { // 需要将 src marshal 转, src: obj => dest: string
			if destFieldInfo.ty.Kind() != reflect.String {
				return errors.New("dest must string")
			}
			b, err := srcFieldInfo.marshal(srcVal.Interface())
			if err != nil {
				return err
			}
			destVal.SetString(string(b))
		} else if srcFieldInfo.unmarshal != nil { // 需要 src unmarshal 转, src: string => dest: obj
			if srcFieldInfo.ty.Kind() != reflect.String {
				return errors.New("src must string")
			}

			if err := srcFieldInfo.unmarshal([]byte(srcVal.String()), destVal.Addr().Interface()); err != nil {
				return err
			}
			return nil
		} else {
			err := convertAssign(destVal.Addr().Interface(), srcVal.Interface())
			if err != nil {
				return err
			}
		}
	}
	return nil
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
			return fmt.Errorf("converting driver.Value type %T (%q) to a %s: %v", src, s, dv.Kind(), err)
		}
		dv.SetInt(i64)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		s := asString(src)
		u64, err := strconv.ParseUint(s, 10, dv.Type().Bits())
		if err != nil {
			err = strconvErr(err)
			return fmt.Errorf("converting driver.Value type %T (%q) to a %s: %v", src, s, dv.Kind(), err)
		}
		dv.SetUint(u64)
		return nil
	case reflect.Float32, reflect.Float64:
		s := asString(src)
		f64, err := strconv.ParseFloat(s, dv.Type().Bits())
		if err != nil {
			err = strconvErr(err)
			return fmt.Errorf("converting driver.Value type %T (%q) to a %s: %v", src, s, dv.Kind(), err)
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

	return fmt.Errorf("unsupported Scan, storing driver.Value type %T into type %T", src, dest)
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
