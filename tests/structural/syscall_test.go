package structural

// An address handed to a native library is converted to uintptr only in the argument list of the call
// that crosses into it, where Go keeps it as a pointer up to the moment of the call. Converted any
// earlier, it is a bare number that a moving goroutine stack leaves pointing at the old copy, which
// the library then reads and writes. That broke a build on 2026-09-14 inside the speech model,
// reproduced the same day by making lines while collections shrank stacks. The calls that cross are
// syscall.SyscallN (marked //go:uintptrkeepalive in Go's source), purego.SyscallN and any function
// named Call; nativelib.Call and the Call method of the procedures x/sys/windows loads are both marked
// //go:uintptrescapes. A pointer already held as unsafe.Pointer and converted with no unsafe.Pointer in
// the expression is not seen here; the speech model's pinned tensors are that case.
//
// A function taking ...uintptr carries addresses across a Go call, so it is allowed only where it is
// marked //go:uintptrescapes. Measured on 2026-09-16 with the compiler's escape analysis over
// speechmodel: with the mark on nativelib.Call, every local whose address crosses was moved to the
// heap; with it replaced by //go:noinline, none was. The stress test in speechmodel passed both ways,
// so this test is the only thing that notices the mark gone.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// uintptrEscapes is the directive that makes a function taking ...uintptr safe to hand addresses to.
const uintptrEscapes = "//go:uintptrescapes"

// TestAddressesAreConvertedOnlyWhereTheCallIsMade fails on uintptr(unsafe.Pointer(...)) anywhere but
// the argument list of a call into a native library. It also fails on any function taking ...uintptr
// that is not marked //go:uintptrescapes.
func TestAddressesAreConvertedOnlyWhereTheCallIsMade(t *testing.T) {
	for _, file := range goFiles(t) {
		fileSet := token.NewFileSet()
		parsed, err := parser.ParseFile(fileSet, file, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parsing %s: %v", file, err)
		}
		crossing := map[ast.Expr]bool{}
		marked := map[*ast.FuncType]bool{}
		ast.Inspect(parsed, func(node ast.Node) bool {
			switch found := node.(type) {
			case *ast.CallExpr:
				if crossesIntoALibrary(found) {
					for _, argument := range found.Args {
						crossing[argument] = true
					}
				}
			case *ast.FuncDecl:
				marked[found.Type] = escapesUintptrs(found.Doc)
			}
			return true
		})
		ast.Inspect(parsed, func(node ast.Node) bool {
			switch found := node.(type) {
			case *ast.CallExpr:
				if addressAsUintptr(found) && !crossing[found] {
					t.Errorf("%s: an address is converted to uintptr outside the call into the library",
						fileSet.Position(found.Pos()))
				}
			case *ast.FuncType:
				if takesUintptrs(found) && !marked[found] {
					t.Errorf("%s: a function takes ...uintptr without %s, which carries addresses across a Go call",
						fileSet.Position(found.Pos()), uintptrEscapes)
				}
			}
			return true
		})
	}
}

// crossesIntoALibrary reports whether call is syscall.SyscallN, purego.SyscallN or a function or
// method named Call.
func crossesIntoALibrary(call *ast.CallExpr) bool {
	if name, ok := call.Fun.(*ast.Ident); ok {
		return name.Name == "Call"
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if selector.Sel.Name == "Call" {
		return true
	}
	pkg, ok := selector.X.(*ast.Ident)
	return ok && (pkg.Name == "syscall" || pkg.Name == "purego") && selector.Sel.Name == "SyscallN"
}

// escapesUintptrs reports whether a declaration's comments carry the //go:uintptrescapes directive.
func escapesUintptrs(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}
	for _, comment := range doc.List {
		if strings.TrimSpace(comment.Text) == uintptrEscapes {
			return true
		}
	}
	return false
}

// addressAsUintptr reports whether call is uintptr(unsafe.Pointer(...)).
func addressAsUintptr(call *ast.CallExpr) bool {
	conversion, ok := call.Fun.(*ast.Ident)
	if !ok || conversion.Name != "uintptr" || len(call.Args) != 1 {
		return false
	}
	inner, ok := call.Args[0].(*ast.CallExpr)
	if !ok {
		return false
	}
	selector, ok := inner.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := selector.X.(*ast.Ident)
	return ok && pkg.Name == "unsafe" && selector.Sel.Name == "Pointer"
}

// takesUintptrs reports whether a function type's last parameter is ...uintptr.
func takesUintptrs(function *ast.FuncType) bool {
	if function.Params == nil || len(function.Params.List) == 0 {
		return false
	}
	variadic, ok := function.Params.List[len(function.Params.List)-1].Type.(*ast.Ellipsis)
	if !ok {
		return false
	}
	element, ok := variadic.Elt.(*ast.Ident)
	return ok && element.Name == "uintptr"
}
