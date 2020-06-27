package sql2go

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"io/ioutil"
	"reflect"
	"sort"
	"strings"
	"text/template"
	"xorm.io/xorm/names"
	"xorm.io/xorm/schemas"
)

var (
	GoXormTmp                 string
	errBadComparisonType      error
	errBadComparison          error
	errNoComparison           error
	created, updated, deleted []string
	supportComment            bool
	snakeMapper               names.SnakeMapper
)

func init() {
	GoXormTmp = `
{{- range .Tables}}
type {{TableMapper .Name}} struct {
{{$table := .}}
{{range .ColumnsSeq}}{{$col := $table.GetColumn .}}	{{ColMapper $col.Name}}	{{Type $col}} {{Tag $table $col}}
{{end}}
}
{{end}}
`
	errBadComparisonType = errors.New("invalid type for comparison")
	errBadComparison = errors.New("incompatible types for comparison")
	errNoComparison = errors.New("missing argument for comparison")
	created, updated, deleted = []string{"created_at"}, []string{"updated_at"}, []string{"deleted_at"}
	supportComment = true
	snakeMapper = names.SnakeMapper{}
}

type convertArgs struct {
	colPrefix   string
	tablePrefix string
	genJson     bool
	genXorm     bool
	tmpl        string
	otherTags   string
}
type GolangTmp struct {
	funcs    template.FuncMap
	formater func([]byte) ([]byte, error)
	args     *convertArgs
}
type kind int

const (
	invalidKind kind = iota
	boolKind
	complexKind
	intKind
	floatKind
	integerKind
	stringKind
	uintKind
)

type TmplType int

const (
	GOTMPL TmplType = iota
	GOXORMTMPL
)

func NewConvertArgs() *convertArgs {

	return &convertArgs{
		colPrefix:   "",
		tablePrefix: "",
		genJson:     false,
		tmpl:        GoXormTmp,
	}
}

func (c *convertArgs) SetColPrefix(prefix string) *convertArgs {
	c.colPrefix = prefix
	return c
}
func (c *convertArgs) SetGenJson(genJson bool) *convertArgs {
	c.genJson = genJson
	return c
}

func (c *convertArgs) SetTmpl(tmplType TmplType) *convertArgs {
	c.tmpl = GoXormTmp
	if tmplType == GOTMPL {
		c.SetGenXorm(false)
	} else {
		c.SetGenXorm(true)
	}
	return c
}

func (c *convertArgs) SetGenXorm(gen bool) *convertArgs {
	c.genXorm = gen
	return c
}

func (c *convertArgs) SetTmplStr(tmpl string) *convertArgs {
	c.tmpl = tmpl
	return c
}

func (c *convertArgs) SetTablePrefix(prefix string) *convertArgs {
	c.tablePrefix = prefix
	return c
}

func (c *convertArgs) SetOtherTags(tags string) *convertArgs {
	c.otherTags = tags
	return c
}

func (g *GolangTmp) GenerateGo(tables []*schemas.Table) ([]byte, error) {
	t := template.New("sql2go")
	t.Funcs(g.funcs)

	tmpl, err := t.Parse(g.args.tmpl)
	if err != nil {
		return nil, err
	}

	newbytes := bytes.NewBufferString("")

	d := &TmpData{Tables: tables}

	err = tmpl.Execute(newbytes, d)

	if err != nil {
		return nil, err
	}

	tplcontent, err := ioutil.ReadAll(newbytes)

	if err != nil {
		return nil, err
	}
	var source []byte
	if g.formater != nil {
		source, err = g.formater(tplcontent)
		if err != nil {
			return nil, err
		}
	} else {
		source = tplcontent
	}
	return source, nil
}

func NewGolangTmp(args *convertArgs) *GolangTmp {
	var colMapper names.Mapper
	var tableMapper names.Mapper
	colMapper = names.SnakeMapper{}
	tableMapper = names.SnakeMapper{}
	if args.colPrefix != "" {
		colMapper = names.NewPrefixMapper(colMapper, args.colPrefix)
	}
	if args.tablePrefix != "" {
		tableMapper = names.NewPrefixMapper(tableMapper, args.tablePrefix)
	}
	otherTags := make([]string, 0)
	for _, s := range strings.Split(args.otherTags, ",") {
		for _, ss := range strings.Split(s, " ") {
			for _, sss := range strings.Split(ss, "\t") {
				for _, ssss := range strings.Split(sss, ";") {
					ts := strings.TrimSpace(ssss)
					if len(ts) != 0 {
						otherTags = append(otherTags, ts)
					}
				}
			}
		}
	}
	return &GolangTmp{
		funcs: template.FuncMap{
			"ColMapper":   colMapper.Table2Obj,
			"TableMapper": tableMapper.Table2Obj,
			"Type":        typestring,
			"Tag":         getTag(colMapper, args.genJson, args.genXorm, otherTags),
			"UnTitle":     unTitle,
			"gt":          gt,
			"getCol":      getCol,
			"UpperTitle":  upTitle,
		},
		formater: formatGo,
		args:     args,
	}
}

