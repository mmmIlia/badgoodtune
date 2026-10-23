package architecture_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const modulePath = "github.com/mmmIlia/badgoodtune"

func TestImportsIncludeNestedPackages(t *testing.T) {
	directory := t.TempDir()
	nested := filepath.Join(directory, "nested")
	require.NoError(t, os.Mkdir(nested, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(nested, "example.go"), []byte("package nested\nimport _ \"context\"\n"), 0o600))
	var imports []string
	assertImports(t, directory, func(importPath string) bool {
		imports = append(imports, importPath)
		return true
	})
	require.Equal(t, []string{"context"}, imports)
}

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
		{directory: "metrics", localImports: []string{"internal/domain", "internal/usecase"}, allowThirdParty: true},
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
	err := filepath.WalkDir(directory, func(filename string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(filename, ".go") || strings.HasSuffix(filename, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), filename, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, specification := range file.Imports {
			importPath, err := strconv.Unquote(specification.Path.Value)
			if err != nil {
				return err
			}
			if !allowed(importPath) {
				t.Errorf("%s imports forbidden dependency %q", filename, importPath)
			}
		}
		return nil
	})
	require.NoError(t, err)
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
