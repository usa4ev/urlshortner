// Package osexitcheck alerts if there are os.Exit() calls
// in main() function in main package.
package osexitcheck

import (
	"go/ast"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "osExitCheck",
	Doc:  "os.Exit() calls in main() of main.go",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	pkgName := ""

	checkFunc := func(node ast.Node) bool {
		if fcall, ok := node.(*ast.CallExpr); ok {
			if s, ok := fcall.Fun.(*ast.SelectorExpr); ok && s.Sel.Name == "Exit" {
				if x, ok := s.X.(*ast.Ident); ok && x.Name == pkgName {
					pass.Reportf(fcall.Pos(), "using os.Exit is not recommended")
				}
			}
			return false
		}
		return true
	}

	osImported := false

	// check if os imported and if there's a synonym for the import
	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			if importer, ok := node.(*ast.ImportSpec); ok && importer.Path.Value == "\"os\"" {
				if importer.Name == nil {
					pkgName = "os"
				} else {
					pkgName = importer.Name.String()
				}

				osImported = true

				return !osImported
			}

			return !osImported
		})
	}

	// if there is no os package imported, no need to look further
	if !osImported {
		return nil, nil
	}

	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			if fdecl, ok := node.(*ast.FuncDecl); ok && fdecl.Name.String() == "main" {
				ast.Inspect(node, checkFunc)
				return false
			}
			return true
		})
	}

	return nil, nil
}
