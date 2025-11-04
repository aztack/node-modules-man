package utils

import "testing"

func TestHumanizeBytes(t *testing.T) {
    cases := []struct{
        in int64
        want string
    }{
        {0, "0 B"},
        {10, "10 B"},
        {1024, "1.00 KB"},
        {1024*1024, "1.00 MB"},
        {1024*1024*5 + 100, "5.00 MB"},
        {1024*1024*1024, "1.00 GB"},
    }
    for _, c := range cases {
        got := HumanizeBytes(c.in)
        if got != c.want {
            t.Fatalf("HumanizeBytes(%d) = %q; want %q", c.in, got, c.want)
        }
    }
}

