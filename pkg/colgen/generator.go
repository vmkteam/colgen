package colgen

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"go/types"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	"github.com/jinzhu/inflection"
	"golang.org/x/tools/go/packages"
)

const (
	CustomRuleUnique = "Unique"
	CustomRuleMap    = "Map"
	CustomRuleMapP   = "MapP"
	CustomRuleIndex  = "Index"
	FieldID          = "ID"

	ColgenPrefix    = "//colgen:"
	InjectionPrefix = "//colgen@"
)

var (
	ErrUnknownLine   = errors.New("unknown line")
	ErrMissingArg    = errors.New("missing arg")
	ErrMissingType   = errors.New("missing type")
	ErrMissingField  = errors.New("missing field")
	ErrMissingEntity = errors.New("missing main entity")
)

type Entity struct {
	Name, List string
}

func NewEntity(name string, useList bool) Entity {
	list, pl := name+"List", ""
	if !useList {
		pl = inflection.Plural(name)
	}

	if !useList && pl != name {
		list = pl
	}
	return Entity{Name: name, List: list}
}

func ParseRules(lines []string, useListSuffix bool) ([]Rule, error) {
	var result []Rule
	for _, line := range lines {
		line = strings.TrimSpace(line)
		var (
			rr  []Rule
			err error
		)

		// skip empty lines
		if line == "" {
			continue
		}

		switch {
		// detect custom generators: // colgen:News:UniqueTagIDs, Map
		case strings.Contains(line, ":"):
			rr, err = parseCustomRule(line)
		// detect main entities like: //colgen:News,Tag or //colgen:News
		case strings.Contains(line, ",") || !strings.Contains(line, " "):
			rr, err = parseEntities(line)
		default: // fail on unknow lines
			return nil, fmt.Errorf("%w: %q", ErrUnknownLine, line)
		}

		if err != nil {
			return nil, fmt.Errorf("%w: %q", err, line)
		}

		result = append(result, rr...)
	}

	merged := mergeRules(result, useListSuffix)
	err := validateRules(merged)

	return merged, err
}

// validateRules validates Rules for BaseGen parameter and MapP/Map.
func validateRules(rules []Rule) error {
	for _, r := range rules {
		if r.BaseGen {
			continue
		}

		for _, cr := range r.CustomRules {
			if !isMapP(cr.Name) {
				return fmt.Errorf("%w: %s for %s", ErrMissingEntity, r.EntityName, cr.Name)
			}
		}
	}

	return nil
}

// mergeRules merges base and custom rules.
func mergeRules(rules []Rule, useListSuffix bool) []Rule {
	idx := make(map[string]Rule)
	for _, r := range rules {
		r.UseListSuffix = useListSuffix

		existing, ok := idx[r.EntityName]
		if !ok {
			// create new rule
			idx[r.EntityName] = r
		} else {
			// detect custom or not
			if len(r.CustomRules) == 0 {
				existing.BaseGen = true
			} else {
				existing.CustomRules = append(existing.CustomRules, r.CustomRules...)
			}

			idx[r.EntityName] = existing
		}

	}

	// create sorted result
	var result []Rule
	for _, r := range idx {
		result = append(result, r)
	}
	slices.SortFunc(result, func(a, b Rule) int { return strings.Compare(a.EntityName, b.EntityName) })

	return result
}

// isMapP checks string for Map/MapP/map/mapp.
func isMapP(s string) bool {
	s = strings.ToLower(s)
	return s == strings.ToLower(CustomRuleMap) || s == strings.ToLower(CustomRuleMapP)
}

// reNameArg is regexp for `Index(db.User)` lookalike string.
var reNameArg = regexp.MustCompile(`(?mi)^(\w+)\(([\w.]+)\)$`)

