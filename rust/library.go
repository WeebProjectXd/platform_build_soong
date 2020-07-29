// Copyright 2019 The Android Open Source Project
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rust

import (
	"regexp"
	"strings"

	"android/soong/android"
)

func init() {
	android.RegisterModuleType("rust_library", RustLibraryFactory)
	android.RegisterModuleType("rust_library_dylib", RustLibraryDylibFactory)
	android.RegisterModuleType("rust_library_rlib", RustLibraryRlibFactory)
	android.RegisterModuleType("rust_library_host", RustLibraryHostFactory)
	android.RegisterModuleType("rust_library_host_dylib", RustLibraryDylibHostFactory)
	android.RegisterModuleType("rust_library_host_rlib", RustLibraryRlibHostFactory)
	android.RegisterModuleType("rust_ffi", RustFFIFactory)
	android.RegisterModuleType("rust_ffi_shared", RustFFISharedFactory)
	android.RegisterModuleType("rust_ffi_static", RustFFIStaticFactory)
	android.RegisterModuleType("rust_ffi_host", RustFFIHostFactory)
	android.RegisterModuleType("rust_ffi_host_shared", RustFFISharedHostFactory)
	android.RegisterModuleType("rust_ffi_host_static", RustFFIStaticHostFactory)
}

type VariantLibraryProperties struct {
	Enabled *bool    `android:"arch_variant"`
	Srcs    []string `android:"path,arch_variant"`
}

type LibraryCompilerProperties struct {
	Rlib   VariantLibraryProperties `android:"arch_variant"`
	Dylib  VariantLibraryProperties `android:"arch_variant"`
	Shared VariantLibraryProperties `android:"arch_variant"`
	Static VariantLibraryProperties `android:"arch_variant"`

	// path to include directories to pass to cc_* modules, only relevant for static/shared variants.
	Include_dirs []string `android:"path,arch_variant"`
}

type LibraryMutatedProperties struct {
	// Build a dylib variant
	BuildDylib bool `blueprint:"mutated"`
	// Build an rlib variant
	BuildRlib bool `blueprint:"mutated"`
	// Build a shared library variant
	BuildShared bool `blueprint:"mutated"`
	// Build a static library variant
	BuildStatic bool `blueprint:"mutated"`

	// This variant is a dylib
	VariantIsDylib bool `blueprint:"mutated"`
	// This variant is an rlib
	VariantIsRlib bool `blueprint:"mutated"`
	// This variant is a shared library
	VariantIsShared bool `blueprint:"mutated"`
	// This variant is a static library
	VariantIsStatic bool `blueprint:"mutated"`
}

type libraryDecorator struct {
	*baseCompiler
	*flagExporter

	Properties        LibraryCompilerProperties
	MutatedProperties LibraryMutatedProperties
	includeDirs       android.Paths
}

type libraryInterface interface {
	rlib() bool
	dylib() bool
	static() bool
	shared() bool

	// Returns true if the build options for the module have selected a particular build type
	buildRlib() bool
	buildDylib() bool
	buildShared() bool
	buildStatic() bool

	// Sets a particular variant type
	setRlib()
	setDylib()
	setShared()
	setStatic()

	// Build a specific library variant
	BuildOnlyFFI()
	BuildOnlyRust()
	BuildOnlyRlib()
	BuildOnlyDylib()
	BuildOnlyStatic()
	BuildOnlyShared()
}

func (library *libraryDecorator) nativeCoverage() bool {
	return true
}

func (library *libraryDecorator) rlib() bool {
	return library.MutatedProperties.VariantIsRlib
}

func (library *libraryDecorator) dylib() bool {
	return library.MutatedProperties.VariantIsDylib
}

func (library *libraryDecorator) shared() bool {
	return library.MutatedProperties.VariantIsShared
}

func (library *libraryDecorator) static() bool {
	return library.MutatedProperties.VariantIsStatic
}

func (library *libraryDecorator) buildRlib() bool {
	return library.MutatedProperties.BuildRlib && BoolDefault(library.Properties.Rlib.Enabled, true)
}

func (library *libraryDecorator) buildDylib() bool {
	return library.MutatedProperties.BuildDylib && BoolDefault(library.Properties.Dylib.Enabled, true)
}

