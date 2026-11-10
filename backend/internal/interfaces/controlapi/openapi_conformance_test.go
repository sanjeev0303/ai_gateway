package controlapi

import (
	"bufio"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const specPath = "../../../api/openapi/openapi.yaml"

// paramName matches a path parameter so {id} and {userID} compare equal.
var paramName = regexp.MustCompile(`\{[^}]+\}`)

// TestOpenAPIMatchesRegisteredRoutes keeps openapi.yaml honest: every
// "METHOD /path" the control-api registers must be documented, and every
// documented route must be registered. ServeMux cannot list its patterns, so
// the routes are read from the mux.Handle calls in this package's source.
func TestOpenAPIMatchesRegisteredRoutes(t *testing.T) {
	registered := registeredRoutes(t)
	documented := specRoutes(t)

	var missing, extra []string
	for route := range registered {
		if !documented[route] {
			missing = append(missing, route)
		}
	}
	for route := range documented {
		if !registered[route] {
			extra = append(extra, route)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)

	if len(missing) > 0 {
		t.Errorf("routes registered but missing from openapi.yaml:\n  %s", strings.Join(missing, "\n  "))
	}
	if len(extra) > 0 {
		t.Errorf("routes in openapi.yaml but not registered:\n  %s", strings.Join(extra, "\n  "))
	}
}

func isHealthProbe(path string) bool {
	return path == "/livez" || path == "/readyz" || path == "/healthz"
}

// registeredRoutes returns every route pattern passed to mux.Handle or
// mux.HandleFunc in the non-test files of this package.
func registeredRoutes(t *testing.T) map[string]bool {
	t.Helper()
	files, err := filepath.Glob("*.go")
	requireNoErr(t, err)

	routes := map[string]bool{}
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		requireNoErr(t, err)

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || (sel.Sel.Name != "Handle" && sel.Sel.Name != "HandleFunc") {
				return true
			}
			if recv, ok := sel.X.(*ast.Ident); !ok || recv.Name != "mux" {
				return true
			}
			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				t.Errorf("%s: mux registration with a non-literal pattern", fset.Position(call.Pos()))
				return true
			}
			pattern, err := strconv.Unquote(lit.Value)
			requireNoErr(t, err)
			method, path, _ := strings.Cut(pattern, " ")
			if !isHealthProbe(path) {
				routes[method+" "+paramName.ReplaceAllString(path, "{}")] = true
			}
			return true
		})
	}
	if len(routes) == 0 {
		t.Fatal("found no route registrations")
	}
	return routes
}

// specRoutes scans the paths block of openapi.yaml. The spec keeps a fixed
// layout (paths at two spaces, methods at four), so a line scan is enough and
// spares the module a YAML dependency.
func specRoutes(t *testing.T) map[string]bool {
	t.Helper()
	f, err := os.Open(specPath)
	requireNoErr(t, err)
	defer f.Close()

	methods := map[string]bool{"get": true, "post": true, "put": true, "patch": true, "delete": true}
	routes := map[string]bool{}
	inPaths := false
	path := ""
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, " ") && line != "" {
			inPaths = line == "paths:"
			continue
		}
		if !inPaths {
			continue
		}
		if strings.HasPrefix(line, "  /") {
			path = strings.TrimSuffix(strings.TrimSpace(line), ":")
			continue
		}
		key := strings.TrimSuffix(strings.TrimPrefix(line, "    "), ":")
		if methods[key] && !isHealthProbe(path) {
			routes[strings.ToUpper(key)+" "+paramName.ReplaceAllString(path, "{}")] = true
		}
	}
	requireNoErr(t, scanner.Err())
	return routes
}
