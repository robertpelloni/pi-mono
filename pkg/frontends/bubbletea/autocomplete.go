package bubbletea

import (
    "strings"
  "sort"

    "github.com/badlogic/pi-mono/pkg/slashcommands"
    "github.com/badlogic/pi-mono/pkg/util"
)

// AutocompleteProvider defines an interface for providing completion suggestions.
// Trigger determines whether this provider should be activated for the given
// textarea value and cursor position. Complete returns the list of possible
// completions for the current context.
type AutocompleteProvider interface {
    Trigger(val string, cursorPos int) bool
    Complete(val string, cursorPos int) []string
    Prefix() string
}

// SlashProvider offers completions for slash commands.
type SlashProvider struct {
    registry *slashcommands.Registry
}

func NewSlashProvider(reg *slashcommands.Registry) *SlashProvider {
    return &SlashProvider{registry: reg}
}

func (p *SlashProvider) Trigger(val string, cursorPos int) bool {
    // Trigger on '/' at the beginning of the input or after a space, with no spaces before cursor.
    if strings.HasPrefix(val, "/") && !strings.Contains(val[:cursorPos], " ") {
        return true
    }
    return false
}

func (p *SlashProvider) Complete(val string, cursorPos int) []string {
    prefix := val[1:cursorPos]
    type scored struct {
        val   string
        score int
    }
    scoredList := []scored{}
    if p.registry != nil {
        for _, cmd := range p.registry.ListCommands() {
            if strings.Contains(strings.ToLower(cmd), strings.ToLower(prefix)) {
                // Simple fuzzy scoring: prioritize prefix matches.
                score := 0
                if strings.HasPrefix(strings.ToLower(cmd), strings.ToLower(prefix)) {
                    score = 100
                } else {
                    score = 10
                }
                scoredList = append(scoredList, scored{val: "/" + cmd, score: score})
            }
        }
    }
    // Sort by score descending, then alphabetically.
    sort.SliceStable(scoredList, func(i, j int) bool {
        if scoredList[i].score != scoredList[j].score {
            return scoredList[i].score > scoredList[j].score
        }
        return scoredList[i].val < scoredList[j].val
    })
    completions := make([]string, len(scoredList))
    for i, s := range scoredList {
        completions[i] = s.val
    }
    return completions
}

func (p *SlashProvider) Prefix() string { return "/" }

// FileProvider offers completions for file paths triggered by '@'.
type FileProvider struct{}

func NewFileProvider() *FileProvider { return &FileProvider{} }

func (p *FileProvider) Trigger(val string, cursorPos int) bool {
    // Find the last '@' before cursor and ensure it starts a token.
    lastAt := strings.LastIndex(val[:cursorPos], "@")
    if lastAt != -1 && (lastAt == 0 || val[lastAt-1] == ' ') {
        return true
    }
    return false
}

func (p *FileProvider) Complete(val string, cursorPos int) []string {
    lastAt := strings.LastIndex(val[:cursorPos], "@")
    if lastAt == -1 {
        return nil
    }
    prefix := val[lastAt+1 : cursorPos]
    files := util.ListFilesRecursively(".", 100)
    type scored struct {
        val   string
        score int
    }
    scoredList := []scored{}
    for _, f := range files {
        if strings.Contains(strings.ToLower(f), strings.ToLower(prefix)) {
            // Simple scoring: prioritize prefix matches.
            score := 0
            if strings.HasPrefix(strings.ToLower(f), strings.ToLower(prefix)) {
                score = 100
            } else {
                score = 10
            }
            scoredList = append(scoredList, scored{val: "@" + f, score: score})
        }
    }
    sort.SliceStable(scoredList, func(i, j int) bool {
        if scoredList[i].score != scoredList[j].score {
            return scoredList[i].score > scoredList[j].score
        }
        return scoredList[i].val < scoredList[j].val
    })
    completions := make([]string, len(scoredList))
    for i, s := range scoredList {
        completions[i] = s.val
    }
    return completions
}

func (p *FileProvider) Prefix() string { return "@" }
