package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Config struct {
	Duration      time.Duration
	Iterations    int
	Concurrency   int
	BaseURL       string
	MonitorServer bool
	OutputFile    string
	UserName      string
	Password      string
	Insecure      bool
}

type Operation string

const (
	OpLogin  Operation = "login"
	OpCreate Operation = "create"
	OpUpdate Operation = "update"
	OpGet    Operation = "get"
	OpDelete Operation = "delete"
	OpLogout Operation = "logout"
)

type OperationMetric struct {
	Operation Operation
	Duration  time.Duration
	Success   bool
	ErrorType string
	Timestamp time.Time
}

type Metrics struct {
	Operation Operation
	Count     int
	Success   int
	Durations []time.Duration
	Errors    map[string]int
}

type SystemMetrics struct {
	CPUUsage    []float64
	MemoryUsage []int64 // MB
	Timestamp   time.Time
}

type SystemMonitor struct {
	stopChan chan struct{}
}

func NewSystemMonitor() *SystemMonitor {
	return &SystemMonitor{
		stopChan: make(chan struct{}),
	}
}

func (sm *SystemMonitor) Start(metrics *SystemMetrics) {
	// For now, just collect basic metrics
	// In a real implementation, this would use system monitoring libraries
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-sm.stopChan:
				return
			case <-ticker.C:
				// Placeholder: collect CPU and memory usage
				// This would use gopsutil or similar on real systems
				metrics.CPUUsage = append(
					metrics.CPUUsage,
					50.0,
				) // Placeholder
				metrics.MemoryUsage = append(
					metrics.MemoryUsage,
					256,
				) // Placeholder MB
			}
		}
	}()
}

func (sm *SystemMonitor) Stop() {
	close(sm.stopChan)
}

type TestResults struct {
	StartTime      time.Time
	EndTime        time.Time
	Config         Config
	OperationStats map[Operation]*Metrics
	SystemMetrics  *SystemMetrics
	TotalRequests  int
	TotalErrors    int
}

func main() {
	config := parseFlags()

	// Validate configuration
	if config.Duration == 0 && config.Iterations == 0 {
		log.Fatal("Must specify either --duration or --iterations")
	}
	if config.BaseURL == "" {
		log.Fatal("Must specify --base-url")
	}

	// Generate output filename if not specified
	if config.OutputFile == "" {
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		config.OutputFile = fmt.Sprintf(
			"tmp/test-reports/load_test_%s.html",
			timestamp,
		)
	}

	log.Printf(
		"Starting load test with %d concurrent users",
		config.Concurrency,
	)
	log.Printf(
		"Target: %v duration or %d iterations",
		config.Duration,
		config.Iterations,
	)
	log.Printf("Output: %s", config.OutputFile)

	results := runLoadTest(config)

	// Generate HTML report
	if err := generateHTMLReport(results); err != nil {
		log.Fatalf("Failed to generate HTML report: %v", err)
	}

	log.Printf("Load test completed. Results saved to %s", config.OutputFile)

	// Print summary to console
	printSummary(results)
}

func parseFlags() Config {
	var durationStr string
	var iterations int
	var concurrency int
	var baseURL string
	var monitorServer bool
	var outputFile string
	var userName, password string
	var insecure bool

	flag.StringVar(
		&durationStr,
		"duration",
		"2m",
		"Test duration (e.g., 2m, 30s)",
	)
	flag.IntVar(
		&iterations,
		"iterations",
		0,
		"Number of iterations per user (alternative to duration)",
	)
	flag.IntVar(&concurrency, "concurrency", 100, "Number of concurrent users")
	flag.StringVar(&baseURL, "base-url", "", "Base URL of the API server")
	flag.StringVar(&userName, "user", "admin", "User to log in as (the seeded admin from sql/fixtures)")
	flag.StringVar(&password, "password", "admin123!", "That user's password")
	flag.BoolVar(&insecure, "insecure", false, "Skip TLS certificate verification (self-signed local cert only)")
	flag.BoolVar(
		&monitorServer,
		"monitor-server",
		false,
		"Monitor server CPU and memory usage",
	)
	flag.StringVar(
		&outputFile,
		"output",
		"",
		"Output HTML file (auto-generated if empty)",
	)
	flag.Parse()

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		log.Fatalf("Invalid duration format: %v", err)
	}

	return Config{
		Duration:      duration,
		Iterations:    iterations,
		Concurrency:   concurrency,
		BaseURL:       baseURL,
		MonitorServer: monitorServer,
		OutputFile:    outputFile,
		UserName:      userName,
		Password:      password,
		Insecure:      insecure,
	}
}

