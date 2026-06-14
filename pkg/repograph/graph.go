package repograph

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// Graph represents a simple import dependency graph where keys are file paths
// and values are slices of imported package paths.
type Graph map[string][]string

// Build constructs a dependency graph for all Go files under root.
func Build(root string) (Graph, error) {
	g := make(Graph)
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(p, ".go") {
			if imports, e := parseImports(p); e == nil {
				g[p] = imports
			} else {
				return e
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return g, nil
}

func parseImports(path string) ([]string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	imports := []string{}
	for _, imp := range node.Imports {
		impPath := strings.Trim(imp.Path.Value, "\"")
		imports = append(imports, impPath)
	}
	return imports, nil
}
