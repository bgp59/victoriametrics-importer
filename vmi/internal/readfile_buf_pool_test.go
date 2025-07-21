package vmi_internal

import (
	"bytes"
	"fmt"
	"os"

	"testing"
)

const (
	TEST_READ_FILE_BUF_POOL_TEST_FILE = "readfile_buf_pool_test.go"
)

func testBufPoolGetReturn(t *testing.T, maxPoolSize int) {
	p := NewBufPool(maxPoolSize)
	numGets := maxPoolSize + 1
	if maxPoolSize <= 0 {
		numGets = 13
	}
	// Retrieve buffers from empty pool. Check that they are empty and that the
	// pool size stays 0.
	for k := 0; k < numGets; k++ {
		b := p.GetBuf()
		if p.poolSize != 0 {
			t.Fatalf("GetBuf(k=%d): poolSize: want: %d, got: %d", k, 0, p.poolSize)
		}
		if b.Len() != 0 {
			t.Fatalf("GetBuf(k=%d): buf.Len(): want: %d, got: %d", k, 0, b.Len())
		}
	}
	// Return seeded buffers. Check that the pool size does not exceed the max, if
	// capped.
	for k := 0; k < numGets; k++ {
		p.ReturnBuf(bytes.NewBuffer([]byte{byte(k >> 24), byte(k >> 16), byte(k >> 8), byte(k & 255)}))
		wantPoolSize := k + 1
		if maxPoolSize > 0 && wantPoolSize > maxPoolSize {
			wantPoolSize = maxPoolSize
		}
		if p.poolSize != wantPoolSize {
			t.Fatalf("ReturnBuff(k=%d): poolSize: want: %d, got: %d", k, wantPoolSize, p.poolSize)
		}
	}
	// Retrieve again and check content; note that the buffers are retrieved from the end:
	if maxPoolSize > 0 && numGets > maxPoolSize {
		numGets = maxPoolSize
	}
	for k := numGets - 1; k >= 0; k-- {
		gotBytes := p.GetBuf().Bytes()
		if p.poolSize != k {
			t.Fatalf("GetBuf(k=%d): poolSize: want: %d, got: %d", k, k, p.poolSize)
		}
		if len(gotBytes) != 0 {
			t.Fatalf("GetBuf(k=%d): buf.Len(): want: %d, got: %d", k, 0, len(gotBytes))
		}
		wantBytes := []byte{byte(k >> 24), byte(k >> 16), byte(k >> 8), byte(k & 255)}
		if cap(gotBytes) < len(wantBytes) {
			t.Fatalf("GetBuf(k=%d): cap(buf): want: >= %d, got: %d", k, len(wantBytes), cap(gotBytes))
		}
		gotBytes = gotBytes[:len(wantBytes)]
		if !bytes.Equal(wantBytes, gotBytes) {
			t.Fatalf("GetBuf(k=%d): content: want: %v, got: %v", k, wantBytes, gotBytes)
		}
	}
}

func TestBufPoolGetReturn(t *testing.T) {
	for _, maxPoolSize := range []int{
		0,
		7,
	} {
		t.Run(
			fmt.Sprintf("maxPoolSize=%d", maxPoolSize),
			func(t *testing.T) { testBufPoolGetReturn(t, maxPoolSize) },
		)
	}
}

func testReadFileBufPoolReadFile(t *testing.T, maxReadSize int64, filePath string, fileSize int64) {
	t.Logf("maxReadSize: %d, fileSize: %d", maxReadSize, fileSize)
	p := NewReadFileBufPool(0, maxReadSize)
	for k := 0; k < 2; k++ {
		b, err := p.ReadFile(filePath)
		if maxReadSize > 0 && maxReadSize <= fileSize {
			if err != ErrReadFileBufPotentialTruncation {
				t.Fatalf("error: want: %v, got: %v", ErrReadFileBufPotentialTruncation, err)
			}
			if int64(b.Len()) != maxReadSize {
				t.Fatalf("size: want: %d, got: %d",
					maxReadSize, b.Len(),
				)
			}
		} else {
			if err != nil {
				t.Fatalf("ReadFile(%s): %v", filePath, err)
			}
			if int64(b.Len()) != fileSize {
				t.Fatalf("ReadFile(%s): size: want: %d, got: %d",
					filePath, fileSize, b.Len(),
				)
			}
		}
		if b != nil {
			p.ReturnBuf(b)
		}
	}
}

func TestReadFileBufPoolReadFile(t *testing.T) {
	filePath := TEST_READ_FILE_BUF_POOL_TEST_FILE
	f, err := os.Open(filePath)
	if err != nil {
		t.Fatal(err)
	}
	fileInfo, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	fileSize := fileInfo.Size()

	for _, maxReadSize := range []int64{
		0,
		fileSize + 1,
		fileSize,
	} {
		t.Run(
			fmt.Sprintf("maxReadSize=%d", maxReadSize),
			func(t *testing.T) { testReadFileBufPoolReadFile(t, maxReadSize, filePath, fileSize) },
		)
	}
}
