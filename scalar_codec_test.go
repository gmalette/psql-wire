package wire

import (
	"math"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestDefaultScalarCodecs(t *testing.T) {
	typeMap := (&Server{}).newTypeMap()

	int8Type, ok := typeMap.TypeForOID(pgtype.Int8OID)
	require.True(t, ok)
	require.IsType(t, &int8Codec{}, int8Type.Codec)

	timestampType, ok := typeMap.TypeForOID(pgtype.TimestampOID)
	require.True(t, ok)
	require.IsType(t, &timestampCodec{}, timestampType.Codec)
}

func TestInt8CodecMatchesPGX(t *testing.T) {
	gotMap := (&Server{}).newTypeMap()
	wantMap := pgtype.NewMap()

	for _, format := range []int16{pgtype.TextFormatCode, pgtype.BinaryFormatCode} {
		for _, value := range []int64{math.MinInt64, -1, 0, 1, math.MaxInt64} {
			got, err := gotMap.Encode(pgtype.Int8OID, format, value, nil)
			require.NoError(t, err)

			want, err := wantMap.Encode(pgtype.Int8OID, format, value, nil)
			require.NoError(t, err)
			require.Equal(t, want, got)
		}
	}
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

func TestTypeExtensionsOverrideScalarCodecs(t *testing.T) {
	server := &Server{
		typeExtension: func(typeMap *pgtype.Map) {
			typeMap.RegisterType(&pgtype.Type{
				Name:  "int8",
				OID:   pgtype.Int8OID,
				Codec: pgtype.Int8Codec{},
			})
		},
	}

	typeMap := server.newTypeMap()
	dataType, ok := typeMap.TypeForOID(pgtype.Int8OID)
	require.True(t, ok)
	require.IsType(t, pgtype.Int8Codec{}, dataType.Codec)
}

func BenchmarkScalarCodecTextEncoding(b *testing.B) {
	benchmarks := []struct {
		name  string
		oid   uint32
		value any
	}{
		{name: "int8", oid: pgtype.Int8OID, value: int64(35_000_000)},
		{name: "timestamp", oid: pgtype.TimestampOID, value: time.Date(2026, time.September, 22, 14, 45, 12, 123456789, time.UTC)},
	}

	for _, benchmark := range benchmarks {
		b.Run(benchmark.name, func(b *testing.B) {
			for _, implementation := range []struct {
				name    string
				typeMap *pgtype.Map
			}{
				{name: "pgx", typeMap: pgtype.NewMap()},
				{name: "psql-wire", typeMap: (&Server{}).newTypeMap()},
			} {
				b.Run(implementation.name, func(b *testing.B) {
					plan := implementation.typeMap.PlanEncode(benchmark.oid, pgtype.TextFormatCode, benchmark.value)
					require.NotNil(b, plan)
					buf := make([]byte, 0, 64)

					b.ReportAllocs()
					b.ResetTimer()
					for b.Loop() {
						encoded, err := plan.Encode(benchmark.value, buf[:0])
						if err != nil {
							b.Fatal(err)
						}
						if len(encoded) == 0 {
							b.Fatal("encoded value is empty")
						}
					}
				})
			}
		})
	}
}
