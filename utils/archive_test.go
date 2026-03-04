package utils

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v2"
)

const testDataDir = "test_data/archive_test/"

func TestInitArchive(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
		wantErr  bool
	}{
		{
			name:     "creates archive with valid .tar.gz filename",
			filename: testDataDir + "test-archive.tar.gz",
			want:     testDataDir + "test-archive.tar.gz",
			wantErr:  false,
		},
		{
			name:     "fails with .tar extension only",
			filename: testDataDir + "test-archive.tar",
			want:     "",
			wantErr:  true,
		},
		{
			name:     "fails with empty filename",
			filename: "",
			want:     "",
			wantErr:  true,
		},
		{
			name:     "fails with no extension",
			filename: testDataDir + "test-archive",
			want:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archive, err := InitArchive(tt.filename)

			if (err != nil) != tt.wantErr {
				t.Errorf("InitArchive() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if archive != nil {
					t.Errorf("InitArchive() expected nil archive on error")
				}
				return
			}

			if archive.ArchiveFilename != tt.want {
				t.Errorf("InitArchive() ArchiveFilename = %v, want %v", archive.ArchiveFilename, tt.want)
			}
		})
	}
}

func TestArchive_AddItem(t *testing.T) {
	tests := []struct {
		name     string
		syncher  string
		filename string
		data     map[string]string
		wantLen  int
		wantErr  bool
	}{
		{
			name:     "Fail test - file does not exist",
			syncher:  "mariadb",
			filename: testDataDir + "idontexist.sql",
			data:     map[string]string{"host": "localhost", "port": "3306"},
			wantLen:  0,
			wantErr:  true,
		},
		{
			name:     "adds item with all fields",
			syncher:  "mariadb",
			filename: testDataDir + "database.sql",
			data:     map[string]string{"host": "localhost", "port": "3306"},
			wantLen:  1,
			wantErr:  false,
		},
		{
			name:     "adds item with nil data",
			syncher:  "files",
			filename: testDataDir + "files.tar.gz",
			data:     nil,
			wantLen:  1,
			wantErr:  false,
		},
		{
			name:     "adds item with empty data",
			syncher:  "postgres",
			filename: testDataDir + "pg_dump.sql",
			data:     map[string]string{},
			wantLen:  1,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			archivePath := filepath.Join(tmpDir, "test.tar.gz")

			archive, err := InitArchive(archivePath)
			if err != nil {
				t.Fatalf("InitArchive() unexpected error: %v", err)
			}

			err = archive.AddItem(tt.syncher, tt.filename, tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("AddItem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(archive.Items) != tt.wantLen {
				t.Errorf("AddItem() items length = %v, want %v", len(archive.Items), tt.wantLen)
				return
			}

			if tt.wantLen > 0 {
				item := archive.Items[0]
				if item.Syncher != tt.syncher {
					t.Errorf("AddItem() Syncher = %v, want %v", item.Syncher, tt.syncher)
				}
				if item.Filename != tt.filename {
					t.Errorf("AddItem() Filename = %v, want %v", item.Filename, tt.filename)
				}
			}
		})
	}
}

func TestArchive_WriteArchive(t *testing.T) {
	tests := []struct {
		name      string
		files     []string
		wantFiles int // expected item files (excluding manifest)
		wantErr   bool
	}{
		{
			name:      "writes single file archive",
			files:     []string{testDataDir + "database.sql"},
			wantFiles: 1,
			wantErr:   false,
		},
		{
			name: "writes multiple files archive",
			files: []string{
				testDataDir + "database.sql",
				testDataDir + "pg_dump.sql",
				testDataDir + "db1.sql",
			},
			wantFiles: 3,
			wantErr:   false,
		},
		{
			name:      "writes empty archive with manifest only",
			files:     []string{},
			wantFiles: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			archivePath := filepath.Join(tmpDir, "output.tar.gz")

			archive, err := InitArchive(archivePath)
			if err != nil {
				t.Fatalf("InitArchive() error: %v", err)
			}

			for _, f := range tt.files {
				err = archive.AddItem("test-syncher", f, nil)
				if err != nil {
					t.Fatalf("AddItem() error: %v", err)
				}
			}

			err = archive.WriteArchive()
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteArchive() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Verify archive file exists
			if _, err := os.Stat(archivePath); os.IsNotExist(err) {
				t.Errorf("WriteArchive() did not create file at %s", archivePath)
				return
			}

			// Verify archive contents (manifest + item files)
			filesInArchive := readTarGzFileNames(t, archivePath)
			wantTotal := tt.wantFiles + 1 // +1 for manifest.yml
			if len(filesInArchive) != wantTotal {
				t.Errorf("WriteArchive() archive contains %d entries, want %d (manifest + %d files)",
					len(filesInArchive), wantTotal, tt.wantFiles)
			}

			// Verify manifest.yml is first entry
			if len(filesInArchive) == 0 || filesInArchive[0] != "manifest.yml" {
				t.Errorf("WriteArchive() expected manifest.yml as first entry, got %v", filesInArchive)
			}

			// Verify each expected item file is in the archive
			for _, expectedFile := range tt.files {
				found := false
				for _, archiveFile := range filesInArchive {
					if archiveFile == expectedFile {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("WriteArchive() archive missing file: %s", expectedFile)
				}
			}
		})
	}
}

