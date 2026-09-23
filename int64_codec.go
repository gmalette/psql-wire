package wire

import (
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
)

func registerInt64Codec(typeMap *pgtype.Map) {
	typeMap.RegisterType(&pgtype.Type{
		Name:  "int8",
		OID:   pgtype.Int8OID,
		Codec: &int64Codec{},
	})
}

type int64Codec struct {
	pgtype.Int8Codec
}

func (c *int64Codec) PlanEncode(m *pgtype.Map, oid uint32, format int16, value any) pgtype.EncodePlan {
	if format == pgtype.TextFormatCode {
		if _, ok := value.(int64); ok {
			return int64TextEncodePlan{}
		}
	}
	return c.Int8Codec.PlanEncode(m, oid, format, value)
}

type int64TextEncodePlan struct{}

func (int64TextEncodePlan) Encode(value any, buf []byte) ([]byte, error) {
	return strconv.AppendInt(buf, value.(int64), 10), nil
}
