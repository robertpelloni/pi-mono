package sourceinfo

// SourceInfo describes where a resource was loaded from.
type SourceInfo struct {
	Path       string `json:"path"`
	Source     string `json:"source"` // "global", "project", "extension"
	Extension  string `json:"extension,omitempty"`
}

// CreateSourceInfo creates a SourceInfo for a given path and source.
func CreateSourceInfo(path, source string) SourceInfo {
	return SourceInfo{
		Path:   path,
		Source: source,
	}
}

// Label returns a human-readable label for the source.
func (si SourceInfo) Label() string {
	if si.Extension != "" {
		return si.Extension + ":" + si.Path
	}
	return si.Source + ":" + si.Path
}
