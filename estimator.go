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
	Name             string  // Name of the preset (e.g., "kimi-k2")
	Description      string  // Description of the preset
	intercept        float64 // Regression coefficients
	coefSymbols      float64
	coefLatinLetters float64
	coefLatinExt     float64
	coefDigits       float64
	coefChinese      float64
	coefJapanese     float64
	coefKorean       float64
	coefRussian      float64
	coefArabic       float64
	coefSpaces       float64

	// Sampling configuration
	EnableSampling    bool // Enable sampling mode for long texts
	SamplingThreshold int  // Minimum text length to trigger sampling (default: 10000)
	SamplingSize      int  // Number of characters to sample (default: 1000)
}

// Predefined estimator presets
var (
	// KimiK2Estimator is an estimator trained on Kimi-K2 tokenizer data.
	// Achieves ~8.5% average relative error.
	KimiK2Estimator = &Estimator{
		Name:             "kimi-k2",
		Description:      "Kimi-K2 tokenizer preset (~8.5% avg error)",
		intercept:        0.0,
		coefSymbols:      0.5671194745036742,
		coefLatinLetters: 0.20601617930567592,
		coefLatinExt:     5.87908499852652,
		coefDigits:       0.8030572147361226,
		coefChinese:      0.6627122076124944,
		coefJapanese:     1.0879350533022305,
		coefKorean:       1.0509515625240804,
		coefRussian:      0.5306900990158002,
		coefArabic:       0.6352704975749803,
		coefSpaces:       0.02578661842488973,
	}

	// presets maps preset names to their estimator instances
	presets = map[string]*Estimator{
		"kimi-k2": KimiK2Estimator,
	}
)

// Stats contains detailed character statistics for a text string.
type Stats struct {
	Symbols       int // Count of punctuation and symbols
	LatinLetters  int // Count of ASCII Latin letters (a-z, A-Z)
	LatinExtended int // Count of Latin extended letters (à, ñ, ü, etc.)
	Digits        int // Count of numeric digits (0-9)
	ChineseChars  int // Count of Chinese (CJK) characters
	JapaneseKana  int // Count of Japanese Hiragana and Katakana
	KoreanHangul  int // Count of Korean Hangul
	RussianChars  int // Count of Russian Cyrillic letters
	ArabicChars   int // Count of Arabic characters
	Spaces        int // Count of whitespace characters
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
		coefSymbols:       e.coefSymbols,
		coefLatinLetters:  e.coefLatinLetters,
		coefLatinExt:      e.coefLatinExt,
		coefDigits:        e.coefDigits,
		coefChinese:       e.coefChinese,
		coefJapanese:      e.coefJapanese,
		coefKorean:        e.coefKorean,
		coefRussian:       e.coefRussian,
		coefArabic:        e.coefArabic,
		coefSpaces:        e.coefSpaces,
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
	stats := e.Analyze(text)
	return e.estimateFromStats(stats)
}

