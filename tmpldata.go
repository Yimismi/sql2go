package sql2go

import (
	"xorm.io/xorm/schemas"
)

type TmpData struct {
	Tables  []*schemas.Table
	Imports map[string]string
	Models  string
}
