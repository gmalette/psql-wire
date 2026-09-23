package wire

import (
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func registerScalarCodecs(typeMap *pgtype.Map) {
	typeMap.RegisterType(&pgtype.Type{
		Name:  "int8",
		OID:   pgtype.Int8OID,
		Codec: &int8Codec{},
	})
	typeMap.RegisterType(&pgtype.Type{
		Name:  "timestamp",
		OID:   pgtype.TimestampOID,
		Codec: &timestampCodec{},
	})
}

type int8Codec struct {
	pgtype.Int8Codec
}

func (c *int8Codec) PlanEncode(m *pgtype.Map, oid uint32, format int16, value any) pgtype.EncodePlan {
	if format == pgtype.TextFormatCode {
		if _, ok := value.(int64); ok {
			return int8TextEncodePlan{}
		}
	}
	return c.Int8Codec.PlanEncode(m, oid, format, value)
}

type int8TextEncodePlan struct{}

func (int8TextEncodePlan) Encode(value any, buf []byte) ([]byte, error) {
	return strconv.AppendInt(buf, value.(int64), 10), nil
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
