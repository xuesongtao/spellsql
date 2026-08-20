package dialect

import (
	"strings"

	"gitee.com/xuesongtao/spellsql/v2/internal"
	"gitee.com/xuesongtao/spellsql/v2/utils"
)

var dialectMap = map[DbType]Dialect{
	MySQL:    &mysql{},
	Postgres: &pgsql{},
	SQLite:   &sqlite{},
}

// Dialect 数据库方言接口, 适配不同数据库, 不变的部分
type Dialect interface {
	GetWarpColSymbol() string             // 获取字段包裹符号
	GetWarpValueStrSymbol() string        // 获取值为字符串的包裹符号
	GetValueEscapeMap() map[byte][]byte   // 获取值转义规则
	GetLimitSql(limit, offset int) string // 获取 limit sql 语句
}

// WarpValue 将值进行包裹, 如果已经包裹过了, 则不再包裹
func WarpValue(d Dialect, value string) string {
	if strings.HasPrefix(value, d.GetWarpValueStrSymbol()) {
		return value
	}
	return d.GetWarpValueStrSymbol() + value + d.GetWarpValueStrSymbol()
}

// GetDialect 获取数据库方言, 如果没有匹配到, 则返回默认的数据库方言
func GetDialect(dbType DbType) Dialect {
	dialect, ok := dialectMap[dbType]
	if ok {
		return dialect
	}
	return dialectMap[DefaultDbType]
}

// Placeholders 获取占位符字符串, 例如: "?, ?, ?"
func Placeholders(n ...int) string {
	nn := 1
	if len(n) > 0 {
		nn = n[0]
	}
	if nn <= 0 {
		return ""
	}
	return strings.Repeat("?, ", nn-1) + "?"
}

// ================= mysql start=====================
type mysql struct{}

func (m *mysql) GetWarpColSymbol() string {
	return "`"
}

func (m *mysql) GetWarpValueStrSymbol() string {
	return "\""
}

func (m *mysql) GetValueEscapeMap() map[byte][]byte {
	return internal.GetValueEscapeMap()
}

// GetLimitSql implements [Dialect].
func (m *mysql) GetLimitSql(limit int, offset int) string {
	return "LIMIT " + utils.Int2Str(int64(limit)) + " OFFSET " + utils.Int2Str(int64(offset))
}

// ================= mysql end=====================

// ================= postgres start=====================
type pgsql struct{}

// GetWarpColSymbol implements [Dialect].
func (p *pgsql) GetWarpColSymbol() string {
	return `"`
}

// GetWarpValueStrSymbol implements [Dialect].
func (p *pgsql) GetWarpValueStrSymbol() string {
	return `'`
}

// GetLimitSql implements [Dialect].
func (p *pgsql) GetLimitSql(limit int, offset int) string {
	return "LIMIT " + utils.Int2Str(int64(limit)) + " OFFSET " + utils.Int2Str(int64(offset))
}

func (p *pgsql) GetValueEscapeMap() map[byte][]byte {
	escapeMap := internal.GetValueEscapeMap()
	// 将 "'" 进行转义
	escapeMap['\''] = []byte{'\'', '\''}
	return escapeMap
}

// ================= postgres end=====================

// ================ sqlite start=====================
type sqlite struct{}

func (m *sqlite) GetWarpColSymbol() string {
	return "\""
}

func (m *sqlite) GetWarpValueStrSymbol() string {
	return "'"
}

func (m *sqlite) GetValueEscapeMap() map[byte][]byte {
	escapeMap := internal.GetValueEscapeMap()
	// 将 "'" 进行转义
	escapeMap['\''] = []byte{'\'', '\''}
	return escapeMap
}

// GetLimitSql implements [Dialect].
func (m *sqlite) GetLimitSql(limit int, offset int) string {
	return "LIMIT " + utils.Int2Str(int64(limit)) + " OFFSET " + utils.Int2Str(int64(offset))
}

// ================ sqlite end=====================
