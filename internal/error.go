package internal

import (
	"errors"
	"fmt"
)

var (
	GetField2ColInfoMapErr = "%q GetField2ColInfoMap initArgs is not ok"

	StructTagErr = fmt.Errorf("you should sure struct is ok, eg: %s", "type User struct {\n"+
		"    Name string `json:\"name\"`\n"+
		"}")
	TableNameIsUnknownErr = errors.New("table name is unknown")
	NullRowErr            = errors.New("row is null")
	FindOneDestTypeErr    = errors.New("dest should is struct/oneField/map")
	FindAllDestTypeErr    = errors.New("dest should is struct/oneField/map slice")
	BuilderIsNilErr       = errors.New("builder is nil, you should check is first call Select/Insert/Update/Delete")
)
