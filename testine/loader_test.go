package testine

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/calumari/jwalk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupLoader(cache bool) *documentLoader {
	reg := &jwalk.Registry{}
	return newDocumentLoader(reg, cache)
}

// helpers

func writeTempJSON(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "ld_*.json")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	f.Close()
	return f.Name()
}

func writeDirFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, data := range files {
		err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0644)
		require.NoError(t, err)
	}
	return dir
}

func writeTemp(t *testing.T, pattern string, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", pattern)
	require.NoError(t, err)
	if content != "" {
		_, err = f.WriteString(content)
		require.NoError(t, err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

func Test_documentLoader_load(t *testing.T) {
	t.Run("loads document without cache succeeds", func(t *testing.T) {
		path := writeTemp(t, "doc_*.json", `[{"Key":"foo","Value":123}]`)
		l := setupLoader(false)
		got, err := l.load(path)
		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, "foo", got[0].Key)
		assert.Equal(t, 123.0, got[0].Value) // json numbers are float64
	})

	t.Run("loads document with cache succeeds", func(t *testing.T) {
		path := writeTemp(t, "doc_*.json", `[{"Key":"bar","Value":456}]`)
		l := setupLoader(true)
		got, err := l.load(path)
		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, "bar", got[0].Key)
		assert.Equal(t, 456.0, got[0].Value)
		// Should hit cache on second load
		got2, err2 := l.load(path)
		require.NoError(t, err2)
		assert.Equal(t, got, got2)
	})

	t.Run("loads merged documents succeeds", func(t *testing.T) {
		dir := writeDirFiles(t, map[string]string{
			"a.json": `[{"Key":"a","Value":1}]`,
			"b.json": `[{"Key":"b","Value":2}]`,
		})
		l := setupLoader(false)
		got, err := l.load(dir)
		require.NoError(t, err)
		assert.Len(t, got, 2)
		assert.Equal(t, "a", got[0].Key)
		assert.Equal(t, "b", got[1].Key)
	})

	t.Run("uppercase extension file succeeds", func(t *testing.T) {
		path := writeTemp(t, "up_*.JSON", `[{"Key":"up","Value":3}]`)
		l := setupLoader(false)
		got, err := l.load(path)
		require.NoError(t, err)
		assert.Equal(t, "up", got[0].Key)
	})

	t.Run("duplicate keys later file overrides succeeds", func(t *testing.T) {
		dir := writeDirFiles(t, map[string]string{
			"a.json": `[{"Key":"dup","Value":1}]`,
			"b.json": `[{"Key":"dup","Value":99}]`,
		})
		l := setupLoader(false)
		got, err := l.load(dir)
		require.NoError(t, err)
		// Only one entry key dup with value 99
		assert.Len(t, got, 1)
		assert.Equal(t, "dup", got[0].Key)
		assert.Equal(t, 99.0, got[0].Value)
	})

	t.Run("glob pattern multiple files succeeds", func(t *testing.T) {
		dir := writeDirFiles(t, map[string]string{
			"c.json": `[{"Key":"c","Value":1}]`,
			"d.json": `[{"Key":"d","Value":2}]`,
		})
		pattern := filepath.Join(dir, "*.json")
		l := setupLoader(false)
		got, err := l.load(pattern)
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})

	t.Run("concurrent loads same path succeed", func(t *testing.T) {
		tmp := writeTempJSON(t, `[{"Key":"cc","Value":7}]`)
		l := setupLoader(true)
		var wg sync.WaitGroup
		const n = 10
		results := make([]jwalk.Document, n)
		errs := make([]error, n)
		for i := range n {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				r, e := l.load(tmp)
				errs[i] = e
				results[i] = r
			}(i)
		}
		wg.Wait()
		for i := range n {
			require.NoError(t, errs[i])
			require.NotNil(t, results[i])
			assert.Equal(t, "cc", results[i][0].Key)
		}
	})

	t.Run("concurrent getFile same file succeeds", func(t *testing.T) {
		if runtime.GOOS == "windows" { // permissions races less predictable, but still fine; keep anyway
		}
		tmp := writeTempJSON(t, `[{"Key":"raw","Value":11}]`)
		l := setupLoader(true)
		var wg sync.WaitGroup
		const n = 8
		docs := make([]jwalk.Document, n)
		errs := make([]error, n)
		for i := range n {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				d, e := l.getFile(tmp)
				errs[i] = e
				docs[i] = d
			}(i)
		}
		wg.Wait()
		for i := range n {
			require.NoError(t, errs[i])
			require.NotNil(t, docs[i])
			assert.Equal(t, "raw", docs[i][0].Key)
		}
	})

	t.Run("loads from cache after initial load succeeds", func(t *testing.T) {
		path := writeTemp(t, "doc_*.json", `[{"Key":"baz","Value":789}]`)
		l := setupLoader(true)
		l.setPathCache(path, jwalk.Document{{Key: "baz", Value: 789}})
		got, err := l.load(path)
		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, "baz", got[0].Key)
		assert.Equal(t, 789, got[0].Value)
	})

	t.Run("empty bundle returns error", func(t *testing.T) {
		dir := t.TempDir()
		l := setupLoader(false)
		got, err := l.load(dir)
		assert.Error(t, err)
		assert.Nil(t, got)
	})

	t.Run("non-existent file returns error", func(t *testing.T) {
		l := setupLoader(false)
		got, err := l.load("/tmp/doesnotexist.json")
		assert.Error(t, err)
		assert.Nil(t, got)
	})

	t.Run("non-json file returns error", func(t *testing.T) {
		path := writeTemp(t, "notjson_*.txt", "")
		l := setupLoader(false)
		got, err := l.load(path)
		assert.Error(t, err)
		assert.Nil(t, got)
	})

	t.Run("glob pattern returns error if no match", func(t *testing.T) {
		l := setupLoader(false)
		got, err := l.load("*.doesnotexist.json")
		assert.Error(t, err)
		assert.Nil(t, got)
	})

	t.Run("file with invalid JSON returns error", func(t *testing.T) {
		path := writeTemp(t, "badjson_*.json", "not valid json")
		l := setupLoader(false)
		got, err := l.load(path)
		assert.Error(t, err)
		assert.Nil(t, got)
	})

	t.Run("directory with valid and invalid JSON files returns error", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "good.json")
		file2 := filepath.Join(dir, "bad.json")
		err := os.WriteFile(file1, []byte(`[{"Key":"ok","Value":1}]`), 0644)
		require.NoError(t, err)
		err = os.WriteFile(file2, []byte("not valid json"), 0644)
		require.NoError(t, err)
		l := setupLoader(false)
		got, err := l.load(dir)
		assert.Error(t, err)
		assert.Nil(t, got)
	})

	t.Run("glob pattern matches valid and invalid files returns error", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "good.json")
		file2 := filepath.Join(dir, "bad.json")
		err := os.WriteFile(file1, []byte(`[{"Key":"ok","Value":1}]`), 0644)
		require.NoError(t, err)
		err = os.WriteFile(file2, []byte("not valid json"), 0644)
		require.NoError(t, err)
		pattern := filepath.Join(dir, "*.json")
		l := setupLoader(false)
		got, err := l.load(pattern)
		assert.Error(t, err)
		assert.Nil(t, got)
	})

	t.Run("file with correct extension but empty content returns error", func(t *testing.T) {
		path := writeTemp(t, "empty_*.json", "")
		l := setupLoader(false)
		got, err := l.load(path)
		assert.Error(t, err)
		assert.Nil(t, got)
	})

	t.Run("cache contains nil returns error", func(t *testing.T) {
		path := writeTemp(t, "nilcache_*.json", "")
		l := setupLoader(true)
		l.setPathCache(path, nil)
		got, err := l.load(path)
		assert.Error(t, err)
		assert.Nil(t, got)
	})

	t.Run("directory not readable returns error", func(t *testing.T) {
		dir := t.TempDir()
		err := os.Chmod(dir, 0000)
		require.NoError(t, err)
		defer os.Chmod(dir, 0755)
		l := setupLoader(false)
		got, err := l.load(dir)
		assert.Error(t, err)
		assert.Nil(t, got)
	})
}
