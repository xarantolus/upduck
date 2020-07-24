package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var errCancelled = fmt.Errorf("request cancelled")

// GenerateZIPFromDir generates a zip file from the given directory
func GenerateZIPFromDir(to io.Writer, directory string, ctx context.Context) (err error) {
	zipW := zip.NewWriter(to)

	err = filepath.Walk(directory, func(path string, f os.FileInfo, err error) error {
		select {
		case <-ctx.Done():
			return errCancelled
		default:
		}

		if err != nil || f.IsDir() {
			return err
		}

		// Get relative path for zip file
		relPath, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}

		// Get the file header that contains all info for the zip file (dates...)
		fh, err := zip.FileInfoHeader(f)
		if err != nil {
			return err
		}
		fh.Name = relPath

		// Open the actual file
		diskFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer diskFile.Close()

		// Create the header in the zip file
		fw, err := zipW.CreateHeader(fh)
		if err != nil {
			return err
		}

		// Copy the file content to the zip file
		_, err = io.Copy(fw, diskFile)

		return err
	})
	if err != nil {
		return err
	}

	// Finalize the zip file
	if err = zipW.Close(); err != nil {
		return err
	}

	return err
}

// GenerateTARFromDir generates a tar file from the given directory
func GenerateTARFromDir(to io.Writer, directory string, ctx context.Context) (err error) {
	tarW := tar.NewWriter(to)

	err = filepath.Walk(directory, func(path string, f os.FileInfo, err error) error {
		select {
		case <-ctx.Done():
			return errCancelled
		default:
		}

		if err != nil || f.IsDir() {
			return err
		}

		// Get relative path for tar file
		relPath, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}

		// Get the file header that contains all info for the tar file (dates...)
		fh, err := tar.FileInfoHeader(f, "")
		if err != nil {
			return err
		}
		fh.Name = relPath

		// Open the actual file
		diskFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer diskFile.Close()

		// Write the header in the tar file
		err = tarW.WriteHeader(fh)
		if err != nil {
			return err
		}

		// Copy the file content to the tar file
		_, err = io.Copy(tarW, diskFile)

		return err
	})
	if err != nil {
		return err
	}

	// Finalize the tar file
	if err = tarW.Close(); err != nil {
		return err
	}

	return err
}

// GenerateTARGZFromDir generates a tar.gz file from the given directory
func GenerateTARGZFromDir(to io.Writer, directory string, ctx context.Context) (err error) {
	w, err := gzip.NewWriterLevel(to, gzip.BestCompression) // I mean, if we want the compressed version then we probably want the best compressed version
	if err != nil {
		return
	}
	defer func() {
		cerr := w.Close()
		if err == nil {
			err = cerr
		}
	}()

	return GenerateTARFromDir(w, directory, ctx)
}
