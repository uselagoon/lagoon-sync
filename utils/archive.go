package utils

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

const TarGzExtension = ".tar.gz"

type Archive struct {
	ArchiveFilename string        `yaml:"archivefilename"` // Used primarily internally for creating the archive.
	Items           []ArchiveItem `yaml:"items"`
	Version         string        `yaml:"version,omitempty"`
}

type ArchiveItem struct {
	Syncher  string            `yaml:"syncher"`        // which syncher we need to use to pull/push the data
	Filename string            `yaml:"filename"`       // the resulting file
	Data     map[string]string `yaml:"data,omitempty"` // any data we need to pass to the syncer
}

func InitArchive(filename, version string) (*Archive, error) {

	if !strings.HasSuffix(filename, TarGzExtension) {
		return nil, fmt.Errorf("Archive filename does not end with .tar.gz")
	}

	return &Archive{
		ArchiveFilename: filename,
		Version:         version,
	}, nil
}

func ExtractManifest(archiveFileName string) (*Archive, error) {
	file, err := os.Open(archiveFileName)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if header.Typeflag == tar.TypeReg && header.Name == "manifest.yml" {
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, tr); err != nil {
				return nil, fmt.Errorf("failed to copy file content: %w", err)
			}
			// else, we potentially have found out manifest - let's pull it out
			archiveManifest := Archive{}
			err = yaml.Unmarshal(buf.Bytes(), &archiveManifest)
			if err != nil {
				return nil, err
			}

			return &archiveManifest, nil
		}
	}

	return nil, fmt.Errorf("Manifest not found in archive")
}

// ExtractError is returned by ExtractFromArchive when an individual archive
// entry cannot be extracted. EntryType is the tar type flag (e.g. tar.TypeReg,
// tar.TypeDir) and Name is the resolved destination path.
type ExtractError struct {
	EntryType byte
	Name      string
	Err       error
}

func (e *ExtractError) Error() string {
	return fmt.Sprintf("extract entry (type %d) %q: %v", e.EntryType, e.Name, e.Err)
}

func (e *ExtractError) Unwrap() error { return e.Err }

// safeExtractPath resolves the destination path for an archive entry within
// base, ensuring the result does not escape base (zip-slip protection).
// Absolute entry names are made relative by stripping the leading separator.
func safeExtractPath(base, entryName string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(entryName))
	if clean == "" || clean == "." {
		return "", fmt.Errorf("empty entry name")
	}
	// Make relative so that absolute archive paths land inside base.
	relative := strings.TrimPrefix(clean, string(os.PathSeparator))
	if relative == "" {
		return "", fmt.Errorf("entry %q has no path components after normalisation", entryName)
	}
	dest := filepath.Join(base, relative)
	rel, err := filepath.Rel(base, dest)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("entry %q would escape target directory", entryName)
	}
	return dest, nil
}

// ExtractFromArchive extracts entries from a .tar.gz archive whose names match
// matchPrefix into targetPath. Pass matchPrefix="" to extract everything.
//
// When ignoreAbsPath is false, entries with absolute paths are rejected.
// When ignoreAbsPath is true, absolute entry paths are normalised — the leading
// separator is stripped and the entry is extracted relative to targetPath.
func ExtractFromArchive(archiveFileName, matchPrefix, targetPath string, ignoreAbsPath bool) error {
	if targetPath == "" {
		return fmt.Errorf("Cannot have an empty extraction directory")
	}

	file, err := os.Open(archiveFileName)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("resolving target path %q: %w", targetPath, err)
	}

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if matchPrefix != "" && !strings.HasPrefix(header.Name, matchPrefix) {
			continue
		}

		if !ignoreAbsPath && filepath.IsAbs(filepath.FromSlash(header.Name)) {
			return fmt.Errorf("archive entry %q has an absolute path", header.Name)
		}

		safeName, err := safeExtractPath(absTarget, header.Name)
		if err != nil {
			return fmt.Errorf("archive entry %q would escape target directory", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			LogProcessStep("Extracting directory "+safeName, nil)
			if info, statErr := os.Stat(safeName); statErr == nil {
				// path already exists — ensure it's a directory and writable
				if !info.IsDir() {
					return &ExtractError{EntryType: tar.TypeDir, Name: safeName, Err: fmt.Errorf("path exists but is not a directory")}
				}
				tmp, err := os.CreateTemp(safeName, ".write-check-*")
				if err != nil {
					return &ExtractError{EntryType: tar.TypeDir, Name: safeName, Err: fmt.Errorf("directory is not writable: %w", err)}
				}
				tmp.Close()
				os.Remove(tmp.Name())
			} else if os.IsNotExist(statErr) {
				if err := os.MkdirAll(safeName, os.FileMode(header.Mode&0777)); err != nil {
					return &ExtractError{EntryType: tar.TypeDir, Name: safeName, Err: fmt.Errorf("creating directory: %w", err)}
				}
			} else {
				return &ExtractError{EntryType: tar.TypeDir, Name: safeName, Err: fmt.Errorf("stat failed: %w", statErr)}
			}

		case tar.TypeReg:
			LogProcessStep("Extracting "+safeName, nil)
			if err := os.MkdirAll(filepath.Dir(safeName), 0750); err != nil {
				return &ExtractError{EntryType: tar.TypeReg, Name: safeName, Err: fmt.Errorf("creating parent dirs: %w", err)}
			}
			out, err := os.OpenFile(safeName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode&0777))
			if err != nil {
				return &ExtractError{EntryType: tar.TypeReg, Name: safeName, Err: fmt.Errorf("creating file: %w", err)}
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return &ExtractError{EntryType: tar.TypeReg, Name: safeName, Err: fmt.Errorf("writing file contents: %w", err)}
			}
			out.Close()
		}
	}

	return nil
}

