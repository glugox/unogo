package orm

import "github.com/glugox/unogo/orm/field"

type Model struct {
	field.IDAttr
	field.CreatedAtAttr
	field.UpdatedAtAttr
}

type SoftDeletes struct {
	field.DeletedAtAttr
}
