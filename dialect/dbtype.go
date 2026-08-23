package dialect

import "strconv"

type DbType int // db 类型

func (d DbType) Is(dt DbType) bool {
	return d == dt
}

func (d DbType) String() string {
	switch d {
	case MySQL:
		return "mysql"
	case Postgres:
		return "postgres"
	case SQLite:
		return "sqlite"
	}
	return strconv.Itoa(int(d))
}

const (
	MySQL DbType = iota
	Postgres
	SQLite
)

var DefaultDbType = MySQL