// parseCustomRule parses custom rules like // colgen:News:UniqueTagIDs,Map
func parseCustomRule(line string) ([]Rule, error) {
	var rule Rule
	ll := strings.Split(line, ":")
	if len(ll) != 2 {
		return nil, fmt.Errorf("%w: %q", ErrUnknownLine, line)
	}

	// set entity name
	rule.EntityName = ll[0]

	// process all custom generators
	for _, l := range strings.Split(ll[1], ",") {
		name, arg := l, ""
		matches := reNameArg.FindStringSubmatch(l)
		if len(matches) == 3 {
			name, arg = matches[1], matches[2]
		}

		var cr CustomRule
		switch {
		case strings.HasPrefix(name, CustomRuleUnique): //UniqueTagIDs, UniqueEpisodeID
			cr.Name = CustomRuleUnique
			cr.Field = strings.TrimPrefix(name, CustomRuleUnique)
		case isMapP(name): // MapP(db), Map(db.User), mapp(db), map(db)
			if arg == "" {
				return nil, fmt.Errorf("%w: %q", ErrMissingArg, l)
			}

			cr.Name = name
			cr.Arg = arg
		case name == CustomRuleIndex: // Index(UserID)
			if arg == "" {
				return nil, fmt.Errorf("%w: %q", ErrMissingArg, l)
			}

			cr.Name = name
			cr.Field = arg
		default: // Field, like ID => IDs()
			cr.Field = name
		}

		rule.CustomRules = append(rule.CustomRules, cr)
	}

	return []Rule{rule}, nil
}

// parseEntities parses main entities like // colgen:News,Tag
func parseEntities(line string) ([]Rule, error) {
	var r []Rule
	ll := strings.Split(line, ",")
	for _, l := range ll {
		r = append(r, Rule{EntityName: l, BaseGen: true})
	}

	//TODO(sergeyfat): check for Entity?
	return r, nil
}

type Generator struct {
	buf bytes.Buffer // current buffer

	err         error    // generation error
	pkgName     string   // main pkg
	funcPkgName string   // for map & mapp
	imports     []string // additional imports

	pkg *packages.Package // parsed go packages
}

// NewGenerator returns new Generator. Do not forget to use `UsePackageDir` method.
func NewGenerator(pkgName, imports, funcPkgName string) *Generator {
	g := &Generator{
		pkgName:     pkgName,
		funcPkgName: funcPkgName,
	}

	if imports != "" {
		g.imports = strings.Split(imports, ",")
		sort.Strings(g.imports)
	}

	return g
}

// UsePackageDir parses path for go packages.
func (g *Generator) UsePackageDir(path string) error {
	g.pkg, g.err = loadPackage(path)

	return g.err
}

// lookupTypes returns type for given struct name or nil if not found.
func (g *Generator) lookupType(s string) types.Object {
	if g.pkg == nil {
		return nil
	}

	return g.pkg.Types.Scope().Lookup(s)
}

func (g *Generator) SetError(err error, msg ...string) {
	if err != nil && g.err == nil {
		// wrap err if msg was set
		if len(msg) > 0 {
			err = fmt.Errorf("%s: %w", strings.Join(msg, ","), err)
		}

		g.err = err
	}
}

type Rule struct {
	EntityName    string       // struct name
	BaseGen       bool         // use base generation: methods, IDs, Index
	UseListSuffix bool         // always use `List` suffix
	CustomRules   []CustomRule // custom generation rules
}

type CustomRule struct {
	Name  string // rule name, might be empty for `Field` generator
	Field string // current `Field`
	Arg   string // Optional Arg in ().
}

// P writes string to Buffer.
func (g *Generator) P(format string, args ...any) *Generator {
	_, err := fmt.Fprintf(&g.buf, format, args...)
	g.SetError(err, "printf")
	return g
}

// L writes  new line to Buffer.
func (g *Generator) L() *Generator {
	g.SetError(g.buf.WriteByte('\n'), "newline")
	return g
}

// T renders text/template to Buffer.
func (g *Generator) T(tmpl string, data TemplateData) {
	t := template.Must(template.New("m").Parse(tmpl))
	g.SetError(t.Execute(&g.buf, data), "template")
}

// genHead generates Header for file with imports.
func (g *Generator) genHead() {
	g.P(`// Code generated by colgen; DO NOT EDIT.`)
	g.L()
	g.P("package %s", g.pkgName).L()
	g.L()

	if len(g.imports) > 0 {
		g.P("import (").L()
		for _, i := range g.imports {
			g.P("%q", i).L()
		}
		g.P(")")
	}
}

// Generate generates all code.
func (g *Generator) Generate(rules []Rule) ([]byte, error) {
	g.genHead()
	g.L()
	for _, r := range rules {
		if err := g.generateByRule(r); err != nil {
			return nil, fmt.Errorf("%w: %s", err, r.EntityName)
		}
	}
	return g.buf.Bytes(), g.err
}

// Format returns current Buffer as `go fmt`.
func (g *Generator) Format() ([]byte, error) {
	return format.Source(g.buf.Bytes())
}

