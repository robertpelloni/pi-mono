package agent

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	negationPrefixes     = []*regexp.Regexp{regexp.MustCompile(`(?i)not\s+`), regexp.MustCompile(`(?i)isn't\s+`), regexp.MustCompile(`(?i)aren't\s+`), regexp.MustCompile(`(?i)won't\s+`), regexp.MustCompile(`(?i)far\s+from\s+`)}
	futurePrefixes       = []*regexp.Regexp{regexp.MustCompile(`(?i)will\s+`), regexp.MustCompile(`(?i)going\s+to\s+`), regexp.MustCompile(`(?i)plan\s+to\s+`), regexp.MustCompile(`(?i)next\s+`), regexp.MustCompile(`(?i)once\s+`), regexp.MustCompile(`(?i)when\s+`)}
	hypotheticalPrefixes = []*regexp.Regexp{regexp.MustCompile(`(?i)if\s+`), regexp.MustCompile(`(?i)assuming\s+`), regexp.MustCompile(`(?i)should\s+`), regexp.MustCompile(`(?i)maybe\s+`), regexp.MustCompile(`(?i)probably\s+`)}
	partialIndicators    = []*regexp.Regexp{regexp.MustCompile(`(?i)partially`), regexp.MustCompile(`(?i)pending`), regexp.MustCompile(`(?i)remaining`), regexp.MustCompile(`(?i)todo`), regexp.MustCompile(`\[\s*\]`)}
	completionPatterns   = []*regexp.Regexp{
		regexp.MustCompile(`(?i)all\s+tasks?\s+(are\s+)?completed?`),
		regexp.MustCompile(`(?i)goal\s+achieved`),
		regexp.MustCompile(`(?i)implementation\s+(is\s+)?complete`),
		regexp.MustCompile(`(?i)nothing\s+(left|remaining)\s+to\s+do`),
		regexp.MustCompile(`(?i)no\s+more\s+tasks?`),
		regexp.MustCompile(`(?i)done\s+with\s+all\s+tasks?`),
		regexp.MustCompile(`(?i)everything\s+is\s+complete`),
		regexp.MustCompile(`(?i)full\s+scope\s+implemented`),
	}
	activeWorkPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)working\s+on`),
		regexp.MustCompile(`(?i)implement(ing|ed)?`),
		regexp.MustCompile(`(?i)fix(ing|ed)?`),
		regexp.MustCompile(`(?i)next\s+task`),
		regexp.MustCompile(`(?i)continu(e|ing)`),
		regexp.MustCompile(`(?i)running\s+tests?`),
		regexp.MustCompile(`(?i)pending`),
	}
	uncertaintyPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)might\s+be\s+done`),
		regexp.MustCompile(`(?i)likely\s+complete`),
		regexp.MustCompile(`(?i)seems\s+complete`),
		regexp.MustCompile(`(?i)probably`),
		regexp.MustCompile(`(?i)maybe`),
	}
)

type ExitResult struct {
	ShouldExit bool
	Reason     string
	Confidence float64
	Reasons    []string
}

type ExitDetector struct {
	failureCount           int
	maxConsecutiveFailures int
}

func NewExitDetector() *ExitDetector {
	return &ExitDetector{
		maxConsecutiveFailures: 5,
	}
}

func (d *ExitDetector) CheckResponse(response string) ExitResult {
	text := strings.TrimSpace(response)
	if text == "" {
		return ExitResult{ShouldExit: false, Confidence: 0, Reasons: []string{"empty response"}}
	}

	sentences := strings.FieldsFunc(text, func(r rune) bool {
		return r == '.' || r == ';' || r == '!' || r == '?' || r == '\n'
	})

	validCompletionSignals := 0
	activeWorkSignals := 0
	uncertaintySignals := 0
	partialSignals := 0

	for _, s := range sentences {
		sentence := strings.TrimSpace(s)
		if sentence == "" {
			continue
		}

		isNegated := matchesAny(sentence, negationPrefixes)
		isFuture := matchesAny(sentence, futurePrefixes)
		isHypothetical := matchesAny(sentence, hypotheticalPrefixes)
		isPartial := matchesAny(sentence, partialIndicators)

		if isPartial {
			partialSignals++
		}

		if matchesAny(sentence, completionPatterns) {
			if !isNegated && !isFuture && !isHypothetical {
				validCompletionSignals++
			}
		}

		if matchesAny(sentence, activeWorkPatterns) {
			activeWorkSignals++
		}

		if matchesAny(sentence, uncertaintyPatterns) {
			uncertaintySignals++
		}
	}

	if strings.Contains(text, "[ ]") {
		partialSignals += 2
	}

	positiveScore := float64(validCompletionSignals) * 0.8
	negativeScore := (float64(activeWorkSignals) * 0.3) + (float64(uncertaintySignals) * 0.2) + (float64(partialSignals) * 0.4)
	confidence := positiveScore - negativeScore
	if confidence < 0 {
		confidence = 0
	} else if confidence > 1 {
		confidence = 1
	}

	shouldExit := validCompletionSignals > 0 && partialSignals == 0 && activeWorkSignals == 0 && confidence >= 0.7

	reasons := []string{
		"validCompletion=" + strings.TrimSpace(fmt.Sprintf("%d", validCompletionSignals)),
		"partial=" + strings.TrimSpace(fmt.Sprintf("%d", partialSignals)),
		"activeWork=" + strings.TrimSpace(fmt.Sprintf("%d", activeWorkSignals)),
		"uncertainty=" + strings.TrimSpace(fmt.Sprintf("%d", uncertaintySignals)),
		"confidence=" + strings.TrimSpace(fmt.Sprintf("%.2f", confidence)),
	}

	if shouldExit {
		return ExitResult{
			ShouldExit: true,
			Reason:     "AI indicated completion",
			Confidence: confidence,
			Reasons:    reasons,
		}
	}

	return ExitResult{
		ShouldExit: false,
		Confidence: confidence,
		Reasons:    reasons,
	}
}

func matchesAny(text string, patterns []*regexp.Regexp) bool {
	for _, p := range patterns {
		if p.MatchString(text) {
			return true
		}
	}
	return false
}

func (d *ExitDetector) ReportFailure() ExitResult {
	d.failureCount++
	if d.failureCount >= d.maxConsecutiveFailures {
		return ExitResult{ShouldExit: true, Reason: "Too many consecutive failures"}
	}
	return ExitResult{ShouldExit: false}
}

func (d *ExitDetector) Reset() {
	d.failureCount = 0
}
