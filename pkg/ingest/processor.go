package ingest

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/badlogic/pi-mono/pkg/ast"
	"github.com/badlogic/pi-mono/pkg/memory"
)

var (
	htmlTagRe      = regexp.MustCompile("<[^>]*>")
	markdownLinkRe = regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`)
)

// DataProcessor handles normalization and ingestion of textual data.
type DataProcessor struct {
	KB *memory.KnowledgeBase
}

// NewDataProcessor creates a new DataProcessor with a knowledge base.
func NewDataProcessor(kb *memory.KnowledgeBase) *DataProcessor {
	return &DataProcessor{KB: kb}
}

// Normalize cleans input text: removes HTML tags, strips markdown links, trims whitespace,
// condenses multiple newlines, and normalizes spaces within lines.
func (p *DataProcessor) Normalize(input string) string {
	if strings.Contains(input, "<") && strings.Contains(input, ">") {
		input = htmlTagRe.ReplaceAllString(input, "")
	}
	if strings.Contains(input, "[") && strings.Contains(input, "]") {
		input = markdownLinkRe.ReplaceAllString(input, "$1")
	}
	cleaned := strings.TrimSpace(input)
	lines := strings.Split(cleaned, "\n")
	for i, line := range lines {
		lines[i] = strings.Join(strings.Fields(line), " ")
	}
	cleaned = strings.Join(lines, "\n")
	for strings.Contains(cleaned, "\n\n\n") {
		cleaned = strings.ReplaceAll(cleaned, "\n\n\n", "\n\n")
	}
	return cleaned
}

// IngestText normalizes a text snippet and stores it in the knowledge base.
func (p *DataProcessor) IngestText(title, content string, tags []string, scope memory.KnowledgeScope) error {
	norm := p.Normalize(content)
	entry := &memory.KnowledgeEntry{
		Title:     title,
		Content:   norm,
		Tags:      tags,
		Scope:     scope,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	return p.KB.Store(entry)
}

// ProcessFile reads a file, normalizes its content, and optionally summarizes Go files.
func (p *DataProcessor) ProcessFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".go" {
		if summary, err := ast.SummarizeGoFile(path, data); err == nil {
			return p.Normalize(summary), nil
		}
	}
	return p.Normalize(string(data)), nil
}

// IngestFile processes a file and stores it with the given tags and scope.
func (p *DataProcessor) IngestFile(path string, tags []string, scope memory.KnowledgeScope) error {
	content, err := p.ProcessFile(path)
	if err != nil {
		return err
	}
	title := filepath.Base(path)
	return p.IngestText(title, content, append(tags, "file", filepath.Ext(path)), scope)
}
