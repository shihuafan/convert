package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

type StructType struct {
	FullName string
	Name     string
	Pkg      *packages.Package
	Content  map[string]string
}

type Convert struct {
	from *StructType
	to   *StructType
	pkg  *packages.Package
}

var raw = []string{"int8", "int16", "int32", "int", "int64"}

var (
	from = flag.String("from", "", "input from")
	to   = flag.String("to", "", "input to")
)

func main() {
	flag.Parse()
	if from == nil || to == nil {
		os.Exit(1)
	}

	cfg := &packages.Config{
		Mode:  packages.LoadAllSyntax,
		Tests: true,
	}
	pkg, _ := packages.Load(cfg)
	locationPkg := pkg[0]
	allPkgs := listAllPkgs(pkg[0])
	convert := Convert{pkg: locationPkg}
	if convert.from = buildStructType(locationPkg, allPkgs, *from); convert.from == nil {
		fmt.Println("build from struct fail")
		return
	}
	if convert.to = buildStructType(locationPkg, allPkgs, *to); convert.to == nil {
		fmt.Println("build to struct fail")
		return
	}
	if convert.write() == nil {
		fmt.Println("ok!")
	} else {
		fmt.Println("sorry, an error has occured!")
	}
}

func buildStructType(locationPkg *packages.Package, allPkgs map[string]*packages.Package, filed string) *StructType {
	result := &StructType{FullName: filed}
	if s := strings.Index(filed, "."); s > 0 {
		result.Name = filed[s+1:]
		if result.Pkg = allPkgs[filed[:s]]; result.Pkg == nil {
			return nil
		}
	} else {
		result.Name = filed
		result.Pkg = locationPkg
	}

	res := findAllFieldsFromPkg(result.Pkg)
	if result.Content = res[result.Name]; result.Content == nil {
		return nil
	}
	return result
}

func indexOf(ss []string, s string) int {
	for i, v := range ss {
		if v == s {
			return i
		}
	}
	return -1
}
func stringfySingle(name string, fromType, toType string) string {
	if fromType == toType {
		return fmt.Sprintf("\tto.%v = from.%v\n", name, name)
	}
	if "*"+fromType == toType {
		return fmt.Sprintf("\tto.%v = &from.%v\n", name, name)
	}
	if fromType == "*"+toType {
		result := fmt.Sprintf("\tif from.%v != nil {\n", name)
		result += fmt.Sprintf("\t\tto.%v = *from.%v\n", name, name)
		result += "\t}\n"
		return result
	}
	// 判断能不能转换
	if indexOf(raw, strings.ReplaceAll(fromType, "*", "")) < indexOf(raw, strings.ReplaceAll(toType, "*", "")) {
		if !strings.HasPrefix(fromType, "*") && !strings.HasPrefix(toType, "*") {
			return fmt.Sprintf("\tto.%v = %v(from.%v)\n", name, toType, name)
		}
		if strings.HasPrefix(fromType, "*") && !strings.HasPrefix(toType, "*") {
			result := fmt.Sprintf("\tif from.%v != nil {\n", name)
			result += fmt.Sprintf("\t\tto.%v = %v(*from.%v)\n", name, toType, name)
			result += "\t}\n"
			return result
		}
		if !strings.HasPrefix(fromType, "*") && strings.HasPrefix(toType, "*") {
			result := fmt.Sprintf("\t%vTemp := %v(from.%v)\n", name, toType, name)
			result += fmt.Sprintf("\tto.%v = &(%vTemp)\n", name, name)
			return result
		}
		if strings.HasPrefix(fromType, "*") && strings.HasPrefix(toType, "*") {
			result := fmt.Sprintf("\tif from.%v != nil {\n", name)
			result += fmt.Sprintf("\t\t%vTemp := %v(*from.%v)\n", name, strings.ReplaceAll(toType, "*", ""), name)
			result += fmt.Sprintf("\t\tto.%v = &%vTemp\n", name, name)
			result += "\t}\n"
			return result
		}
	}
	return ""

}

