package main

import (
	"bytes"
	"path/filepath"
	"sort"
	"strings"
)

type Entry struct {
	Name string

	// Files have revisions
	Revisions [][]byte

	// Directories have children
	Children map[string]*Entry
}

func NewFilesystem() *Entry {
	return NewEntry("/")
}

func NewEntry(name string) *Entry {
	return &Entry{
		Name:     name,
		Children: make(map[string]*Entry),
	}
}

func (e *Entry) GetEntry(path string) *Entry {
	path = filepath.Clean(path)
	if path == "/" {
		return e
	}
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}
	parts := strings.Split(path, "/")

	current := e
	for _, part := range parts {
		if _, ok := current.Children[part]; !ok {
			current.Children[part] = NewEntry(part)
		}
		current = current.Children[part]
	}

	return current
}

func (e *Entry) ListDir(dir string) []Entry {
	var entries []Entry

	root := e.GetEntry(dir)
	if root != nil {
		for _, entry := range root.Children {
			entries = append(entries, *entry)
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	return entries
}

func (e *Entry) Get(path string, revision int) []byte {
	entry := e.GetEntry(path)

	if revision <= -1 {
		revision = len(entry.Revisions)
	}

	if revision == 0 || revision > len(entry.Revisions) {
		return nil
	}

	return entry.Revisions[revision-1]
}

func (e *Entry) Put(path string, bs []byte) int {
	entry := e.GetEntry(path)

	var last []byte
	if len(entry.Revisions) > 0 {
		last = entry.Revisions[len(entry.Revisions)-1]
	}

	if !bytes.Equal(last, bs) {
		entry.Revisions = append(entry.Revisions, bs)
	}

	return len(entry.Revisions)
}
