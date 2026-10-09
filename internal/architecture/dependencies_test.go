package architecture_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const modulePath = "github.com/mmmIlia/badgoodtune"

func TestLayerDependencies(t *testing.T) {
	repositoryRoot := repositoryRoot(t)

	rules := []struct {
		directory       string
		localImports    []string
		allowThirdParty bool
	}{
		{directory: "domain"},
		{directory: "usecase", localImports: []string{"internal/domain"}},
		{directory: "clock"},
		{directory: "config", allowThirdParty: true},
		{directory: "logging"},
		{directory: "security", allowThirdParty: true},
		{
			directory:       filepath.Join("repository", "postgres"),
			localImports:    []string{"db/migrations", "internal/domain", "internal/usecase", "internal/repository/postgres/sqlc"},
			allowThirdParty: true,
		},
		{
			directory:       filepath.Join("transport", "http"),
			localImports:    []string{"internal/domain", "internal/usecase", "internal/transport/http/openapi"},
			allowThirdParty: true,
		},
	}

	for _, rule := range rules {
		t.Run(rule.directory, func(t *testing.T) {
			allowedLocalImports := make(map[string]struct{}, len(rule.localImports))
			for _, dependency := range rule.localImports {
				allowedLocalImports[modulePath+"/"+filepath.ToSlash(dependency)] = struct{}{}
			}
			assertImports(t, filepath.Join(repositoryRoot, "internal", rule.directory), func(importPath string) bool {
				if !strings.HasPrefix(importPath, modulePath) {
					return rule.allowThirdParty || !isThirdParty(importPath)
				}
				_, allowed := allowedLocalImports[importPath]
				return allowed
			})
		})
	}
}

func assertImports(t *testing.T, directory string, allowed func(string) bool) {
	t.Helper()
	packages, err := parser.ParseDir(token.NewFileSet(), directory, func(info fs.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, parser.ImportsOnly)
	require.NoError(t, err)

	for _, parsedPackage := range packages {
		for filename, file := range parsedPackage.Files {
			for _, declaration := range file.Decls {
				importDeclaration, ok := declaration.(*ast.GenDecl)
				if !ok || importDeclaration.Tok != token.IMPORT {
					continue
				}
				for _, specification := range importDeclaration.Specs {
					importSpecification := specification.(*ast.ImportSpec)
					importPath, unquoteErr := strconv.Unquote(importSpecification.Path.Value)
					require.NoError(t, unquoteErr)
					if !allowed(importPath) {
						t.Errorf("%s imports forbidden dependency %q", filename, importPath)
					}
				}
			}
		}
	}
}

func isThirdParty(importPath string) bool {
	firstSegment, _, _ := strings.Cut(importPath, "/")
	return strings.Contains(firstSegment, ".")
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}
