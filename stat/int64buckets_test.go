package stat

import (
	"testing"
	"time"
)

func TestInt64Buckets(t *testing.T) {
	buckets := NewInt64Buckets(5 * time.Millisecond)

	b := buckets.Get(1 * time.Millisecond)
	b.Value = 10

	b = buckets.Get(11 * time.Millisecond)
	b.Value = 20

	if len(buckets.buckets) != 3 {
		t.Fatal("buckets len != 3")
	}
	if buckets.buckets[0].Value != 10 {
		t.Fatal("buckets[0].Value != 10")
	}
	if buckets.buckets[1].Value != 10 {
		t.Fatal("buckets[1].Value != 10")
	}
	if buckets.buckets[2].Value != 20 {
		t.Fatal("buckets[2].Value != 20")
	}
}
