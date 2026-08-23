package sqldb

import (
	"context"
	"database/sql"
	"fmt"

	"gitee.com/xuesongtao/spellsql/v2/internal"
)

type PgSQL struct{}

func (p *PgSQL) GetColInfoMap(ctx context.Context, db DBer, tableName string) (map[string]*TableColInfo, error) {
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
            a.attrelid = ('public.%s')::regclass
            AND a.attnum > 0                   
            AND NOT a.attisdropped              
        ORDER BY a.attnum;
		`, tableName,
	)
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

func (p *PgSQL) GetDefaultVal(col string, colInfo *TableColInfo) internal.RawSql {
	return internal.DEFAULT
}
