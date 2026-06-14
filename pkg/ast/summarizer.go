package ast

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// SummarizeGoFile parses a Go source file and returns a summary containing
// package declaration, imports, types, interfaces, constants, variables, and function signatures.
func SummarizeGoFile(filepath string, src []byte) (string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filepath, src, parser.ParseComments)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("package %s\n\n", node.Name.Name))

	var imports []string
	var consts []string
	var vars []string
	var types []string
	var funcs []string

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.GenDecl:
			switch x.Tok {
			case token.IMPORT:
				for _, spec := range x.Specs {
					if is, ok := spec.(*ast.ImportSpec); ok {
						imports = append(imports, is.Path.Value)
					}
				}
			case token.CONST:
				for _, spec := range x.Specs {
					if vs, ok := spec.(*ast.ValueSpec); ok {
						names := []string{}
						for _, name := range vs.Names {
							names = append(names, name.Name)
						}
						typeStr := ""
						if vs.Type != nil {
							typeStr = " " + formatNode(fset, vs.Type)
						}
						consts = append(consts, fmt.Sprintf("const %s%s", strings.Join(names, ", "), typeStr))
					}
				}
			case token.VAR:
				for _, spec := range x.Specs {
					if vs, ok := spec.(*ast.ValueSpec); ok {
						names := []string{}
						for _, name := range vs.Names {
							names = append(names, name.Name)
						}
						typeStr := ""
						if vs.Type != nil {
							typeStr = " " + formatNode(fset, vs.Type)
						}
						vars = append(vars, fmt.Sprintf("var %s%s", strings.Join(names, ", "), typeStr))
					}
				}
			case token.TYPE:
				for _, spec := range x.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok {
						typeStr := formatNode(fset, ts.Type)
						types = append(types, fmt.Sprintf("type %s %s", ts.Name.Name, typeStr))
					}
				}
			}
		case *ast.FuncDecl:
			recv := ""
			if x.Recv != nil && len(x.Recv.List) > 0 {
				recvStr := formatNode(fset, x.Recv.List[0].Type)
				recv = fmt.Sprintf("(%s) ", recvStr)
			}

			params := formatFieldList(fset, x.Type.Params)
			results := formatFieldList(fset, x.Type.Results)

			retStr := ""
			if results != "" {
				if strings.Contains(results, ",") {
					retStr = fmt.Sprintf(" (%s)", results)
				} else {
					retStr = fmt.Sprintf(" %s", results)
				}
			}

			funcs = append(funcs, fmt.Sprintf("func %s%s(%s)%s { ... }", recv, x.Name.Name, params, retStr))
		}
		return true
	})

	if len(imports) > 0 {
		sb.WriteString(fmt.Sprintf("import (%s\n)\n\n", "\n\t"+strings.Join(imports, "\n\t")))
	}

	if len(consts) > 0 {
		for _, c := range consts {
			sb.WriteString(c + "\n")
		}
		sb.WriteString("\n")
	}

	if len(vars) > 0 {
		for _, v := range vars {
			sb.WriteString(v + "\n")
		}
		sb.WriteString("\n")
	}

	if len(types) > 0 {
		for _, t := range types {
			sb.WriteString(t + "\n\n")
		}
	}

	if len(funcs) > 0 {
		for _, f := range funcs {
			sb.WriteString(f + "\n")
		}
	}

	return sb.String(), nil
}

func formatNode(fset *token.FileSet, n ast.Node) string {
	if n == nil {
		return ""
	}
	var buf bytes.Buffer
	ast.Fprint(&buf, fset, n, nil)

	switch x := n.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.StarExpr:
		return "*" + formatNode(fset, x.X)
	case *ast.ArrayType:
		return "[]" + formatNode(fset, x.Elt)
	case *ast.SelectorExpr:
		return formatNode(fset, x.X) + "." + x.Sel.Name
	case *ast.MapType:
		return "map[" + formatNode(fset, x.Key) + "]" + formatNode(fset, x.Value)
	case *ast.ChanType:
		dir := ""
		switch x.Dir {
		case ast.SEND:
			dir = "chan<- "
		case ast.RECV:
			dir = "<-chan "
		default:
			dir = "chan "
		}
		return dir + formatNode(fset, x.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{}"
	default:
		return buf.String()
	}
}

func formatFieldList(fset *token.FileSet, fl *ast.FieldList) string {
	if fl == nil || len(fl.List) == 0 {
		return ""
	}

	var parts []string
	for _, field := range fl.List {
		typeStr := formatNode(fset, field.Type)
		if len(field.Names) == 0 {
			parts = append(parts, typeStr)
		} else {
			names := []string{}
			for _, name := range field.Names {
				names = append(names, name.Name)
			}
			parts = append(parts, strings.Join(names, ", ")+" "+typeStr)
		}
	}
	return strings.Join(parts, ", ")
}