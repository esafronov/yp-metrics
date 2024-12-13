// Package customcheck inludes static analyser for os.exit in main func of main package
package customcheck

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// lookFunctionDeclaration declaration of outer function we check
const lookFunctionDeclaration string = "main"

// lookFunctionName function name we check
const lookFunctionName string = "Exit"

// lookFunctionPackageName function package name we check
const lookFunctionPackageName string = "os"

// lookPackageName package name we check
const lookPackageName string = "main"

var OsExitInMainAnalyzer = &analysis.Analyzer{
	Name: "osexitinmaincheck",
	Doc:  "check for os.exit in main function of main package",
	Run:  run,
}

type Pass struct {
	Fset         *token.FileSet
	Pkg          *types.Package
	TypesInfo    *types.Info
	Files        []*ast.File
	OtherFiles   []string
	IgnoredFiles []string
}

// checkFunClass function checks and reports if funcDecl has os.exit call
func checkFunCall(funcDecl *ast.FuncDecl, pass *analysis.Pass) {
	//iterate over all statements inside function
	for _, stmt := range funcDecl.Body.List {
		if exp, ok := stmt.(*ast.ExprStmt); ok {
			//get function call
			if call, ok := exp.X.(*ast.CallExpr); ok {
				if fun, ok := call.Fun.(*ast.SelectorExpr); ok {
					// get function name
					funcName := fun.Sel.Name
					//by pass not exit calls
					if funcName != lookFunctionName {
						continue
					}
					if pkg, ok := fun.X.(*ast.Ident); ok {
						//get package name
						pkgName := pkg.Name
						//check if it's os package
						if pkgName == lookFunctionPackageName {
							pass.Reportf(stmt.Pos(), "os.exit in main func of main package")
						}
					}
				}
			}
		}
	}
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		//process only main package and skip other
		if file.Name.String() != lookPackageName {
			continue
		}
		// iterate over AST tree
		ast.Inspect(file, func(node ast.Node) bool {
			switch x := node.(type) {
			case *ast.FuncDecl: // function declaration
				//process only main function declarations
				if x.Name.String() != lookFunctionDeclaration {
					return true
				}
				checkFunCall(x, pass)
			}
			return true
		})
	}
	return nil, nil
}
