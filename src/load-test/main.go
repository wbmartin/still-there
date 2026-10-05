package main

import (
	"bufio"
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
	Duration    time.Duration
	Iterations  int
	Concurrency int
	BaseURL     string
	OutputFile  string
	UserName    string
	Password    string
	Insecure    bool
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

type TestResults struct {
	StartTime      time.Time
	EndTime        time.Time
	Config         Config
	OperationStats map[Operation]*Metrics
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

	// One login before any worker starts: a wrong password, a user outside
	// the test tenant or an API that is not there stops the run here
	// instead of filling the report with failures.
	preflight := NewAPIClient(
		config.BaseURL, config.UserName, config.Password, config.Insecure,
	)
	if err := preflight.Login(); err != nil {
		log.Fatalf("Login as %s failed: %v", config.UserName, err)
	}
	if err := preflight.Logout(); err != nil {
		log.Fatalf("Logout after the first login failed: %v", err)
	}

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
	var outputFile string
	var userName, envFile string
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
	flag.StringVar(
		&userName,
		"user",
		"",
		"User to log in as; must be in the test tenant (2)",
	)
	// The password never goes on the command line, where ps and shell
	// history would keep it: it is read from a KEY=VALUE file.
	flag.StringVar(
		&envFile,
		"env",
		"",
		"KEY=VALUE file holding "+passwordKey+" for -user",
	)
	flag.BoolVar(
		&insecure,
		"insecure",
		false,
		"Skip TLS certificate verification (self-signed local cert only)",
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
	if userName == "" {
		log.Fatal("Must specify --user")
	}
	if envFile == "" {
		log.Fatalf("Must specify --env (a file holding %s)", passwordKey)
	}
	password, err := readEnvKey(envFile, passwordKey)
	if err != nil {
		log.Fatal(err)
	}

	return Config{
		Duration:    duration,
		Iterations:  iterations,
		Concurrency: concurrency,
		BaseURL:     baseURL,
		OutputFile:  outputFile,
		UserName:    userName,
		Password:    password,
		Insecure:    insecure,
	}
}

// passwordKey names the -user's password in the -env file.
const passwordKey = "TEST_PERSONA_PASSWORD"

// readEnvKey returns key's value from a flat KEY=VALUE file, read the way
// the API reads its config (src/api-srv/internal/config ReadEnvFile):
// blank and # lines skipped, split at the first =, whitespace trimmed, one
// pair of matching quotes removed. A leading "export " is dropped too, so
// a sourced shell file works. A missing or empty key is an error.
func readEnvKey(path, key string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("cannot read %s: %w", path, err)
	}
	defer file.Close()

	value := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		name, rest, found := strings.Cut(line, "=")
		if !found || strings.TrimSpace(name) != key {
			continue
		}
		value = strings.TrimSpace(rest)
		if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') &&
			value[len(value)-1] == value[0] {
			value = value[1 : len(value)-1]
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("cannot read %s: %w", path, err)
	}
	if value == "" {
		return "", fmt.Errorf("%s not set in %s", key, path)
	}
	return value, nil
}

func runLoadTest(config Config) *TestResults {
	results := &TestResults{
		StartTime:      time.Now(),
		Config:         config,
		OperationStats: make(map[Operation]*Metrics),
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
	// Closed once, by whichever comes first: Ctrl-C or the normal end.
	stopChan := make(chan struct{})
	var stopOnce sync.Once
	stop := func() { stopOnce.Do(func() { close(stopChan) }) }

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
		stop()
	}()

	// Collect metrics. collectorDone closes once the channel is drained,
	// so the report never reads a half-counted result.
	collectorDone := make(chan struct{})
	go func() {
		defer close(collectorDone)
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

	// Wait for test completion (or Ctrl-C)
	if config.Iterations > 0 {
		log.Printf(
			"Waiting for %d iterations to complete across %d workers...",
			config.Iterations*config.Concurrency,
			config.Concurrency,
		)
		wg.Wait()
		log.Printf("All iterations completed. Stopping workers...")
	} else {
		log.Printf(
			"Running load test for duration %v with %d concurrent workers...",
			config.Duration,
			config.Concurrency,
		)
		select {
		case <-time.After(config.Duration):
			log.Printf(
				"Duration %v completed. Stopping workers...",
				config.Duration,
			)
		case <-stopChan:
		}
	}
	stop()
	wg.Wait()
	signal.Stop(sigChan)

	close(metricsChan)
	<-collectorDone
	log.Printf("All workers stopped. Generating report...")

	results.EndTime = time.Now()
	return results
}

func runWorker(
	workerID int,
	config Config,
	metricsChan chan<- OperationMetric,
	stopChan <-chan struct{},
) {
	client := NewAPIClient(
		config.BaseURL, config.UserName, config.Password, config.Insecure,
	)

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

// runCycle is one login, create, update, get, delete, logout, each timed
// on its own. A failed create leaves nothing to update or delete; a failed
// update leaves the row at its create timestamp, which the delete then uses.
func runCycle(client *APIClient, metricsChan chan<- OperationMetric) {
	record := func(operation Operation, start time.Time, err error) {
		metricsChan <- OperationMetric{
			Operation: operation,
			Duration:  time.Since(start),
			Success:   err == nil,
			ErrorType: getErrorType(err),
			Timestamp: time.Now(),
		}
	}

	start := time.Now()
	err := client.Login()
	record(OpLogin, start, err)
	if err != nil {
		log.Printf("Authentication failed (continuing): %v", err)
		return
	}

	start = time.Now()
	exampleID, updateTs, err := client.CreateExample()
	record(OpCreate, start, err)
	if err != nil {
		client.Logout() // Cleanup
		return
	}

	start = time.Now()
	newUpdateTs, err := client.UpdateExample(exampleID, updateTs)
	record(OpUpdate, start, err)
	if err == nil {
		updateTs = newUpdateTs
	}

	start = time.Now()
	err = client.GetExample(exampleID)
	record(OpGet, start, err)

	start = time.Now()
	err = client.DeleteExample(exampleID, updateTs)
	record(OpDelete, start, err)

	start = time.Now()
	err = client.Logout()
	record(OpLogout, start, err)
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

	if results.TotalErrors > 0 {
		fmt.Printf(
			"Errors:             %d (%.1f%%)\n",
			results.TotalErrors,
			float64(results.TotalErrors)/float64(results.TotalRequests)*100,
		)
	}
}
