package buffer

import (
	"errors"
	"testing"
)

func TestErrMessageSizeExceeded(t *testing.T) {
	limit := DefaultBufferSize
	size := limit + 1024

	err := NewMessageSizeExceeded(limit, size)

	if !errors.Is(err, ErrMessageSizeExceeded) {
		t.Error("unexpected comparison, error should contain message size exceeded type")
	}

	exceeded, has := UnwrapMessageSizeExceeded(err)
	if !has {
		t.Fatal("unexpected result, expected message size exceeded to be wrapped")
	}

	if exceeded.Max != limit {
		t.Errorf("unexpected max size %d, expected %d", exceeded.Max, limit)
	}

	if exceeded.Size != size {
		t.Errorf("unexpected message size %d, expected %d", exceeded.Size, size)
	}
}
