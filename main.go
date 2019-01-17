package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"github.com/cznic/golex/lex"
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
	"go/token"
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

func ignoreErrors(token.Pos, string) {}

func newParser(source io.Reader, name string) parser.Parser {
	switch php {
	case 5:
		p := php5.NewParser(source, name)
		if !debug {
			lex.ErrorFunc(ignoreErrors)(p.Lexer.Lexer)
		}
		return p
	case 7:
		p := php7.NewParser(source, name)
		if !debug {
			lex.ErrorFunc(ignoreErrors)(p.Lexer.Lexer)
		}
		return p
	default:
		return nil
	}
}

type convert struct {
	buf *bytes.Buffer
	pos parser.Positions
}

func (c *convert) toSpan(n node.Node) *[2]int {
	pos := c.pos[n]
	return &[2]int{pos.StartPos - 1, pos.EndPos - 1}
}

func (c *convert) getContent(n node.Node) string {
	pos := c.pos[n]
	if pos == nil {
		return ""
	}
	return c.buf.String()[pos.StartPos-1 : pos.EndPos]
}

// helper function for seek
func skipComment(buf []byte, start int, end int) int {
	if start+1 >= end {
		return start
	}
	offset := 0
	length := 0
	switch true {
	case buf[start] == '#':
		offset = bytes.IndexByte(buf[start:end+1], byte('\n'))
		length = 1
	case buf[start] == '/' && buf[start+1] == '/':
		offset = bytes.IndexByte(buf[start:end+1], byte('\n'))
		length = 1
	case buf[start] == '/' && buf[start+1] == '*':
		offset = bytes.Index(buf[start:end+1], []byte("*/"))
		length = 2
	}
	if offset == -1 {
		return end
	}
	return start + offset + length
}

