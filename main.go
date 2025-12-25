package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

var BasicTypeMap = map[string]string{
	"string":  "string",
	"int":     "number",
	"int32":   "number",
	"int64":   "number",
	"float32": "number",
	"float64": "number",
	"bool":    "boolean",
	"error":   "Error",
}

func MapType(goType string) string {
	if tsType, ok := BasicTypeMap[goType]; ok {
		return tsType
	}

	return goType
}

type Action struct {
	Name string
	Req  string
	Res  string
}

type StructDef struct {
	Name   string
	Fields map[string]string
}

func main() {
	fset := token.NewFileSet()
	pkgs, _ := parser.ParseDir(fset, "./internal/actions", nil, parser.ParseComments)

	var actions []Action
	structs := make(map[string]StructDef)

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.TypeSpec:
					if st, ok := x.Type.(*ast.StructType); ok {
						def := StructDef{Name: x.Name.Name, Fields: make(map[string]string)}
						for _, field := range st.Fields.List {
							fieldType := field.Type.(*ast.Ident).Name
							// フィールド名
							fieldName := field.Names[0].Name
							def.Fields[fieldName] = MapType(fieldType)
						}
						structs[x.Name.Name] = def
					}

				// Actionの解析
				case *ast.FuncDecl:
					if x.Doc != nil && hasActionTag(x.Doc) {
						req := x.Type.Params.List[0].Type.(*ast.Ident).Name
						res := x.Type.Results.List[0].Type.(*ast.Ident).Name
						actions = append(actions, Action{Name: x.Name.Name, Req: req, Res: res})
					}
				}
				return true
			})
		}
	}

	generateTS(actions, structs)
	generateGoRouter(actions)
}

func hasActionTag(doc *ast.CommentGroup) bool {
	for _, c := range doc.List {
		if strings.Contains(c.Text, "@action") {
			return true
		}
	}
	return false
}

func generateGoRouter(actions []Action) {
	var cases []string
	for _, a := range actions {
		cases = append(cases, fmt.Sprintf(`
	case "%s":
		var req actions.%s
		json.Unmarshal(body.Args, &req)
		res, err := actions.%s(req)
		return res, err`, a.Name, a.Req, a.Name))
	}

	tpl := `package main
import (
	"encoding/json"
	"net/http"
	"yourproject/internal/actions"
)

type RPCRequest struct {
	Fn   string          ` + "`json:\"fn\"`" + `
	Args json.RawMessage ` + "`json:\"args\"`" + `
}

func ActionHandler(w http.ResponseWriter, r *http.Request) {
	var body RPCRequest
	json.NewDecoder(r.Body).Decode(&body)

	var result interface{}
	var err error

	switch body.Fn {%s
	}

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(result)
}`
	os.WriteFile("internal/generated/router.go", []byte(fmt.Sprintf(tpl, strings.Join(cases, ""))), 0644)
}

func generateTS(actions []Action, structs map[string]StructDef) {
	var sb strings.Builder

	for _, s := range structs {
		sb.WriteString(fmt.Sprintf("export interface %s {\n", s.Name))
		for name, tsType := range s.Fields {
			sb.WriteString(fmt.Sprintf("  %s: %s;\n", strings.ToLower(name), tsType))
		}
		sb.WriteString("}\n\n")
	}

	for _, a := range actions {
		sb.WriteString(fmt.Sprintf(`export const %s = async (args: %s): Promise<%s> => {
  const res = await fetch("/api/actions", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ fn: "%s", args }),
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
};

`, a.Name, a.Req, a.Res, a.Name))
	}

	os.WriteFile("src/actions.ts", []byte(sb.String()), 0644)
}