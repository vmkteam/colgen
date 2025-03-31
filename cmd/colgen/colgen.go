// Colgen is a tool to automate the creation of collections methods.
// Tool will search `//colgen:` comments and generates specific methods and types.
// Example:
//
//	//go:generate colgen
//	//colgen:News,Category,Tag
//	//colgen:News:TagIDs,UniqueTagIDs,Map(db),UUID
//	//colgen:Episode:ShowIDs,MapP(db.SiteUser),Index(MovieID)
//	//colgen:Show:MapP(db)
//	//colgen:Season:mapp(db)
//
// Flags:
// -list: use List suffix for collection, default false.
// -imports: use custom imports: e.g pkg/db, pkg/domain.
//
// Base Generators (by default) will be created for `//colgen:<struct>,<struct>,...`.
// - Collection type `type <structs> []<struct>` and methods for this type:
// - IDs() []<id type>: if ID field exists. Returns all IDs in slice.
// - Index() map[<id type>]<struct>: if ID filed exists. Returns all structs as map[ID]struct.
//
// Custom generators
// - `Index` can accept another field for creating index. By default, it is ID.
// - <Field>: collect all values from field.
// - Unique<Field>: collect unique values from field.
// - MapP: `func NewUsers(in []<arg>) <structs> { return <func pkg>MapP(in, New<struct>) }`
// - Map: same as MapP. Map or MapP can accept package or struct as arg. Can be lower for private constructors.
//
// Inline mode via //go:generate
// //colgen@NewCall(db)
// //colgen@newUserSummary(newsportal.User,full,json)
package main

import (
	"bufio"
	"bytes"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/vmkteam/colgen/pkg/colgen"
)

var (
	flList    = flag.Bool("list", false, "use List suffix for collection")
	flImports = flag.String("imports", "", "use custom imports: e.g pkg/db, pkg/domain")
	flFuncPkg = flag.String("funcpkg", "", "use funcpkg for Map & MapP functions")
)

func exitOnErr(err error) {
	if err != nil {
		log.Fatal("generation failed: ", err)
	}
}

func main() {
	log.SetFlags(log.Lshortfile)
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		args = []string{"."}
	}

	// set filename from go:generate
	filename := os.Getenv("GOFILE")
	if filename == "" {
		log.Fatal("GOFILE environment variable is not set. Run via `go generate`")
	}

	// get colgen lines from file
	cl, err := readFile(filename)
	exitOnErr(err)

	if len(cl.injection) > 0 {
		log.Println("replacing injections")
		replaceFile(cl, filename)
	}

	if len(cl.lines) == 0 {
		log.Println("no colgen lines found")
		return
	}
	generateFile(cl, filename)
}

func replaceFile(cl colgenLines, filename string) {
	r := colgen.NewReplacer()
	// load go packages
	err := r.UsePackageDir(filepath.Dir(filename))
	exitOnErr(err)

	rr, err := r.Generate(cl.injection)
	exitOnErr(err)

	// read file
	content, err := os.ReadFile(filename)
	exitOnErr(err)

	// replace
	for _, r := range rr {
		content = bytes.ReplaceAll(content, []byte(r.Find), []byte(r.Replace))
	}

	// write file
	err = os.WriteFile(filename, content, os.ModePerm)
	exitOnErr(err)
}

func generateFile(cl colgenLines, filename string) {
	// init generator and rules
	g := colgen.NewGenerator(cl.pkgName, *flImports, *flFuncPkg)
	rules, err := colgen.ParseRules(cl.lines, *flList)
	exitOnErr(err)

	// load go packages
	err = g.UsePackageDir(filepath.Dir(filename))
	exitOnErr(err)

	// generate code
	data, err := g.Generate(rules)
	exitOnErr(err)

	// try to save formatted file
	formatted, err := g.Format()
	if err != nil {
		log.Println("failed to format:", err)
		log.Println("saving anyway")
	} else {
		data = formatted
	}

	// save file to FS
	err = os.WriteFile(baseName(filename)+"_colgen.go", data, os.ModePerm)
	exitOnErr(err)
}

type colgenLines struct {
	lines     []string
	injection []string
	pkgName   string
}

// readFile parses file line by line and returns all colgen lines without prefix.
func readFile(filename string) (result colgenLines, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		//TODO(sergeyfast):remove
		if strings.HasPrefix(s.Text(), "package ") {
			result.pkgName = strings.TrimPrefix(s.Text(), "package ")
		}

		// find injection lines
		if strings.HasPrefix(s.Text(), colgen.InjectionPrefix) {
			result.injection = append(result.injection, s.Text())
		}

		// find normal lines
		if !strings.HasPrefix(s.Text(), colgen.ColgenPrefix) {
			continue
		}

		l, ok := strings.CutPrefix(s.Text(), colgen.ColgenPrefix)
		if ok {
			result.lines = append(result.lines, l)
		}
	}

	return result, s.Err()
}

// baseName returns baseName from path without extension.
func baseName(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}
