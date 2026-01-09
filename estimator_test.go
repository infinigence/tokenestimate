package tokenestimate

import (
	"bufio"
	"encoding/json"
	"math"
	"os"
	"testing"
)

const (
	TestDatasetPath = "testset-sample.jsonl"
)

func TestEstimator_Estimate(t *testing.T) {
	estimator := NewEstimator()

	tests := []struct {
		name string
		text string
		min  int // Minimum acceptable token count
		max  int // Maximum acceptable token count
	}{
		{
			name: "Empty string",
			text: "",
			min:  0,
			max:  0,
		},
		{
			name: "Simple English",
			text: "Hello, world!",
			min:  0,
			max:  10,
		},
		{
			name: "Chinese text",
			text: "你好，世界！",
			min:  1,
			max:  10,
		},
		{
			name: "Mixed English and Chinese",
			text: "Hello 世界",
			min:  0,
			max:  10,
		},
		{
			name: "Numbers and symbols",
			text: "Price: $99.99",
			min:  1,
			max:  15,
		},
		{
			name: "Long English text",
			text: "The quick brown fox jumps over the lazy dog. This is a test sentence.",
			min:  5,
			max:  30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := estimator.Estimate(tt.text)
			if result < tt.min || result > tt.max {
				t.Errorf("Estimate(%q) = %d, want between %d and %d",
					tt.text, result, tt.min, tt.max)
			}
		})
	}
}

func TestEstimator_Analyze(t *testing.T) {
	estimator := NewEstimator()

	tests := []struct {
		name     string
		text     string
		expected Stats
	}{
		{
			name:     "Empty string",
			text:     "",
			expected: Stats{},
		},
		{
			name: "English letters only",
			text: "Hello",
			expected: Stats{
				EnglishLetters: 5,
			},
		},
		{
			name: "Mixed characters",
			text: "Hello, 世界! 123",
			expected: Stats{
				EnglishLetters: 5,
				EnglishSymbols: 2, // , and !
				CJKChars:       2,
				Digits:         3,
				Spaces:         2,
			},
		},
		{
			name: "Symbols and spaces",
			text: "!@# $%^",
			expected: Stats{
				EnglishSymbols: 6,
				Spaces:         1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := estimator.Analyze(tt.text)
			if result != tt.expected {
				t.Errorf("Analyze(%q) = %+v, want %+v", tt.text, result, tt.expected)
			}
		})
	}
}

func TestEstimator_EstimateFromStats(t *testing.T) {
	estimator := NewEstimator()

	stats := Stats{
		EnglishLetters: 10,
		Spaces:         2,
		EnglishSymbols: 1,
	}

	result := estimator.estimateFromStats(stats)
	if result < 0 {
		t.Errorf("EstimateFromStats should not return negative values, got %d", result)
	}
}

func BenchmarkEstimator_Estimate(b *testing.B) {
	estimator := NewEstimator()
	text := "This is a benchmark test for token estimation. It contains mixed content: 中文字符，English letters, numbers 12345, and symbols !@#$%."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		estimator.Estimate(text)
	}
}

func BenchmarkEstimator_Analyze(b *testing.B) {
	estimator := NewEstimator()
	text := "This is a benchmark test for character analysis. 这是一个基准测试。"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		estimator.Analyze(text)
	}
}

// TestCase represents a test case from the JSONL test dataset
type TestCase struct {
	TokenCount int    `json:"token_count"`
	Text       string `json:"text"`
}

