package dialect

import (
	"context"
	"fmt"

	"gitee.com/xuesongtao/spellsql/v2/internal"
	"gitee.com/xuesongtao/spellsql/v2/utils"
)

type MysqlTable struct {
}

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

func (m *MysqlTable) GetAdapterName() string {
	return "mysql"
}

func (m *MysqlTable) GetColInfoMap(ctx context.Context, db DBer, tableName string) (map[string]*TableColInfo, error) {
	sqlStr := fmt.Sprintf("SHOW COLUMNS FROM %s", tableName)
	rows, err := db.QueryContext(ctx, sqlStr)
	if err != nil {
		return nil, fmt.Errorf("mysql query is failed, err: %v, sqlStr: %v", err, sqlStr)
	}
	defer rows.Close()

	cacheCol2InfoMap := make(map[string]*TableColInfo)
	var index int
	for rows.Next() {
		var info TableColInfo
		err = rows.Scan(&info.Field, &info.Type, &info.Null, &info.Key, &info.Default, &info.Extra)
		if err != nil {
			return nil, fmt.Errorf("mysql scan is failed, err: %v", err)
		}
		info.Index = index
		cacheCol2InfoMap[info.Field] = &info
		index++
	}
	return cacheCol2InfoMap, nil
}

func (m *MysqlTable) GetDefaultVal(col string, colInfo *TableColInfo) interface{} {
	return internal.DEFAULT
}
