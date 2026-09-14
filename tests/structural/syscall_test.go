package structural

// An address handed to a Windows DLL is converted to uintptr only in the argument list of the call
// that crosses into the DLL, where Go keeps it as a pointer up to the moment of the call. Converted any
// earlier, it is a bare number that a moving goroutine stack leaves pointing at the old copy, which
// the DLL then reads and writes. That broke a build on 2026-09-14 inside the speech model, reproduced
// the same day by making lines while collections shrank stacks. The calls that cross are
// syscall.SyscallN (marked //go:uintptrkeepalive in Go's source) and the Call method of the procedures
// x/sys/windows loads (marked //go:uintptrescapes there, both read on 2026-09-14); a method named Call
// is taken for the second by its name alone. A pointer already held as unsafe.Pointer and converted with
// no unsafe.Pointer in the expression is not seen here; the speech model's pinned tensors are that case.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// TestAddressesAreConvertedOnlyWhereTheCallIsMade fails on uintptr(unsafe.Pointer(...)) anywhere but
// the argument list of a call into a DLL. It also fails on any function taking ...uintptr, the shape
// that carried the addresses across a Go call.
func TestAddressesAreConvertedOnlyWhereTheCallIsMade(t *testing.T) {
	for _, file := range goFiles(t) {
		fileSet := token.NewFileSet()
		parsed, err := parser.ParseFile(fileSet, file, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", file, err)
		}
		crossing := map[ast.Expr]bool{}
		ast.Inspect(parsed, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok && crossesIntoADLL(call) {
				for _, argument := range call.Args {
					crossing[argument] = true
				}
			}
			return true
		})
		ast.Inspect(parsed, func(node ast.Node) bool {
			switch found := node.(type) {
			case *ast.CallExpr:
				if addressAsUintptr(found) && !crossing[found] {
					t.Errorf("%s: an address is converted to uintptr outside the call into the DLL",
						fileSet.Position(found.Pos()))
				}
			case *ast.FuncType:
				if takesUintptrs(found) {
					t.Errorf("%s: a function takes ...uintptr, which carries addresses across a Go call",
						fileSet.Position(found.Pos()))
				}
			}
			return true
		})
	}
}

// crossesIntoADLL reports whether call is syscall.SyscallN or a method named Call.
func crossesIntoADLL(call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if selector.Sel.Name == "Call" {
		return true
	}
	pkg, ok := selector.X.(*ast.Ident)
	return ok && pkg.Name == "syscall" && selector.Sel.Name == "SyscallN"
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
