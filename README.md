# TokenEstimate

A fast Go library for estimating token counts without actual tokenization.

## Features

- 🚀 **Fast**: Pure Go implementation with zero heap allocations
- 🎯 **Accurate**: ~10% average relative error with zero-intercept model
- 🌏 **Multi-language**: Supports English, Chinese (CJK), and mixed content
- 📊 **Detailed Stats**: Get character breakdown by type
- ⚡ **Sampling Mode**: 2-3x faster for long texts (>10K characters)
- 🎨 **Multiple Presets**: Choose from different model configurations
- 🔧 **Extensible**: Register custom presets with your own coefficients
- 🧪 **Well-tested**: Comprehensive test suite with >15 test cases

## Installation

```bash
go get github.com/infinigence/tokenestimate
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/infinigence/tokenestimate"
)

func main() {
    // Create estimator (uses default kimi-k2 preset)
    estimator := tokenestimate.NewEstimator()

    // Estimate tokens
    text := "Hello, world! 你好世界！"
    tokens := estimator.Estimate(text)
    fmt.Printf("Estimated tokens: %d\n", tokens)
}
```

## Usage Examples

### Basic Estimation

```go
estimator := tokenestimate.NewEstimator()

// Simple text
tokens := estimator.Estimate("Hello, world!")
fmt.Println(tokens) // Output: ~7 tokens
```

### Using Different Presets

```go
// Use default preset (kimi-k2 with zero intercept)
estimator := tokenestimate.NewEstimator()

// Access preset directly
estimator = tokenestimate.KimiK2Estimator

// Get preset by name
estimator, err := tokenestimate.NewEstimatorWithName("kimi-k2")
if err != nil {
    log.Fatal(err)
}

// List all available presets
presets := tokenestimate.ListPresets()
fmt.Println("Available presets:", presets)
```

### Sampling Mode for Long Texts

```go
// Enable sampling for texts longer than 1,000 characters
// Sample 500 characters to estimate
estimator := tokenestimate.NewEstimator().WithSampling(1000, 500)

// This will be 2-3x faster for very long texts
longText := strings.Repeat("Lorem ipsum dolor sit amet...", 1000)
tokens := estimator.Estimate(longText)
```

### Register Custom Preset

```go
// Create a custom estimator with your own coefficients
customEstimator := &tokenestimate.Estimator{
    Name:           "my-tokenizer",
    Description:    "Custom tokenizer model",
    // Set your coefficients here
}

// Register it
tokenestimate.RegisterPreset(customEstimator)

// Use it
estimator, _ := tokenestimate.NewEstimatorWithName("my-tokenizer")
```

### Clone and Modify Estimator

```go
// Clone an existing estimator
original := tokenestimate.KimiK2Estimator
modified := original.Clone()

// Modify without affecting the original
modified.EnableSampling = true
modified.SamplingThreshold = 5000
```

## API Reference

### Creating Estimators

#### `NewEstimator() *Estimator`
Creates a new estimator with the default preset (kimi-k2 with zero intercept).

#### `NewEstimatorWithName(name string) (*Estimator, error)`
Creates an estimator using a named preset. Returns error if preset not found.

#### `GetPresetByName(name string) (*Estimator, error)`
Gets a preset by name without creating a new instance.

#### `ListPresets() []string`
Returns a list of all available preset names.

#### `RegisterPreset(estimator *Estimator)`
Registers a custom preset for later use.

### Estimator Methods

#### `Estimate(text string) int`
Returns the estimated token count for the given text. Main method for token estimation.

#### `Clone() *Estimator`
Creates a deep copy of the estimator.

#### `WithSampling(threshold, sampleSize int) *Estimator`
Returns a clone with sampling mode enabled.
- `threshold`: minimum text length to trigger sampling (e.g., 10000)
- `sampleSize`: number of characters to sample (e.g., 1000)

### Available Presets

| Preset Name | Description | Avg Error | Intercept |
|------------|-------------|-----------|-----------|
| `kimi-k2` | Kimi-K2 tokenizer | ~10% | 0.0 |

### Stats Structure

```go
type Stats struct {
    EnglishSymbols int // Count of English punctuation and symbols
    EnglishLetters int // Count of ASCII letters (a-z, A-Z)
    Digits         int // Count of numeric digits (0-9)
    CJKChars       int // Count of CJK characters
    Spaces         int // Count of whitespace characters
    OtherChars     int // Count of other characters
}
```

## How It Works

The estimator uses a linear regression model trained on actual Kimi-K2 tokenization data.

### Standard Model

1. **Analyzes** the input text and counts different character types:
   - English letters (a-z, A-Z)
   - English symbols/punctuation (!"#$%&'()*+,-./:;<=>?@[\]^_`{|}~)
   - Digits (0-9)
   - CJK characters (Chinese, Japanese, Korean)
   - Spaces (all whitespace)
   - Other characters

2. **Applies** trained coefficients (zero-intercept model):
   ```
   tokens ≈ 0.488 × English Symbols
       + 0.206 × English Letters
       + 0.746 × Digits
       + 0.507 × CJK Characters
       + 0.043 × Spaces
       + 1.830 × Other Characters
   ```

3. **Returns** the estimated token count

### Sampling Mode

For long texts (when enabled):

1. **Converts** text to runes to handle Unicode correctly
2. **Samples** evenly distributed characters across the text
3. **Analyzes** only the sampled characters
4. **Scales up** the statistics proportionally
5. **Applies** the same regression formula

This provides 2-3x speedup with minimal accuracy loss (<20% error).

## Use Cases

- 📝 Pre-validate input length before API calls
- 💰 Estimate costs for language model usage
- 🔒 Implement rate limiting based on token counts
- 📊 Monitor token usage without full tokenization
- ⚡ Fast preliminary checks in high-throughput systems
- 🚀 Process very long documents efficiently with sampling mode

## Advanced Usage

### Custom Coefficients

If you have your own tokenizer and training data:

```go
// Train your own model and get coefficients
customPreset := &tokenestimate.Estimator{
    Name:           "my-model",
    Description:    "My custom tokenizer model",
    intercept:      0.0,  // or your intercept
    coefEngSymbols: 0.5,
    coefEngLetters: 0.2,
    coefDigits:     0.7,
    coefCJK:        0.5,
    coefSpaces:     0.04,
    coefOthers:     1.8,
}

// Register and use
tokenestimate.RegisterPreset(customPreset)
estimator, _ := tokenestimate.NewEstimatorWithName("my-model")
```

### Sampling Configuration

```go
// Small texts - don't sample
estimator := tokenestimate.NewEstimator()

// Medium texts - sample if > 5K chars, use 500 samples
estimator := tokenestimate.NewEstimator().WithSampling(5000, 500)

// Large texts - sample if > 10K chars, use 1000 samples
estimator := tokenestimate.NewEstimator().WithSampling(10000, 1000)

// Very large texts - sample if > 50K chars, use 2000 samples
estimator := tokenestimate.NewEstimator().WithSampling(50000, 2000)
```

## Limitations

- The model is trained on Kimi-K2 tokenizer data and may have different accuracy for other tokenizers
- Very short texts (<100 tokens) have higher relative error (but low absolute error)
- Emoji and special Unicode characters are counted as "other" characters
- Sampling mode introduces additional error (~5-20% depending on sample size)
- Sampling mode has one heap allocation (for rune array)

## License

MIT License - See LICENSE file for details

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
