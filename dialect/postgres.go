package dialect

import (
	"context"
	"database/sql"
	"fmt"

	"gitee.com/xuesongtao/spellsql/v2/internal"
	"gitee.com/xuesongtao/spellsql/v2/utils"
)

type PgTable struct {
	initArgs []string
}

// Pg, 默认模式: public
// initArgs 允许自定义两个参数
// initArgs[0] 为 schema
// initArgs[1] 为 table name (此参数可以忽略, 因为 orm 内部会处理该值)
func Pg(initArgs ...string) *PgTable {
	obj := &PgTable{initArgs: make([]string, 2)}
	l := len(initArgs)
	switch l {
	case 1:
		obj.initArgs[0] = initArgs[0]
	case 2:
		obj.initArgs[0] = initArgs[0]
		obj.initArgs[1] = initArgs[1]
	}
	if l == 0 {
		obj.initArgs[0] = "public"
	}
	return obj
}

// GetWarpColSymbol implements [Dialect].
func (p *PgTable) GetWarpColSymbol() string {
	return `"`
}

// GetWarpValueStrSymbol implements [Dialect].
func (p *PgTable) GetWarpValueStrSymbol() string {
	return `'`
}

func (p *PgTable) GetAdapterName() string {
	return "pg"
}

// GetLimitSql implements [Dialect].
func (p *PgTable) GetLimitSql(limit int, offset int) string {
	return "LIMIT " + utils.Int2Str(int64(limit)) + " OFFSET " + utils.Int2Str(int64(offset))
}

func (p *PgTable) SetTableName(name string) {
	p.initArgs[1] = name
}

func (p *PgTable) GetValueEscapeMap() map[byte][]byte {
	escapeMap := internal.GetValueEscapeMap()
	// 将 "'" 进行转义
	escapeMap['\''] = []byte{'\'', '\''}
	return escapeMap
}

func (p *PgTable) GetColInfoMap(ctx context.Context, db DBer, tableName string) (map[string]*TableColInfo, error) {
	if len(p.initArgs) != 2 {
		return nil, fmt.Errorf(internal.GetField2ColInfoMapErr, p.GetAdapterName())
	}
	sqlStr := fmt.Sprintf(
		"SELECT c.column_name,c.data_type,c.is_nullable,tc.constraint_type,c.column_default FROM information_schema.columns AS c "+
			"LEFT JOIN information_schema.constraint_column_usage AS ccu USING (column_name,table_name) "+
			"LEFT JOIN information_schema.table_constraints tc ON tc.constraint_name=ccu.constraint_name "+
			"WHERE c.table_schema='%s' AND c.table_name='%s'", p.initArgs[0], p.initArgs[1])
	rows, err := db.QueryContext(ctx, sqlStr)
	if err != nil {
		return nil, fmt.Errorf("pg query is failed, err: %v, sqlStr: %v", err, sqlStr)
	}
	defer rows.Close()

	cacheCol2InfoMap := make(map[string]*TableColInfo)
	var index int
	for rows.Next() {
		var (
			info TableColInfo
			key  sql.NullString
		)
		err = rows.Scan(&info.Field, &info.Type, &info.Null, &key, &info.Default)
		if err != nil {
			return nil, fmt.Errorf("pg scan is failed, err: %v", err)
		}
		if key.String == "PRIMARY KEY" {
			info.Key = PriFlag
		}
		info.Index = index
		cacheCol2InfoMap[info.Field] = &info
		index++
	}
	return cacheCol2InfoMap, nil
}
