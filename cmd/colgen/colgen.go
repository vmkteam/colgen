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

//go:generate colgen
//colgen@ai:review

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/vmkteam/colgen/pkg/colgen"

	"github.com/BurntSushi/toml"
)

var (
	flList      = flag.Bool("list", false, "use List suffix for collection")
	flImports   = flag.String("imports", "", "use custom imports: e.g pkg/db, pkg/domain")
	flFuncPkg   = flag.String("funcpkg", "", "use funcpkg for Map & MapP functions")
	flWriteKey  = flag.String("write-key", "", "write assistant key to ~/.colgen file")
	flAssistant = flag.String("ai", "", "use it to redefining assistant while writing a key to ~/.colgen file")
	flVersion   = flag.Bool("v", false, "print version and exit")
)

const (
	configFile = ".colgen"
)

type Config struct {
	DeepSeekKey string
	ClaudeKey   string
	Assistant   colgen.AssistantName
}

func NewConfig(key string, name colgen.AssistantName) Config {
	cfg := Config{
		Assistant: name,
	}
	switch name {
	case colgen.AssistantDeepSeek:
		cfg.DeepSeekKey = key
	case colgen.AssistantClaude:
		cfg.ClaudeKey = key
	default:
		panic("invalid assistant name")
	}

	return cfg
}

func (cfg Config) Key() string {
	switch cfg.Assistant {
	case colgen.AssistantDeepSeek:
		return cfg.DeepSeekKey
	case colgen.AssistantClaude:
		return cfg.ClaudeKey
	}

	return ""
}

func exitOnErr(err error) {
	if err != nil {
		log.Fatal("generation failed: ", err)
	}
}

func main() {
	log.SetFlags(log.Lshortfile)
	flag.Parse()

	switch {
	case *flVersion:
		fmt.Printf("colgen version: %v\n", appVersion())
		return // quit
	case *flWriteKey != "":
		err := writeConfig(*flWriteKey, colgen.AssistantName(*flAssistant))
		exitOnErr(err)
		return // quits
	}

	// read config
	cfg, err := readConfig(*flAssistant)
	exitOnErr(err)

	// set filename from go:generate
	filename := os.Getenv("GOFILE")
	if filename == "" {
		log.Fatal("GOFILE environment variable is not set. Run via `go generate`")
	}

	// get colgen lines from file
	cl, err := readFile(filename, cfg.Assistant != "")
	exitOnErr(err)

	// if assistant was found, process only one instruction
	if len(cl.assistant) > 0 {
		now := time.Now()
		log.Println("assisting: ", cl.assistant[0])
		assistFile(cfg, cl.assistant[0], filename)
		log.Println("assisting done", time.Since(now))
		return
	}

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

func assistFile(cfg Config, assistPrompt, filename string) {
	aa := colgen.NewAssistant(cfg.Assistant, cfg.Key())
	am := colgen.AssistMode(assistPrompt)

	if err := aa.IsValidMode(am); err != nil {
		exitOnErr(err)
	}

	content, err := os.ReadFile(filename)
	exitOnErr(err)

	// normal cases
	if am != colgen.ModeTests {
		r, err := aa.Generate(am, string(content))
		exitOnErr(err)

		// write file
		err = os.WriteFile(filename+".md", []byte(r), os.ModePerm)
		exitOnErr(err)
	} else { // tests
		tp, err := colgen.UserPromptForTests(content, filename)
		exitOnErr(err)

		r, err := aa.Generate(am, tp.TestPrompt)
		exitOnErr(err)

		if tp.AppendToFile {
			file, er := os.OpenFile(tp.TestFilename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
			exitOnErr(er)
			defer file.Close()

			_, er = file.WriteString(r)
			exitOnErr(er)
			return
		}

		// full
		err = os.WriteFile(tp.TestFilename, []byte(r), os.ModePerm)
		exitOnErr(err)
	}
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
	g := colgen.NewGenerator(cl.pkgName, *flImports, *flFuncPkg, appVersion())
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
	assistant []string
	pkgName   string
}

// readFile parses file line by line and returns all colgen lines without prefix.
func readFile(filename string, withAssistant bool) (result colgenLines, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		// is it possible to get package from gopackages, but we will do it in simple way.
		if strings.HasPrefix(line, "package ") {
			result.pkgName = strings.TrimPrefix(line, "package ")
		}

		switch {
		// find assistant lines
		case withAssistant && strings.HasPrefix(line, colgen.AssistantPrefix):
			if l, ok := strings.CutPrefix(line, colgen.AssistantPrefix); ok {
				result.assistant = append(result.assistant, l)
			}
		// find injection lines
		case strings.HasPrefix(line, colgen.InjectionPrefix):
			result.injection = append(result.injection, line)
		// find normal lines
		case strings.HasPrefix(line, colgen.ColgenPrefix):
			if l, ok := strings.CutPrefix(line, colgen.ColgenPrefix); ok {
				result.lines = append(result.lines, l)
			}
		}
	}

	return result, s.Err()
}

// baseName returns baseName from path without extension.
func baseName(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

// appVersion returns app version from VCS info.
func appVersion() string {
	result := "devel"
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return result
	}

	if info.Main.Version != "" {
		return info.Main.Version
	}

	for _, v := range info.Settings {
		if v.Key == "vcs.revision" {
			result = v.Value
		}
	}

	if len(result) > 8 {
		result = result[:8]
	}

	return result
}

// writeConfig creates config in home dir.
func writeConfig(key string, name colgen.AssistantName) error {
	cp, err := configPath()
	if err != nil {
		return err
	}

	if name == "" {
		name = colgen.AssistantDeepSeek
	}

	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(NewConfig(key, name)); err != nil {
		return fmt.Errorf("create config failed: %w", err)
	}

	if err := os.WriteFile(cp, buf.Bytes(), 0600); err != nil {
		return fmt.Errorf("write config to %s failed: %w", cp, err)
	}

	return nil
}

// configPath gets config path.
func configPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(homeDir, configFile)
	return path, nil
}

// readConfig reads default config from home dir.
func readConfig(flAssistant string) (Config, error) {
	cp, err := configPath()
	var cfg Config
	if err != nil {
		return cfg, err
	}

	if _, err = os.Stat(cp); errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}

	_, err = toml.DecodeFile(cp, &cfg)

	if flAssistant != "" {
		cfg.Assistant = colgen.AssistantName(flAssistant)
	}

	return cfg, err
}