func runLoadTest(config Config) *TestResults {
	results := &TestResults{
		StartTime:      time.Now(),
		Config:         config,
		OperationStats: make(map[Operation]*Metrics),
		SystemMetrics:  &SystemMetrics{},
	}

	// Initialize metrics for each operation
	for _, op := range []Operation{
		OpLogin, OpCreate, OpUpdate, OpGet, OpDelete, OpLogout,
	} {
		results.OperationStats[op] = &Metrics{
			Operation: op,
			Errors:    make(map[string]int),
		}
	}

	// Channel for metrics collection
	metricsChan := make(chan OperationMetric, 10000)
	stopChan := make(chan struct{})

	// Start system monitoring if requested
	var systemMonitor *SystemMonitor
	if config.MonitorServer {
		systemMonitor = NewSystemMonitor()
		go systemMonitor.Start(results.SystemMetrics)
		defer systemMonitor.Stop()
	}

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			runWorker(workerID, config, metricsChan, stopChan)
		}(i)
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal, stopping workers...")
		close(stopChan)
	}()

	// Collect metrics
	go func() {
		for metric := range metricsChan {
			stats := results.OperationStats[metric.Operation]
			stats.Count++
			if metric.Success {
				stats.Success++
				stats.Durations = append(stats.Durations, metric.Duration)
			} else {
				stats.Errors[metric.ErrorType]++
				results.TotalErrors++
			}
			results.TotalRequests++
		}
	}()

	// Wait for test completion
	if config.Iterations > 0 {
		log.Printf(
			"Waiting for %d iterations to complete across %d workers...",
			config.Iterations*config.Concurrency,
			config.Concurrency,
		)
		// Wait for all iterations to complete
		wg.Wait()
		log.Printf("All iterations completed. Stopping workers...")
	} else {
		log.Printf("Running load test for duration %v with %d concurrent workers...", config.Duration, config.Concurrency)
		// Wait for duration
		time.Sleep(config.Duration)
		log.Printf("Duration %v completed. Stopping workers...", config.Duration)
	}
	close(stopChan)
	wg.Wait()
	log.Printf("All workers stopped. Generating report...")

	// Close metrics channel
	close(metricsChan)

	results.EndTime = time.Now()
	return results
}

func runWorker(
	workerID int,
	config Config,
	metricsChan chan<- OperationMetric,
	stopChan <-chan struct{},
) {
	client := NewAPIClient(config.BaseURL, config.UserName, config.Password, config.Insecure)

	iteration := 0
	for {
		select {
		case <-stopChan:
			return
		default:
			// Check if we've reached iteration limit
			if config.Iterations > 0 && iteration >= config.Iterations {
				return
			}

			// Run one complete cycle
			runCycle(client, metricsChan)
			iteration++
		}
	}
}

func runCycle(client *APIClient, metricsChan chan<- OperationMetric) {
	// 1. Login
	start := time.Now()
	err := client.Login()
	duration := time.Since(start)

	success := err == nil
	errorType := ""
	if err != nil {
		errorType = err.Error()
		if errorType != "" {
			log.Printf("Authentication failed (continuing): %v", err)

		}
	}

	metricsChan <- OperationMetric{
		Operation: OpLogin,
		Duration:  duration,
		Success:   success,
		ErrorType: errorType,
		Timestamp: time.Now(),
	}

	if !success {
		return
	}

	// 2. Create example
	start = time.Now()
	exampleID, updateTs, err := client.CreateExample()
	duration = time.Since(start)

	metricsChan <- OperationMetric{
		Operation: OpCreate,
		Duration:  duration,
		Success:   err == nil,
		ErrorType: getErrorType(err),
		Timestamp: time.Now(),
	}

	if err != nil {
		client.Logout() // Cleanup
		return
	}

	metricsChan <- OperationMetric{
		Operation: OpLogin,
		Duration:  duration,
		Success:   success,
		ErrorType: errorType,
		Timestamp: time.Now(),
	}

	if !success {
		return
	}

	// 2. Create example
	start = time.Now()
	exampleID, updateTs, err = client.CreateExample()
	duration = time.Since(start)

	metricsChan <- OperationMetric{
		Operation: OpCreate,
		Duration:  duration,
		Success:   err == nil,
		ErrorType: getErrorType(err),
		Timestamp: time.Now(),
	}

	if err != nil {
		client.Logout() // Cleanup
		return
	}

	// 2. Create example
	start = time.Now()
	exampleID, updateTs, err = client.CreateExample()
	duration = time.Since(start)

	metricsChan <- OperationMetric{
		Operation: OpUpdate,
		Duration:  duration,
		Success:   err == nil,
		ErrorType: getErrorType(err),
		Timestamp: time.Now(),
	}

	// 4. Get example
	start = time.Now()
	err = client.GetExample(exampleID)
	duration = time.Since(start)

	metricsChan <- OperationMetric{
		Operation: OpGet,
		Duration:  duration,
		Success:   err == nil,
		ErrorType: getErrorType(err),
		Timestamp: time.Now(),
	}

	// 5. Delete example
	start = time.Now()
	err = client.DeleteExample(exampleID, updateTs)
	duration = time.Since(start)

	metricsChan <- OperationMetric{
		Operation: OpDelete,
		Duration:  duration,
		Success:   err == nil,
		ErrorType: getErrorType(err),
		Timestamp: time.Now(),
	}

	// 6. Logout
	start = time.Now()
	err = client.Logout()
	duration = time.Since(start)

	metricsChan <- OperationMetric{
		Operation: OpLogout,
		Duration:  duration,
		Success:   err == nil,
		ErrorType: getErrorType(err),
		Timestamp: time.Now(),
	}
}

