package validator

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/theoremus-urban-solutions/netex-validator/testutil"
)

func TestConcurrentValidation_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	t.Run("High concurrency validation", func(t *testing.T) {
		options := testOptions(t).
			WithCodespace(testutil.TestCodespace).
			WithSkipSchema(true).
			WithConcurrentFiles(4)

		validator, err := NewWithOptions(options)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		const numGoroutines = 50
		const validationsPerGoroutine = 10

		var wg sync.WaitGroup
		var successCount int64
		var errorCount int64
		results := make(chan time.Duration, numGoroutines*validationsPerGoroutine)

		startTime := time.Now()

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < validationsPerGoroutine; j++ {
					// Create unique content for each validation
					content := strings.ReplaceAll(testutil.NetEXTestFragment,
						"TEST:Line:1", fmt.Sprintf("TEST:Line:%d_%d", workerID, j))

					validationStart := time.Now()
					result, err := validator.ValidateContent([]byte(content),
						fmt.Sprintf("stress_test_%d_%d.xml", workerID, j))
					duration := time.Since(validationStart)

					if err != nil {
						atomic.AddInt64(&errorCount, 1)
						t.Logf("Validation error in worker %d: %v", workerID, err)
					} else {
						atomic.AddInt64(&successCount, 1)
						results <- duration

						// Log occasionally to track progress
						if j == 0 {
							t.Logf("Worker %d: first validation completed in %v (%d issues)",
								workerID, duration, len(result.ValidationReportEntries))
						}
					}
				}
			}(i)
		}

		wg.Wait()
		close(results)
		totalTime := time.Since(startTime)

		// Collect timing statistics
		var durations []time.Duration
		for duration := range results {
			durations = append(durations, duration)
		}

		if len(durations) > 0 {
			var total time.Duration
			min := durations[0]
			max := durations[0]

			for _, d := range durations {
				total += d
				if d < min {
					min = d
				}
				if d > max {
					max = d
				}
			}

			avgDuration := total / time.Duration(len(durations))

			t.Logf("Stress test results:")
			t.Logf("  Total time: %v", totalTime)
			t.Logf("  Successful validations: %d", successCount)
			t.Logf("  Failed validations: %d", errorCount)
			t.Logf("  Average validation time: %v", avgDuration)
			t.Logf("  Min validation time: %v", min)
			t.Logf("  Max validation time: %v", max)
			t.Logf("  Validations per second: %.2f", float64(successCount)/totalTime.Seconds())
		}

		if errorCount > 0 {
			t.Errorf("Got %d errors out of %d validations", errorCount, numGoroutines*validationsPerGoroutine)
		}

		expectedValidations := int64(numGoroutines * validationsPerGoroutine)
		if successCount != expectedValidations {
			t.Errorf("Expected %d successful validations, got %d", expectedValidations, successCount)
		}
	})

	t.Run("Memory usage under load", func(t *testing.T) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		options := testOptions(t).
			WithCodespace(testutil.TestCodespace).
			WithSkipSchema(true)

		validator, err := NewWithOptions(options)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		const numValidations = 100
		var wg sync.WaitGroup

		for i := 0; i < numValidations; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				content := createLargeNetEXContent(testutil.NetEXTestFragment, 5)
				_, _ = validator.ValidateContent([]byte(content), fmt.Sprintf("memory_test_%d.xml", id))
			}(i)
		}

		wg.Wait()
		runtime.GC()
		runtime.ReadMemStats(&m2)

		memoryUsed := m2.Alloc - m1.Alloc
		memoryAllocated := m2.TotalAlloc - m1.TotalAlloc

		t.Logf("Memory usage:")
		t.Logf("  Memory used: %d KB", memoryUsed/1024)
		t.Logf("  Memory allocated: %d KB", memoryAllocated/1024)
		t.Logf("  GC cycles: %d", m2.NumGC-m1.NumGC)

		// Check for potential memory leaks (very rough heuristic)
		if memoryUsed > 100*1024*1024 { // 100MB
			t.Logf("Warning: High memory usage detected: %d MB", memoryUsed/(1024*1024))
		}
	})
}

