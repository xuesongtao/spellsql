package dialect

import (
	"gitee.com/xuesongtao/spellsql/v2/internal"
	"gitee.com/xuesongtao/spellsql/v2/utils"
)

type SqliteTable struct{}

// Sqlite
func Sqlite() *SqliteTable {
	return &SqliteTable{}
}

func (m *SqliteTable) GetWarpColSymbol() string {
	return "\""
}

func (m *SqliteTable) GetWarpValueStrSymbol() string {
	return "'"
}

func (m *SqliteTable) GetValueEscapeMap() map[byte][]byte {
	escapeMap := internal.GetValueEscapeMap()
	// 将 "'" 进行转义
	escapeMap['\''] = []byte{'\'', '\''}
	return escapeMap
}

// GetLimitSql implements [Dialect].
func (m *SqliteTable) GetLimitSql(limit int, offset int) string {
	return "LIMIT " + utils.Int2Str(int64(limit)) + " OFFSET " + utils.Int2Str(int64(offset))
}
