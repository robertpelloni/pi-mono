package initialmessage

import "strings"

// InitialMessageResult holds the combined initial prompt.
type InitialMessageResult struct {
	InitialMessage *string
	InitialImages  []map[string]interface{}
}

// BuildInitialMessage combines stdin content, @file text, and CLI messages.
func BuildInitialMessage(
	messages []string,
	fileText *string,
	fileImages []map[string]interface{},
	stdinContent *string,
) InitialMessageResult {
	var parts []string

	if stdinContent != nil {
		parts = append(parts, *stdinContent)
	}
	if fileText != nil {
		parts = append(parts, *fileText)
	}
	if len(messages) > 0 {
		parts = append(parts, messages[0])
	}

	var result InitialMessageResult
	if len(parts) > 0 {
		combined := joinNonEmpty(parts, "")
		result.InitialMessage = &combined
	}

	if len(fileImages) > 0 {
		result.InitialImages = fileImages
	}

	return result
}

func joinNonEmpty(parts []string, sep string) string {
	var nonEmpty []string
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	return strings.Join(nonEmpty, sep)
}