func (c *convert) seek(it byte, start, stop int) int {
	buf := c.buf.Bytes()
	if start > stop {
		tmp := stop
		stop = start
		start = tmp
	}

	// look for it
	for i := start; i != stop; i++ {
		i = skipComment(buf, i, stop)
		if buf[i] == it {
			return i + 1
		}
	}

	// look for newline
	for i := start; i != stop; i++ {
		i = skipComment(buf, i, stop)
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
	if len(name) > 0 {
		return name[1:]
	} else {
		return ""
	}
}

func (c *convert) getName(n node.Node) string {
	switch v := n.(type) {
	case *stmt.AltIf:
		return c.getContent(v.Cond)

	case *stmt.AltFor:
		return c.getContentList(v.Init) + "; " + c.getContentList(v.Cond) + "; " + c.getContentList(v.Loop)

	case *stmt.AltForeach:
		return c.getContent(v.Expr) + " as " + c.getContent(v.Variable)

	case *stmt.AltSwitch:
		return c.getContent(v.Cond)

	case *stmt.AltWhile:
		return c.getContent(v.Cond)

	case *expr.ArrayDimFetch:
		return c.getContent(n)

	case *assign.Assign:
		return c.getName(v.Variable)

	case *stmt.Catch:
		return c.getContentList(v.Types)

	case *stmt.Class:
		return c.getName(v.ClassName)

	case *stmt.ClassConstList:
		return c.getNameList(v.Consts)

	case *stmt.ClassMethod:
		return c.getName(v.MethodName)

	case *assign.Concat:
		return c.getName(v.Variable)

	case *stmt.ConstList:
		return c.getNameList(v.Consts)

	case *stmt.Constant:
		return c.getName(v.ConstantName)

	case *expr.Die:
		return c.getContent(v.Expr)

	case *stmt.Do:
		return c.getContent(v.Cond)

	case *stmt.Echo:
		return c.getContentList(v.Exprs)

	case *expr.Exit:
		return c.getContent(v.Expr)

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

	case *stmt.Goto:
		return c.getName(v.Label)

	case *stmt.If:
		return c.getContent(v.Cond)

	case *node.Identifier:
		return v.Value

	case *expr.Include:
		return c.getContent(v.Expr)

	case *expr.IncludeOnce:
		return c.getContent(v.Expr)

	case *stmt.Interface:
		return c.getName(v.InterfaceName)

	case *stmt.Label:
		return c.getName(v.LabelName)

	case *name.Name:
		return c.getNameList(v.Parts)

	case *name.NamePart:
		return v.Value

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

	case *expr.StaticCall:
		return c.getContent(v.Class) + "::" + c.getName(v.Call)

	case *stmt.Switch:
		return c.getContent(v.Cond)

	case *expr.Ternary:
		return c.getContent(v.Condition)

	case *stmt.Unset:
		return c.getNameList(v.Vars)

	case *stmt.Use:
		return c.getContent(v.Use)

	case *stmt.UseList:
		return c.getNameList(v.Uses)

	case *expr.Variable:
		return c.getName(v.VarName)

	case *stmt.While:
		return c.getContent(v.Cond)

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
	case *stmt.AltIf:
		r.Kind = "if"

	case *stmt.AltFor:
		r.Kind = "for"

	case *stmt.AltForeach:
		r.Kind = "foreach"

	case *stmt.AltSwitch:
		r.Kind = "switch"

	case *stmt.AltWhile:
		r.Kind = "while"

	case *assign.Assign:
		r.Kind = "assign"

	case *stmt.Catch:
		r.Kind = "catch"

	case *stmt.Class:
		r.Kind = "class"
		contained = v.Stmts

	case *stmt.ClassConstList:
		r.Kind = "constant"

	case *stmt.ClassMethod:
		r.Kind = "method"

	case *assign.Concat:
		r.Kind = "concat"

	case *stmt.ConstList:
		r.Kind = "constant"

	case *expr.Die:
		r.Kind = "die"

	case *stmt.Do:
		r.Kind = "do"

	case *stmt.Expression:
		r = c.toNode(v.Expr)

	case *stmt.Echo:
		r.Kind = "echo"

	case *expr.Exit:
		r.Kind = "exit"

	case *expr.ErrorSuppress:
		r = c.toNode(v.Expr)

	case *stmt.Finally:
		r.Kind = "finally"

	case *stmt.Foreach:
		r.Kind = "foreach"

	case *stmt.For:
		r.Kind = "for"

	case *stmt.Function:
		r.Kind = "function"

	case *expr.FunctionCall:
		r.Kind = "call_function"

	case *stmt.Global:
		r.Kind = "global"

	case *stmt.Goto:
		r.Kind = "goto"

	case *stmt.HaltCompiler:
		r.Kind = "halt_compiler"

	case *stmt.If:
		r.Kind = "if"

	case *expr.Include:
		r.Kind = "include"

	case *expr.IncludeOnce:
		r.Kind = "include_once"

	case *stmt.InlineHtml:
		r.Kind = "inline_text"

	case *stmt.Interface:
		r.Kind = "interface"
		contained = v.Stmts

	case *stmt.Label:
		r.Kind = "label"

	case *expr.MethodCall:
		r.Kind = "call_method"

	case *stmt.Namespace:
		r.Kind = "namespace"
		contained = v.Stmts

	case *expr.Print:
		r.Kind = "print"

	case *stmt.PropertyList:
		r.Kind = "properties"

	case *expr.Require:
		r.Kind = "require"

	case *expr.RequireOnce:
		r.Kind = "require_once"

	case *expr.StaticCall:
		r.Kind = "call_static"

	case *stmt.Switch:
		r.Kind = "switch"

	case *expr.Ternary:
		r.Kind = "ternary"

	case *stmt.Try:
		r.Kind = "try"
		contained = append(v.Catches, v.Finally)

	case *stmt.Unset:
		r.Kind = "unset"

	case *stmt.UseList:
		r.Kind = "use"

	case *stmt.While:
		r.Kind = "while"

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
		if stmt == nil {
			continue
		}
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
	v := root.(*node.Root)

	children := []ast.Node{}
	for _, stmt := range v.Stmts {
		t := c.toNode(stmt)
		if t != nil {
			children = append(children, *t)
		}
	}

	// insert the header if necessary
	if len(c.buf.Bytes()) != 0 && len(children) != 0 && children[0].Span[0] != 0 {
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

func Parse(source []byte, name string) (*ast.File, error) {
	c := convert{}
	c.buf = bytes.NewBuffer(source)
	parse := newParser(bytes.NewReader(source), name)
	if parse == nil {
		return nil, errors.New("invalid php version")
	}

	parse.Parse()
	parseErrors := parse.GetErrors()
	tree := parse.GetRootNode()
	c.pos = parse.GetPositions()
	var file *ast.File
	if tree == nil {
		if len(parseErrors) == 0 {
			return nil, errors.New("parser returned nil with no parse errors")
		}
		file = &ast.File{Kind: "file"}
	} else {
		file = c.toFile(tree)
	}
	file.Name = name
	file.Numbering = ast.NumberingBytes

	file.ParsingErrorsDetected = len(parseErrors) > 0
	// we can only use the first parsing error...
	if file.ParsingErrorsDetected {
		file.ParsingError = &ast.ParsingError{
			Position: parseErrors[0].Pos.StartPos,
			Message:  parseErrors[0].Msg,
		}
	}

	return file, nil
}

func main() {
	flag.Parse()
	api.Run(Parse)
}
