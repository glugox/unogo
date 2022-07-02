package formatter

import "github.com/glugox/unogo/log/record"

type Formatter interface {
	Format(r record.Record) string
	FormatBatch(rs []record.Record) string
}
