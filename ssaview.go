// ssaview is a small utlity that renders SSA code alongside input Go code
//
// Runs via HTTP on :8080
package main

import (
	"bytes"
	"encoding/json"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"os"
	"sort"

	"code.google.com/p/go.tools/importer"
	"code.google.com/p/go.tools/ssa"
)

const indexPage = "index.html"

// toSSA converts go source to SSA
func toSSA(source io.Reader) ([]byte, error) {

	// Construct an importer.  Imports will be loaded as if by 'go build'.
	imp := importer.New(&importer.Config{Build: &build.Default})

	// Parse the input file.
	file, err := parser.ParseFile(imp.Fset, "hello.go", source, 0)
	if err != nil {
		return nil, err
	}

	// Create single-file main package and import its dependencies.
	mainInfo := imp.CreatePackage("main", file)

	// Create SSA-form program representation.
	var mode ssa.BuilderMode
	prog := ssa.NewProgram(imp.Fset, mode)
	if err := prog.CreatePackages(imp); err != nil {
		return nil, err
	}
	mainPkg := prog.Package(mainInfo.Pkg)

	// Print out the package.
	out := new(bytes.Buffer)
	mainPkg.SetDebugMode(true)
	mainPkg.DumpTo(out)

	// Build SSA code for bodies of functions in mainPkg.
	mainPkg.Build()

	funcs := members([]ssa.Member{})
	for _, obj := range mainPkg.Members {
		if obj.Token() == token.FUNC {
			funcs = append(funcs, obj)
		}
	}
	sort.Sort(funcs)
	for _, f := range funcs {
		mainPkg.Func(f.Name()).DumpTo(out)
	}

	return out.Bytes(), nil
}

type members []ssa.Member

func (m members) Len() int           { return len(m) }
func (m members) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }
func (m members) Less(i, j int) bool { return m[i].Pos() < m[j].Pos() }

// writeJSON attempts to serialize data and write it to w
// On error it will write an HTTP status of 400
func writeJSON(w http.ResponseWriter, data interface{}) error {

	if err, ok := data.(error); ok {
		data = struct{ Error string }{err.Error()}
		w.WriteHeader(400)
	}
	o, err := json.MarshalIndent(data, "", "   ")
	if err != nil {
		return err
	}
	_, err = w.Write(o)
	return err
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open(indexPage)
		if err != nil {
			writeJSON(w, err)
		}
		io.Copy(w, f)
	})
	http.HandleFunc("/ssa", func(w http.ResponseWriter, r *http.Request) {
		ssa, err := toSSA(r.Body)
		if err != nil {
			writeJSON(w, err)
			return
		}
		defer r.Body.Close()
		writeJSON(w, struct{ All string }{string(ssa)})
	})
	http.ListenAndServe(":8080", nil)
}
