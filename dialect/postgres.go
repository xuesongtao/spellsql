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
	sqlStr := fmt.Sprintf(
		`
		SELECT 
            a.attname AS column_name,
            pg_catalog.format_type(a.atttypid, a.atttypmod) AS data_type,
            NOT a.attnotnull AS is_nullable,
            pg_catalog.pg_get_expr(d.adbin, d.adrelid) AS default_value,
            COALESCE(
                (SELECT STRING_AGG(DISTINCT ct.contype, '') 
                 FROM pg_catalog.pg_constraint ct 
                 WHERE ct.conrelid = a.attrelid 
                   AND a.attnum = ANY(ct.conkey)
                ), ''
            ) AS constraint_types
        FROM 
            pg_catalog.pg_attribute a
        LEFT JOIN pg_catalog.pg_attrdef d 
            ON (a.attrelid = d.adrelid AND a.attnum = d.adnum)
        WHERE 
            a.attrelid = ('%s.%s')::regclass
            AND a.attnum > 0                   
            AND NOT a.attisdropped              
        ORDER BY a.attnum;
		`, p.initArgs[0], tableName)
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
		err = rows.Scan(&info.Field, &info.Type, &info.Null, &info.Default, &key)
		if err != nil {
			return nil, fmt.Errorf("pg scan is failed, err: %v", err)
		}
		if key.String == "np" {
			info.Key = PriFlag
		}
		if info.Null == "[v]" {
			info.Null = NotNullFlag
		}
		info.Index = index
		cacheCol2InfoMap[info.Field] = &info
		index++
	}
	return cacheCol2InfoMap, nil
}

func (p *PgTable) GetDefaultVal(col string, colInfo *TableColInfo) internal.RawSql {
	return internal.DEFAULT
}
