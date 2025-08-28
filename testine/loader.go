package testine

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/calumari/jwalk"
	"github.com/go-json-experiment/json"
	"golang.org/x/sync/singleflight"
)

type documentLoader struct {
	reg   *jwalk.Registry
	cache bool

	mu        sync.RWMutex
	fileCache map[string]jwalk.Document
	pathCache map[string]jwalk.Document

	group singleflight.Group
}

func newDocumentLoader(reg *jwalk.Registry, cache bool) *documentLoader {
	fl := &documentLoader{reg: reg, cache: cache}
	if cache {
		fl.fileCache = make(map[string]jwalk.Document)
		fl.pathCache = make(map[string]jwalk.Document)
	}
	return fl
}

func (l *documentLoader) load(path string) (jwalk.Document, error) {
	if !l.cache { // no caching, just resolve and merge
		return l.loadPath(path)
	}
	// cached path singleflight
	if doc := l.getPathCache(path); doc != nil {
		return copyDoc(doc), nil
	}
	v, err, _ := l.group.Do(path, func() (any, error) {
		if doc := l.getPathCache(path); doc != nil { // re-check inside flight
			return copyDoc(doc), nil
		}
		doc, err := l.loadPath(path)
		if err != nil {
			return nil, err
		}
		l.setPathCache(path, doc)
		return copyDoc(doc), nil
	})
	if err != nil {
		return nil, err
	}
	return v.(jwalk.Document), nil
}

func (l *documentLoader) getPathCache(path string) jwalk.Document {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.pathCache[path]
}

func (l *documentLoader) setPathCache(path string, doc jwalk.Document) {
	l.mu.Lock()
	l.pathCache[path] = doc
	l.mu.Unlock()
}

func (l *documentLoader) loadPath(path string) (jwalk.Document, error) {
	files, err := l.resolve(path)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, errors.New("no json files resolved")
	}
	if len(files) == 1 {
		return l.getFile(files[0])
	}
	sort.Strings(files)
	var readErr error
	doc := mergeDocs(func(yield func(jwalk.Document)) {
		for _, f := range files {
			if readErr != nil {
				return
			}
			d, err := l.getFile(f)
			if err != nil {
				readErr = err
				return
			}
			yield(d)
		}
	})
	if readErr != nil {
		return nil, readErr
	}
	return doc, nil
}

func (l *documentLoader) resolve(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			if !isJSONFile(path) {
				return nil, errors.New("not a json file")
			}
			return []string{path}, nil
		}
		ents, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}
		var out []string
		for _, e := range ents {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if isJSONFile(name) {
				out = append(out, filepath.Join(path, name))
			}
		}
		return out, nil
	}
	if !strings.ContainsAny(path, "*?[") { // not a glob
		return nil, errors.New("path not found")
	}
	matches, gerr := filepath.Glob(path)
	if gerr != nil {
		return nil, gerr
	}
	var out []string
	for _, m := range matches {
		if info, err := os.Stat(m); err == nil && !info.IsDir() && isJSONFile(m) {
			out = append(out, m)
		}
	}
	return out, nil
}

func (l *documentLoader) getFile(file string) (jwalk.Document, error) {
	if !l.cache {
		return l.readFile(file)
	}
	// cache + singleflight keyed per file path (still via group but distinct key)
	if d := l.getFileCache(file); d != nil {
		return copyDoc(d), nil
	}
	v, err, _ := l.group.Do("file::"+file, func() (any, error) {
		if d := l.getFileCache(file); d != nil {
			return copyDoc(d), nil
		}
		doc, err := l.readFile(file)
		if err != nil {
			return nil, err
		}
		l.setFileCache(file, doc)
		return copyDoc(doc), nil
	})
	if err != nil {
		return nil, err
	}
	return v.(jwalk.Document), nil
}

func (l *documentLoader) getFileCache(f string) jwalk.Document {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.fileCache[f]
}
func (l *documentLoader) setFileCache(f string, d jwalk.Document) {
	l.mu.Lock()
	l.fileCache[f] = d
	l.mu.Unlock()
}

func (l *documentLoader) readFile(file string) (jwalk.Document, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var doc jwalk.Document
	if err := json.UnmarshalRead(f, &doc, json.WithUnmarshalers(jwalk.Unmarshalers(l.reg))); err != nil {
		return nil, err
	}
	return doc, nil
}

// helpers

func copyDoc(in jwalk.Document) jwalk.Document {
	cp := make(jwalk.Document, len(in))
	copy(cp, in)
	return cp
}

func isJSONFile(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".json")
}

func mergeDocs(iter func(func(jwalk.Document))) jwalk.Document {
	var merged jwalk.Document
	index := make(map[string]int)
	iter(func(doc jwalk.Document) {
		for _, e := range doc {
			if pos, ok := index[e.Key]; ok {
				merged[pos].Value = e.Value
				continue
			}
			index[e.Key] = len(merged)
			merged = append(merged, e)
		}
	})
	return merged
}
