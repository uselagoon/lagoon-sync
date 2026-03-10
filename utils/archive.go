package utils

import (
	"archive/tar"
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
}

type ArchiveItem struct {
	Syncher  string            `yaml:"syncher"`        // which syncher we need to use to pull/push the data
	Filename string            `yaml:"filename"`       // the resulting file
	Data     map[string]string `yaml:"data,omitempty"` // any data we need to pass to the syncer
}

// At this point we assume the synchers will have been

func InitArchive(filename string) (*Archive, error) {

	if !strings.HasSuffix(filename, TarGzExtension) {
		return nil, fmt.Errorf("Archive filename does not end with .tar.gz")
	}

	return &Archive{
		ArchiveFilename: filename,
	}, nil
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
				return err
			}
		}
		return nil
	}

	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return err
	}

	header.Name = fn

	err = tarWriter.WriteHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(tarWriter, file)

	return err // will be nil or not depending on io.Copy's success

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