func (library *libraryDecorator) buildShared() bool {
	return library.MutatedProperties.BuildShared && BoolDefault(library.Properties.Shared.Enabled, true)
}

func (library *libraryDecorator) buildStatic() bool {
	return library.MutatedProperties.BuildStatic && BoolDefault(library.Properties.Static.Enabled, true)
}

func (library *libraryDecorator) setRlib() {
	library.MutatedProperties.VariantIsRlib = true
	library.MutatedProperties.VariantIsDylib = false
	library.MutatedProperties.VariantIsStatic = false
	library.MutatedProperties.VariantIsShared = false
}

func (library *libraryDecorator) setDylib() {
	library.MutatedProperties.VariantIsRlib = false
	library.MutatedProperties.VariantIsDylib = true
	library.MutatedProperties.VariantIsStatic = false
	library.MutatedProperties.VariantIsShared = false
}

func (library *libraryDecorator) setShared() {
	library.MutatedProperties.VariantIsStatic = false
	library.MutatedProperties.VariantIsShared = true
	library.MutatedProperties.VariantIsRlib = false
	library.MutatedProperties.VariantIsDylib = false
}

func (library *libraryDecorator) setStatic() {
	library.MutatedProperties.VariantIsStatic = true
	library.MutatedProperties.VariantIsShared = false
	library.MutatedProperties.VariantIsRlib = false
	library.MutatedProperties.VariantIsDylib = false
}

func (library *libraryDecorator) autoDep() autoDep {
	if library.rlib() || library.static() {
		return rlibAutoDep
	} else if library.dylib() || library.shared() {
		return dylibAutoDep
	} else {
		return rlibAutoDep
	}
}

var _ compiler = (*libraryDecorator)(nil)
var _ libraryInterface = (*libraryDecorator)(nil)
var _ exportedFlagsProducer = (*libraryDecorator)(nil)

// rust_library produces all rust variants.
func RustLibraryFactory() android.Module {
	module, library := NewRustLibrary(android.HostAndDeviceSupported)
	library.BuildOnlyRust()
	return module.Init()
}

// rust_ffi produces all ffi variants.
func RustFFIFactory() android.Module {
	module, library := NewRustLibrary(android.HostAndDeviceSupported)
	library.BuildOnlyFFI()
	return module.Init()
}

// rust_library_dylib produces a dylib.
func RustLibraryDylibFactory() android.Module {
	module, library := NewRustLibrary(android.HostAndDeviceSupported)
	library.BuildOnlyDylib()
	return module.Init()
}

// rust_library_rlib produces an rlib.
func RustLibraryRlibFactory() android.Module {
	module, library := NewRustLibrary(android.HostAndDeviceSupported)
	library.BuildOnlyRlib()
	return module.Init()
}

// rust_ffi_shared produces a shared library.
func RustFFISharedFactory() android.Module {
	module, library := NewRustLibrary(android.HostAndDeviceSupported)
	library.BuildOnlyShared()
	return module.Init()
}

// rust_ffi_static produces a static library.
func RustFFIStaticFactory() android.Module {
	module, library := NewRustLibrary(android.HostAndDeviceSupported)
	library.BuildOnlyStatic()
	return module.Init()
}

// rust_library_host produces all rust variants.
func RustLibraryHostFactory() android.Module {
	module, library := NewRustLibrary(android.HostSupported)
	library.BuildOnlyRust()
	return module.Init()
}

// rust_ffi_host produces all FFI variants.
func RustFFIHostFactory() android.Module {
	module, library := NewRustLibrary(android.HostSupported)
	library.BuildOnlyFFI()
	return module.Init()
}

// rust_library_dylib_host produces a dylib.
func RustLibraryDylibHostFactory() android.Module {
	module, library := NewRustLibrary(android.HostSupported)
	library.BuildOnlyDylib()
	return module.Init()
}

// rust_library_rlib_host produces an rlib.
func RustLibraryRlibHostFactory() android.Module {
	module, library := NewRustLibrary(android.HostSupported)
	library.BuildOnlyRlib()
	return module.Init()
}

// rust_ffi_static_host produces a static library.
func RustFFIStaticHostFactory() android.Module {
	module, library := NewRustLibrary(android.HostSupported)
	library.BuildOnlyStatic()
	return module.Init()
}

