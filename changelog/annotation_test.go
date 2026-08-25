package changelog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveAnnotationRemovesHyphenatedAndCanonicalFiles(t *testing.T) {
	originalRemove := osRemove
	t.Cleanup(func() {
		osRemove = originalRemove
	})

	var paths []string
	osRemove = func(path string) error {
		paths = append(paths, path)
		return os.ErrNotExist
	}

	root := t.TempDir()
	annotation := Annotation{ID: "0b05ca0c-b38f-49bd-b566-3deaeb2e06c5"}
	if err := RemoveAnnotation(root, annotation); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := []string{
		filepath.Join(root, changelogDir, annotation.ID+".json"),
		filepath.Join(root, changelogDir, strings.ReplaceAll(annotation.ID, "-", "")+".json"),
	}
	if len(paths) != len(want) {
		t.Fatalf("expected %d paths, got %d: %v", len(want), len(paths), paths)
	}

	for i := range want {
		if paths[i] != want[i] {
			t.Errorf("expected path %d to be %q, got %q", i, want[i], paths[i])
		}
	}
}
