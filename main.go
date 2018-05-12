package main

import (
	"bytes"
	"errors"
	"flag"
	"github.com/imuli/go-semantic/api"
	"github.com/imuli/go-semantic/ast"
	"github.com/z7zmey/php-parser/node"
	"github.com/z7zmey/php-parser/node/expr"
	"github.com/z7zmey/php-parser/node/stmt"
	"github.com/z7zmey/php-parser/parser"
	"github.com/z7zmey/php-parser/php5"
	"github.com/z7zmey/php-parser/php7"
	"github.com/z7zmey/php-parser/position"
	"io"
)

var php int

func init() {
	flag.IntVar(&php, "php", 7, "parse using php `version` (5 or 7)")
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
}

func (c *convert) toSpan(n node.Node) *[2]int {
	pos := c.pos[n]
	return &[2]int{pos.StartPos - 1, pos.EndPos}
}

func (c *convert) getContent(n node.Node) string {
	pos := c.pos[n]
	return c.buf.String()[pos.StartPos-1:pos.EndPos]
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

func (c *convert) getName(n node.Node) string {
	switch v := n.(type) {
	case *stmt.Class:
		return c.getName(v.ClassName)

	case *stmt.ClassMethod:
		return c.getName(v.MethodName)

	case *stmt.Function:
		return c.getName(v.FunctionName)

	case *stmt.Global:
		return c.getNameList(v.Vars)

	case *node.Identifier:
		return v.Value

	case *stmt.Property:
		return c.getName(v.Variable)

	case *stmt.PropertyList:
		return c.getNameList(v.Properties)

	case *expr.Require:
		return c.getContent(v.Expr)

	case *expr.RequireOnce:
		return c.getContent(v.Expr)

	case *expr.Variable:
		return c.getName(v.VarName)

	default:
		return ""
	}
}

func (c *convert) toNode(n node.Node) *ast.Node {

	r := ast.Node{
		Span: c.toSpan(n),
		Name: c.getName(n),
	}
	contained := []node.Node{} // containers set this for recursion

	switch v := n.(type) {
	case *stmt.Class:
		r.Kind = "class"
		contained = v.Stmts

	case *stmt.ClassMethod:
		r.Kind = "method"

	case *stmt.Expression:
		expr := c.toNode(v.Expr)
		if expr == nil {
			return nil
		}
		expr.Span = r.Span
		r = *expr

	case *stmt.Function:
		r.Kind = "function"

	case *stmt.Global:
		r.Kind = "global"

	case *stmt.PropertyList:
		r.Kind = "properties"

	case *expr.Require:
		r.Kind = "require"

	case *expr.RequireOnce:
		r.Kind = "require_once"

	case *stmt.StmtList:
		r.Kind = "statement_list"
		contained = v.Stmts

	default:
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

	return &r
}

func (c *convert) toFile(root node.Node) ast.File {
	v := root.(*stmt.StmtList)

	children := []ast.Node{}
	for _, stmt := range v.Stmts {
		t := c.toNode(stmt)
		if t != nil {
			children = append(children, *t)
		}
	}

	return ast.File{
		Kind:     "file",
		Children: children,
	}
}

func Parse(source io.Reader, name string) (ast.File, error) {
	c := convert{}
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
	file = *ast.CleanFile(&file, ast.MakeLines(c.buf.Bytes()))

	return file, nil
}

func main() {
	flag.Parse()
	api.Run(Parse)
}