func getTag(mapper names.Mapper, genJson bool, genXorm bool, otherTags []string) func(table *schemas.Table, col *schemas.Column) string {
	return func(table *schemas.Table, col *schemas.Column) string {
		isNameId := (mapper.Table2Obj(col.Name) == "Id")
		isIdPk := isNameId && typestring(col) == "int64"

		var res []string
		if !col.Nullable {
			if !isIdPk {
				res = append(res, "not null")
			}
		}
		if col.IsPrimaryKey {
			res = append(res, "pk")
		}
		if col.Default != "" {
			res = append(res, "default "+col.Default)
		}
		if col.IsAutoIncrement {
			res = append(res, "autoincr")
		}

		if col.SQLType.IsTime() && include(created, col.Name) {
			res = append(res, "created")
		}

		if col.SQLType.IsTime() && include(updated, col.Name) {
			res = append(res, "updated")
		}

		if col.SQLType.IsTime() && include(deleted, col.Name) {
			res = append(res, "deleted")
		}

		if supportComment && col.Comment != "" {
			res = append(res, fmt.Sprintf("comment('%s')", col.Comment))
		}

		colNames := make([]string, 0, len(col.Indexes))
		for name := range col.Indexes {
			colNames = append(colNames, name)
		}
		sort.Strings(colNames)

		for _, name := range colNames {
			index := table.Indexes[name]
			var uistr string
			if index.Type == schemas.UniqueType {
				uistr = "unique"
			} else if index.Type == schemas.IndexType {
				uistr = "index"
			}
			if len(index.Cols) > 1 {
				uistr += "(" + index.Name + ")"
			}
			res = append(res, uistr)
		}

		nstr := col.SQLType.Name
		if col.Length != 0 {
			if col.Length2 != 0 {
				nstr += fmt.Sprintf("(%v,%v)", col.Length, col.Length2)
			} else {
				nstr += fmt.Sprintf("(%v)", col.Length)
			}
		} else if len(col.EnumOptions) > 0 { //enum
			nstr += "("
			opts := ""

			enumOptions := make([]string, 0, len(col.EnumOptions))
			for enumOption := range col.EnumOptions {
				enumOptions = append(enumOptions, enumOption)
			}
			sort.Strings(enumOptions)

			for _, v := range enumOptions {
				opts += fmt.Sprintf(",'%v'", v)
			}
			nstr += strings.TrimLeft(opts, ",")
			nstr += ")"
		} else if len(col.SetOptions) > 0 { //enum
			nstr += "("
			opts := ""

			setOptions := make([]string, 0, len(col.SetOptions))
			for setOption := range col.SetOptions {
				setOptions = append(setOptions, setOption)
			}
			sort.Strings(setOptions)

			for _, v := range setOptions {
				opts += fmt.Sprintf(",'%v'", v)
			}
			nstr += strings.TrimLeft(opts, ",")
			nstr += ")"
		}
		res = append(res, nstr, "'"+col.Name+"'")
		var tags []string
		jsonName := mapper.Table2Obj(col.Name)
		jsonName = snakeMapper.Obj2Table(jsonName)
		if genJson {
			tags = append(tags, "json:\""+jsonName+"\"")
		}
		if len(res) > 0 && genXorm {
			tags = append(tags, "xorm:\""+strings.Join(res, " ")+"\"")
		}
		if len(otherTags) != 0 {
			for _, t := range otherTags {
				if t == "json" {
					continue
				}
				tags = append(tags, fmt.Sprintf("%s:\"%s\"", t, jsonName))
			}
		}
		if len(tags) > 0 {
			return "`" + strings.Join(tags, " ") + "`"
		} else {
			return ""
		}
	}
}

