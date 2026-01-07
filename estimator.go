// Package tokenestimate provides fast token count estimation for text strings.
// It uses linear regression based on character classification to estimate
// token counts without actual tokenization.
package tokenestimate

import (
	"fmt"
	"unicode"
)

// Estimator estimates token counts for text strings using a trained
// linear regression model based on character classification.
type Estimator struct {
	Name           string  // Name of the preset (e.g., "kimi-k2")
	Description    string  // Description of the preset
	intercept      float64 // Regression coefficients
	coefEngSymbols float64
	coefEngLetters float64
	coefDigits     float64
	coefCJK        float64
	coefSpaces     float64
	coefOthers     float64

	// Sampling configuration
	EnableSampling    bool // Enable sampling mode for long texts
	SamplingThreshold int  // Minimum text length to trigger sampling (default: 10000)
	SamplingSize      int  // Number of characters to sample (default: 1000)
}

// Predefined estimator presets
var (
	// KimiK2Estimator is an estimator trained on Kimi-K2 tokenizer data.
	// Achieves ~11% average relative error.
	KimiK2Estimator = &Estimator{
		Name:           "kimi-k2",
		Description:    "Kimi-K2 tokenizer preset (~11% avg error)",
		intercept:      0.0,
		coefEngSymbols: 0.4878931629843917,
		coefEngLetters: 0.2058159462091778,
		coefDigits:     0.7456747100691167,
		coefCJK:        0.5073823638376703,
		coefSpaces:     0.04303848300732736,
		coefOthers:     1.8299151693307378,
	}

	// presets maps preset names to their estimator instances
	presets = map[string]*Estimator{
		"kimi-k2": KimiK2Estimator,
	}
)

// Stats contains detailed character statistics for a text string.
type Stats struct {
	EnglishSymbols int // Count of English punctuation and symbols
	EnglishLetters int // Count of ASCII letters (a-z, A-Z)
	Digits         int // Count of numeric digits (0-9)
	CJKChars       int // Count of CJK (Chinese, Japanese, Korean) characters
	Spaces         int // Count of whitespace characters
	OtherChars     int // Count of other characters
}

// NewEstimator creates a new token count estimator with pre-trained coefficients.
// By default, it returns the Kimi-K2 estimator which achieves ~11% average relative error.
func NewEstimator() *Estimator {
	return KimiK2Estimator
}

// NewEstimatorWithName creates a new estimator using a preset name.
// Returns an error if the preset name is not found.
func NewEstimatorWithName(name string) (*Estimator, error) {
	estimator, ok := presets[name]
	if !ok {
		return nil, fmt.Errorf("unknown preset: %s", name)
	}
	return estimator, nil
}

// ListPresets returns a list of all available preset names.
func ListPresets() []string {
	names := make([]string, 0, len(presets))
	for name := range presets {
		names = append(names, name)
	}
	return names
}

// GetPresetByName returns an estimator preset by name, or an error if not found.
func GetPresetByName(name string) (*Estimator, error) {
	estimator, ok := presets[name]
	if !ok {
		return nil, fmt.Errorf("unknown preset: %s", name)
	}
	return estimator, nil
}

// RegisterPreset allows users to register custom estimator presets.
// If an estimator with the same name already exists, it will be overwritten.
func RegisterPreset(estimator *Estimator) {
	if estimator.Name != "" {
		presets[estimator.Name] = estimator
	}
}

// Clone creates a deep copy of the estimator.
// This is useful when you want to modify a preset without affecting the original.
func (e *Estimator) Clone() *Estimator {
	return &Estimator{
		Name:              e.Name,
		Description:       e.Description,
		intercept:         e.intercept,
		coefEngSymbols:    e.coefEngSymbols,
		coefEngLetters:    e.coefEngLetters,
		coefDigits:        e.coefDigits,
		coefCJK:           e.coefCJK,
		coefSpaces:        e.coefSpaces,
		coefOthers:        e.coefOthers,
		EnableSampling:    e.EnableSampling,
		SamplingThreshold: e.SamplingThreshold,
		SamplingSize:      e.SamplingSize,
	}
}

// WithSampling returns a clone of the estimator with sampling enabled.
// threshold: minimum text length to trigger sampling (e.g., 1000)
// sampleSize: number of characters to sample (e.g., 500)
func (e *Estimator) WithSampling(threshold, sampleSize int) *Estimator {
	clone := e.Clone()
	clone.EnableSampling = true
	clone.SamplingThreshold = threshold
	clone.SamplingSize = sampleSize
	return clone
}

// Estimate returns the estimated token count for the given text.
// This is the main method for quick token estimation.
func (e *Estimator) Estimate(text string) int {
	stats := e.analyze(text)
	return e.estimateFromStats(stats)
}