func TestConcurrentValidation_DeadlockDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping deadlock test in short mode")
	}

	t.Run("Deadlock prevention", func(t *testing.T) {
		options := testOptions(t).
			WithCodespace(testutil.TestCodespace).
			WithSkipSchema(true).
			WithValidationCache(true, 100, 10, 1).
			WithConcurrentFiles(2)

		validator, err := NewWithOptions(options)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		// Create a scenario that could potentially cause deadlocks
		// by having multiple goroutines access shared resources
		const numWorkers = 10
		const opsPerWorker = 20

		var wg sync.WaitGroup
		timeout := time.After(30 * time.Second)
		done := make(chan bool, 1)

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < opsPerWorker; j++ {
					// Mix of different operations that could cause contention
					switch j % 4 {
					case 0:
						// Validation with caching
						content := testutil.NetEXTestFragment
						_, _ = validator.ValidateContent([]byte(content), fmt.Sprintf("deadlock_test_%d.xml", workerID))
					case 1:
						// Validation with different content
						content := strings.ReplaceAll(testutil.NetEXTestFragment, "TEST:Line:1", fmt.Sprintf("TEST:Line:%d", workerID))
						_, _ = validator.ValidateContent([]byte(content), fmt.Sprintf("deadlock_test_var_%d.xml", workerID))
					case 2:
						// Small delay to change timing
						time.Sleep(1 * time.Millisecond)
						_, _ = validator.ValidateContent([]byte(testutil.NetEXTestFragment), fmt.Sprintf("deadlock_test_delayed_%d.xml", workerID))
					case 3:
						// Quick validation
						_, _ = validator.ValidateContent([]byte(testutil.NetEXTestFragment), fmt.Sprintf("deadlock_test_quick_%d.xml", workerID))
					}
				}
			}(i)
		}

		go func() {
			wg.Wait()
			done <- true
		}()

		select {
		case <-done:
			t.Log("Deadlock test completed successfully - no deadlocks detected")
		case <-timeout:
			t.Fatal("Deadlock test timed out - possible deadlock detected")
		}
	})
}

func TestConcurrentValidation_ResourceContention(t *testing.T) {
	t.Run("Resource contention under load", func(t *testing.T) {
		// Test with limited concurrent files setting
		options := testOptions(t).
			WithCodespace(testutil.TestCodespace).
			WithSkipSchema(true).
			WithConcurrentFiles(2). // Limited concurrency
			WithMaxFindings(50)

		validator, err := NewWithOptions(options)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		const numRequests = 20
		var wg sync.WaitGroup
		startTimes := make([]time.Time, numRequests)
		endTimes := make([]time.Time, numRequests)
		var mu sync.Mutex

		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				mu.Lock()
				startTimes[id] = time.Now()
				mu.Unlock()

				content := createLargeNetEXContent(testutil.NetEXTestFragment, 3)
				_, err := validator.ValidateContent([]byte(content), fmt.Sprintf("contention_test_%d.xml", id))

				mu.Lock()
				endTimes[id] = time.Now()
				mu.Unlock()

				if err != nil {
					t.Logf("Validation %d failed: %v", id, err)
				}
			}(i)
		}

		overallStart := time.Now()
		wg.Wait()
		overallEnd := time.Now()

		// Analyze timing patterns
		var totalDuration time.Duration
		maxDuration := time.Duration(0)
		minDuration := time.Hour

		for i := 0; i < numRequests; i++ {
			if !endTimes[i].IsZero() && !startTimes[i].IsZero() {
				duration := endTimes[i].Sub(startTimes[i])
				totalDuration += duration
				if duration > maxDuration {
					maxDuration = duration
				}
				if duration < minDuration {
					minDuration = duration
				}
			}
		}

		avgDuration := totalDuration / time.Duration(numRequests)
		overallDuration := overallEnd.Sub(overallStart)

		t.Logf("Resource contention test results:")
		t.Logf("  Overall time: %v", overallDuration)
		t.Logf("  Average validation time: %v", avgDuration)
		t.Logf("  Min validation time: %v", minDuration)
		t.Logf("  Max validation time: %v", maxDuration)
		t.Logf("  Concurrent efficiency: %.2f", float64(totalDuration)/float64(overallDuration))

		// If efficiency is close to the concurrency limit, resource contention is working
		expectedEfficiency := float64(2) // With ConcurrentFiles=2
		if efficiency := float64(totalDuration) / float64(overallDuration); efficiency > expectedEfficiency*2 {
			t.Logf("Warning: Low concurrent efficiency (%.2f), may indicate resource contention issues", efficiency)
		}
	})
}

