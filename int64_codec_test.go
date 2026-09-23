package wire

import (
	"math"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestDefaultInt64Codec(t *testing.T) {
	typeMap := (&Server{}).newTypeMap()

	int8Type, ok := typeMap.TypeForOID(pgtype.Int8OID)
	require.True(t, ok)
	require.IsType(t, &int64Codec{}, int8Type.Codec)
}

func TestInt64CodecMatchesPGX(t *testing.T) {
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

func TestTypeExtensionsOverrideInt64Codec(t *testing.T) {
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

func BenchmarkInt64TextEncoding(b *testing.B) {
	value := int64(35_000_000)
	for _, implementation := range []struct {
		name    string
		typeMap *pgtype.Map
	}{
		{name: "pgx", typeMap: pgtype.NewMap()},
		{name: "psql-wire", typeMap: (&Server{}).newTypeMap()},
	} {
		b.Run(implementation.name, func(b *testing.B) {
			plan := implementation.typeMap.PlanEncode(pgtype.Int8OID, pgtype.TextFormatCode, value)
			require.NotNil(b, plan)
			buf := make([]byte, 0, 32)

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

func BenchmarkInt64Rows(b *testing.B) {
	columns := Columns{{Oid: pgtype.Int8OID}}
	rows := make([][]any, 10_000)
	for i := range rows {
		rows[i] = []any{int64(i)}
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
