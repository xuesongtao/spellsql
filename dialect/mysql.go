package dialect

import (
	"gitee.com/xuesongtao/spellsql/v2/internal"
	"gitee.com/xuesongtao/spellsql/v2/utils"
)

type MysqlTable struct{}

// Mysql
func Mysql() *MysqlTable {
	return &MysqlTable{}
}

func (m *MysqlTable) GetWarpColSymbol() string {
	return "`"
}

func (m *MysqlTable) GetWarpValueStrSymbol() string {
	return "\""
}

func (m *MysqlTable) GetValueEscapeMap() map[byte][]byte {
	return internal.GetValueEscapeMap()
}

// GetLimitSql implements [Dialect].
func (m *MysqlTable) GetLimitSql(limit int, offset int) string {
	return "LIMIT " + utils.Int2Str(int64(limit)) + " OFFSET " + utils.Int2Str(int64(offset))
}
