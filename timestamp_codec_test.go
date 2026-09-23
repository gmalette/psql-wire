package wire

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestDefaultTimestampCodec(t *testing.T) {
	typeMap := (&Server{}).newTypeMap()

	timestampType, ok := typeMap.TypeForOID(pgtype.TimestampOID)
	require.True(t, ok)
	require.IsType(t, &timestampCodec{}, timestampType.Codec)
}

func TestTimestampCodecMatchesPGX(t *testing.T) {
	gotMap := (&Server{}).newTypeMap()
	wantMap := pgtype.NewMap()
	location := time.FixedZone("test", -5*60*60)
	timestamps := []pgtype.Timestamp{
		{},
		{Time: time.Date(10_000, time.December, 31, 23, 59, 59, 999999999, time.UTC), Valid: true},
		{Time: time.Date(2026, time.September, 22, 14, 45, 12, 123456789, time.UTC), Valid: true},
		{Time: time.Date(2026, time.September, 22, 14, 45, 12, 1000, time.UTC), Valid: true},
		{Time: time.Date(2026, time.September, 22, 14, 45, 12, 123456789, location), Valid: true},
		{Time: time.Date(0, time.January, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		{Time: time.Date(-10, time.January, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		{InfinityModifier: pgtype.Infinity, Valid: true},
		{InfinityModifier: pgtype.NegativeInfinity, Valid: true},
	}

	for _, format := range []int16{pgtype.TextFormatCode, pgtype.BinaryFormatCode} {
		for _, value := range timestamps {
			got, err := gotMap.Encode(pgtype.TimestampOID, format, value, nil)
			require.NoError(t, err)

			want, err := wantMap.Encode(pgtype.TimestampOID, format, value, nil)
			require.NoError(t, err)
			require.Equal(t, want, got)
		}
	}
}

func TestTimestampCodecTimeMatchesPGX(t *testing.T) {
	gotMap := (&Server{}).newTypeMap()
	wantMap := pgtype.NewMap()
	location := time.FixedZone("test", -5*60*60)
	timestamps := []time.Time{
		time.Date(10_000, time.December, 31, 23, 59, 59, 999999999, time.UTC),
		time.Date(2026, time.September, 22, 14, 45, 12, 123456789, time.UTC),
		time.Date(2026, time.September, 22, 14, 45, 12, 1000, time.UTC),
		time.Date(2026, time.September, 22, 14, 45, 12, 123456789, location),
		time.Date(0, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(-10, time.January, 1, 0, 0, 0, 0, time.UTC),
	}

	for _, format := range []int16{pgtype.TextFormatCode, pgtype.BinaryFormatCode} {
		for _, value := range timestamps {
			got, err := gotMap.Encode(pgtype.TimestampOID, format, value, nil)
			require.NoError(t, err)

			want, err := wantMap.Encode(pgtype.TimestampOID, format, value, nil)
			require.NoError(t, err)
			require.Equal(t, want, got)
		}
	}
}

func TestTypeExtensionsOverrideTimestampCodec(t *testing.T) {
	server := &Server{
		typeExtension: func(typeMap *pgtype.Map) {
			typeMap.RegisterType(&pgtype.Type{
				Name:  "timestamp",
				OID:   pgtype.TimestampOID,
				Codec: pgtype.TimestampCodec{},
			})
		},
	}

	typeMap := server.newTypeMap()
	dataType, ok := typeMap.TypeForOID(pgtype.TimestampOID)
	require.True(t, ok)
	require.IsType(t, pgtype.TimestampCodec{}, dataType.Codec)
}

func BenchmarkTimestampTextEncoding(b *testing.B) {
	var value any = time.Date(2026, time.September, 22, 14, 45, 12, 123456789, time.UTC)
	for _, implementation := range []struct {
		name    string
		typeMap *pgtype.Map
	}{
		{name: "pgx", typeMap: pgtype.NewMap()},
		{name: "psql-wire", typeMap: (&Server{}).newTypeMap()},
	} {
		b.Run(implementation.name, func(b *testing.B) {
			plan := implementation.typeMap.PlanEncode(pgtype.TimestampOID, pgtype.TextFormatCode, value)
			require.NotNil(b, plan)
			buf := make([]byte, 0, 64)

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				encoded, err := plan.Encode(value, buf[:0])
				if err != nil {
					b.Fatal(err)
				}
				if len(encoded) == 0 {
					b.Fatal("encoded value is empty")
				}
			}
		})
	}
}

func BenchmarkTimestampRows(b *testing.B) {
	columns := Columns{{Oid: pgtype.TimestampOID}}
	value := time.Date(2026, time.September, 22, 14, 45, 12, 123456789, time.UTC)
	rows := make([][]any, 10_000)
	for i := range rows {
		rows[i] = []any{value}
	}

	for _, implementation := range []struct {
		name    string
		typeMap *pgtype.Map
	}{
		{name: "pgx", typeMap: pgtype.NewMap()},
		{name: "psql-wire", typeMap: (&Server{}).newTypeMap()},
	} {
		b.Run(implementation.name, func(b *testing.B) {
			benchmarkDataWriterRowsWith(b, true, columns, rows, implementation.typeMap)
		})
	}
}
