package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	totalRequests   = 100
	workerCount     = 10
	concurrentTasks = 100
)

type Result struct {
	workerID   int
	taskNum    int
	statusCode int
	status     string
	duration   time.Duration
	err        error
}

func makeRequest(workerID, taskNum int) Result {
	start := time.Now()

	url := "https://api.divar.ir/v8/postlist/w/search"
	body := `{"city_ids":["1"],"pagination_data":{"@type":"type.googleapis.com/post_list.PaginationData","last_post_date":"2026-05-10T13:54:00.065731Z","page":10,"layer_page":10,"search_uid":"7425395b-2c81-45f0-86b6-663397c7ff4b","cumulative_widgets_count":241,"viewed_tokens":"","search_bookmark_info":{"search_hash":"3c08c51403d09b9ea300772577379267","bookmark_state":{},"alert_state":{}},"first_page_viewed_at":"2026-05-10T13:55:19.742595902Z"},"disable_recommendation":false,"map_state":{"camera_info":{"bbox":{}}},"search_data":{"form_data":{"data":{"category":{"str":{"value":"ROOT"}}}},"server_payload":{"@type":"type.googleapis.com/widgets.SearchData.ServerPayload","additional_form_data":{"data":{"sort":{"str":{"value":"sort_date"}}}}}}}`

	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		return Result{workerID: workerID, taskNum: taskNum, err: err, duration: time.Since(start)}
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:146.0) Gecko/20100101 Firefox/146.0")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", "https://divar.ir/")
	req.Header.Set("X-Screen-Size", "921x868")
	req.Header.Set("X-Standard-Divar-Error", "true")
	req.Header.Set("X-Render-Type", "CSR")
	req.Header.Set("traceparent", "00-60fecc7db8f7a9147ad5039aa1f1954e-0779fb2005a4387a-00")
	req.Header.Set("Origin", "https://divar.ir")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("Connection", "keep-alive")
	// req.Header.Set("Cookie", "_ga=GA1.1.1815580441.1681409742; multi-city=tehran%7C; city=tehran; theme=dark; did=c50b3249-36a9-452e-8639-382c28d94d90; _ga_1G1K17N77F=GS2.1.s1766260883$o8$g1$t1766260962$j58$l0$h0; referrer=; cdid=5fb267d7-9dc1-43f3-8a22-24aab51af0d1; ff=%7B%22f%22%3A%7B%22foreigner_payment_enabled%22%3Atrue%2C%22enable_filter_post_count_web%22%3Atrue%2C%22device_fp_enable%22%3Atrue%2C%22enable-places-selector-online-search-web%22%3Atrue%2C%22location-row-v2-in-post-list-enabled-web%22%3Atrue%2C%22chat_message_disabled%22%3Atrue%2C%22web_sentry_sample_rate%22%3A0.2%2C%22enable-screen-size-metric%22%3Atrue%7D%2C%22e%22%3A1778422814590%2C%22r%22%3A1778505614590%7D; csid=")

	req.Header.Set("Cookie", "_ga=GA1.1.1815580441.1681409742; multi-city=tehran%7C; city=tehran; theme=dark; did=c50b3249-36a9-452e-8639-382c28d94d90; _ga_1G1K17N77F=GS2.1.s1766260883$o8$g1$t1766260962$j58$l0$h0; referrer=https%3A%2F%2Fdivar.ir%2Fs%2Ftehran; cdid=5fb267d7-9dc1-43f3-8a22-24aab51af0d1; ff=%7B%22f%22%3A%7B%22foreigner_payment_enabled%22%3Atrue%2C%22enable_filter_post_count_web%22%3Atrue%2C%22device_fp_enable%22%3Atrue%2C%22enable-places-selector-online-search-web%22%3Atrue%2C%22location-row-v2-in-post-list-enabled-web%22%3Atrue%2C%22chat_message_disabled%22%3Atrue%2C%22web_sentry_sample_rate%22%3A0.2%2C%22enable-screen-size-metric%22%3Atrue%7D%2C%22e%22%3A1778506794576%2C%22r%22%3A1778589594576%7D; csid=ddf418af4e77f4a087; sAccessToken=eyJraWQiOiJkLTE3Nzg0MTEzMTkwNjAiLCJ0eXAiOiJKV1QiLCJ2ZXJzaW9uIjoiNCIsImFsZyI6IlJTMjU2In0.eyJpYXQiOjE3Nzg1MDM2MTgsImV4cCI6MTc3ODUwNzIxOCwic3ViIjoiMTRmM2FlNDktMDdhOC00MjViLWJkOWEtNzg5MzJkZmYxMGUzIiwidElkIjoicHVibGljIiwic2Vzc2lvbkhhbmRsZSI6IjdkOWVkOTc2LTE2MmMtNGZlMS1hYWNhLTAwOTJiMWYwYTIxZCIsInJlZnJlc2hUb2tlbkhhc2gxIjoiZDUxNzZlOWU2YzU1NmE5ODYyN2QxYjAyZGRlNmZjZWYwYzYxODNhN2EwNGNiM2UyOTM2MDgyNzZmZmJhYTliMiIsInBhcmVudFJlZnJlc2hUb2tlbkhhc2gxIjpudWxsLCJhbnRpQ3NyZlRva2VuIjpudWxsLCJpc3MiOiJodHRwczovL2FwaS5kaXZhci5pci92OC9hdXRoZW50aWNhdGUiLCJwaG9uZU51bWJlciI6Iis5ODk5MzMzNDU4MjEiLCJzdC1wZXJtIjp7InQiOjE3Nzg1MDM2MTgwMDcsInYiOltdfSwic3Qtcm9sZSI6eyJ0IjoxNzc4NTAzNjE4MDA2LCJ2IjpbXX19.jyzgw_AXiNnCi0uXrlNtizQFI_MbImE3z_y4b-uiqWEqHhI-sHNXVJr05ST6UpppM5OlZr4jqdKjHRWCVUkx7FnarVGCZHBt7R3FB6bOtDnia-_KNuPrtTyHO9tE-e8XBlkyd-iey88qNq8PUOV_E0yMABe6D0lwixmLUH9lsodipc5Qw_pQ8WH25Tr7oWsvdo3XeMRx0b7JlLhj7BHz1xWwgG-X-uX9YTsIE5sweElZsCkjdNvMQ1z70c4XmO0Xa6VCi3KNU9lAy6VbIDjuXGsYxtixTsuvfbMkz9SG1-xgWloC7w47KDn0aYAZNQ7tX5rJ68QQnrFU1dXlCCJC-A; sFrontToken=eyJ1aWQiOiIxNGYzYWU0OS0wN2E4LTQyNWItYmQ5YS03ODkzMmRmZjEwZTMiLCJhdGUiOjE3Nzg1MDcyMTgwMDAsInVwIjp7ImFudGlDc3JmVG9rZW4iOm51bGwsImV4cCI6MTc3ODUwNzIxOCwiaWF0IjoxNzc4NTAzNjE4LCJpc3MiOiJodHRwczovL2FwaS5kaXZhci5pci92OC9hdXRoZW50aWNhdGUiLCJwYXJlbnRSZWZyZXNoVG9rZW5IYXNoMSI6bnVsbCwicGhvbmVOdW1iZXIiOiIrOTg5OTMzMzQ1ODIxIiwicmVmcmVzaFRva2VuSGFzaDEiOiJkNTE3NmU5ZTZjNTU2YTk4NjI3ZDFiMDJkZGU2ZmNlZjBjNjE4M2E3YTA0Y2IzZTI5MzYwODI3NmZmYmFhOWIyIiwic2Vzc2lvbkhhbmRsZSI6IjdkOWVkOTc2LTE2MmMtNGZlMS1hYWNhLTAwOTJiMWYwYTIxZCIsInN0LXBlcm0iOnsidCI6MTc3ODUwMzYxODAwNywidiI6W119LCJzdC1yb2xlIjp7InQiOjE3Nzg1MDM2MTgwMDYsInYiOltdfSwic3ViIjoiMTRmM2FlNDktMDdhOC00MjViLWJkOWEtNzg5MzJkZmYxMGUzIiwidElkIjoicHVibGljIn19; player_id=cdacbe7b-ea9d-47a0-a17d-c2ba5093279b; _vid_t=DTRWUyWsUXYlddPq9SsNBb6FW64Fd405MPiNuH0j5srGWgPI7hZ8yiZ2H87bzwEl+aJzr0iTHW6SfA==")

	req.Header.Set("Priority", "u=0")
	req.Header.Set("TE", "trailers")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return Result{workerID: workerID, taskNum: taskNum, err: err, duration: time.Since(start)}
	}
	defer resp.Body.Close()

	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(resp.Body)
		if err == nil {
			defer gzReader.Close()
			reader = gzReader
		}
	}
	io.Copy(io.Discard, reader) // drain body

	return Result{
		workerID:   workerID,
		taskNum:    taskNum,
		statusCode: resp.StatusCode,
		status:     resp.Status,
		duration:   time.Since(start),
	}
}

