package sqldb

import (
	"context"
	"fmt"

	"gitee.com/xuesongtao/spellsql/v2/dialect"
	"gitee.com/xuesongtao/spellsql/v2/internal"
)

func init() {
    dialect.RegisterTabaleMeter(dialect.MySQL, func() dialect.TableMeter {
		return &MySql{}
	})
}

type MySql struct{}

func (m *MySql) GetColInfoMap(ctx context.Context, db dialect.DBer, tableName string) (map[string]*dialect.TableColInfo, error) {
	sqlStr := fmt.Sprintf("SHOW COLUMNS FROM %s", tableName)
	rows, err := db.QueryContext(ctx, sqlStr)
	if err != nil {
		return nil, fmt.Errorf("mysql query is failed, err: %v, sqlStr: %v", err, sqlStr)
	}
	defer rows.Close()

	cacheCol2InfoMap := make(map[string]*dialect.TableColInfo)
	var index int
	for rows.Next() {
		var info dialect.TableColInfo
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

func (m *MySql) GetDefaultVal(col string, colInfo *dialect.TableColInfo) internal.RawSql {
	return internal.DEFAULT
}