func TestArchive_WriteArchive_VerifyContents(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "verify-contents.tar.gz")

	archive, err := InitArchive(archivePath)
	if err != nil {
		t.Fatalf("InitArchive() error: %v", err)
	}

	testFile := testDataDir + "database.sql"
	err = archive.AddItem("mariadb", testFile, nil)
	if err != nil {
		t.Fatalf("AddItem() error: %v", err)
	}

	err = archive.WriteArchive()
	if err != nil {
		t.Fatalf("WriteArchive() error: %v", err)
	}

	// Read original file content
	originalContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read original file: %v", err)
	}

	// Read content from archive
	archivedContent := readFileFromTarGz(t, archivePath, testFile)

	if string(archivedContent) != string(originalContent) {
		t.Errorf("Archived content does not match original.\nOriginal: %s\nArchived: %s",
			string(originalContent), string(archivedContent))
	}
}

func TestArchive_WriteArchive_ManifestContent(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "manifest-check.tar.gz")

	archive, err := InitArchive(archivePath)
	if err != nil {
		t.Fatalf("InitArchive() error: %v", err)
	}

	testFiles := []struct {
		syncher  string
		filename string
		data     map[string]string
	}{
		{"mariadb", testDataDir + "database.sql", map[string]string{"host": "localhost"}},
		{"postgres", testDataDir + "pg_dump.sql", nil},
	}

	for _, tf := range testFiles {
		err = archive.AddItem(tf.syncher, tf.filename, tf.data)
		if err != nil {
			t.Fatalf("AddItem() error: %v", err)
		}
	}

	err = archive.WriteArchive()
	if err != nil {
		t.Fatalf("WriteArchive() error: %v", err)
	}

	// Read manifest from archive
	manifestBytes := readFileFromTarGz(t, archivePath, "manifest.yml")

	// Unmarshal and verify
	var manifest Archive
	err = yaml.Unmarshal(manifestBytes, &manifest)
	if err != nil {
		t.Fatalf("Failed to unmarshal manifest: %v", err)
	}

	if manifest.ArchiveFilename != archivePath {
		t.Errorf("Manifest ArchiveFilename = %v, want %v", manifest.ArchiveFilename, archivePath)
	}

	if len(manifest.Items) != len(testFiles) {
		t.Fatalf("Manifest items count = %d, want %d", len(manifest.Items), len(testFiles))
	}

	for i, tf := range testFiles {
		item := manifest.Items[i]
		if item.Syncher != tf.syncher {
			t.Errorf("Manifest item[%d].Syncher = %v, want %v", i, item.Syncher, tf.syncher)
		}
		if item.Filename != tf.filename {
			t.Errorf("Manifest item[%d].Filename = %v, want %v", i, item.Filename, tf.filename)
		}
	}
}

// Helper: read file names from a tar.gz archive
func readTarGzFileNames(t *testing.T, archivePath string) []string {
	t.Helper()

	f, err := os.Open(archivePath)
	if err != nil {
		t.Fatalf("Failed to open archive: %v", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	var files []string

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Error reading tar: %v", err)
		}
		files = append(files, header.Name)
	}

	return files
}

// Helper: read a specific file's content from a tar.gz archive
func readFileFromTarGz(t *testing.T, archivePath, fileName string) []byte {
	t.Helper()

	f, err := os.Open(archivePath)
	if err != nil {
		t.Fatalf("Failed to open archive: %v", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			t.Fatalf("File %s not found in archive", fileName)
		}
		if err != nil {
			t.Fatalf("Error reading tar: %v", err)
		}

		if header.Name == fileName {
			content, err := io.ReadAll(tr)
			if err != nil {
				t.Fatalf("Failed to read file content: %v", err)
			}
			return content
		}
	}
}