func TestConcurrentValidation_ErrorHandling(t *testing.T) {
	t.Run("Concurrent error handling", func(t *testing.T) {
		options := testOptions(t).
			WithCodespace(testutil.TestCodespace).
			WithSkipSchema(true)

		validator, err := NewWithOptions(options)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		const numWorkers = 10
		var wg sync.WaitGroup
		var successCount, errorCount int64

		// Mix of valid and invalid content
		testCases := []struct {
			name    string
			content string
			isValid bool
		}{
			{"valid", testutil.NetEXTestFragment, true},
			{"malformed_xml", "<invalid>xml<content>", false},
			{"empty", "", false},
			{"invalid_netex", `<?xml version="1.0"?><root></root>`, false},
		}

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j, testCase := range testCases {
					result, err := validator.ValidateContent([]byte(testCase.content),
						fmt.Sprintf("error_test_%d_%d.xml", workerID, j))

					if err != nil {
						atomic.AddInt64(&errorCount, 1)
						if testCase.isValid {
							t.Logf("Unexpected error for valid content in worker %d: %v", workerID, err)
						}
					} else {
						atomic.AddInt64(&successCount, 1)
						if !testCase.isValid && len(result.ValidationReportEntries) == 0 {
							t.Logf("Expected validation issues for invalid content in worker %d", workerID)
						}
					}
				}
			}(i)
		}

		wg.Wait()

		t.Logf("Concurrent error handling results:")
		t.Logf("  Successful validations: %d", successCount)
		t.Logf("  Error validations: %d", errorCount)

		// We expect some errors from malformed content
		if errorCount == 0 {
			t.Log("No validation errors - malformed content might not be properly rejected")
		}
	})
}

func TestPerformanceRegression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	t.Run("Performance baseline", func(t *testing.T) {
		options := testOptions(t).
			WithCodespace(testutil.TestCodespace).
			WithSkipSchema(true)

		validator, err := NewWithOptions(options)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		datasets := []struct {
			name     string
			content  string
			baseline time.Duration // Expected maximum time
		}{
			{"small", testutil.GetBenchmarkData().SmallDataset, 100 * time.Millisecond},
			{"medium", testutil.GetBenchmarkData().MediumDataset, 500 * time.Millisecond},
			{"large", testutil.GetBenchmarkData().LargeDataset, 2 * time.Second},
		}

		for _, dataset := range datasets {
			t.Run(dataset.name, func(t *testing.T) {
				const numRuns = 5
				var totalTime time.Duration

				for i := 0; i < numRuns; i++ {
					start := time.Now()
					result, err := validator.ValidateContent([]byte(dataset.content),
						fmt.Sprintf("%s_perf_%d.xml", dataset.name, i))
					duration := time.Since(start)
					totalTime += duration

					if err != nil {
						t.Fatalf("Performance test failed: %v", err)
					}

					t.Logf("Run %d: %v (%d issues)", i+1, duration, len(result.ValidationReportEntries))
				}

				avgTime := totalTime / numRuns
				t.Logf("Average time for %s dataset: %v", dataset.name, avgTime)

				if avgTime > dataset.baseline {
					t.Logf("Warning: Performance regression detected. Average time %v exceeds baseline %v",
						avgTime, dataset.baseline)
				} else {
					t.Logf("Performance within baseline: %v <= %v", avgTime, dataset.baseline)
				}
			})
		}
	})
}

func BenchmarkConcurrentValidation_Goroutines(b *testing.B) {
	options := testOptions(b).
		WithCodespace(testutil.TestCodespace).
		WithSkipSchema(true)

	validator, err := NewWithOptions(options)
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	content := []byte(testutil.GetBenchmarkData().MediumDataset)

	b.Run("Sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = validator.ValidateContent(content, "benchmark.xml")
		}
	})

	b.Run("Concurrent_2", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = validator.ValidateContent(content, "benchmark.xml")
			}
		})
	})
}

func BenchmarkValidation_DatasetSizes(b *testing.B) {
	options := testOptions(b).
		WithCodespace(testutil.TestCodespace).
		WithSkipSchema(true)

	validator, err := NewWithOptions(options)
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	benchData := testutil.GetBenchmarkData()

	b.Run("Small", func(b *testing.B) {
		content := []byte(benchData.SmallDataset)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = validator.ValidateContent(content, "small.xml")
		}
	})

	b.Run("Medium", func(b *testing.B) {
		content := []byte(benchData.MediumDataset)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = validator.ValidateContent(content, "medium.xml")
		}
	})

	b.Run("Large", func(b *testing.B) {
		content := []byte(benchData.LargeDataset)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = validator.ValidateContent(content, "large.xml")
		}
	})
}

func BenchmarkValidation_RuleComplexity(b *testing.B) {
	options1 := testOptions(b)
	options1.Codespace = testutil.TestCodespace
	options1.SkipSchema = true
	options1.SkipValidators = true
	validator1, err := NewWithOptions(options1)
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	options2 := testOptions(b)
	options2.Codespace = testutil.TestCodespace
	options2.SkipSchema = true
	options2.SkipValidators = false
	validator2, err := NewWithOptions(options2)
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	content := []byte(testutil.GetBenchmarkData().MediumDataset)

	b.Run("NoRules", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = validator1.ValidateContent(content, "norules.xml")
		}
	})

	b.Run("AllRules", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = validator2.ValidateContent(content, "allrules.xml")
		}
	})
}
