package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"github.com/imuli/go-semantic/api"
	"github.com/imuli/go-semantic/ast"
	"github.com/z7zmey/php-parser/node"
	"github.com/z7zmey/php-parser/node/expr"
	"github.com/z7zmey/php-parser/node/expr/assign"
	"github.com/z7zmey/php-parser/node/name"
	"github.com/z7zmey/php-parser/node/stmt"
	"github.com/z7zmey/php-parser/parser"
	"github.com/z7zmey/php-parser/php5"
	"github.com/z7zmey/php-parser/php7"
	"github.com/z7zmey/php-parser/position"
	"golang.org/x/text/encoding"
	"io"
	"os"
	"strings"
)

var php int
var debug bool

func init() {
	flag.IntVar(&php, "php", 7, "parse using php `version` (5 or 7)")
	flag.BoolVar(&debug, "debug", false, "log extra parse info to stderr")
}

func newParser(source io.Reader, name string) parser.Parser {
	switch php {
	case 5:
		return php5.NewParser(source, name)
	case 7:
		return php7.NewParser(source, name)
	default:
		return nil
	}
}

type convert struct {
	buf   bytes.Buffer
	pos   position.Positions
	lines []int // offsets of lines
	decoder *encoding.Decoder
}

func (c *convert) toSpan(n node.Node) *[2]int {
	pos := c.pos[n]
	return &[2]int{pos.StartPos - 1, pos.EndPos - 1}
}

func (c *convert) decode(str string) string {
	utf, err := c.decoder.String(str)
	if err != nil {
		return str
	}
	return utf
}

func (c *convert) getContent(n node.Node) string {
	pos := c.pos[n]
	return c.decode(c.buf.String()[pos.StartPos-1 : pos.EndPos])
}

func (c *convert) seek(it byte, start, stop int) int {
	buf := c.buf.Bytes()
	var dir int
	if start > stop {
		dir = -1
	} else {
		dir = 1
	}

	// look for it
	for i := start; i != stop; i += dir {
		if buf[i] == it {
			return i + 1
		}
	}

	// look for newline
	for i := start; i != stop; i += dir {
		if buf[i] == '\n' {
			return i + 1
		}
	}

	return stop
}

func (c *convert) getNameList(ns []node.Node) string {
	name := ""
	for _, prop := range ns {
		name = name + "," + c.getName(prop)
	}
	return name[1:]
}

func (c *convert) getContentList(ns []node.Node) string {
	name := ""
	for _, prop := range ns {
		name = name + "," + c.getContent(prop)
	}
	return name[1:]
}

func (c *convert) getName(n node.Node) string {
	switch v := n.(type) {
	case *expr.ArrayDimFetch:
		return c.getContent(n)

	case *assign.Assign:
		return c.getName(v.Variable)

	case *stmt.Class:
		return c.getName(v.ClassName)

	case *stmt.ClassConstList:
		return c.getNameList(v.Consts)

	case *stmt.ClassMethod:
		return c.getName(v.MethodName)

	case *stmt.ConstList:
		return c.getNameList(v.Consts)

	case *stmt.Constant:
		return c.getName(v.ConstantName)

	case *stmt.Echo:
		return c.getContentList(v.Exprs)

	case *expr.Print:
		return c.getContent(v.Expr)

	case *stmt.Function:
		return c.getName(v.FunctionName)

	case *stmt.For:
		return c.getContentList(v.Init) + "; " + c.getContentList(v.Cond) + "; " + c.getContentList(v.Loop)

	case *stmt.Foreach:
		return c.getContent(v.Expr) + " as " + c.getContent(v.Variable)

	case *expr.FunctionCall:
		return c.getName(v.Function)

	case *expr.MethodCall:
		return c.getContent(v.Variable) + "->" + c.getName(v.Method)

	case *stmt.Global:
		return c.getNameList(v.Vars)

	case *stmt.If:
		return c.getContent(v.Cond)

	case *node.Identifier:
		return c.decode(v.Value)

	case *name.Name:
		return c.getNameList(v.Parts)

	case *name.NamePart:
		return c.decode(v.Value)

	case *stmt.Namespace:
		return c.getContent(v.NamespaceName)

	case *stmt.Property:
		return c.getName(v.Variable)

	case *stmt.PropertyList:
		return c.getNameList(v.Properties)

	case *expr.Require:
		return c.getContent(v.Expr)

	case *expr.RequireOnce:
		return c.getContent(v.Expr)

	case *stmt.Unset:
		return c.getNameList(v.Vars)

	case *stmt.Use:
		return c.getContent(v.Use)

	case *stmt.UseList:
		return c.getNameList(v.Uses)

	case *expr.Variable:
		return c.getName(v.VarName)

	default:
		return ""
	}
}

