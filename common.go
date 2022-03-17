package spellsql

import "reflect"

// removeValuePtr 移除多指针
func removeValuePtr(t reflect.Value) reflect.Value {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// isExported 是可导出
func isExported(filedName string) bool {
	if filedName == "" {
		return false
	}
	first := filedName[0]
	return first >= 'A' && first <= 'Z'
}
