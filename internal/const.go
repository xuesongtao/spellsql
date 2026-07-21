package internal

type OpType = uint8

const (
	// sql 操作数字
	None OpType = iota
	INSERT
	INSERT_REPLACE
	INSERT_IGNORE
	INSERT_ON_DUPLICATE
	DELETE
	SELECT
	SELECT_AND
	SELECT_OR
	UPDATE

	// sql LIKE 语句
	ALK // 全模糊 如: xxx LIKE "%xxx%"
	RLK // 右模糊 如: xxx LIKE "xxx%"
	LLK // 左模糊 如: xxx LIKE "%xxx"

	// sql join 语句
	LJI // 左连接
	RJI // 右连接
)

// 原样输入
const (
	NULL    RawSql = "NULL"
	DEFAULT RawSql = "DEFAULT"
)

const (
	PriFlag     = "PRI" // 主键标识
	NotNullFlag = "NO"  // 非空标识
)

const (
	DefaultTableTag        = "json"
	DefaultBatchSelectSize = 10 // 批量查询默认条数
)

// RawSql 内部使用的原始 sql, 主要是为了在 insert/update 时, 直接原样输出, 不进行 sql 解析, 例如: DEFAULT, NULL 等
type RawSql string

func (r RawSql) Is(v RawSql) bool {
	return r == v
}
