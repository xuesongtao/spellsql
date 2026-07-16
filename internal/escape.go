package internal

// GetValueEscapeMap 获取值的转义处理
func GetValueEscapeMap() map[byte][]byte {
	// key 为待转义的字符, value [0]为如何处理转义 [1]转义为
	escapeMap := map[byte][]byte{
		'\'':   {'\\', '\''},
		'"':    {'\\', '"'},
		'\x00': {'\\', '0'},
		'\n':   {'\\', 'n'},
		'\r':   {'\\', 'r'},
		'\t':   {'\\', 't'},
		'\x1a': {'\\', 'Z'},
		'\\':   {'\\', '\\'},
	}
	return escapeMap
}

// Escape 转义字符
func Escape(val []byte, escapeMap map[byte][]byte) []byte {
	if escapeMap == nil {
		escapeMap = GetValueEscapeMap()
	}
	return toEscapeBytes(val, false, escapeMap)
}

// EscapeOfHasNum 转义
func EscapeOfHasNum(val string, is2Num bool, escapeMap map[byte][]byte) string {
	return string(toEscapeBytes([]byte(val), is2Num, escapeMap))
}

// toEscapeBytes 转义
func toEscapeBytes(val []byte, is2Num bool, escapeMap map[byte][]byte) []byte {
	pos := 0
	vLen := len(val)

	buf := make([]byte, vLen*2)
	for i := 0; i < len(val); i++ {
		v := val[i]
		bytes, ok := escapeMap[v]
		if ok {
			buf[pos] = bytes[0]
			buf[pos+1] = bytes[1]
			pos += 2
		} else {
			// 这里需要判断下在占位符: ?d 时是否包含字母, 如果有的话就转为 0, 防止数字型注入
			if is2Num && ((v >= 'A' && v <= 'Z') || (v >= 'a' && v <= 'z')) {
				v = '0'
			}
			buf[pos] = v
			pos++
		}
	}
	return buf[:pos]
}
