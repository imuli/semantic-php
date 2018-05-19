package main

import (
	"fmt"
	"github.com/go-yaml/yaml"
	"github.com/imuli/go-semantic/ast"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helper functions for tests

func parse(t *testing.T, source string) ast.File {
	file, err := Parse(strings.NewReader(source), "testing")
	if err != nil {
		t.Errorf("got an error: %v", err)
	}
	if file.Name != "testing" {
		t.Errorf("name was '%s' instead of 'testing'", file.Name)
	}
	if file.FooterSpan[1]+1 != len(source) {
		t.Errorf("file length was %d instead of %d", len(source), file.FooterSpan[1]+1)
	}
	return file
}

func getSpan(source string, span *[2]int) string {
	if span != nil {
		return source[span[0] : span[1]+1]
	} else {
		return ""
	}
}

func testNodeContinuity(t *testing.T, prefix string, n ast.Node, pos int) {
	if n.HeaderSpan != nil {
		if pos+1 != n.HeaderSpan[0] {
			t.Errorf("gap before %s.Header", prefix)
		}
		pos = n.HeaderSpan[1]
	}
	for i, child := range n.Children {
		childPrefix := fmt.Sprintf("%s.Children[%d]", prefix, i)
		testNodeContinuity(t, childPrefix, child, pos)
		if pos+1 != child.Span[0] {
			t.Errorf("gap before %s.Children[%d]", prefix, i)
		}
		pos = child.Span[1]
	}
	if n.HeaderSpan != nil {
		if pos+1 != n.FooterSpan[0] {
			t.Errorf("gap before %s.Footer", prefix)
		}
	}
}

func testFileContinuity(t *testing.T, prefix string, n ast.File) {
	pos := -1
	for i, child := range n.Children {
		childPrefix := fmt.Sprintf("%s.Children[%d]", prefix, i)
		testNodeContinuity(t, childPrefix, child, pos)
		if pos+1 != child.Span[0] {
			t.Errorf("gap before %s.Children[%d]", prefix, i)
		}
		pos = child.Span[1]
	}
	if pos+1 != n.FooterSpan[0] {
		t.Errorf("gap before %s.Footer", prefix)
	}
}

func equalSpan(a *[2]int, b *[2]int) bool {
	return a == b || (a != nil && b != nil && *a == *b)
}

func testNodeEquality(t *testing.T, prefix string, left ast.Node, right ast.Node) {
	if left.Kind != right.Kind {
		t.Errorf("%s.Kind '%s' != '%s'", prefix, left.Kind, right.Kind)
	}
	if left.Name != right.Name {
		t.Errorf("%s.Name '%s' != '%s'", prefix, left.Name, right.Name)
	}
	if left.LocationSpan != right.LocationSpan {
		t.Errorf("%s.LocationSpan %v != %v", prefix, left.LocationSpan, right.LocationSpan)
	}
	if !equalSpan(left.Span, right.Span) {
		t.Errorf("%s.Span %v != %v", prefix, left.Span, right.Span)
	}
	if !equalSpan(left.HeaderSpan, right.HeaderSpan) {
		t.Errorf("%s.HeaderSpan %v != %v", prefix, left.HeaderSpan, right.HeaderSpan)
	}
	if !equalSpan(left.FooterSpan, right.FooterSpan) {
		t.Errorf("%s.FooterSpan %v != %v", prefix, left.FooterSpan, right.FooterSpan)
	}
	if len(left.Children) != len(right.Children) {
		t.Errorf("%s.len(Children) %v != %v", prefix, len(left.Children), len(right.Children))
	}
	for i := 0; i < len(left.Children) && i < len(right.Children); i++ {
		childPrefix := fmt.Sprintf("%s.Children[%d]", prefix, i)
		testNodeEquality(t, childPrefix, left.Children[i], right.Children[i])
	}
}

func testFileEquality(t *testing.T, prefix string, left ast.File, right ast.File) {
	if left.Kind != right.Kind {
		t.Errorf("%s.Kind '%s' != '%s'", prefix, left.Kind, right.Kind)
	}
	if left.Name != right.Name {
		t.Errorf("%s.Name '%s' != '%s'", prefix, left.Name, right.Name)
	}
	if left.LocationSpan != right.LocationSpan {
		t.Errorf("%s.LocationSpan %v != %v", prefix, left.LocationSpan, right.LocationSpan)
	}
	if left.FooterSpan != right.FooterSpan {
		t.Errorf("%s.FooterSpan %v != %v", prefix, left.FooterSpan, right.FooterSpan)
	}
	if left.ParsingErrorsDetected != right.ParsingErrorsDetected {
		t.Errorf("%s.ParsingErrorsDetected %v != %v", prefix, left.ParsingErrorsDetected, right.ParsingErrorsDetected)
	}
	// don't check ParsingError, as it may change with underlying library
	if len(left.Children) != len(right.Children) {
		t.Errorf("%s.len(Children) %v != %v", prefix, len(left.Children), len(right.Children))
	}
	for i := 0; i < len(left.Children) && i < len(right.Children); i++ {
		childPrefix := fmt.Sprintf("%s.Children[%d]", prefix, i)
		testNodeEquality(t, childPrefix, left.Children[i], right.Children[i])
	}
}

// Tests start here

func TestSnippets(t *testing.T) {
	files, _ := filepath.Glob("snippets/*.php")
	for _, file := range files {
		src, _ := os.Open(file)
		tree, err := Parse(src, file[9:])
		src.Close()
		if err != nil {
			t.Errorf("got an error reading %s: %v", file, err)
		}

		testFileContinuity(t, file, tree)

		yml, err := ioutil.ReadFile(strings.TrimSuffix(file, ".php") + ".yml")
		if err == nil {
			var good ast.File
			err = yaml.Unmarshal(yml, &good)
			if err == nil {
				testFileEquality(t, strings.TrimSuffix(file[9:], ".php"), tree, good)
			} else {
				t.Errorf("bad yaml in %s.yml", file)
			}
		}
	}
}

func TestEmpty(t *testing.T) {
	file := parse(t, "")
	if len(file.Children) != 0 {
		t.Errorf("empty file with %d children", len(file.Children))
	}
}
