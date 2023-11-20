package xopen

import (
	"compress/bzip2"
	"compress/gzip"
	"errors"
	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
	"io"
	"os"
	"regexp"
)

var (
	pattern = regexp.MustCompile(`^.*(\.zst|\.xz|\.gz|\.bz2|)$`)
)

type readerCloser struct {
	io.Reader
	closers []func() error
}

func (r *readerCloser) Close() error {
	var errs []error
	if r.closers != nil {
		for _, closer := range r.closers {
			if err := closer(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

func openZSTD(filename string) (io.ReadCloser, error) {
	f, err := Open(filename)
	if err != nil {
		return nil, err
	}
	decoder, err := zstd.NewReader(f)
	if err != nil {
		_ = f.Close()
		return nil, err
	}

	return &readerCloser{
		Reader: decoder,
		closers: []func() error{func() error {
			decoder.Close()
			return nil
		}, f.Close},
	}, nil

}
func openXZ(filename string) (io.ReadCloser, error) {
	f, err := Open(filename)
	if err != nil {
		return nil, err
	}
	r, err := xz.NewReader(f)
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	return &readerCloser{
		Reader:  r,
		closers: []func() error{f.Close},
	}, nil
}

func openGZ(filename string) (io.ReadCloser, error) {
	f, err := Open(filename)
	if err != nil {
		return nil, err
	}
	r, err := gzip.NewReader(f)
	if err != nil {
		_ = f.Close()
		return nil, err
	}

	return &readerCloser{Reader: r, closers: []func() error{
		r.Close, f.Close,
	}}, nil
}

func openBZ2(filename string) (io.ReadCloser, error) {
	f, err := Open(filename)
	if err != nil {
		return nil, err
	}
	r := bzip2.NewReader(f)
	return &readerCloser{Reader: r, closers: []func() error{
		f.Close,
	}}, nil
}

// Open opens a file for reading. If the filename ends with .zst, .xz, .gz, .bz2, it will be decompressed automatically.
// If the filename is "-", it will read from stdin.
func Open(filename string) (io.ReadCloser, error) {
	if filename == "-" {
		return io.NopCloser(os.Stdin), nil
	}
	match := pattern.FindStringSubmatch(filename)
	if match == nil {
		return os.Open(filename)
	}
	switch match[1] {
	case ".zst":
		return openZSTD(filename)
	case ".xz":
		return openXZ(filename)
	case ".gz":
		return openGZ(filename)
	case ".bz2":
		return openBZ2(filename)
	default:
		return os.Open(filename)
	}
}
