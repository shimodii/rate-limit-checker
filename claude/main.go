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
	req.Header.Set("Cookie", "_ga=GA1.1.1815580441.1681409742; multi-city=tehran%7C; city=tehran; theme=dark; did=c50b3249-36a9-452e-8639-382c28d94d90; _ga_1G1K17N77F=GS2.1.s1766260883$o8$g1$t1766260962$j58$l0$h0; referrer=; cdid=5fb267d7-9dc1-43f3-8a22-24aab51af0d1; ff=%7B%22f%22%3A%7B%22foreigner_payment_enabled%22%3Atrue%2C%22enable_filter_post_count_web%22%3Atrue%2C%22device_fp_enable%22%3Atrue%2C%22enable-places-selector-online-search-web%22%3Atrue%2C%22location-row-v2-in-post-list-enabled-web%22%3Atrue%2C%22chat_message_disabled%22%3Atrue%2C%22web_sentry_sample_rate%22%3A0.2%2C%22enable-screen-size-metric%22%3Atrue%7D%2C%22e%22%3A1778422814590%2C%22r%22%3A1778505614590%7D; csid=")
	req.Header.Set("Priority", "u=0")
	req.Header.Set("TE", "trailers")

	client := &http.Client{Timeout: 15 * time.Second}
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
