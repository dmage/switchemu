package stat

import (
	"bufio"
	"io"
	"os"
)

const (
	writerBufferSize = 1 << 20
)

type fileWriter struct {
	file *os.File
	w    *bufio.Writer
}

func newFileWriter(filename string) (*fileWriter, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	w := bufio.NewWriterSize(file, writerBufferSize)

	return &fileWriter{
		file: file,
		w:    w,
	}, nil
}

func (d *fileWriter) Close() error {
	err := d.w.Flush()
	if err != nil {
		_ = d.file.Close()
		return err
	}
	return d.file.Close()
}

func CreateFile(filename string, fn func(w io.Writer) error) (err error) {
	var f *fileWriter
	f, err = newFileWriter(filename)
	if err != nil {
		return
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			err = cerr
		}
	}()
	err = fn(f.w)
	return
}