func unTitle(src string) string {
	if src == "" {
		return ""
	}

	if len(src) == 1 {
		return strings.ToLower(string(src[0]))
	} else {
		return strings.ToLower(string(src[0])) + src[1:]
	}
}

func upTitle(src string) string {
	if src == "" {
		return ""
	}
	return strings.ToUpper(src)
}

func basicKind(v reflect.Value) (kind, error) {
	switch v.Kind() {
	case reflect.Bool:
		return boolKind, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intKind, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return uintKind, nil
	case reflect.Float32, reflect.Float64:
		return floatKind, nil
	case reflect.Complex64, reflect.Complex128:
		return complexKind, nil
	case reflect.String:
		return stringKind, nil
	}
	return invalidKind, errBadComparisonType
}

// eq evaluates the comparison a == b || a == c || ...
func eq(arg1 interface{}, arg2 ...interface{}) (bool, error) {
	v1 := reflect.ValueOf(arg1)
	k1, err := basicKind(v1)
	if err != nil {
		return false, err
	}
	if len(arg2) == 0 {
		return false, errNoComparison
	}
	for _, arg := range arg2 {
		v2 := reflect.ValueOf(arg)
		k2, err := basicKind(v2)
		if err != nil {
			return false, err
		}
		if k1 != k2 {
			return false, errBadComparison
		}
		truth := false
		switch k1 {
		case boolKind:
			truth = v1.Bool() == v2.Bool()
		case complexKind:
			truth = v1.Complex() == v2.Complex()
		case floatKind:
			truth = v1.Float() == v2.Float()
		case intKind:
			truth = v1.Int() == v2.Int()
		case stringKind:
			truth = v1.String() == v2.String()
		case uintKind:
			truth = v1.Uint() == v2.Uint()
		default:
			panic("invalid kind")
		}
		if truth {
			return true, nil
		}
	}
	return false, nil
}

// lt evaluates the comparison a < b.
func lt(arg1, arg2 interface{}) (bool, error) {
	v1 := reflect.ValueOf(arg1)
	k1, err := basicKind(v1)
	if err != nil {
		return false, err
	}
	v2 := reflect.ValueOf(arg2)
	k2, err := basicKind(v2)
	if err != nil {
		return false, err
	}
	if k1 != k2 {
		return false, errBadComparison
	}
	truth := false
	switch k1 {
	case boolKind, complexKind:
		return false, errBadComparisonType
	case floatKind:
		truth = v1.Float() < v2.Float()
	case intKind:
		truth = v1.Int() < v2.Int()
	case stringKind:
		truth = v1.String() < v2.String()
	case uintKind:
		truth = v1.Uint() < v2.Uint()
	default:
		panic("invalid kind")
	}
	return truth, nil
}

// le evaluates the comparison <= b.
func le(arg1, arg2 interface{}) (bool, error) {
	// <= is < or ==.
	lessThan, err := lt(arg1, arg2)
	if lessThan || err != nil {
		return lessThan, err
	}
	return eq(arg1, arg2)
}

// gt evaluates the comparison a > b.
func gt(arg1, arg2 interface{}) (bool, error) {
	// > is the inverse of <=.
	lessOrEqual, err := le(arg1, arg2)
	if err != nil {
		return false, err
	}
	return !lessOrEqual, nil
}

func getCol(cols map[string]*schemas.Column, name string) *schemas.Column {
	return cols[strings.ToLower(name)]
}

func formatGo(src []byte) ([]byte, error) {
	source, err := format.Source(src)
	if err != nil {
		return nil, err
	}
	return source, nil
}

func typestring(col *schemas.Column) string {
	st := col.SQLType
	t := schemas.SQLType2Type(st)
	s := t.String()
	if s == "[]uint8" {
		return "[]byte"
	}
	if col.Nullable {
		switch t.Kind() {
		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint8, reflect.Uint16:
			return "sql.NullInt32"
		case reflect.Int, reflect.Int64, reflect.Uint32, reflect.Uint64:
			return "sql.NullInt64"
		case reflect.Bool:
			return "sql.NullBool"
		case reflect.Float32, reflect.Float64:
			return "sql.NullFloat64"
		case reflect.String:
			return "sql.NullString"
		}
		if t.String() == "time.Time" {
			return "sql.NullTime"
		}
	}
	return s
}

func include(source []string, target string) bool {
	for _, s := range source {
		if s == target {
			return true
		}
	}
	return false
}
