package sqldb

import (
	"context"
	"fmt"

	"gitee.com/xuesongtao/spellsql/v2/dialect"
	"gitee.com/xuesongtao/spellsql/v2/internal"
)

func init() {
    dialect.RegisterTabaleMeter(dialect.SQLite, func() dialect.TableMeter {
		return &Sqlite{}
	})
}

type Sqlite struct{}

func (m *Sqlite) GetColInfoMap(ctx context.Context, db dialect.DBer, tableName string) (map[string]*dialect.TableColInfo, error) {
	sqlStr := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	rows, err := db.QueryContext(ctx, sqlStr)
	if err != nil {
		return nil, fmt.Errorf("sqlite query is failed, err: %v, sqlStr: %v", err, sqlStr)
	}
	defer rows.Close()

	cacheCol2InfoMap := make(map[string]*dialect.TableColInfo)
	var index int
	for rows.Next() {
		var tmpId int
		var info dialect.TableColInfo
		var pkNo int
		var nullFlag int
		err = rows.Scan(&tmpId, &info.Field, &info.Type, &nullFlag, &info.Default, &pkNo)
		if err != nil {
			return nil, fmt.Errorf("sqlite scan is failed, err: %v", err)
		}
		info.Index = index
		if pkNo == 1 {
			info.Key = dialect.PriFlag
		}
		if nullFlag == 1 {
			info.Null = dialect.NotNullFlag
		}
		cacheCol2InfoMap[info.Field] = &info
		index++
	}
	return cacheCol2InfoMap, nil
}

func (m *Sqlite) GetDefaultVal(col string, colInfo *dialect.TableColInfo) internal.RawSql {
	val := internal.RawSql(colInfo.Default.String)
	if val == "" {
		if colInfo.Null == dialect.NotNullFlag {
			val = "''"
		} else {
			val = internal.NULL
		}
	}
	return val
}
