package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type Result struct {
	StatusCode int
	Duration   time.Duration
}

func fetch(client *http.Client, url string) Result {
	start := time.Now()

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return Result{StatusCode: 0, Duration: time.Since(start)}
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body) // drain body

	return Result{StatusCode: resp.StatusCode, Duration: time.Since(start)}
}

func main() {
	url := "https://divar.ir" // 👈 change this
	concurrency := 20         // 👈 parallel workers
	totalReqs := 1000         // 👈 total requests to send

	client := &http.Client{Timeout: 10 * time.Second}

	var (
		wg          sync.WaitGroup
		mu          sync.Mutex
		success     int64
		rateLimited int64
		failed      int64
		statusCount = make(map[int]int)
	)

	sem := make(chan struct{}, concurrency)
	start := time.Now()

	for i := 0; i < totalReqs; i++ {
		wg.Add(1)
		sem <- struct{}{}

		go func(reqNum int) {
			defer wg.Done()
			defer func() { <-sem }()

			result := fetch(client, url)

			mu.Lock()
			statusCount[result.StatusCode]++
			mu.Unlock()

			switch {
			case result.StatusCode == 429:
				atomic.AddInt64(&rateLimited, 1)
				fmt.Printf("[%d] ⚠️  429 Rate Limited (%.2fs)\n", reqNum, result.Duration.Seconds())
			case result.StatusCode >= 200 && result.StatusCode < 300:
				atomic.AddInt64(&success, 1)
				fmt.Printf("[%d] ✅ %d OK (%.2fs)\n", reqNum, result.StatusCode, result.Duration.Seconds())
			default:
				atomic.AddInt64(&failed, 1)
				fmt.Printf("[%d] ❌ %d Error (%.2fs)\n", reqNum, result.StatusCode, result.Duration.Seconds())
			}
		}(i + 1)
	}

	wg.Wait()
	elapsed := time.Since(start)

	fmt.Println("\n========== RESULTS ==========")
	fmt.Printf("Total requests  : %d\n", totalReqs)
	fmt.Printf("Concurrency     : %d workers\n", concurrency)
	fmt.Printf("Total time      : %.2fs\n", elapsed.Seconds())
	fmt.Printf("Req/sec         : %.2f\n", float64(totalReqs)/elapsed.Seconds())
	fmt.Println("-----------------------------")
	fmt.Printf("✅ Success       : %d\n", success)
	fmt.Printf("⚠️  Rate limited  : %d\n", rateLimited)
	fmt.Printf("❌ Failed        : %d\n", failed)
	fmt.Println("-----------------------------")
	fmt.Println("Status code breakdown:")
	for code, count := range statusCount {
		fmt.Printf("  HTTP %d : %d requests\n", code, count)
	}
}
