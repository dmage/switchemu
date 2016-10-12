package stat

import (
	"fmt"
	"io"
)

type FastInt64Counter struct {
	first []int64
	rest  map[int64]int64
}

func NewFastInt64Counter(sliceSize int64) FastInt64Counter {
	return FastInt64Counter{
		first: make([]int64, sliceSize),
		rest:  make(map[int64]int64),
	}
}

func (c FastInt64Counter) Add(key int64, value int64) {
	if key < int64(len(c.first)) {
		c.first[key] += value
	} else {
		c.rest[key] += value
	}
}

func (c FastInt64Counter) Increment(key int64) {
	c.Add(key, 1)
}

func (c FastInt64Counter) Dump(filename string) error {
	return CreateFile(filename, func(w io.Writer) error {
		for key, value := range c.first {
			if value != 0 {
				fmt.Fprintf(w, "%d\t%d\n", key, value)
			}
		}
		for key, value := range c.rest {
			fmt.Fprintf(w, "%d\t%d\n", key, value)
		}
		return nil
	})
}
