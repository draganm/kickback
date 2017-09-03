package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	reactor "github.com/draganm/go-reactor"

	"gopkg.in/urfave/cli.v2"
)

func main() {
	app := &cli.App{

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "p",
				Aliases: []string{"package"},
			},
		},

		Action: func(c *cli.Context) error {

			pkg := c.String("package")
			if pkg == "" {
				pkg = "main"
			}

			files := []string{}
			err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return fmt.Errorf("Error listing %s: %s", path, err)
				}
				if strings.HasSuffix(path, ".xml") && !info.IsDir() {
					files = append(files, path)
				}
				return nil
			})

			if err != nil {
				return err
			}

			dms := map[string]*reactor.DisplayModel{}

			for _, f := range files {
				name, dm, e := parseDisplayModel(f)
				if e != nil {
					return fmt.Errorf("Error parsing %s: %s", f, e)
				}
				dms[name] = dm
			}

			fs := token.NewFileSet()

			decls := []ast.Decl{
				&ast.GenDecl{
					Tok: token.IMPORT,
					Specs: []ast.Spec{
						&ast.ImportSpec{
							Name: &ast.Ident{Name: "reactor"},
							Path: &ast.BasicLit{
								Kind:  token.STRING,
								Value: `"github.com/draganm/go-reactor"`,
							},
						},
					},
				},
			}

			modelNames := []string{}

			for n := range dms {
				modelNames = append(modelNames, n)
			}

			for _, n := range modelNames {
				dm := dms[n]
				decls = append(decls, modelDecl(n, dm))
			}

			f := &ast.File{
				Name:  &ast.Ident{Name: pkg},
				Decls: decls,
			}
			if err != nil {
				return err
			}

			buf := &bytes.Buffer{}

			err = printer.Fprint(buf, fs, f)
			if err != nil {
				return err
			}

			err = ioutil.WriteFile("kickback-generated.go", buf.Bytes(), 0770)
			if err != nil {
				return err
			}

			return nil
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func modelDecl(name string, dm *reactor.DisplayModel) ast.Decl {
	return &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{
					&ast.Ident{Name: name},
				},
				Values: []ast.Expr{
					displayModelToAST(dm),
				},
			},
		},
	}

}

func parseDisplayModel(fileName string) (string, *reactor.DisplayModel, error) {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return "", nil, err
	}

	dm, err := reactor.ParseDisplayModel(string(data))
	if err != nil {
		return "", nil, err
	}
	name := strings.TrimSuffix(filepath.Base(fileName), ".xml")

	return name, dm, nil
}

func displayModelToAST(dm *reactor.DisplayModel) *ast.UnaryExpr {
	elts := []ast.Expr{}
	if dm.ID != "" {
		elts = append(elts, stringKeyValueAssignmentAST("ID", dm.ID))
	}

	if dm.Element != "" {
		elts = append(elts, stringKeyValueAssignmentAST("Element", dm.Element))
	}

	if dm.Text != "" {
		elts = append(elts, stringKeyValueAssignmentAST("Text", dm.Text))
	}

	if dm.Attributes != nil {
		elts = append(elts, attributesMapAST(dm.Attributes))
	}

	if dm.ReportEvents != nil {
		elts = append(elts, reportEventsSliceAST(dm.ReportEvents))
	}

	if dm.Children != nil {
		elts = append(elts, childrenSliceAST(dm.Children))
	}

	ast := &ast.UnaryExpr{
		Op: token.AND,
		X: &ast.CompositeLit{
			Type: &ast.SelectorExpr{
				X: &ast.Ident{
					Name: "reactor",
				},
				Sel: &ast.Ident{
					Name: "DisplayModel",
				},
			},
			Elts: elts,
		},
	}
	return ast
}

func attributesMapAST(m map[string]interface{}) *ast.KeyValueExpr {
	elts := []ast.Expr{}

	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := m[k]
		elts = append(
			elts,
			&ast.KeyValueExpr{
				Key:   &ast.Ident{Name: fmt.Sprintf("%#v", k)},
				Value: &ast.Ident{Name: fmt.Sprintf("%#v", v)},
			})
	}
	exp := &ast.KeyValueExpr{
		Key: &ast.Ident{Name: "Attributes"},
		Value: &ast.CompositeLit{
			Type: &ast.MapType{
				Key: &ast.Ident{Name: "string"},
				Value: &ast.InterfaceType{
					Methods: &ast.FieldList{},
				},
			},
			Elts: elts,
		},
	}
	return exp
}

func stringKeyValueAssignmentAST(key string, value interface{}) *ast.KeyValueExpr {
	return &ast.KeyValueExpr{
		Key: &ast.Ident{Name: key},
		Value: &ast.BasicLit{
			Kind:  token.STRING,
			Value: fmt.Sprintf("%#v", value),
		},
	}
}

func reportEventAST(re reactor.ReportEvent) *ast.CompositeLit {
	elts := []ast.Expr{}

	if re.Name != "" {
		elts = append(elts, stringKeyValueAssignmentAST("Name", re.Name))
	}

	if re.PreventDefault != false {
		elts = append(elts, stringKeyValueAssignmentAST("PreventDefault", re.PreventDefault))
	}

	if re.StopPropagation != false {
		elts = append(elts, stringKeyValueAssignmentAST("StopPropagation", re.StopPropagation))
	}

	// TODO: re.ExtraValues

	cl := &ast.CompositeLit{
		Type: &ast.SelectorExpr{
			X: &ast.Ident{
				Name: "reactor",
			},
			Sel: &ast.Ident{
				Name: "ReportEvent",
			},
		},
		Elts: elts,
	}

	return cl

}

func reportEventsSliceAST(res []reactor.ReportEvent) *ast.KeyValueExpr {

	elts := []ast.Expr{}

	for _, re := range res {
		elts = append(elts, reportEventAST(re))
	}

	expr := &ast.KeyValueExpr{
		Key: &ast.Ident{Name: "ReportEvents"},
		Value: &ast.CompositeLit{

			Type: &ast.ArrayType{

				Elt: &ast.SelectorExpr{

					X: &ast.Ident{
						Name: "reactor",
					},
					Sel: &ast.Ident{
						Name: "ReportEvent",
					},
				},
			},
			Elts: elts,
		},
	}

	return expr
}

func childrenSliceAST(children []*reactor.DisplayModel) *ast.KeyValueExpr {

	elts := []ast.Expr{}

	for _, ch := range children {
		elts = append(elts, displayModelToAST(ch))
	}

	expr := &ast.KeyValueExpr{
		Key: &ast.Ident{Name: "Children"},
		Value: &ast.CompositeLit{
			Type: &ast.ArrayType{
				Elt: &ast.UnaryExpr{
					Op: token.MUL,
					X: &ast.SelectorExpr{
						X: &ast.Ident{
							Name: "reactor",
						},
						Sel: &ast.Ident{
							Name: "DisplayModel",
						},
					},
				},
			},
			Elts: elts,
		},
	}

	return expr
}
