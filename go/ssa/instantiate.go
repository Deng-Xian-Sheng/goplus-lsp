// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssa

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/Deng-Xian-Sheng/goplus-lsp/internal/typeparams"
)

// Instances returns all of the instances generated by runtime types for this function in an unspecified order.
//
// Thread-safe.
//
// This is an experimental interface! It may change without warning.
func (prog *Program) _Instances(fn *Function) []*Function {
	if fn.typeparams.Len() == 0 || len(fn.typeargs) > 0 {
		return nil
	}

	prog.methodsMu.Lock()
	defer prog.methodsMu.Unlock()
	return prog.instances[fn].list()
}

// A set of instantiations of a generic function fn.
type instanceSet struct {
	fn        *Function               // fn.typeparams.Len() > 0 and len(fn.typeargs) == 0.
	instances map[*typeList]*Function // canonical type arguments to an instance.
	syntax    *ast.FuncDecl           // fn.syntax copy for instantiating after fn is done. nil on synthetic packages.
	info      *types.Info             // fn.pkg.info copy for building after fn is done.. nil on synthetic packages.

	// TODO(taking): Consider ways to allow for clearing syntax and info when done building.
	// May require a public API change as MethodValue can request these be built after prog.Build() is done.
}

func (insts *instanceSet) list() []*Function {
	if insts == nil {
		return nil
	}

	fns := make([]*Function, 0, len(insts.instances))
	for _, fn := range insts.instances {
		fns = append(fns, fn)
	}
	return fns
}

// createInstanceSet adds a new instanceSet for a generic function fn if one does not exist.
//
// Precondition: fn is a package level declaration (function or method).
//
// EXCLUSIVE_LOCKS_ACQUIRED(prog.methodMu)
func (prog *Program) createInstanceSet(fn *Function) {
	assert(fn.typeparams.Len() > 0 && len(fn.typeargs) == 0, "Can only create instance sets for generic functions")

	prog.methodsMu.Lock()
	defer prog.methodsMu.Unlock()

	syntax, _ := fn.syntax.(*ast.FuncDecl)
	assert((syntax == nil) == (fn.syntax == nil), "fn.syntax is either nil or a *ast.FuncDecl")

	if _, ok := prog.instances[fn]; !ok {
		prog.instances[fn] = &instanceSet{
			fn:     fn,
			syntax: syntax,
			info:   fn.info,
		}
	}
}

// needsInstance returns a Function that is the instantiation of fn with the type arguments targs.
//
// Any CREATEd instance is added to cr.
//
// EXCLUSIVE_LOCKS_ACQUIRED(prog.methodMu)
func (prog *Program) needsInstance(fn *Function, targs []types.Type, cr *creator) *Function {
	prog.methodsMu.Lock()
	defer prog.methodsMu.Unlock()

	return prog.lookupOrCreateInstance(fn, targs, cr)
}

// lookupOrCreateInstance returns a Function that is the instantiation of fn with the type arguments targs.
//
// Any CREATEd instance is added to cr.
//
// EXCLUSIVE_LOCKS_REQUIRED(prog.methodMu)
func (prog *Program) lookupOrCreateInstance(fn *Function, targs []types.Type, cr *creator) *Function {
	return prog.instances[fn].lookupOrCreate(targs, &prog.parameterized, cr)
}

// lookupOrCreate returns the instantiation of insts.fn using targs.
// If the instantiation is created, this is added to cr.
func (insts *instanceSet) lookupOrCreate(targs []types.Type, parameterized *tpWalker, cr *creator) *Function {
	if insts.instances == nil {
		insts.instances = make(map[*typeList]*Function)
	}

	fn := insts.fn
	prog := fn.Prog

	// canonicalize on a tuple of targs. Sig is not unique.
	//
	// func A[T any]() {
	//   var x T
	//   fmt.Println("%T", x)
	// }
	key := prog.canon.List(targs)
	if inst, ok := insts.instances[key]; ok {
		return inst
	}

	// CREATE instance/instantiation wrapper
	var syntax ast.Node
	if insts.syntax != nil {
		syntax = insts.syntax
	}

	var sig *types.Signature
	var obj *types.Func
	if recv := fn.Signature.Recv(); recv != nil {
		// method
		m := fn.object.(*types.Func)
		obj = prog.canon.instantiateMethod(m, targs, prog.ctxt)
		sig = obj.Type().(*types.Signature)
	} else {
		instSig, err := typeparams.Instantiate(prog.ctxt, fn.Signature, targs, false)
		if err != nil {
			panic(err)
		}
		instance, ok := instSig.(*types.Signature)
		if !ok {
			panic("Instantiate of a Signature returned a non-signature")
		}
		obj = fn.object.(*types.Func) // instantiation does not exist yet
		sig = prog.canon.Type(instance).(*types.Signature)
	}

	var synthetic string
	var subst *subster

	concrete := !parameterized.anyParameterized(targs)

	if prog.mode&InstantiateGenerics != 0 && concrete {
		synthetic = fmt.Sprintf("instance of %s", fn.Name())
		subst = makeSubster(prog.ctxt, fn.typeparams, targs, false)
	} else {
		synthetic = fmt.Sprintf("instantiation wrapper of %s", fn.Name())
	}

	name := fmt.Sprintf("%s%s", fn.Name(), targs) // may not be unique
	instance := &Function{
		name:           name,
		object:         obj,
		Signature:      sig,
		Synthetic:      synthetic,
		syntax:         syntax,
		topLevelOrigin: fn,
		pos:            obj.Pos(),
		Pkg:            nil,
		Prog:           fn.Prog,
		typeparams:     fn.typeparams, // share with origin
		typeargs:       targs,
		info:           insts.info, // on synthetic packages info is nil.
		subst:          subst,
	}

	cr.Add(instance)
	insts.instances[key] = instance
	return instance
}
