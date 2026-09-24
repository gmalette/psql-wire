package wire

import (
	"context"
	"io"
	"log/slog"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jeroenrinzema/psql-wire/pkg/buffer"
)

type countingRowSink struct {
	io.Writer
	writes int
	bytes  int
}

func (s *countingRowSink) Write(p []byte) (int, error) {
	s.writes++
	n, err := s.Writer.Write(p)
	s.bytes += n
	return n, err
}

func benchmarkRows() (Columns, [][]any) {
	columns := Columns{{Oid: pgtype.Int8OID}, {Oid: pgtype.Int8OID}, {Oid: pgtype.Int8OID}, {Oid: pgtype.Int8OID}, {Oid: pgtype.Int8OID}, {Oid: pgtype.TimestampOID}}
	rows := make([][]any, 10000)
	stamp := time.Date(2026, 9, 16, 1, 2, 3, 456000000, time.UTC)
	for i := range rows {
		rows[i] = []any{int64(100000 + i), int64(200000 + i), int64(300000 + i), int64(400000 + i), int64(500000 + i), stamp}
	}
	return columns, rows
}

// BenchmarkDataWriterRows measures one materialized 10,000-row response. The
// counting wrapper records underlying Write calls (socket write attempts in the
// loopback case), not packets. Input construction and connection setup are
// outside the timed region; writer buffers are warm, as on a reused connection.
func BenchmarkDataWriterRows(b *testing.B) {
	benchmarkDataWriterRows(b, true, false)
}

func BenchmarkDataWriterRowsObserved(b *testing.B) {
	benchmarkDataWriterRows(b, true, true)
}

func BenchmarkDataWriterRowLoop(b *testing.B) {
	benchmarkDataWriterRows(b, false, false)
}

func benchmarkDataWriterRows(b *testing.B, batch bool, observe bool) {
	for _, sink := range []string{"discard", "loopback"} {
		b.Run(sink, func(b *testing.B) {
			output := io.Discard
			if sink == "loopback" {
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					b.Fatal(err)
				}
				defer func() { _ = listener.Close() }()
				conn, err := net.Dial("tcp", listener.Addr().String())
				if err != nil {
					b.Fatal(err)
				}
				peer, err := listener.Accept()
				if err != nil {
					_ = conn.Close()
					b.Fatal(err)
				}
				done := make(chan struct{})
				go func() {
					defer close(done)
					_, _ = io.Copy(io.Discard, peer)
				}()
				defer func() {
					_ = conn.Close()
					<-done
					_ = peer.Close()
				}()
				output = conn
			}
			columns, rows := benchmarkRows()
			ctx := setTypeInfo(context.Background(), pgtype.NewMap())
			// Model a request context with twenty layers above connection state.
			type depthKey int
			for i := range 20 {
				ctx = context.WithValue(ctx, depthKey(i), i)
			}
			counted := &countingRowSink{Writer: output}
			var observedValues atomic.Uint64
			var observedBytes atomic.Uint64
			writer := &dataWriter{
				ctx: ctx, columns: columns, formats: []FormatCode{TextFormat},
				client:    buffer.NewWriter(slog.New(slog.NewTextHandler(io.Discard, nil)), counted),
				yield:     func(struct{}) bool { return true },
				batchable: true,
			}
			if observe {
				writer.encodeObserver = func(_ context.Context, _ FormatCode, _ uint32, count uint64, encodedBytes uint64) {
					observedValues.Add(count)
					observedBytes.Add(encodedBytes)
				}
			}
			write := func() {
				if batch {
					if err := WriteRows(writer, rows); err != nil {
						b.Fatal(err)
					}
					return
				}
				for _, row := range rows {
					if err := writer.Row(row); err != nil {
						b.Fatal(err)
					}
				}
			}
			write()
			counted.writes, counted.bytes = 0, 0
			observedValues.Store(0)
			observedBytes.Store(0)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				write()
			}
			b.StopTimer()
			if observe {
				want := uint64(len(rows) * len(columns) * b.N)
				if observedValues.Load() != want {
					b.Fatalf("observed %d values, want %d", observedValues.Load(), want)
				}
			}
			b.ReportMetric(float64(counted.writes)/float64(b.N), "writes/op")
			b.ReportMetric(float64(counted.bytes)/float64(b.N), "wire-B/op")
		})
	}
}