func (c *Convert) write() error {
	functionName := fmt.Sprintf("Convert%v%vTo%v%v", capitalize(c.from.Pkg.Name), c.from.Name, capitalize(c.to.Pkg.Name), c.to.Name)
	functionContent := fmt.Sprintf("func %v(from *%v) *%v {\n",
		functionName, c.from.FullName, c.to.FullName)
	functionContent += fmt.Sprintf("\tto := &%v{}\n", c.to.FullName)
	var names []string
	for name := range c.from.Content {
		if _, ok := c.to.Content[name]; ok {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		functionContent += stringfySingle(name, c.from.Content[name], c.to.Content[name])
	}
	functionContent += "\treturn to\n}"

	convertFile, err := os.OpenFile("convert.go", os.O_RDWR, os.ModeAppend)
	if pathErr := (*os.PathError)(nil); errors.As(err, &pathErr) {
		convertFile, _ = os.Create("convert.go")
	}
	content, _ := io.ReadAll(convertFile)
	defer convertFile.Close()
	buf := bytes.Buffer{}
	f, _ := parser.ParseFile(token.NewFileSet(), "", content, 0)
	if len(content) == 0 || f == nil {
		buf.WriteString(fmt.Sprintf("package %v\n\n", c.pkg.Name))
		buf.WriteString(fmt.Sprintf("import (\n\t\"%v\"\n", c.from.Pkg.PkgPath))
		if c.from.Pkg.PkgPath != c.to.Pkg.PkgPath {
			buf.WriteString(fmt.Sprintf("\t\"%v\"\n", c.to.Pkg.PkgPath))
		}
		buf.WriteString(")\n\n")
		buf.WriteString(functionContent)
	} else {
		for _, decl := range f.Decls {
			if v, ok := decl.(*ast.FuncDecl); ok && v.Name.Name == functionName {
				buf.Write(content[:v.Pos()-1])
				buf.WriteString(functionContent)
				buf.Write(content[v.End()-1:])
				break
			}
		}
		if buf.Len() == 0 {
			buf.Write(content)
			buf.WriteString("\n\n")
			buf.WriteString(functionContent)
		}
	}

	convertFile.Seek(0, 0)
	_, err = convertFile.Write(buf.Bytes())
	return err
}

func findAllFieldsFromPkg(pkg *packages.Package) (fields map[string]map[string]string) {
	fields = make(map[string]map[string]string)
	for _, goFile := range pkg.GoFiles {
		rawFile, _ := os.Open(goFile)
		content, _ := io.ReadAll(rawFile)
		f, _ := parser.ParseFile(token.NewFileSet(), "", content, 0)
		for _, v := range f.Decls {
			if g, ok := v.(*ast.GenDecl); ok && g.Tok == token.TYPE {
				for _, v := range g.Specs {
					if typeSpec, ok := v.(*ast.TypeSpec); ok {
						if structType, ok := typeSpec.Type.(*ast.StructType); ok {
							fields[typeSpec.Name.Name] = make(map[string]string)
							for _, field := range structType.Fields.List {
								if len(field.Names) > 0 {
									if starExpr, ok := field.Type.(*ast.StarExpr); ok {
										if identType, ok := (starExpr.X).(*ast.Ident); ok {
											fields[typeSpec.Name.Name][field.Names[0].Name] = "*" + identType.Name
										}
									} else if identType, ok := (field.Type).(*ast.Ident); ok {
										fields[typeSpec.Name.Name][field.Names[0].Name] = identType.Name
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return
}

func listAllPkgs(pkg *packages.Package) map[string]*packages.Package {
	result := make(map[string]*packages.Package)
	for _, value := range pkg.Imports {
		result[value.Name] = value
	}
	return result
}

func capitalize(str string) string {
	r := []rune(str)
	if r[0] >= 97 && r[0] <= 122 {
		r[0] -= 32
	}
	return string(r)
}
