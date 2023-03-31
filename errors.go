package spellsql

import (
	"errors"
	"fmt"
)

// orm 部分
var (
	structTagErr = fmt.Errorf("you should sure struct is ok, eg: %s", "type User struct {\n"+
		"    Name string `json:\"name\"`\n"+
		"}")
	tableNameIsUnknownErr = errors.New("table name is unknown")
	nullRowErr            = errors.New("row is null")
	findOneDestTypeErr    = errors.New("dest should is struct/oneField/map")
	findAllDestTypeErr    = errors.New("dest should is struct/oneField/map slice")
)