func (a *Archive) AddItem(syncher, fileName string, data map[string]string) error {

	// first we check this item actually exists
	_, err := os.Stat(fileName)

	if err != nil {
		return err
	}

	newItem := ArchiveItem{
		Syncher:  syncher,
		Filename: fileName,
		Data:     data,
	}
	a.Items = append(a.Items, newItem)
	return nil
}

func (a *Archive) WriteArchive() error {
	if a.ArchiveFilename == "" {
		return fmt.Errorf("No filename set for archive")
	}

	// Working from https://www.arthurkoziel.com/writing-tar-gz-files-in-go/
	// and the std library docs

	// TODO: think about having the possibility of failing if the file exists

	out, err := os.Create(a.ArchiveFilename)
	if err != nil {
		return err
	}

	defer out.Close() // TODO: do we remove the file if something goes wrong?

	gw := gzip.NewWriter(out)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// now we create a manifest file

	manifest, err := yaml.Marshal(a)
	if err != nil {
		return err
	}

	// Write manifest directly into the tar archive (no temp file)
	manifestHeader := &tar.Header{
		Name:    "manifest.yml",
		Mode:    0600,
		Size:    int64(len(manifest)),
		ModTime: time.Now(),
	}

	if err := tw.WriteHeader(manifestHeader); err != nil {
		return fmt.Errorf("writing manifest header: %w", err)
	}
	if _, err := tw.Write(manifest); err != nil {
		return fmt.Errorf("writing manifest body: %w", err)
	}

	// now we iterate over the files and add 'em to the archive

	for _, file := range a.Items {

		err = writeToTar(tw, file.Filename)

		if err != nil {
			return err
		}

	}

	return nil

}

func writeToTar(tarWriter *tar.Writer, fn string) error {

	file, err := os.Open(fn)
	if err != nil {
		return err
	}

	defer file.Close()

	// we need to ensure that this isn't we recurse

	info, err := file.Stat()
	if err != nil {
		return err
	}

	if info.IsDir() {
		files, err := unwindFolder(fn)
		if err != nil {
			return err
		}
		for _, f := range files {
			if err := writeToTar(tarWriter, f); err != nil {
				return fmt.Errorf("writing %s to tar: %w", f, err)
			}
		}
		return nil
	}

	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return err
	}

	// Use PAX format: no name-length limit (USTAR caps at 255 bytes total).
	header.Format = tar.FormatPAX
	header.Name = fn

	err = tarWriter.WriteHeader(header)
	if err != nil {
		return err
	}

	// LimitReader caps the copy at exactly header.Size bytes.
	// If the file is still being appended to (e.g. a live log), io.Copy
	// would write more bytes than the header declared and corrupt the archive.
	// LimitReader gives us a consistent point-in-time snapshot with zero
	// buffering — we stream directly from source to tar writer.
	_, err = io.Copy(tarWriter, io.LimitReader(file, info.Size()))

	return err

}

// unwindFolder takes a file or directory path and returns a flat list of all
// contained file paths. Directories are walked recursively; empty directories
// are silently skipped.
func unwindFolder(folderName string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(folderName, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}