// TestEstimator_TestDataset tests the estimator against the test dataset
// with a maximum error of 15% or 20 tokens (whichever is larger)
func TestEstimator_TestDataset(t *testing.T) {
	estimator := NewEstimator()

	// Find the test dataset file
	file, err := os.Open(TestDatasetPath)
	if err != nil {
		t.Fatalf("Failed to open test dataset: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	var failedCases []struct {
		line      int
		text      string
		expected  int
		estimated int
		error     float64
	}

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line == "" {
			continue
		}

		var testCase TestCase
		if err := json.Unmarshal([]byte(line), &testCase); err != nil {
			t.Logf("Warning: Failed to parse line %d: %v", lineNum, err)
			continue
		}

		// Skip empty text cases
		if testCase.Text == "" {
			continue
		}

		estimated := estimator.Estimate(testCase.Text)
		expected := testCase.TokenCount

		// Calculate error thresholds
		// Error must not exceed 15% OR 20 tokens (whichever is larger)
		percentError := math.Abs(float64(estimated-expected)) / float64(expected) * 100
		absoluteError := math.Abs(float64(estimated - expected))
		t.Logf("Line %d: expected=%d, estimated=%d, percentError=%.2f%%, absoluteError=%.2f",
			lineNum, expected, estimated, percentError, absoluteError)
		maxPercentThreshold := 15.0
		maxAbsoluteThreshold := 20.0

		// Check if error exceeds both thresholds
		if percentError > maxPercentThreshold && absoluteError > maxAbsoluteThreshold {
			failedCases = append(failedCases, struct {
				line      int
				text      string
				expected  int
				estimated int
				error     float64
			}{
				line:      lineNum,
				text:      testCase.Text,
				expected:  expected,
				estimated: estimated,
				error:     percentError,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("Error reading test dataset: %v", err)
	}

	// Report results
	if len(failedCases) > 0 {
		t.Errorf("Failed %d test cases out of %d:", len(failedCases), lineNum)
		for i, fc := range failedCases {
			if i < 10 { // Show first 10 failures
				textPreview := fc.text
				if len(textPreview) > 100 {
					textPreview = textPreview[:100] + "..."
				}
				t.Logf("  Line %d: expected=%d, estimated=%d, error=%.2f%%, text=%q",
					fc.line, fc.expected, fc.estimated, fc.error, textPreview)
			}
		}
		if len(failedCases) > 10 {
			t.Logf("  ... and %d more failures", len(failedCases)-10)
		}
	} else {
		t.Logf("All %d test cases passed with error ≤ 15%% or ≤ 20 tokens", lineNum)
	}
}

// TestPresetSystem tests the preset system functionality
func TestPresetSystem(t *testing.T) {
	t.Run("NewEstimator returns KimiK2Estimator", func(t *testing.T) {
		estimator := NewEstimator()
		if estimator.Name != "kimi-k2" {
			t.Errorf("Expected default estimator name 'kimi-k2', got %q", estimator.Name)
		}
		if estimator != KimiK2Estimator {
			t.Error("Expected NewEstimator to return KimiK2Estimator")
		}
	})

	t.Run("KimiK2Estimator is accessible", func(t *testing.T) {
		if KimiK2Estimator == nil {
			t.Fatal("KimiK2Estimator should not be nil")
		}
		if KimiK2Estimator.Name != "kimi-k2" {
			t.Errorf("Expected KimiK2Estimator name 'kimi-k2', got %q", KimiK2Estimator.Name)
		}
	})

	t.Run("NewEstimatorWithName valid", func(t *testing.T) {
		estimator, err := NewEstimatorWithName("kimi-k2")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if estimator == nil {
			t.Fatal("Expected non-nil estimator")
		}
		if estimator != KimiK2Estimator {
			t.Error("Expected to get KimiK2Estimator")
		}
	})

	t.Run("NewEstimatorWithName invalid", func(t *testing.T) {
		estimator, err := NewEstimatorWithName("nonexistent")
		if err == nil {
			t.Error("Expected error for nonexistent preset")
		}
		if estimator != nil {
			t.Error("Expected nil estimator for nonexistent preset")
		}
	})

	t.Run("ListPresets", func(t *testing.T) {
		presets := ListPresets()
		if len(presets) == 0 {
			t.Error("Expected at least one preset")
		}
		found := false
		for _, name := range presets {
			if name == "kimi-k2" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected 'kimi-k2' in preset list")
		}
	})

	t.Run("GetPresetByName valid", func(t *testing.T) {
		estimator, err := GetPresetByName("kimi-k2")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if estimator.Name != "kimi-k2" {
			t.Errorf("Expected estimator name 'kimi-k2', got %q", estimator.Name)
		}
		if estimator != KimiK2Estimator {
			t.Error("Expected to get KimiK2Estimator")
		}
	})

	t.Run("GetPresetByName invalid", func(t *testing.T) {
		_, err := GetPresetByName("nonexistent")
		if err == nil {
			t.Error("Expected error for nonexistent preset")
		}
	})

	t.Run("RegisterPreset and retrieve", func(t *testing.T) {
		customEstimator := &Estimator{
			Name:           "custom-test",
			Description:    "Custom test estimator",
			intercept:      1.0,
			coefEngSymbols: 0.5,
			coefEngLetters: 0.3,
			coefDigits:     0.8,
			coefCJK:        0.6,
			coefSpaces:     0.1,
		}
		RegisterPreset(customEstimator)

		// Verify it was registered
		estimator, err := GetPresetByName("custom-test")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if estimator.Name != "custom-test" {
			t.Errorf("Expected estimator name 'custom-test', got %q", estimator.Name)
		}
		if estimator != customEstimator {
			t.Error("Expected to get the same estimator instance")
		}

		// Verify it can be retrieved by NewEstimatorWithName
		estimator2, err := NewEstimatorWithName("custom-test")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if estimator2 != customEstimator {
			t.Error("Expected to get the same estimator instance")
		}
	})

	t.Run("Clone estimator", func(t *testing.T) {
		original := KimiK2Estimator
		cloned := original.Clone()

		if cloned == original {
			t.Error("Clone should return a different instance")
		}
		if cloned.Name != original.Name {
			t.Errorf("Expected cloned name %q, got %q", original.Name, cloned.Name)
		}
		if cloned.Description != original.Description {
			t.Errorf("Expected cloned description %q, got %q", original.Description, cloned.Description)
		}

		// Test that clone produces same results
		testText := "Hello world! 你好世界 123"
		if original.Estimate(testText) != cloned.Estimate(testText) {
			t.Error("Clone should produce same estimation results as original")
		}
	})

	t.Run("WithSampling", func(t *testing.T) {
		original := KimiK2Estimator
		sampled := original.WithSampling(10000, 1000)

		if sampled == original {
			t.Error("WithSampling should return a different instance")
		}
		if !sampled.EnableSampling {
			t.Error("Expected EnableSampling to be true")
		}
		if sampled.SamplingThreshold != 10000 {
			t.Errorf("Expected SamplingThreshold 10000, got %d", sampled.SamplingThreshold)
		}
		if sampled.SamplingSize != 1000 {
			t.Errorf("Expected SamplingSize 1000, got %d", sampled.SamplingSize)
		}
	})
}

// TestSamplingMode tests the sampling mode for long texts
func TestSamplingMode(t *testing.T) {
	t.Run("Short text doesn't trigger sampling", func(t *testing.T) {
		estimator := NewEstimator().WithSampling(10000, 1000)
		shortText := "Hello world! 你好世界 123"

		stats := estimator.Analyze(shortText)
		// Should use full analysis since text is short
		expectedStats := Stats{
			EnglishLetters: 10,
			EnglishSymbols: 1,
			Spaces:         3,
			CJKChars:       4,
			Digits:         3,
		}

		if stats != expectedStats {
			t.Errorf("Expected stats %+v, got %+v", expectedStats, stats)
		}
	})

	t.Run("Long text triggers sampling", func(t *testing.T) {
		estimator := NewEstimator().WithSampling(100, 10)

		// Create a long repetitive text (200 chars)
		longText := ""
		for i := 0; i < 100; i++ {
			longText += "ab"
		}

		stats := estimator.Analyze(longText)

		// With sampling, we should get approximate results
		// All characters are 'a' and 'b', so all should be EnglishLetters
		if stats.EnglishLetters == 0 {
			t.Error("Expected some EnglishLetters in sampled stats")
		}

		// The total should be close to the text length (200)
		total := stats.EnglishLetters + stats.EnglishSymbols + stats.Digits +
			stats.CJKChars + stats.ArabicChars + stats.Spaces

		if total < 180 || total > 220 {
			t.Errorf("Expected total around 200, got %d", total)
		}
	})

	t.Run("Sampling accuracy on mixed text", func(t *testing.T) {
		estimator := NewEstimator().WithSampling(1000, 100)

		// Create a long mixed text (2000 chars: 1000 'a' + 1000 '中')
		longText := ""
		for i := 0; i < 1000; i++ {
			longText += "a"
		}
		for i := 0; i < 1000; i++ {
			longText += "中"
		}

		sampledEstimator := estimator
		fullEstimator := NewEstimator()

		sampledResult := sampledEstimator.Estimate(longText)
		fullResult := fullEstimator.Estimate(longText)

		// Sampled result should be reasonably close to full result
		diff := float64(sampledResult-fullResult) / float64(fullResult) * 100
		if diff < 0 {
			diff = -diff
		}

		if diff > 20.0 {
			t.Errorf("Sampling error too large: %.2f%% (sampled=%d, full=%d)",
				diff, sampledResult, fullResult)
		}
	})

	t.Run("Sampling disabled by default", func(t *testing.T) {
		estimator := NewEstimator()
		if estimator.EnableSampling {
			t.Error("Expected sampling to be disabled by default")
		}
	})
}

func TestEstimator_TestDataset_Sampling(t *testing.T) {
	estimator := NewEstimator().WithSampling(1000, 1000)

	// Find the test dataset file
	file, err := os.Open(TestDatasetPath)
	if err != nil {
		t.Fatalf("Failed to open test dataset: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	var failedCases []struct {
		line      int
		text      string
		expected  int
		estimated int
		error     float64
	}

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line == "" {
			continue
		}

		var testCase TestCase
		if err := json.Unmarshal([]byte(line), &testCase); err != nil {
			t.Logf("Warning: Failed to parse line %d: %v", lineNum, err)
			continue
		}

		// Skip empty text cases
		if testCase.Text == "" {
			continue
		}

		estimated := estimator.Estimate(testCase.Text)
		expected := testCase.TokenCount

		// Calculate error thresholds
		// Error must not exceed 15% OR 20 tokens (whichever is larger)
		percentError := math.Abs(float64(estimated-expected)) / float64(expected) * 100
		absoluteError := math.Abs(float64(estimated - expected))
		t.Logf("Line %d: expected=%d, estimated=%d, percentError=%.2f%%, absoluteError=%.2f",
			lineNum, expected, estimated, percentError, absoluteError)
		maxPercentThreshold := 15.0
		maxAbsoluteThreshold := 20.0

		// Check if error exceeds both thresholds
		if percentError > maxPercentThreshold && absoluteError > maxAbsoluteThreshold {
			failedCases = append(failedCases, struct {
				line      int
				text      string
				expected  int
				estimated int
				error     float64
			}{
				line:      lineNum,
				text:      testCase.Text,
				expected:  expected,
				estimated: estimated,
				error:     percentError,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("Error reading test dataset: %v", err)
	}

	// Report results
	if len(failedCases) > 0 {
		t.Errorf("Failed %d test cases out of %d:", len(failedCases), lineNum)
		for i, fc := range failedCases {
			if i < 10 { // Show first 10 failures
				textPreview := fc.text
				if len(textPreview) > 100 {
					textPreview = textPreview[:100] + "..."
				}
				t.Logf("  Line %d: expected=%d, estimated=%d, error=%.2f%%, text=%q",
					fc.line, fc.expected, fc.estimated, fc.error, textPreview)
			}
		}
		if len(failedCases) > 10 {
			t.Logf("  ... and %d more failures", len(failedCases)-10)
		}
	} else {
		t.Logf("All %d test cases passed with error ≤ 15%% or ≤ 20 tokens", lineNum)
	}
}

// BenchmarkEstimator_TestDataset benchmarks the estimator performance using the test dataset
func BenchmarkEstimator_TestDataset(b *testing.B) {
	estimator := NewEstimator()

	// Load test dataset once
	file, err := os.Open(TestDatasetPath)
	if err != nil {
		b.Fatalf("Failed to open test dataset: %v", err)
	}
	defer file.Close()

	// Read all test cases into memory
	var testCases []TestCase
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var testCase TestCase
		if err := json.Unmarshal([]byte(line), &testCase); err != nil {
			continue
		}

		if testCase.Text != "" {
			testCases = append(testCases, testCase)
		}
	}

	if err := scanner.Err(); err != nil {
		b.Fatalf("Error reading test dataset: %v", err)
	}

	if len(testCases) == 0 {
		b.Fatal("No test cases loaded")
	}

	b.Logf("Loaded %d test cases", len(testCases))

	// Reset timer after setup
	b.ResetTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			estimator.Estimate(tc.Text)
		}
	}
}

// BenchmarkEstimator_TestDatasetAnalyze benchmarks just the Analyze phase using test dataset
func BenchmarkEstimator_TestDatasetAnalyze(b *testing.B) {
	estimator := NewEstimator()

	// Load test dataset once
	file, err := os.Open(TestDatasetPath)
	if err != nil {
		b.Fatalf("Failed to open test dataset: %v", err)
	}
	defer file.Close()

	var testCases []TestCase
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var testCase TestCase
		if err := json.Unmarshal([]byte(line), &testCase); err != nil {
			continue
		}

		if testCase.Text != "" {
			testCases = append(testCases, testCase)
		}
	}

	if err := scanner.Err(); err != nil {
		b.Fatalf("Error reading test dataset: %v", err)
	}

	b.Logf("Loaded %d test cases", len(testCases))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			estimator.Analyze(tc.Text)
		}
	}
}