func (c *convert) toNode(n node.Node) *ast.Node {

	r := &ast.Node{
		Span: c.toSpan(n),
		Name: c.getName(n),
	}
	contained := []node.Node{} // containers set this for recursion

	switch v := n.(type) {
	case *assign.Assign:
		r.Kind = "assign"

	case *stmt.Class:
		r.Kind = "class"
		contained = v.Stmts

	case *stmt.ClassConstList:
		r.Kind = "constant"

	case *stmt.ClassMethod:
		r.Kind = "method"

	case *stmt.ConstList:
		r.Kind = "constant"

	case *stmt.Expression:
		r = c.toNode(v.Expr)

	case *stmt.Echo:
		r.Kind = "echo"

	case *expr.Print:
		r.Kind = "print"

	case *expr.ErrorSuppress:
		r = c.toNode(v.Expr)

	case *stmt.Foreach:
		r.Kind = "foreach"

	case *stmt.For:
		r.Kind = "for"

	case *stmt.Function:
		r.Kind = "function"

	case *expr.FunctionCall:
		r.Kind = "call_function"

	case *stmt.If:
		r.Kind = "if"

	case *stmt.Global:
		r.Kind = "global"

	case *stmt.InlineHtml:
		r.Kind = "inline_text"

	case *expr.MethodCall:
		r.Kind = "call_method"

	case *stmt.Namespace:
		r.Kind = "namespace"

	case *stmt.PropertyList:
		r.Kind = "properties"

	case *expr.Require:
		r.Kind = "require"

	case *expr.RequireOnce:
		r.Kind = "require_once"

	case *stmt.StmtList:
		r.Kind = "statement_list"
		contained = v.Stmts

	case *stmt.Unset:
		r.Kind = "unset"

	case *stmt.UseList:
		r.Kind = "use"

	default:
		if debug {
			fmt.Fprintf(os.Stderr, "%T\n", v)
		}
		r = nil
	}

	if r == nil {
		return nil
	}

	for _, stmt := range contained {
		t := c.toNode(stmt)
		if t != nil {
			r.Children = append(r.Children, *t)
		}
	}

	if len(r.Children) > 0 {
		// from the start of the container to the start of the first child
		r.HeaderSpan = &[2]int{r.Span[0], c.seek('{', r.Span[0], r.Children[0].Span[0]-1)}
		// from the end of the last child to the end of the container
		r.FooterSpan = &[2]int{c.seek('}', r.Span[1], r.Children[len(r.Children)-1].Span[1]+1), r.Span[1]}
	}

	return r
}

func (c *convert) toFile(root node.Node) *ast.File {
	v := root.(*stmt.StmtList)

	children := []ast.Node{}
	for _, stmt := range v.Stmts {
		t := c.toNode(stmt)
		if t != nil {
			children = append(children, *t)
		}
	}

	// insert the header if necessary
	if len(c.buf.Bytes()) != 0 && children[0].Span[0] != 0 {
		offset := strings.Index(c.buf.String(), "\n\n")
		if offset < 0 || offset > children[0].Span[0] {
			offset = children[0].Span[0] - 1
		}
		children = append([]ast.Node{{
			Kind: "header",
			Span: &[2]int{0, offset},
		}}, children...)
	}

	return &ast.File{
		Kind:     "file",
		Children: children,
	}
}

func Parse(source io.Reader, name string, code encoding.Encoding) (ast.File, error) {
	c := convert{}
	c.decoder = code.NewDecoder()
	tee := io.TeeReader(source, &c.buf)
	parse := newParser(tee, name)
	if parse == nil {
		return ast.File{}, errors.New("invalid php version")
	}

	parse.Parse()
	tree := parse.GetRootNode()
	c.pos = parse.GetPositions()

	file := c.toFile(tree)
	file.Name = name

	lines := ast.MakeLines(c.buf.Bytes())

	parseErrors := parse.GetErrors()
	file.ParsingErrorsDetected = len(parseErrors) > 0
	// we can only use the first parsing error...
	if file.ParsingErrorsDetected {
		file.ParsingError = &ast.ParsingError{
			Location: [2]int{
				parseErrors[0].Pos.StartLine,
				parseErrors[0].Pos.StartPos - lines[parseErrors[0].Pos.StartLine],
			},
			Message: parseErrors[0].Msg,
		}
	}

	file = ast.CleanFile(file, lines)

	if file == nil {
		return ast.File{}, errors.New("something didn't clean up properly")
	}

	return *file, nil
}

func main() {
	flag.Parse()
	api.Run(Parse)
}
