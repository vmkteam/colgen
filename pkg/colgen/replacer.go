package colgen

import (
	"bytes"
	"fmt"
	"go/types"
	"io"
	"regexp"
	"strings"
	"text/template"

	"golang.org/x/tools/go/packages"
)

// Replacer.
//   //colgen@NewCall(db)
//   //colgen@NewUser(db)
//   //colgen@newUserSummary(dating.User,full,json)

type Field struct {
	Name string
	Type string
	Tag  string
}

type ReplaceRule struct {
	Find    string
	Replace string

	Cmd      string
	Entity   string
	Arg      string
	IsFull   bool
	WithJSON bool

	Fields []Field
}

var reNewFullNameArg = regexp.MustCompile(`(?mi)^//colgen@(New|new)(\w+)\(([\w.,]+)\)$`)

// ParseReplaceRule parses replaceRule to struct. Examples:
//
//	//colgen@NewCall(db)
//	//colgen@NewUser(db)
//	//colgen@newUserSummary(dating.User,full,json)
func ParseReplaceRule(rule string) (ReplaceRule, error) {
	r := ReplaceRule{Find: rule}
	matches := reNewFullNameArg.FindStringSubmatch(rule)
	if len(matches) != 4 {
		return r, fmt.Errorf("%w: %s", ErrUnknownLine, rule)
	}

	r.Cmd = matches[1]
	r.Entity = matches[2]

	for i, arg := range strings.Split(matches[3], ",") {
		if i == 0 {
			r.Arg = arg
			continue
		}

		switch arg {
		case "full":
			r.IsFull = true
		case "json":
			r.WithJSON = true
		default:
			return r, fmt.Errorf("%w: %s", ErrUnknownLine, arg)
		}
	}

	// validate conflicts
	if r.WithJSON && !r.IsFull {
		return r, fmt.Errorf("%w: %s", ErrMissingArg, "full")
	}

	// convert db => db.Entity if needed
	if !strings.Contains(r.Arg, ".") {
		r.Arg += "." + r.Entity
	}

	r.Fields = nil

	return r, nil
}

func ParseReplaceRules(rules []string) ([]ReplaceRule, error) {
	rr := make([]ReplaceRule, 0, len(rules))
	for _, rule := range rules {
		r, err := ParseReplaceRule(rule)
		if err != nil {
			return nil, err
		}
		rr = append(rr, r)
	}

	return rr, nil
}

type Replacer struct {
	pkg *packages.Package // parsed go packages
}

func NewReplacer() *Replacer {
	return &Replacer{}
}

// UsePackageDir parses path for go packages.
func (rl *Replacer) UsePackageDir(path string) (err error) {
	rl.pkg, err = loadPackage(path)
	return
}

func (rl *Replacer) findImportedType(fullTypeName string) types.Object {
	if rl.pkg == nil {
		return nil
	}

	// split db.User to db and User.
	tp := strings.Split(fullTypeName, ".")

	// try to find by pkg suffix
	for _, imp := range rl.pkg.Imports {
		if strings.HasSuffix(imp.PkgPath, tp[0]) {
			if found := imp.Types.Scope().Lookup(tp[1]); found != nil {
				return found
			}
		}
	}
	return nil // не найде
}

func newFields(rule ReplaceRule, fields []entityField) []Field {
	if !rule.IsFull {
		return nil
	}

	ff := make([]Field, 0, len(fields))
	for _, f := range fields {
		if !f.IsExported {
			continue
		}

		// create json tag
		tag := ""
		if rule.WithJSON {
			t := f.Name

			// convet ID to entityId
			if f.Name == FieldID {
				t = rule.Entity + "Id"
			}

			// first lower, last D to lower
			t = firsRuneToLower(t)
			if strings.HasSuffix(t, "ID") {
				t = lastRuneToLower(t)
			}

			// creat tag
			tag = fmt.Sprintf("`json:%q`", t)
		}

		ff = append(ff, Field{Name: f.Name, Type: f.Type, Tag: tag})
	}

	return ff
}

// Generate generates Replace code for Rule.
func (rl *Replacer) Generate(rules []string) ([]ReplaceRule, error) {
	// parse rules
	rr, err := ParseReplaceRules(rules)
	if err != nil {
		return nil, err
	}

	// process rules
	for i, r := range rr {
		// extract field for FullMode
		if r.IsFull {
			fields := typeSliceFromType(rl.findImportedType(r.Arg))
			if len(fields) == 0 {
				return nil, fmt.Errorf("%w: %s", ErrMissingType, r.Arg)
			}
			r.Fields = newFields(r, fields)
		}

		// generate code
		r.Replace, err = rl.generateByRule(r)
		if err != nil {
			return nil, err
		}

		// save results
		rr[i] = r
	}

	return rr, nil
}

func (rl *Replacer) generateByRule(rule ReplaceRule) (string, error) {
	const tmpl = `
type {{.Entity}} struct { {{if .IsFull}}{{range .Fields}}
    {{.Name}} {{.Type}} {{.Tag}}{{end}}{{else}}
    {{.Arg}}{{end}}
}

func {{.Cmd}}{{.Entity}}(in *{{.Arg}}) *{{.Entity}} {
	if in == nil {
		return nil
	}

	return &{{.Entity}}{ {{if .IsFull}}{{range .Fields}}
        {{.Name}}: in.{{.Name}},{{end}}{{else}}
        {{.Entity}}: *in,{{end}}
	}
}
`
	var buf bytes.Buffer
	err := rl.T(&buf, tmpl, rule)

	return buf.String(), err
}

// T renders text/template to Writer.
func (rl *Replacer) T(wr io.Writer, tmpl string, data any) error {
	t := template.Must(template.New("m").Parse(tmpl))
	return t.Execute(wr, data)
}
