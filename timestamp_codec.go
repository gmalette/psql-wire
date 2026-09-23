package wire

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func registerTimestampCodec(typeMap *pgtype.Map) {
	typeMap.RegisterType(&pgtype.Type{
		Name:  "timestamp",
		OID:   pgtype.TimestampOID,
		Codec: &timestampCodec{},
	})
}

type timestampCodec struct {
	pgtype.TimestampCodec
}

func (c *timestampCodec) PlanEncode(m *pgtype.Map, oid uint32, format int16, value any) pgtype.EncodePlan {
	if format == pgtype.TextFormatCode {
		if _, ok := value.(time.Time); ok {
			return timestampTimeTextEncodePlan{}
		}
	}
	return c.TimestampCodec.PlanEncode(m, oid, format, value)
}

type timestampTimeTextEncodePlan struct{}

func (timestampTimeTextEncodePlan) Encode(value any, buf []byte) ([]byte, error) {
	return appendTimestamp(buf, value.(time.Time)), nil
}

func appendTimestamp(buf []byte, value time.Time) []byte {
	bc := false
	if year := value.Year(); year <= 0 {
		year = -year + 1
		value = time.Date(
			year,
			value.Month(),
			value.Day(),
			value.Hour(),
			value.Minute(),
			value.Second(),
			value.Nanosecond(),
			time.UTC,
		)
		bc = true
	}

	buf = value.Truncate(time.Microsecond).AppendFormat(buf, "2006-01-02 15:04:05.999999999")
	if bc {
		buf = append(buf, " BC"...)
	}
	return buf
}
