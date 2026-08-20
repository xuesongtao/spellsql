package dialect

import (
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


