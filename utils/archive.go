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

// At this point we assume the synchers will have been

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

// ExtractFromArchive extracts entries from a .tar.gz archive whose names match
// matchPrefix into targetPath. Pass matchPrefix="" to extract everything.
//
// Security: archive entry paths are sanitised before writing — leading slashes
// and ".." components are stripped to prevent path traversal (zip-slip).
func ExtractFromArchive(archiveFileName, matchPrefix, targetPath string, ignoreAbsPath bool) error {
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

		// Sanitize the entry name against path traversal (zip-slip).
		cleanName := filepath.Clean(filepath.FromSlash(header.Name))

		// Reject empty names (Clean("") == ".").
		if cleanName == "" || cleanName == "." {
			return fmt.Errorf("archive entry %q has an empty name", header.Name)
		}
		// Reject absolute paths (handles both Unix '/' and Windows 'C:\' forms).
		if (filepath.IsAbs(cleanName) && targetPath == "") && !ignoreAbsPath {
			return fmt.Errorf("archive entry %q has an absolute path and no targetPath set", header.Name)
		}
		// Reject any path whose components include "..".
		for _, component := range strings.Split(cleanName, string(os.PathSeparator)) {
			if component == ".." {
				return fmt.Errorf("archive entry %q would escape target directory", header.Name)
			}
		}

		safeName := filepath.Join(targetPath, cleanName)

		// Final containment check: verify the resolved destination is inside absTarget.
		absSafe, err := filepath.Abs(safeName)
		if err != nil {
			return fmt.Errorf("resolving path for archive entry %q: %w", header.Name, err)
		}
		if absSafe != absTarget && !strings.HasPrefix(absSafe, absTarget+string(os.PathSeparator)) {
			return fmt.Errorf("archive entry %q would escape target directory", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			LogProcessStep("Extracting directory "+safeName, nil)
			if err := os.MkdirAll(safeName, os.FileMode(header.Mode&0777)); err != nil {
				return fmt.Errorf("creating directory %q: %w", safeName, err)
			}

		case tar.TypeReg:
			LogProcessStep("Extracting "+safeName, nil)
			if err := os.MkdirAll(filepath.Dir(safeName), 0750); err != nil {
				return fmt.Errorf("creating parent dirs for %q: %w", safeName, err)
			}
			out, err := os.OpenFile(safeName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode&0777))
			if err != nil {
				return fmt.Errorf("creating file %q: %w", safeName, err)
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return fmt.Errorf("extracting %q: %w", safeName, err)
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
			fmt.Println("Writing " + f)
			if err := writeToTar(tarWriter, f); err != nil {
				fmt.Println("Failed")
				return err
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