func worker(id int, jobs <-chan int, results chan<- Result, sem chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	for taskNum := range jobs {
		sem <- struct{}{} // acquire concurrency slot
		result := makeRequest(id, taskNum)
		<-sem // release concurrency slot
		results <- result
	}
}

func main() {
	jobs := make(chan int, totalRequests)
	results := make(chan Result, totalRequests)
	sem := make(chan struct{}, concurrentTasks) // concurrency limiter

	var wg sync.WaitGroup
	for w := 1; w <= workerCount; w++ {
		wg.Add(1)
		go worker(w, jobs, results, sem, &wg)
	}

	for i := 1; i <= totalRequests; i++ {
		jobs <- i
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect and summarize
	statusCounts := make(map[int]int)
	var totalDuration time.Duration
	var errCount int32
	var completed int32

	fmt.Println("Running 1000 requests with 10 workers / 10 concurrent tasks...")
	fmt.Println("─────────────────────────────────────────────────────────────")

	for r := range results {
		atomic.AddInt32(&completed, 1)
		n := atomic.LoadInt32(&completed)

		if r.err != nil {
			atomic.AddInt32(&errCount, 1)
			fmt.Printf("[%4d] worker=%d  ERROR: %v\n", n, r.workerID, r.err)
		} else {
			statusCounts[r.statusCode]++
			totalDuration += r.duration
			marker := ""
			if r.statusCode == 429 {
				marker = " ⚠ RATE LIMITED"
			} else if r.statusCode >= 500 {
				marker = " ✗ SERVER ERROR"
			}
			fmt.Printf("[%4d] worker=%d  %s  (%dms)%s\n",
				n, r.workerID, r.status, r.duration.Milliseconds(), marker)
		}
	}

	// Summary
	successCount := statusCounts[200]
	rateLimited := statusCounts[429]
	avgMs := int64(0)
	if completed > 0 {
		avgMs = totalDuration.Milliseconds() / int64(completed)
	}

	fmt.Println("\n═════════════════════════════════════════════════════════════")
	fmt.Println("SUMMARY")
	fmt.Println("═════════════════════════════════════════════════════════════")
	fmt.Printf("Total requests : %d\n", totalRequests)
	fmt.Printf("Completed      : %d\n", completed)
	fmt.Printf("Errors         : %d\n", errCount)
	fmt.Printf("Avg latency    : %dms\n", avgMs)
	fmt.Println("\nStatus code breakdown:")
	for code, count := range statusCounts {
		label := ""
		if code == 429 {
			label = " ← RATE LIMIT HIT"
		}
		fmt.Printf("  HTTP %d : %d%s\n", code, count, label)
	}
	fmt.Printf("\n✓ 200 OK       : %d\n", successCount)
	fmt.Printf("⚠ 429 Limited  : %d\n", rateLimited)
}