// analyze analyzes the text and returns detailed character statistics.
// This is useful if you want to see the breakdown of character types.
// If EnableSampling is true and text length exceeds SamplingThreshold,
// it will use sampling mode for better performance.
func (e *Estimator) analyze(text string) Stats {
	// Check if we should use sampling mode
	textLen := len([]rune(text))
	if e.EnableSampling && e.SamplingThreshold > 0 && e.SamplingSize > 0 && textLen > e.SamplingThreshold {
		return e.analyzeSampling(text, textLen)
	}

	// Full analysis mode
	return e.analyzeFull(text)
}

// analyzeFull performs full character-by-character analysis
func (e *Estimator) analyzeFull(text string) Stats {
	stats := Stats{}

	for _, r := range text {
		switch {
		case unicode.IsLetter(r) && r < 128:
			// English letters (ASCII)
			stats.EnglishLetters++
		case unicode.IsDigit(r):
			stats.Digits++
		case isCJK(r):
			stats.CJKChars++
		case isEnglishSymbol(r):
			stats.EnglishSymbols++
		case unicode.IsSpace(r):
			stats.Spaces++
		default:
			stats.OtherChars++
		}
	}

	return stats
}

// analyzeSampling performs sampling-based analysis for long texts
func (e *Estimator) analyzeSampling(text string, textLen int) Stats {
	runes := []rune(text)
	sampleSize := e.SamplingSize
	if sampleSize > textLen {
		sampleSize = textLen
	}

	// Calculate sampling interval
	interval := textLen / sampleSize
	if interval < 1 {
		interval = 1
	}

	// Sample characters evenly distributed across the text
	sampledStats := Stats{}
	for i := 0; i < sampleSize && i*interval < textLen; i++ {
		r := runes[i*interval]

		switch {
		case unicode.IsLetter(r) && r < 128:
			sampledStats.EnglishLetters++
		case unicode.IsDigit(r):
			sampledStats.Digits++
		case isCJK(r):
			sampledStats.CJKChars++
		case isEnglishSymbol(r):
			sampledStats.EnglishSymbols++
		case unicode.IsSpace(r):
			sampledStats.Spaces++
		default:
			sampledStats.OtherChars++
		}
	}

	// Scale up the sampled statistics to the full text length
	scaleFactor := float64(textLen) / float64(sampleSize)

	stats := Stats{
		EnglishSymbols: int(float64(sampledStats.EnglishSymbols)*scaleFactor + 0.5),
		EnglishLetters: int(float64(sampledStats.EnglishLetters)*scaleFactor + 0.5),
		Digits:         int(float64(sampledStats.Digits)*scaleFactor + 0.5),
		CJKChars:       int(float64(sampledStats.CJKChars)*scaleFactor + 0.5),
		Spaces:         int(float64(sampledStats.Spaces)*scaleFactor + 0.5),
		OtherChars:     int(float64(sampledStats.OtherChars)*scaleFactor + 0.5),
	}

	return stats
}

// estimateFromStats calculates the estimated token count from pre-computed statistics.
// This is useful when you already have the character statistics.
func (e *Estimator) estimateFromStats(stats Stats) int {
	count := e.calculateTokenCount(stats)
	if count < 0 {
		return 0
	}
	return int(count + 0.5) // Round to nearest integer
}

// calculateTokenCount applies the linear regression formula to compute token count.
func (e *Estimator) calculateTokenCount(stats Stats) float64 {
	return e.intercept +
		e.coefEngSymbols*float64(stats.EnglishSymbols) +
		e.coefEngLetters*float64(stats.EnglishLetters) +
		e.coefDigits*float64(stats.Digits) +
		e.coefCJK*float64(stats.CJKChars) +
		e.coefSpaces*float64(stats.Spaces) +
		e.coefOthers*float64(stats.OtherChars)
}

// isCJK checks if a rune is a CJK (Chinese, Japanese, Korean) character.
func isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
		(r >= 0x3400 && r <= 0x4DBF) || // CJK Extension A
		(r >= 0x20000 && r <= 0x2A6DF) || // CJK Extension B
		(r >= 0x2A700 && r <= 0x2B73F) || // CJK Extension C
		(r >= 0x2B740 && r <= 0x2B81F) || // CJK Extension D
		(r >= 0x2B820 && r <= 0x2CEAF) || // CJK Extension E
		(r >= 0x2CEB0 && r <= 0x2EBEF) || // CJK Extension F
		(r >= 0x30000 && r <= 0x3134F) // CJK Extension G
}

// isEnglishSymbol checks if a rune is an ASCII punctuation or symbol.
func isEnglishSymbol(r rune) bool {
	return (r >= 0x21 && r <= 0x2F) || // !"#$%&'()*+,-./
		(r >= 0x3A && r <= 0x40) || // :;<=>?@
		(r >= 0x5B && r <= 0x60) || // [\]^_`
		(r >= 0x7B && r <= 0x7E) // {|}~
}