func getErrorType(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

type OperationStatsDisplay struct {
	Operation   Operation
	Count       int
	SuccessRate float64
	AvgMs       float64
	MinMs       float64
	MaxMs       float64
	P95Ms       float64
	P99Ms       float64
}

type SystemMetricsDisplay struct {
	CPUAvg    float64
	CPUMin    float64
	CPUMax    float64
	MemoryAvg int64
	MemoryMin int64
	MemoryMax int64
}

type TemplateData struct {
	StartTime          time.Time
	Duration           time.Duration
	Config             Config
	TotalRequests      int
	TotalErrors        int
	RequestsPerSecond  float64
	SuccessRate        float64
	ErrorRate          float64
	OperationStats     map[Operation]*Metrics
	OperationStatsList []OperationStatsDisplay
	SystemMetrics      SystemMetricsDisplay
}

func generateHTMLReport(results *TestResults) error {
	// Calculate statistics for each operation
	for _, stats := range results.OperationStats {
		if len(stats.Durations) > 0 {
			sort.Slice(stats.Durations, func(i, j int) bool {
				return stats.Durations[i] < stats.Durations[j]
			})
		}
	}

	// Create HTML template
	htmlTemplate := `<!DOCTYPE html>
<html>
<head>
    <title>API Load Test Results - {{.StartTime.Format "2006-01-02 15:04:05"}}</title>
    <style>
        body {
            font-family: 'Courier New', monospace;
            margin: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        .header {
            border-bottom: 2px solid #333;
            padding-bottom: 10px;
            margin-bottom: 20px;
        }
        .summary {
            background: #f8f9fa;
            padding: 15px;
            border-radius: 5px;
            margin-bottom: 20px;
            border-left: 4px solid #007bff;
        }
        table {
            border-collapse: collapse;
            width: 100%;
            margin-bottom: 20px;
        }
        th, td {
            border: 1px solid #ddd;
            padding: 12px;
            text-align: left;
        }
        th {
            background-color: #f8f9fa;
            font-weight: bold;
        }
        tr:nth-child(even) {
            background-color: #f8f9fa;
        }
        .success { color: #28a745; font-weight: bold; }
        .warning { color: #ffc107; font-weight: bold; }
        .error { color: #dc3545; font-weight: bold; }
        .metric { font-family: 'Courier New', monospace; }
        .errors { background: #fff5f5; padding: 15px; border-radius: 5px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>API Load Test Results</h1>
            <p><strong>Timestamp:</strong> {{.StartTime.Format "2006-01-02 15:04:05"}}</p>
            <p><strong>Duration:</strong> {{printf "%.1f" .Duration.Seconds}}s</p>
            <p><strong>Concurrency:</strong> {{.Config.Concurrency}} users</p>
            <p><strong>Test Mode:</strong> {{if .Config.Iterations}}Iterations ({{.Config.Iterations}} per user){{else}}Duration ({{.Config.Duration}}){{end}}</p>
        </div>

        <div class="summary">
            <h2>Overall Summary</h2>
            <p><strong>Total Requests:</strong> {{.TotalRequests}}</p>
            <p><strong>Requests/sec:</strong> {{printf "%.1f" .RequestsPerSecond}}</p>
            <p><strong>Success Rate:</strong>
                <span class="{{if ge .SuccessRate 99.5}}success{{else if ge .SuccessRate 95.0}}warning{{else}}error{{end}}">
                    {{printf "%.1f" .SuccessRate}}%
                </span>
            </p>
            {{if .Config.MonitorServer}}
            <p><strong>Server CPU:</strong> {{printf "%.1f" .SystemMetrics.CPUAvg}}% avg ({{printf "%.1f" .SystemMetrics.CPUMin}}% - {{printf "%.1f" .SystemMetrics.CPUMax}}%)</p>
            <p><strong>Server Memory:</strong> {{.SystemMetrics.MemoryAvg}}MB avg ({{.SystemMetrics.MemoryMin}}MB - {{.SystemMetrics.MemoryMax}}MB)</p>
            {{end}}
        </div>

        <h2>Detailed Metrics</h2>
        <table>
            <tr>
                <th>Operation</th>
                <th>Requests</th>
                <th>Success Rate</th>
                <th>Avg (ms)</th>
                <th>Min (ms)</th>
                <th>Max (ms)</th>
                <th>P95 (ms)</th>
                <th>P99 (ms)</th>
            </tr>
            {{range .OperationStatsList}}
            <tr>
                <td>{{.Operation}}</td>
                <td>{{.Count}}</td>
                <td class="{{if ge .SuccessRate 99.5}}success{{else if ge .SuccessRate 95.0}}warning{{else}}error{{end}}">
                    {{printf "%.1f" .SuccessRate}}%
                </td>
                <td class="metric">{{printf "%.1f" .AvgMs}}</td>
                <td class="metric">{{printf "%.1f" .MinMs}}</td>
                <td class="metric">{{printf "%.1f" .MaxMs}}</td>
                <td class="metric">{{printf "%.1f" .P95Ms}}</td>
                <td class="metric">{{printf "%.1f" .P99Ms}}</td>
            </tr>
            {{end}}
        </table>

        {{if gt .TotalErrors 0}}
        <div class="errors">
            <h2>Errors</h2>
            <p><strong>Total Errors:</strong> {{.TotalErrors}} ({{printf "%.1f" .ErrorRate}}%)</p>
            <ul>
                {{range $op, $stats := .OperationStats}}
                    {{range $error, $count := $stats.Errors}}
                        <li>{{$op}} - {{$error}}: {{$count}}</li>
                    {{end}}
                {{end}}
            </ul>
        </div>
        {{end}}
    </div>
</body>
</html>`

	// Prepare template data
	templateData := TemplateData{
		StartTime:      results.StartTime,
		Duration:       results.EndTime.Sub(results.StartTime),
		Config:         results.Config,
		TotalRequests:  results.TotalRequests,
		TotalErrors:    results.TotalErrors,
		OperationStats: results.OperationStats,
		SystemMetrics:  SystemMetricsDisplay{},
	}

	// Calculate overall metrics
	if templateData.Duration.Seconds() > 0 {
		templateData.RequestsPerSecond = float64(
			results.TotalRequests,
		) / templateData.Duration.Seconds()
	}
	if results.TotalRequests > 0 {
		templateData.SuccessRate = float64(
			results.TotalRequests-results.TotalErrors,
		) / float64(
			results.TotalRequests,
		) * 100
		templateData.ErrorRate = float64(
			results.TotalErrors,
		) / float64(
			results.TotalRequests,
		) * 100
	}

	// Calculate per-operation statistics
	for _, op := range []Operation{OpLogin, OpCreate, OpUpdate, OpGet, OpDelete, OpLogout} {
		stats := results.OperationStats[op]
		display := OperationStatsDisplay{
			Operation: op,
			Count:     stats.Count,
		}

		if stats.Count > 0 {
			display.SuccessRate = float64(
				stats.Success,
			) / float64(
				stats.Count,
			) * 100
		}

		if len(stats.Durations) > 0 {
			// Calculate average
			var total time.Duration
			for _, d := range stats.Durations {
				total += d
			}
			display.AvgMs = float64(
				total.Milliseconds(),
			) / float64(
				len(stats.Durations),
			)

			// Min/Max
			display.MinMs = float64(stats.Durations[0].Milliseconds())
			display.MaxMs = float64(
				stats.Durations[len(stats.Durations)-1].Milliseconds(),
			)

			// Percentiles
			p95Index := int(float64(len(stats.Durations)) * 0.95)
			p99Index := int(float64(len(stats.Durations)) * 0.99)
			if p95Index < len(stats.Durations) {
				display.P95Ms = float64(
					stats.Durations[p95Index].Milliseconds(),
				)
			}
			if p99Index < len(stats.Durations) {
				display.P99Ms = float64(
					stats.Durations[p99Index].Milliseconds(),
				)
			}
		}

		templateData.OperationStatsList = append(
			templateData.OperationStatsList,
			display,
		)
	}

	// Calculate system metrics
	if results.Config.MonitorServer && len(results.SystemMetrics.CPUUsage) > 0 {
		minCPU, maxCPU := results.SystemMetrics.CPUUsage[0], results.SystemMetrics.CPUUsage[0]
		var totalCPU float64
		for _, cpu := range results.SystemMetrics.CPUUsage {
			if cpu < minCPU {
				minCPU = cpu
			}
			if cpu > maxCPU {
				maxCPU = cpu
			}
			totalCPU += cpu
		}
		templateData.SystemMetrics.CPUAvg = totalCPU / float64(
			len(results.SystemMetrics.CPUUsage),
		)
		templateData.SystemMetrics.CPUMin = minCPU
		templateData.SystemMetrics.CPUMax = maxCPU
	}

	if results.Config.MonitorServer &&
		len(results.SystemMetrics.MemoryUsage) > 0 {
		minMem, maxMem := results.SystemMetrics.MemoryUsage[0], results.SystemMetrics.MemoryUsage[0]
		var totalMem int64
		for _, mem := range results.SystemMetrics.MemoryUsage {
			if mem < minMem {
				minMem = mem
			}
			if mem > maxMem {
				maxMem = mem
			}
			totalMem += mem
		}
		templateData.SystemMetrics.MemoryAvg = totalMem / int64(
			len(results.SystemMetrics.MemoryUsage),
		)
		templateData.SystemMetrics.MemoryMin = minMem
		templateData.SystemMetrics.MemoryMax = maxMem
	}

	// A bare file name lands in tmp/test-reports (bin/330 passes a full path)
	reportsDir := "tmp/test-reports"
	if !strings.Contains(results.Config.OutputFile, "/") &&
		!strings.Contains(results.Config.OutputFile, "\\") {
		results.Config.OutputFile = filepath.Join(
			reportsDir,
			results.Config.OutputFile,
		)
	}
	if err := os.MkdirAll(filepath.Dir(results.Config.OutputFile), 0o755); err != nil {
		return fmt.Errorf("failed to create report directory: %w", err)
	}

	// Generate HTML
	tmpl, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	file, err := os.Create(results.Config.OutputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, templateData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

func printSummary(results *TestResults) {
	duration := results.EndTime.Sub(results.StartTime)
	requestsPerSecond := float64(results.TotalRequests) / duration.Seconds()
	successRate := float64(
		results.TotalRequests-results.TotalErrors,
	) / float64(
		results.TotalRequests,
	) * 100

	fmt.Printf("\nLoad Test Summary:\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Duration:           %.1fs\n", duration.Seconds())
	fmt.Printf("Total Requests:     %d\n", results.TotalRequests)
	fmt.Printf("Requests/sec:       %.1f\n", requestsPerSecond)
	fmt.Printf("Success Rate:       %.1f%%\n", successRate)
	fmt.Printf("Concurrency:        %d users\n", results.Config.Concurrency)

	if results.Config.MonitorServer {
		if len(results.SystemMetrics.CPUUsage) > 0 {
			var minCPU, maxCPU float64 = results.SystemMetrics.CPUUsage[0], results.SystemMetrics.CPUUsage[0]
			var totalCPU float64
			for _, cpu := range results.SystemMetrics.CPUUsage {
				if cpu < minCPU {
					minCPU = cpu
				}
				if cpu > maxCPU {
					maxCPU = cpu
				}
				totalCPU += cpu
			}
			avgCPU := totalCPU / float64(len(results.SystemMetrics.CPUUsage))
			fmt.Printf(
				"Server CPU:         %.1f%% avg (%.1f%% - %.1f%%)\n",
				avgCPU,
				minCPU,
				maxCPU,
			)
		}

		if len(results.SystemMetrics.MemoryUsage) > 0 {
			minMem, maxMem := results.SystemMetrics.MemoryUsage[0], results.SystemMetrics.MemoryUsage[0]
			var totalMem int64
			for _, mem := range results.SystemMetrics.MemoryUsage {
				if mem < minMem {
					minMem = mem
				}
				if mem > maxMem {
					maxMem = mem
				}
				totalMem += mem
			}
			avgMem := totalMem / int64(len(results.SystemMetrics.MemoryUsage))
			fmt.Printf(
				"Server Memory:      %dMB avg (%dMB - %dMB)\n",
				avgMem,
				minMem,
				maxMem,
			)
		}
	}

	if results.TotalErrors > 0 {
		fmt.Printf(
			"Errors:             %d (%.1f%%)\n",
			results.TotalErrors,
			float64(results.TotalErrors)/float64(results.TotalRequests)*100,
		)
	}
}
