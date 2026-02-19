package input_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hyperwapp/input"
)

// Helper function to create dummy files/directories for testing
func createDummyFile(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create directory %s: %v", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create dummy file %s: %v", path, err)
	}
}

// Helper function to create a dummy fff directory structure
func createDummyFFFDir(t *testing.T, baseDir string) {
	t.Helper()
	domainPath := filepath.Join(baseDir, "example.com")
	filepath.Join(domainPath, "test_path")
	hash := "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3" // SHA1 of "test"
	createDummyFile(t, filepath.Join(domainPath, hash+".headers"), "HTTP/1.1 200 OK\nServer: nginx")
	createDummyFile(t, filepath.Join(domainPath, hash+".body"), "<html><body>Hello</body></html>")
}

// Helper function to create a dummy katana directory structure
func createDummyKatanaDir(t *testing.T, baseDir string) {
	t.Helper()
	domainPath := filepath.Join(baseDir, "example.com")
	createDummyFile(t, filepath.Join(domainPath, "index.txt"), "some index content")
	createDummyFile(t, filepath.Join(domainPath, "hash.txt"), "GET / HTTP/1.1\nHost: example.com\n\nHTTP/1.1 200 OK\n\nBody")
}

// Helper function to create a dummy single katana file
func createDummyKatanaFile(t *testing.T, filePath string) {
	t.Helper()
	content := "GET / HTTP/1.1\nHost: example.com\n\nHTTP/1.1 200 OK\n\nBody"
	createDummyFile(t, filePath, content)
}

// Helper function to create a dummy raw HTTP file
func createDummyRawHTTPFile(t *testing.T, filePath string) {
	t.Helper()
	content := "HTTP/1.1 200 OK\nContent-Type: text/html\n\n<html><body>Raw</body></html>"
	createDummyFile(t, filePath, content)
}

// Helper function to create a dummy body-only file
func createDummyBodyOnlyFile(t *testing.T, filePath string) {
	t.Helper()
	content := "<html><body>BodyOnly</body></html>"
	createDummyFile(t, filePath, content)
}

func TestDetectOfflineFormat(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name         string
		setupFunc    func(t *testing.T, path string)
		path         string
		expectedFormat input.OfflineFormat
	}{
		{
			name:         "FFF Directory",
			setupFunc:    createDummyFFFDir,
			path:         filepath.Join(tmpDir, "fff_test_dir"),
			expectedFormat: input.FormatFFF,
		},
		{
			name:         "Katana Directory",
			setupFunc:    createDummyKatanaDir,
			path:         filepath.Join(tmpDir, "katana_test_dir"),
			expectedFormat: input.FormatKatanaDir,
		},
		{
			name:         "Single Katana File",
			setupFunc:    func(t *testing.T, path string) { createDummyKatanaFile(t, filepath.Join(path, "katana.txt")) },
			path:         filepath.Join(tmpDir, "katana_test_file", "katana.txt"),
			expectedFormat: input.FormatKatanaFile,
		},
		{
			name:         "Raw HTTP File",
			setupFunc:    func(t *testing.T, path string) { createDummyRawHTTPFile(t, filepath.Join(path, "raw.txt")) },
			path:         filepath.Join(tmpDir, "raw_test_file", "raw.txt"),
			expectedFormat: input.FormatRawHTTP,
		},
		{
			name:         "Body Only File",
			setupFunc:    func(t *testing.T, path string) { createDummyBodyOnlyFile(t, filepath.Join(path, "body.html")) },
			path:         filepath.Join(tmpDir, "body_test_file", "body.html"),
			expectedFormat: input.FormatBodyOnly,
		},
		{
			name:         "Non-existent Path",
			setupFunc:    func(t *testing.T, path string) {}, // No setup, path won't exist
			path:         filepath.Join(tmpDir, "non_existent"),
			expectedFormat: input.FormatUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testBasePath := tt.path
			if strings.Contains(tt.name, "File") { // For file-based tests, create in parent directory
				testBasePath = filepath.Dir(tt.path)
			}
			tt.setupFunc(t, testBasePath)
			
			format := input.DetectOfflineFormat(tt.path)
			if format != tt.expectedFormat {
				t.Errorf("DetectOfflineFormat(%s) = %s; want %s", tt.path, format, tt.expectedFormat)
			}
		})
	}
}

// TestIsDirectoryDetection is added to specifically test the directory detection helpers
func TestIsDirectoryDetection(t *testing.T) {
	tmpDir := t.TempDir()

	// Test FFF directory
	fffDir := filepath.Join(tmpDir, "fff_dir")
	createDummyFFFDir(t, fffDir) // This creates fff_dir/example.com/a94a8fe5ccb19ba61c4c0873d391e987982fbbd3.{headers,body}
	
	// `createDummyFFFDir` creates content inside a subdirectory (example.com)
	// We need to pass the actual directory where the FFF pattern exists to IsFFFDirectory
	isFFF := input.IsFFFDirectory(fffDir)
	if !isFFF {
		t.Errorf("Expected %s to be an FFF directory, but it was not", fffDir)
	}

	// Test Katana directory
	katanaDir := filepath.Join(tmpDir, "katana_dir")
	createDummyKatanaDir(t, katanaDir) // This creates katana_dir/example.com/index.txt and hash.txt
	
	// `createDummyKatanaDir` creates content inside a subdirectory (example.com)
	isKatana := input.IsKatanaDirectory(katanaDir)
	if !isKatana {
		t.Errorf("Expected %s to be a Katana directory, but it was not", katanaDir)
	}

	// Test a directory that is neither FFF nor Katana
	neitherDir := filepath.Join(tmpDir, "neither_dir")
	os.MkdirAll(neitherDir, 0755)
	createDummyFile(t, filepath.Join(neitherDir, "random.txt"), "some content")
	
	isFFF = input.IsFFFDirectory(neitherDir)
	if isFFF {
		t.Errorf("Expected %s NOT to be an FFF directory, but it was", neitherDir)
	}
	isKatana = input.IsKatanaDirectory(neitherDir)
	if isKatana {
		t.Errorf("Expected %s NOT to be a Katana directory, but it was", neitherDir)
	}
}
