// Copyright 2017 The panchangtao Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/panchangtao/wagon/exec"
	"github.com/panchangtao/wagon/validate"
	"github.com/panchangtao/wagon/wasm"
	"reflect"
)

func main() {
	log.SetPrefix("wasm-run: ")
	log.SetFlags(0)

	verbose := flag.Bool("v", false, "enable/disable verbose mode")
	verify := flag.Bool("verify-module", false, "run module verification")

	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		flag.PrintDefaults()
		os.Exit(1)
	}

	wasm.SetDebugMode(*verbose)

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	m, err := wasm.ReadModule(f, mImporter)
	if err != nil {
		log.Fatalf("could not read module: %v", err)
	}

	if *verify {
		err = validate.VerifyModule(m)
		if err != nil {
			log.Fatalf("could not verify module: %v", err)
		}
	}

	if m.Export == nil {
		log.Fatalf("module has no export section")
	}

	vm, err := exec.NewVM(m)
	if err != nil {
		log.Fatalf("could not create VM: %v", err)
	}

	for name, e := range m.Export.Entries {
		if e.Kind != 0 {
			continue
		}
		i := int64(e.Index)
		fidx := m.Function.Types[int(i)]
		ftype := m.Types.Entries[int(fidx)]
		//data := m.Data.Entries[int(i)].Data
		switch len(ftype.ReturnTypes) {
		case 1:
			fmt.Printf("%s() %s => ", name, ftype.ReturnTypes[0])
		case 0:
			fmt.Printf("%s() => ", name)
		default:
			log.Printf("running exported functions with more than one return value is not supported")
			continue
		}
		if len(ftype.ParamTypes) > 0 {
			log.Printf("running exported functions with input parameters is not supported")
			continue
		}

		o, err := vm.ExecCode(i)
		if err != nil {
			fmt.Printf("\n")
			log.Printf("err=%v", err)
		}
		if len(ftype.ReturnTypes) == 0 {
			fmt.Printf("\n")
			continue
		}
		fmt.Printf("%[1]v (%[1]T)\n", o)
	}
}

func importer(name string) (*wasm.Module, error) {
	f, err := os.Open(name + ".wasm")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	m, err := wasm.ReadModule(f, nil)
	if err != nil {
		return nil, err
	}
	err = validate.VerifyModule(m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func add3(x int32) int32 {
	fmt.Println("add3 call")
	return x + 3
}
func add5(x int32) int32 {
	fmt.Println("add5 call")
	return x + 5
}

func Println(p int32){
	fmt.Println("Println call")
	fmt.Println(p)
}

func mImporter(name string) (*wasm.Module, error) {
	fmt.Println("import name:", name)
	m := wasm.NewModule()
	m.Types = &wasm.SectionTypes{
		// List of all function types available in this module.
		// There is only one: (func [int32] -> [int32])
		Entries: []wasm.FunctionSig{
			{
				Form:        0,
				ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32},
				ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
			},
			{
				Form:        0,
				ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32},
				ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
			},
			{
				Form:        0,
				ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32},
				ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
			},
		},
	}
	m.FunctionIndexSpace = []wasm.Function{
		{
			Sig:  &m.Types.Entries[0],
			Host: reflect.ValueOf(add3),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &m.Types.Entries[1],
			Host: reflect.ValueOf(add5),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &m.Types.Entries[2],
			Host: reflect.ValueOf(Println),
			Body: &wasm.FunctionBody{},
		},
	}
	m.Export = &wasm.SectionExports{
		Entries: map[string]wasm.ExportEntry{
			"add3": {
				FieldStr: "add3",
				Kind:     wasm.ExternalFunction,
				Index:    0,
			},
			"add5": {
				FieldStr: "add5",
				Kind:     wasm.ExternalFunction,
				Index:    1,
			},
			"Println": {
				FieldStr: "Println",
				Kind:     wasm.ExternalFunction,
				Index:    2,
			},
		},
	}

	return m, nil
}