// Analyze analyzes the text and returns detailed character statistics.
// This is useful if you want to see the breakdown of character types.
// If EnableSampling is true and text length exceeds SamplingThreshold,
// it will use sampling mode for better performance.
func (e *Estimator) Analyze(text string) Stats {
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
			// Latin letters (ASCII)
			stats.LatinLetters++
		case isLatinExtended(r):
			stats.LatinExtended++
		case unicode.IsDigit(r):
			stats.Digits++
		case isJapaneseKana(r):
			stats.JapaneseKana++
		case isKoreanHangul(r):
			stats.KoreanHangul++
		case isCJK(r):
			stats.ChineseChars++
		case isRussian(r):
			stats.RussianChars++
		case isArabic(r):
			stats.ArabicChars++
		case isEnglishSymbol(r):
			stats.Symbols++
		case unicode.IsSpace(r):
			stats.Spaces++
		default:
			// treat other chars as symbols
			stats.Symbols++
		}
	}

	// prevent too many latin ext
	if adj := (stats.LatinExtended - stats.LatinLetters/15); adj > 0 {
		stats.Symbols += adj
		stats.LatinExtended -= adj
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
			sampledStats.LatinLetters++
		case isLatinExtended(r):
			sampledStats.LatinExtended++
		case unicode.IsDigit(r):
			sampledStats.Digits++
		case isJapaneseKana(r):
			sampledStats.JapaneseKana++
		case isKoreanHangul(r):
			sampledStats.KoreanHangul++
		case isCJK(r):
			sampledStats.ChineseChars++
		case isRussian(r):
			sampledStats.RussianChars++
		case isArabic(r):
			sampledStats.ArabicChars++
		case isEnglishSymbol(r):
			sampledStats.Symbols++
		case unicode.IsSpace(r):
			sampledStats.Spaces++
		default:
			sampledStats.Symbols++
		}
	}

	// Scale up the sampled statistics to the full text length
	scaleFactor := float64(textLen) / float64(sampleSize)

	stats := Stats{
		Symbols:       int(float64(sampledStats.Symbols)*scaleFactor + 0.5),
		LatinLetters:  int(float64(sampledStats.LatinLetters)*scaleFactor + 0.5),
		LatinExtended: int(float64(sampledStats.LatinExtended)*scaleFactor + 0.5),
		Digits:        int(float64(sampledStats.Digits)*scaleFactor + 0.5),
		ChineseChars:  int(float64(sampledStats.ChineseChars)*scaleFactor + 0.5),
		JapaneseKana:  int(float64(sampledStats.JapaneseKana)*scaleFactor + 0.5),
		KoreanHangul:  int(float64(sampledStats.KoreanHangul)*scaleFactor + 0.5),
		RussianChars:  int(float64(sampledStats.RussianChars)*scaleFactor + 0.5),
		ArabicChars:   int(float64(sampledStats.ArabicChars)*scaleFactor + 0.5),
		Spaces:        int(float64(sampledStats.Spaces)*scaleFactor + 0.5),
	}

	// prevent too many latin ext
	if adj := (stats.LatinExtended - stats.LatinLetters/15); adj > 0 {
		stats.Symbols += adj
		stats.LatinExtended -= adj
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
		e.coefSymbols*float64(stats.Symbols) +
		e.coefLatinLetters*float64(stats.LatinLetters) +
		e.coefLatinExt*float64(stats.LatinExtended) +
		e.coefDigits*float64(stats.Digits) +
		e.coefChinese*float64(stats.ChineseChars) +
		e.coefJapanese*float64(stats.JapaneseKana) +
		e.coefKorean*float64(stats.KoreanHangul) +
		e.coefRussian*float64(stats.RussianChars) +
		e.coefArabic*float64(stats.ArabicChars) +
		e.coefSpaces*float64(stats.Spaces)
}

// isJapaneseKana checks if a rune is Japanese Hiragana or Katakana.
func isJapaneseKana(r rune) bool {
	return (r >= 0x3040 && r <= 0x309F) || // Hiragana
		(r >= 0x30A0 && r <= 0x30FF) // Katakana
}

// isLatinExtended checks if a rune is a Latin extended letter (non-ASCII Latin).
func isLatinExtended(r rune) bool {
	return (r >= 0x00C0 && r <= 0x00FF) || // Latin-1 Supplement (à, ñ, ü, etc.)
		(r >= 0x0100 && r <= 0x017F) || // Latin Extended-A (ā, ē, œ, etc.)
		(r >= 0x0180 && r <= 0x024F) || // Latin Extended-B
		(r >= 0x1E00 && r <= 0x1EFF) // Latin Extended Additional
}

// isKoreanHangul checks if a rune is Korean Hangul.
func isKoreanHangul(r rune) bool {
	return (r >= 0xAC00 && r <= 0xD7AF) || // Hangul Syllables
		(r >= 0x1100 && r <= 0x11FF) || // Hangul Jamo
		(r >= 0x3130 && r <= 0x318F) || // Hangul Compatibility Jamo
		(r >= 0xA960 && r <= 0xA97F) || // Hangul Jamo Extended-A
		(r >= 0xD7B0 && r <= 0xD7FF) // Hangul Jamo Extended-B
}

// isCJK checks if a rune is a CJK (Chinese) character,
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

// isArabic checks if a rune is an Arabic character.
func isArabic(r rune) bool {
	return (r >= 0x0600 && r <= 0x06FF) || // Arabic
		(r >= 0x0750 && r <= 0x077F) || // Arabic Supplement
		(r >= 0x08A0 && r <= 0x08FF) || // Arabic Extended-A
		(r >= 0xFB50 && r <= 0xFDFF) || // Arabic Presentation Forms-A
		(r >= 0xFE70 && r <= 0xFEFF) // Arabic Presentation Forms-B
}

// isRussian checks if a rune is a Russian Cyrillic character.
func isRussian(r rune) bool {
	return (r >= 0x0400 && r <= 0x04FF) || // Cyrillic
		(r >= 0x0500 && r <= 0x052F) || // Cyrillic Supplement
		(r >= 0x2DE0 && r <= 0x2DFF) || // Cyrillic Extended-A
		(r >= 0xA640 && r <= 0xA69F) || // Cyrillic Extended-B
		(r >= 0x1C80 && r <= 0x1C8F) // Cyrillic Extended-C
}
