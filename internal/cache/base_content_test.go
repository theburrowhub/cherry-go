package cache

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBaseContentManager_SaveAndGetSnapshot(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "base-content-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create manager with custom base dir
	manager := &BaseContentManager{baseDir: tempDir}

	// Test data
	sourceName := "test-source"
	pathSpec := "src/utils"
	files := map[string][]byte{
		"helper.go":   []byte("package utils\n\nfunc Helper() {}\n"),
		"config.go":   []byte("package utils\n\nvar Config = struct{}{}\n"),
		"sub/deep.go": []byte("package sub\n\nfunc Deep() {}\n"),
	}

	// Save snapshot
	err = manager.SaveSnapshot(sourceName, pathSpec, files)
	if err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	// Verify HasSnapshot
	if !manager.HasSnapshot(sourceName, pathSpec) {
		t.Error("HasSnapshot should return true after saving")
	}

	// Get snapshot
	retrieved, err := manager.GetSnapshot(sourceName, pathSpec)
	if err != nil {
		t.Fatalf("GetSnapshot failed: %v", err)
	}

	// Verify content
	if len(retrieved) != len(files) {
		t.Errorf("Expected %d files, got %d", len(files), len(retrieved))
	}

	for path, content := range files {
		if string(retrieved[path]) != string(content) {
			t.Errorf("Content mismatch for %s: expected %q, got %q", path, content, retrieved[path])
		}
	}
}

func TestBaseContentManager_GetFileContent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "base-content-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := &BaseContentManager{baseDir: tempDir}

	sourceName := "test-source"
	pathSpec := "lib"
	files := map[string][]byte{
		"main.go": []byte("package main\n"),
	}

	err = manager.SaveSnapshot(sourceName, pathSpec, files)
	if err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	// Get single file content
	content, err := manager.GetFileContent(sourceName, pathSpec, "main.go")
	if err != nil {
		t.Fatalf("GetFileContent failed: %v", err)
	}

	if string(content) != "package main\n" {
		t.Errorf("Unexpected content: %q", content)
	}

	// Get non-existent file
	content, err = manager.GetFileContent(sourceName, pathSpec, "nonexistent.go")
	if err != nil {
		t.Errorf("GetFileContent should not error for missing file, got: %v", err)
	}
	if content != nil {
		t.Error("Content should be nil for non-existent file")
	}
}

func TestBaseContentManager_DeleteSnapshot(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "base-content-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := &BaseContentManager{baseDir: tempDir}

	sourceName := "test-source"
	pathSpec := "src"
	files := map[string][]byte{
		"file.go": []byte("content"),
	}

	// Save and verify
	if err := manager.SaveSnapshot(sourceName, pathSpec, files); err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}
	if !manager.HasSnapshot(sourceName, pathSpec) {
		t.Fatal("Snapshot should exist after saving")
	}

	// Delete
	err = manager.DeleteSnapshot(sourceName, pathSpec)
	if err != nil {
		t.Fatalf("DeleteSnapshot failed: %v", err)
	}

	// Verify deleted
	if manager.HasSnapshot(sourceName, pathSpec) {
		t.Error("Snapshot should not exist after deletion")
	}
}

func TestBaseContentManager_DeleteSourceSnapshots(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "base-content-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := &BaseContentManager{baseDir: tempDir}

	sourceName := "test-source"
	files := map[string][]byte{"file.go": []byte("content")}

	// Save multiple snapshots for same source
	if err := manager.SaveSnapshot(sourceName, "path1", files); err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}
	if err := manager.SaveSnapshot(sourceName, "path2", files); err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}
	if err := manager.SaveSnapshot("other-source", "path1", files); err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	// Delete all snapshots for source
	err = manager.DeleteSourceSnapshots(sourceName)
	if err != nil {
		t.Fatalf("DeleteSourceSnapshots failed: %v", err)
	}

	// Verify test-source snapshots deleted
	if manager.HasSnapshot(sourceName, "path1") {
		t.Error("path1 snapshot should be deleted")
	}
	if manager.HasSnapshot(sourceName, "path2") {
		t.Error("path2 snapshot should be deleted")
	}

	// Verify other-source still exists
	if !manager.HasSnapshot("other-source", "path1") {
		t.Error("other-source snapshot should still exist")
	}
}

func TestBaseContentManager_CleanOrphanedSnapshots(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "base-content-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := &BaseContentManager{baseDir: tempDir}

	files := map[string][]byte{"file.go": []byte("content")}

	// Save snapshots for multiple sources
	manager.SaveSnapshot("source1", "path", files)
	manager.SaveSnapshot("source2", "path", files)
	manager.SaveSnapshot("source3", "path", files)

	// Clean orphaned (only source1 and source3 are valid)
	validSources := []string{"source1", "source3"}
	err = manager.CleanOrphanedSnapshots(validSources)
	if err != nil {
		t.Fatalf("CleanOrphanedSnapshots failed: %v", err)
	}

	// Verify
	if !manager.HasSnapshot("source1", "path") {
		t.Error("source1 should still exist")
	}
	if manager.HasSnapshot("source2", "path") {
		t.Error("source2 should be cleaned up")
	}
	if !manager.HasSnapshot("source3", "path") {
		t.Error("source3 should still exist")
	}
}

func TestBaseContentManager_GetSnapshot_NonExistent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "base-content-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := &BaseContentManager{baseDir: tempDir}

	// Get non-existent snapshot
	snapshot, err := manager.GetSnapshot("nonexistent", "path")
	if err != nil {
		t.Errorf("GetSnapshot should not error for non-existent snapshot: %v", err)
	}
	if snapshot != nil {
		t.Error("Snapshot should be nil for non-existent path")
	}
}

func TestBaseContentManager_OverwriteSnapshot(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "base-content-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := &BaseContentManager{baseDir: tempDir}

	sourceName := "test-source"
	pathSpec := "src"

	// Save initial
	files1 := map[string][]byte{
		"file1.go": []byte("original"),
		"file2.go": []byte("original2"),
	}
	manager.SaveSnapshot(sourceName, pathSpec, files1)

	// Save new version (different files)
	files2 := map[string][]byte{
		"file1.go": []byte("updated"),
		"file3.go": []byte("new file"),
	}
	manager.SaveSnapshot(sourceName, pathSpec, files2)

	// Get and verify
	retrieved, err := manager.GetSnapshot(sourceName, pathSpec)
	if err != nil {
		t.Fatalf("GetSnapshot failed: %v", err)
	}

	// Should have new content
	if string(retrieved["file1.go"]) != "updated" {
		t.Error("file1.go should be updated")
	}
	if string(retrieved["file3.go"]) != "new file" {
		t.Error("file3.go should exist")
	}
	// file2.go should not exist (old snapshot was replaced)
	if _, exists := retrieved["file2.go"]; exists {
		t.Error("file2.go should not exist in new snapshot")
	}
}

func TestBaseContentManager_PathHashing(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "base-content-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := &BaseContentManager{baseDir: tempDir}

	// Different path specs should have different snapshot locations
	path1 := manager.getSnapshotPath("source", "path/to/something")
	path2 := manager.getSnapshotPath("source", "path/to/other")
	path3 := manager.getSnapshotPath("source", "path/to/something") // Same as path1

	if path1 == path2 {
		t.Error("Different path specs should have different snapshot paths")
	}
	if path1 != path3 {
		t.Error("Same path spec should have same snapshot path")
	}

	// Verify paths are under source directory
	expectedBase := filepath.Join(tempDir, "source")
	if !strings.HasPrefix(path1, expectedBase) {
		t.Errorf("Snapshot path should be under source directory: %s", path1)
	}
}
