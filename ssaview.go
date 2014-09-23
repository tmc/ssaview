// ssaview is a small utlity that renders SSA code alongside input Go code
//
// Runs via HTTP on :8080 or the PORT environment variable
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/token"
	"io"
	"net/http"
	"os"
	"sort"

	"code.google.com/p/go.tools/go/loader"
	"code.google.com/p/go.tools/go/ssa"
)

const indexPage = "index.html"

type members []ssa.Member

func (m members) Len() int           { return len(m) }
func (m members) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }
func (m members) Less(i, j int) bool { return m[i].Pos() < m[j].Pos() }

// toSSA converts go source to SSA
func toSSA(source io.Reader, fileName, packageName string, debug bool) ([]byte, error) {
	// adopted from ssa package example
	var conf loader.Config

	// Parse the input file.
	file, err := conf.ParseFile(fileName, source)
	if err != nil {
		return nil, err
	}

	// Create single-file main package.
	conf.CreateFromFiles("main", file)

	// Load the main package and its dependencies.
	iprog, err := conf.Load()
	if err != nil {
		return nil, err
	}

	// Create SSA-form program representation.
	prog := ssa.Create(iprog, ssa.SanityCheckFunctions)
	mainPkg := prog.Package(iprog.Created[0].Pkg)

	// Write the package to a buffer.
	out := new(bytes.Buffer)
	mainPkg.SetDebugMode(debug)
	mainPkg.WriteTo(out)
	mainPkg.Build()

	// grab just the functions
	funcs := members([]ssa.Member{})
	for _, obj := range mainPkg.Members {
		if obj.Token() == token.FUNC {
			funcs = append(funcs, obj)
		}
	}
	// sort by Pos()
	sort.Sort(funcs)
	for _, f := range funcs {
		mainPkg.Func(f.Name()).WriteTo(out)
	}
	return out.Bytes(), nil
}

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
		ssa, err := toSSA(r.Body, "main.go", "main", false)
		if err != nil {
			writeJSON(w, err)
			return
		}
		defer r.Body.Close()
		writeJSON(w, struct{ All string }{string(ssa)})
	})
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println(http.ListenAndServe(":"+port, nil))
}