// generateByRule generates code by Rule to Buffer.
func (g *Generator) generateByRule(rule Rule) error {
	fields := typeMapFromType(g.lookupType(rule.EntityName))
	if len(fields) == 0 {
		return fmt.Errorf("%w: %s", ErrMissingType, rule.EntityName)
	}

	// create entity
	e := NewEntity(rule.EntityName, rule.UseListSuffix)

	// process base generation
	idType, hasID := fields[FieldID]
	if rule.BaseGen {
		g.genType(e)
		g.L()
		if hasID {
			g.L()
			g.genField(TemplateData{FieldType: idType, FieldName: FieldID, Entity: e})
			g.L()
			g.genIndex(TemplateData{FieldType: idType, FieldName: FieldID, Entity: e})
			g.L()
		}
	}

	// process custom generation
	for _, cr := range rule.CustomRules {
		fType, hasF := fields[cr.Field]
		switch cr.Name {
		case CustomRuleMap, CustomRuleMapP:
			g.genMap(cr.Name, TemplateData{FieldType: cr.Arg, Entity: e}, false, rule.BaseGen)
		case strings.ToLower(CustomRuleMap):
			g.genMap(CustomRuleMap, TemplateData{FieldType: cr.Arg, Entity: e}, true, rule.BaseGen)
		case strings.ToLower(CustomRuleMapP):
			g.genMap(CustomRuleMapP, TemplateData{FieldType: cr.Arg, Entity: e}, true, rule.BaseGen)
		case CustomRuleUnique:
			if strings.HasPrefix(fType, "[]") {
				g.genUniqueFieldSlice(TemplateData{FieldType: strings.TrimPrefix(fType, "[]"), FieldName: cr.Field, Entity: e})
			} else {
				g.genUniqueField(TemplateData{FieldType: fType, FieldName: cr.Field, Entity: e})
			}
		case CustomRuleIndex:
			g.genIndex(TemplateData{FieldType: fType, FieldName: cr.Field, FuncName: "By" + cr.Field, Entity: e})
		case "":
			g.genField(TemplateData{FieldType: fType, FieldName: cr.Field, Entity: e})
		}
		g.L()

		// check for good type and name
		if !hasF && (!isMapP(cr.Name)) {
			return fmt.Errorf("%w: %s", ErrMissingField, cr.Field)
		}

	}

	return nil
}

type TemplateData struct {
	Entity    Entity
	FieldType string
	FieldName string
	FuncName  string
}

// genType writes collection Type to Buffer.
func (g *Generator) genType(e Entity) {
	g.P("type %s []%s", e.List, e.Name)
}

// genField generates Field to Buffer.
func (g *Generator) genField(data TemplateData) {
	const tmpl = `
func (ll {{.Entity.List}}) {{.FuncName}}() []{{.FieldType}} {
	r := make([]{{.FieldType}}, len(ll))
	for i := range ll {
		r[i] = ll[i].{{.FieldName}}
	}
	return r
}`

	data.FuncName = lastRuneToLower(inflection.Plural(data.FieldName))
	g.T(tmpl, data)
}

// genField generates Index to Buffer.
func (g *Generator) genIndex(data TemplateData) {
	const tmpl = `
func (ll {{.Entity.List}}) Index{{.FuncName}}() map[{{.FieldType}}]{{.Entity.Name}} {
	r := make(map[{{.FieldType}}]{{.Entity.Name}}, len(ll))
	for i := range ll {
		r[ll[i].{{.FieldName}}] = ll[i]
	}
	return r
}`

	g.T(tmpl, data)
}

// genUniqueField generates Unique Field to Buffer.
func (g *Generator) genUniqueField(data TemplateData) {
	const tmpl = `
func (ll {{.Entity.List}}) Unique{{.FuncName}}() []{{.FieldType}} {
	idx := make(map[{{.FieldType}}]struct{})
	for i := range ll {
		if _, ok := idx[ll[i].{{.FieldName}}]; !ok {
             idx[ll[i].{{.FieldName}}] = struct{}{}
        }
	}

	r, i := make([]{{.FieldType}}, len(idx)), 0
	for k := range idx {
		r[i] = k
        i++
	}
	return r    
}`
	data.FuncName = lastRuneToLower(inflection.Plural(data.FieldName))
	g.T(tmpl, data)
}

