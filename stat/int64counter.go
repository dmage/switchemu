package stat

import (
	"fmt"
	"io"
)

type Int64Counter map[int64]int64

func NewInt64Counter() Int64Counter {
	return make(Int64Counter)
}

func (c Int64Counter) Add(key int64, value int64) {
	c[key] += value
}

func (c Int64Counter) Increment(key int64) {
	c[key] += 1
}

func (c Int64Counter) Dump(filename string) error {
	return CreateFile(filename, func(w io.Writer) error {
		for key, value := range c {
			fmt.Fprintf(w, "%d\t%d\n", key, value)
		}
		return nil
	})
}