// BenchmarkEstimator_LongText benchmarks performance on very long texts
func BenchmarkEstimator_LongText(b *testing.B) {
	// Create a long text (100K characters)
	longText := ""
	sampleText := "The quick brown fox jumps over the lazy dog. 快速的棕色狐狸跳过懒狗。1234567890!@#$%^&*()"
	for i := 0; i < 1000; i++ {
		longText += sampleText
	}

	b.Run("FullAnalysis", func(b *testing.B) {
		estimator := NewEstimator()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			estimator.Estimate(longText)
		}
	})

	b.Run("Sampling_1000", func(b *testing.B) {
		estimator := NewEstimator().WithSampling(10000, 1000)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			estimator.Estimate(longText)
		}
	})

	b.Run("Sampling_500", func(b *testing.B) {
		estimator := NewEstimator().WithSampling(10000, 500)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			estimator.Estimate(longText)
		}
	})

	b.Run("Sampling_2000", func(b *testing.B) {
		estimator := NewEstimator().WithSampling(10000, 2000)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			estimator.Estimate(longText)
		}
	})
}

// BenchmarkEstimator_VaryingTextSizes benchmarks performance across different text sizes
func BenchmarkEstimator_VaryingTextSizes(b *testing.B) {
	baseText := "The quick brown fox jumps over the lazy dog. 快速的棕色狐狸跳过懒狗。"

	sizes := []struct {
		name       string
		repetition int
	}{
		{"1K", 10},
		{"10K", 100},
		{"100K", 1000},
		{"1M", 10000},
	}

	for _, size := range sizes {
		text := ""
		for i := 0; i < size.repetition; i++ {
			text += baseText
		}

		b.Run("Full_"+size.name, func(b *testing.B) {
			estimator := NewEstimator()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				estimator.Estimate(text)
			}
		})

		b.Run("Sampled_"+size.name, func(b *testing.B) {
			estimator := NewEstimator().WithSampling(5000, 1000)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				estimator.Estimate(text)
			}
		})
	}
}