// genUniqueFieldSlice generates Unique Field (slice) to Buffer.
func (g *Generator) genUniqueFieldSlice(data TemplateData) {
	const tmpl = `
func (ll {{.Entity.List}}) Unique{{.FuncName}}() []{{.FieldType}} {
	idx := make(map[{{.FieldType}}]struct{})
	for i := range ll {
		for _, v := range ll[i].{{.FieldName}} {
		    if _, ok := idx[v]; !ok {
                idx[v] = struct{}{}
            }
        }
	}

	r, i := make([]{{.FieldType}}, len(idx)), 0
	for k := range idx {
		r[i] = k
        i++
	}
	return r    
}
`
	data.FuncName = lastRuneToLower(inflection.Plural(data.FieldName))
	g.T(tmpl, data)
}

// genMap generates List Converter to Buffer.
func (g *Generator) genMap(method string, data TemplateData, isLower, hasType bool) {
	s := "func New%s(in []%s) %s { return %s(in, New%s) }"
	if isLower {
		s = "func new%s(in []%s) %s { return %s(in, new%s) }"
	}

	// use New Plural type OR []
	returnType := data.Entity.List
	if !hasType {
		returnType = "[]" + data.Entity.Name
	}

	// set input Type
	inputType := data.FieldType
	if !strings.Contains(data.FieldType, ".") {
		inputType += "." + data.Entity.Name
	}

	if g.funcPkgName != "" {
		method = g.funcPkgName + "." + method
	}

	g.L()
	g.P(s, data.Entity.List, inputType, returnType, method, data.Entity.Name)
	g.L()
}

// lastRuneToLower returns string with last lower rune. It is useful for converting IDS to IDs.
func lastRuneToLower(s string) string {
	if len(s) == 0 {
		return s
	}

	lastRune, size := utf8.DecodeLastRuneInString(s)
	if lastRune == utf8.RuneError {
		return s
	}

	lowerRune := unicode.ToLower(lastRune)
	if lowerRune == lastRune {
		return s
	}

	return s[:len(s)-size] + string(lowerRune)
}

func firsRuneToLower(s string) string {
	if len(s) == 0 {
		return s
	}

	r := []rune(s)
	r[0] = unicode.ToLower(r[0])

	return string(r)
}

// loadPackage loads go pkg.
func loadPackage(path string) (*packages.Package, error) {
	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedDeps | packages.NeedImports}
	pkgs, err := packages.Load(cfg, path)
	if err != nil {
		return nil, fmt.Errorf("failed to load package '%s' for inspection: %w", path, err)
	}

	if packages.PrintErrors(pkgs) > 0 {
		return nil, fmt.Errorf("package errors: %v", packages.PrintErrors(pkgs))
	}

	return pkgs[0], nil
}

type entityField struct {
	Name       string
	Type       string
	FullType   string
	IsExported bool
	Level      int
}

// fillStructTypes fills sTypes with all fields.
func fillStructTypes(typ types.Type, indentLevel int, eTypes *[]entityField) {
	// pointers (*T → T)
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem()
	}

	// underlying types (type MyStruct struct{...})
	if named, ok := typ.(*types.Named); ok {
		typ = named.Underlying()
	}

	// display struct types
	if st, ok := typ.(*types.Struct); ok {
		for i := 0; i < st.NumFields(); i++ {
			field := st.Field(i)
			indent := ""
			for j := 0; j < indentLevel; j++ {
				indent += "  "
			}

			if field.Embedded() {
				//fmt.Printf("%s[mbed] %s (%s)\n", indent, field.Name(), field.Type())
				fillStructTypes(field.Type(), indentLevel+1, eTypes)
			} else {
				//fmt.Printf("%s%s: %s\n", indent, field.Name(), field.Type())
				*eTypes = append(*eTypes, entityField{
					Name:       field.Name(),
					Type:       field.Type().String(),
					Level:      indentLevel,
					IsExported: field.Exported(),
				})
			}
		}
	}
}

// typeMapFromType returns field => type for given type.
func typeMapFromType(t types.Object) map[string]string {
	eTypes := typeSliceFromType(t)
	sTypes := make(map[string]string)
	for _, v := range eTypes {
		sTypes[v.Name] = v.Type
	}

	return sTypes
}

// typeMapFromType returns field => type for given type.
func typeSliceFromType(t types.Object) []entityField {
	if t == nil {
		return nil
	}

	var eTypes []entityField
	fillStructTypes(t.Type(), 0, &eTypes)
	for i, e := range eTypes {
		eTypes[i].FullType = e.Type
		eTypes[i].Type = path.Base(e.Type)
	}

	return eTypes
}
