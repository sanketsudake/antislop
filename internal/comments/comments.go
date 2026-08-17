// Package comments finds the "// SAFETY:" justifications that antislop
// requires in front of an escape hatch, and the //go:linkname directives that
// need one. A justification counts only where a reader would look for it: on
// the statement that owns the expression, or inline just before it. A doc
// comment on the enclosing function is too far away to say which invariant a
// single line relies on.
package comments

import (
	"go/ast"
	"go/token"
	"regexp"

	"golang.org/x/tools/go/analysis"
)

// safetyRe matches the marker of a justification comment.
var safetyRe = regexp.MustCompile(`\bSAFETY\s*:`)

// Safety caches, per file, the comment groups that carry a SAFETY marker.
// Create one per analyzer run: Safety{}.
type Safety map[*ast.File][]*ast.CommentGroup

// Has reports whether node is justified by a SAFETY comment. stack is the
// node stack from an inspector.WithStack callback, with the file at stack[0].
// A comment justifies node when it ends before node starts and either
//
//   - it sits on the line above a statement or declaration that owns node,
//     searched outwards up to (but not into) the enclosing function, or
//   - it ends on node's own line, as in /* SAFETY: ... */ x.(T).
//
// A justification must also stand inside the body of the function that holds
// node. A comment before the opening brace is a doc comment on the function,
// or belongs to an outer statement, and explains other code -- which stays
// true when the body is written on one line.
func (s Safety) Has(pass *analysis.Pass, node ast.Node, stack []ast.Node) bool {
	if len(stack) == 0 {
		return false
	}
	file, ok := stack[0].(*ast.File)
	if !ok {
		return false
	}
	groups := s.groups(file)
	if len(groups) == 0 {
		return false
	}
	body := bodyStart(stack)
	reaches := func(g *ast.CommentGroup) bool { return g.End() < node.Pos() && g.Pos() > body }
	for _, g := range groups {
		if reaches(g) && line(pass, g.End()) == line(pass, node.Pos()) {
			return true
		}
	}
	for i := len(stack) - 1; i >= 0; i-- {
		switch stack[i].(type) {
		case *ast.FuncDecl, *ast.FuncLit:
			return false // the function's own doc comment is too far away
		case *ast.BlockStmt:
			continue // a block starts at "{", so its line belongs to the header
		case ast.Stmt, *ast.GenDecl, *ast.ValueSpec:
		default:
			continue
		}
		owner := line(pass, stack[i].Pos())
		for _, g := range groups {
			if reaches(g) && line(pass, g.End()) == owner-1 {
				return true
			}
		}
	}
	return false
}

// bodyStart returns the position of the opening brace of the innermost
// function body on stack, or token.NoPos when the node sits at file scope and
// no body encloses it.
func bodyStart(stack []ast.Node) token.Pos {
	for i := len(stack) - 1; i >= 0; i-- {
		switch fn := stack[i].(type) {
		case *ast.FuncDecl:
			if fn.Body == nil {
				return token.NoPos
			}
			return fn.Body.Lbrace
		case *ast.FuncLit:
			return fn.Body.Lbrace
		}
	}
	return token.NoPos
}

// HasForDirective reports whether the directive comment c is justified: by a
// SAFETY comment earlier in its own comment group, or by a group ending on the
// line directly above the group c belongs to.
func (s Safety) HasForDirective(pass *analysis.Pass, file *ast.File, c *ast.Comment) bool {
	own := groupOf(file, c)
	if own == nil {
		return false
	}
	for _, entry := range own.List {
		if entry.Pos() < c.Pos() && safetyRe.MatchString(entry.Text) {
			return true
		}
	}
	above := line(pass, own.Pos()) - 1
	for _, g := range s.groups(file) {
		if g.End() < c.Pos() && line(pass, g.End()) == above {
			return true
		}
	}
	return false
}

// LinknameDirectives returns the //go:linkname directive comments of file.
func LinknameDirectives(file *ast.File) []*ast.Comment {
	var out []*ast.Comment
	for _, g := range file.Comments {
		for _, c := range g.List {
			d, ok := ast.ParseDirective(c.Slash, c.Text)
			if ok && d.Tool == "go" && d.Name == "linkname" {
				out = append(out, c)
			}
		}
	}
	return out
}

func (s Safety) groups(file *ast.File) []*ast.CommentGroup {
	if groups, cached := s[file]; cached {
		return groups
	}
	groups := []*ast.CommentGroup{}
	for _, g := range file.Comments {
		for _, c := range g.List {
			if safetyRe.MatchString(c.Text) {
				groups = append(groups, g)
				break
			}
		}
	}
	s[file] = groups
	return groups
}

func groupOf(file *ast.File, c *ast.Comment) *ast.CommentGroup {
	for _, g := range file.Comments {
		if g.Pos() <= c.Pos() && c.End() <= g.End() {
			return g
		}
	}
	return nil
}

func line(pass *analysis.Pass, pos token.Pos) int {
	return pass.Fset.Position(pos).Line
}