// rust_ffi_shared_host produces an shared library.
func RustFFISharedHostFactory() android.Module {
	module, library := NewRustLibrary(android.HostSupported)
	library.BuildOnlyShared()
	return module.Init()
}

func (library *libraryDecorator) BuildOnlyFFI() {
	library.MutatedProperties.BuildDylib = false
	library.MutatedProperties.BuildRlib = false
	library.MutatedProperties.BuildShared = true
	library.MutatedProperties.BuildStatic = true
}

func (library *libraryDecorator) BuildOnlyRust() {
	library.MutatedProperties.BuildDylib = true
	library.MutatedProperties.BuildRlib = true
	library.MutatedProperties.BuildShared = false
	library.MutatedProperties.BuildStatic = false
}

func (library *libraryDecorator) BuildOnlyDylib() {
	library.MutatedProperties.BuildDylib = true
	library.MutatedProperties.BuildRlib = false
	library.MutatedProperties.BuildShared = false
	library.MutatedProperties.BuildStatic = false
}

func (library *libraryDecorator) BuildOnlyRlib() {
	library.MutatedProperties.BuildDylib = false
	library.MutatedProperties.BuildRlib = true
	library.MutatedProperties.BuildShared = false
	library.MutatedProperties.BuildStatic = false
}

func (library *libraryDecorator) BuildOnlyStatic() {
	library.MutatedProperties.BuildRlib = false
	library.MutatedProperties.BuildDylib = false
	library.MutatedProperties.BuildShared = false
	library.MutatedProperties.BuildStatic = true
}

func (library *libraryDecorator) BuildOnlyShared() {
	library.MutatedProperties.BuildRlib = false
	library.MutatedProperties.BuildDylib = false
	library.MutatedProperties.BuildStatic = false
	library.MutatedProperties.BuildShared = true
}

func NewRustLibrary(hod android.HostOrDeviceSupported) (*Module, *libraryDecorator) {
	module := newModule(hod, android.MultilibBoth)

	library := &libraryDecorator{
		MutatedProperties: LibraryMutatedProperties{
			BuildDylib:  false,
			BuildRlib:   false,
			BuildShared: false,
			BuildStatic: false,
		},
		baseCompiler: NewBaseCompiler("lib", "lib64", InstallInSystem),
		flagExporter: NewFlagExporter(),
	}

	module.compiler = library

	return module, library
}

func (library *libraryDecorator) compilerProps() []interface{} {
	return append(library.baseCompiler.compilerProps(),
		&library.Properties,
		&library.MutatedProperties)
}

func (library *libraryDecorator) compilerDeps(ctx DepsContext, deps Deps) Deps {
	deps = library.baseCompiler.compilerDeps(ctx, deps)

	if ctx.toolchain().Bionic() && (library.dylib() || library.shared()) {
		deps = bionicDeps(deps)
		deps.CrtBegin = "crtbegin_so"
		deps.CrtEnd = "crtend_so"
	}

	return deps
}

func (library *libraryDecorator) sharedLibFilename(ctx ModuleContext) string {
	return library.getStem(ctx) + ctx.toolchain().SharedLibSuffix()
}

func (library *libraryDecorator) compilerFlags(ctx ModuleContext, flags Flags) Flags {
	flags.RustFlags = append(flags.RustFlags, "-C metadata="+ctx.ModuleName())
	flags = library.baseCompiler.compilerFlags(ctx, flags)
	if library.shared() || library.static() {
		library.includeDirs = append(library.includeDirs, android.PathsForModuleSrc(ctx, library.Properties.Include_dirs)...)
	}
	if library.shared() {
		flags.LinkFlags = append(flags.LinkFlags, "-Wl,-soname="+library.sharedLibFilename(ctx))
	}

	return flags
}

