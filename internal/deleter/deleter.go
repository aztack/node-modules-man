package deleter

import (
	"context"
	"os"
	"sync"
)

type Target struct {
	Path string
	Size int64
}

type Progress struct {
	Completed int
	Total     int
	Path      string
	Err       error
}

type Failure struct {
	Path string
	Err  error
}

type Summary struct {
	Successes []Target
	Failures  []Failure
	Freed     int64
}

// DeleteTargets deletes all targets concurrently. It sends a Progress update
// for each finished target on the provided progress channel. It returns a
// final Summary when all work is done. The progress channel is not closed here
// (caller may close it after consuming the returned summary if needed).
func DeleteTargets(ctx context.Context, targets []Target, concurrency int, progress chan<- Progress, dryRun bool) Summary {
	if ctx == nil {
		ctx = context.Background()
	}
	if concurrency < 1 {
		concurrency = 1
	}
	total := len(targets)
	type job struct{ t Target }
	jobs := make(chan job)
	var wg sync.WaitGroup

	var mu sync.Mutex
	sum := Summary{}
	completed := 0

	worker := func() {
		defer wg.Done()
		for j := range jobs {
			var err error
			select {
			case <-ctx.Done():
				err = ctx.Err()
			default:
				if dryRun {
					// simulate success without deleting
					err = nil
				} else {
					err = os.RemoveAll(j.t.Path)
				}
			}
			mu.Lock()
			if err != nil {
				sum.Failures = append(sum.Failures, Failure{Path: j.t.Path, Err: err})
			} else {
				sum.Successes = append(sum.Successes, j.t)
				sum.Freed += j.t.Size
			}
			completed++
			if progress != nil {
				// Non-blocking best-effort send; avoid deadlock if receiver slow
				select {
				case progress <- Progress{Completed: completed, Total: total, Path: j.t.Path, Err: err}:
				default:
				}
			}
			mu.Unlock()
		}
	}

	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go worker()
	}
	go func() {
		defer close(jobs)
		for _, t := range targets {
			select {
			case <-ctx.Done():
				return
			case jobs <- job{t: t}:
			}
		}
	}()
	wg.Wait()
	return sum
}
