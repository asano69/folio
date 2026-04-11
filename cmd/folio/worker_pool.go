package main

import (
	"runtime"
	"sync"
)

// runWorkerPool runs fn on each job concurrently using a pool of GOMAXPROCS
// workers and collects the results. Result order is not guaranteed.
func runWorkerPool[J any, R any](jobs []J, fn func(J) R) []R {
	if len(jobs) == 0 {
		return nil
	}

	jobCh := make(chan J, len(jobs))
	for _, j := range jobs {
		jobCh <- j
	}
	close(jobCh)

	resultCh := make(chan R, len(jobs))

	var wg sync.WaitGroup
	for range runtime.GOMAXPROCS(0) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobCh {
				resultCh <- fn(j)
			}
		}()
	}
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	results := make([]R, 0, len(jobs))
	for r := range resultCh {
		results = append(results, r)
	}
	return results
}