func (library *libraryDecorator) compile(ctx ModuleContext, flags Flags, deps PathDeps) android.Path {
	var outputFile android.WritablePath

	srcPath, _ := srcPathFromModuleSrcs(ctx, library.baseCompiler.Properties.Srcs)

	flags.RustFlags = append(flags.RustFlags, deps.depFlags...)

	if library.dylib() {
		// We need prefer-dynamic for now to avoid linking in the static stdlib. See:
		// https://github.com/rust-lang/rust/issues/19680
		// https://github.com/rust-lang/rust/issues/34909
		flags.RustFlags = append(flags.RustFlags, "-C prefer-dynamic")
	}

	if library.rlib() {
		fileName := library.getStem(ctx) + ctx.toolchain().RlibSuffix()
		outputFile = android.PathForModuleOut(ctx, fileName)

		outputs := TransformSrctoRlib(ctx, srcPath, deps, flags, outputFile, deps.linkDirs)
		library.coverageFile = outputs.coverageFile
	} else if library.dylib() {
		fileName := library.getStem(ctx) + ctx.toolchain().DylibSuffix()
		outputFile = android.PathForModuleOut(ctx, fileName)

		outputs := TransformSrctoDylib(ctx, srcPath, deps, flags, outputFile, deps.linkDirs)
		library.coverageFile = outputs.coverageFile
	} else if library.static() {
		fileName := library.getStem(ctx) + ctx.toolchain().StaticLibSuffix()
		outputFile = android.PathForModuleOut(ctx, fileName)

		outputs := TransformSrctoStatic(ctx, srcPath, deps, flags, outputFile, deps.linkDirs)
		library.coverageFile = outputs.coverageFile
	} else if library.shared() {
		fileName := library.sharedLibFilename(ctx)
		outputFile = android.PathForModuleOut(ctx, fileName)

		outputs := TransformSrctoShared(ctx, srcPath, deps, flags, outputFile, deps.linkDirs)
		library.coverageFile = outputs.coverageFile
	}

	var coverageFiles android.Paths
	if library.coverageFile != nil {
		coverageFiles = append(coverageFiles, library.coverageFile)
	}
	if len(deps.coverageFiles) > 0 {
		coverageFiles = append(coverageFiles, deps.coverageFiles...)
	}
	library.coverageOutputZipFile = TransformCoverageFilesToZip(ctx, coverageFiles, library.getStem(ctx))

	if library.rlib() || library.dylib() {
		library.exportLinkDirs(deps.linkDirs...)
		library.exportDepFlags(deps.depFlags...)
	}
	library.unstrippedOutputFile = outputFile

	return outputFile
}

func (library *libraryDecorator) getStem(ctx ModuleContext) string {
	stem := library.baseCompiler.getStemWithoutSuffix(ctx)
	validateLibraryStem(ctx, stem, library.crateName())

	return stem + String(library.baseCompiler.Properties.Suffix)
}

var validCrateName = regexp.MustCompile("[^a-zA-Z0-9_]+")

func validateLibraryStem(ctx BaseModuleContext, filename string, crate_name string) {
	if crate_name == "" {
		ctx.PropertyErrorf("crate_name", "crate_name must be defined.")
	}

	// crate_names are used for the library output file, and rustc expects these
	// to be alphanumeric with underscores allowed.
	if validCrateName.MatchString(crate_name) {
		ctx.PropertyErrorf("crate_name",
			"library crate_names must be alphanumeric with underscores allowed")
	}

	// Libraries are expected to begin with "lib" followed by the crate_name
	if !strings.HasPrefix(filename, "lib"+crate_name) {
		ctx.ModuleErrorf("Invalid name or stem property; library filenames must start with lib<crate_name>")
	}
}

func LibraryMutator(mctx android.BottomUpMutatorContext) {
	if m, ok := mctx.Module().(*Module); ok && m.compiler != nil {
		switch library := m.compiler.(type) {
		case libraryInterface:

			// We only build the rust library variants here. This assumes that
			// LinkageMutator runs first and there's an empty variant
			// if rust variants are required.
			if !library.static() && !library.shared() {
				if library.buildRlib() && library.buildDylib() {
					modules := mctx.CreateLocalVariations("rlib", "dylib")
					rlib := modules[0].(*Module)
					dylib := modules[1].(*Module)

					rlib.compiler.(libraryInterface).setRlib()
					dylib.compiler.(libraryInterface).setDylib()
				} else if library.buildRlib() {
					modules := mctx.CreateLocalVariations("rlib")
					modules[0].(*Module).compiler.(libraryInterface).setRlib()
				} else if library.buildDylib() {
					modules := mctx.CreateLocalVariations("dylib")
					modules[0].(*Module).compiler.(libraryInterface).setDylib()
				}
			}
		}
	}
}
