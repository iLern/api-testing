package cmd

import (
	"context"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"io"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"github.com/cucumber/godog"
	"github.com/linuxsuren/api-testing/pkg/render"
	"github.com/spf13/cobra"
)

func createFunctionCmd() (c *cobra.Command) {
	opt := &funcPrinterOption{}
	c = &cobra.Command{
		Use:   "func",
		Short: "Print all the supported functions",
		RunE:  opt.runE,
	}
	flags := c.Flags()
	flags.StringVarP(&opt.feature, "feature", "", "", "The feature query")
	return
}

type funcPrinterOption struct {
	feature string
}

func (o *funcPrinterOption) runE(cmd *cobra.Command, args []string) (err error) {
	if len(args) > 0 {
		name := args[0]
		if fn, ok := render.FuncMap()[name]; ok {
			cmd.Println(reflect.TypeOf(fn))
			desc := FuncDescription(fn)
			if desc != "" {
				cmd.Println(desc)
			}
		} else {
			cmd.Println("No such function")
		}
	} else if o.feature != "" {
		ctx := context.WithValue(cmd.Context(), render.ContextBufferKey, cmd.OutOrStdout())

		suite := godog.TestSuite{
			ScenarioInitializer: initializeScenario,
			Options: &godog.Options{
				Format:              "pretty",
				ShowStepDefinitions: false,
				FeatureContents: []godog.Feature{{
					Name: "test",
					Contents: []byte(fmt.Sprintf(`Feature: title
					Scenario: generate
					  Given %s`, o.feature)),
				}},
				DefaultContext: ctx,
				Output:         io.Discard,
			},
		}

		if suite.Run() != 0 {
			err = fmt.Errorf("failed to query the feature")
		} else {
			cmd.Println()
		}
	} else {
		for name, fn := range render.FuncMap() {
			cmd.Println(name, reflect.TypeOf(fn))
		}
	}
	return
}

func initializeScenario(ctx *godog.ScenarioContext) {
	funcs := render.GetAdvancedFuncs()
	for _, fn := range funcs {
		if fn.GoDogExper != "" && fn.Generator != nil {
			ctx.Step(fn.GoDogExper, fn.Generator)
		}
	}
}

// Get the name and path of a func
func FuncPathAndName(f interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

// Get the name of a func (with package path)
func FuncName(f interface{}) string {
	splitFuncName := strings.Split(FuncPathAndName(f), ".")
	return splitFuncName[len(splitFuncName)-1]
}

// Get description of a func
func FuncDescription(f interface{}) (desc string) {
	fileName, _ := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).FileLine(0)
	funcName := FuncName(f)
	fset := token.NewFileSet()

	// Parse src
	parsedAst, err := parser.ParseFile(fset, fileName, nil, parser.ParseComments)
	if err == nil {
		pkg := &ast.Package{
			Name:  "Any",
			Files: make(map[string]*ast.File),
		}
		pkg.Files[fileName] = parsedAst

		importPath, _ := filepath.Abs("/")
		myDoc := doc.New(pkg, importPath, doc.AllDecls)
		for _, theFunc := range myDoc.Funcs {
			if theFunc.Name == funcName {
				desc = theFunc.Doc
				break
			}
		}
	}
	return
}
