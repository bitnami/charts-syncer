// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type testFile struct {
	path, data string
}

var testDirContents = []testFile{
	{path: "file-at-root.txt", data: "somedata in the root"},
	{path: ".hiddenfile", data: "somehiddenfile data"},
	{path: "emptyfile"},
	{path: "dir1/somefile", data: "more data"},
	{path: "dir1/dir2/somedeepfile", data: "deep data"},
}

func newDir(label string) string {
	dir, err := os.MkdirTemp("", label+"-*")
	if err != nil {
		log.Fatalf("failed to create test temp dir for %s: %v", label, err)
	}
	return dir
}

func tarTestMemoryFiles(tarFile string, testFiles []testFile) error {
	tfw, err := newTarFileWriter(tarFile)
	if err != nil {
		return fmt.Errorf("failed to create tar file %s: %w", tarFile, err)
	}
	defer tfw.Close()
	for _, testFile := range testFiles {
		if err := tfw.WriteMemFile(testFile.path, []byte(testFile.data), defaultPerm); err != nil {
			log.Fatalf("failed to create test file %s: %v", testFile.path, err)
		}
	}
	return nil
}

func tarTestIOFiles(tarFile string, testFiles []testFile) error {
	tfw, err := newTarFileWriter(tarFile)
	if err != nil {
		return fmt.Errorf("failed to create tar file %s: %w", tarFile, err)
	}
	defer tfw.Close()
	for _, testFile := range testFiles {
		r := bytes.NewBufferString(testFile.data)
		if err := tfw.WriteIOFile(testFile.path, int64(len(testFile.data)), r, defaultPerm); err != nil {
			log.Fatalf("failed to create test file %s: %v", testFile.path, err)
		}
	}
	return nil
}

func newTarFilePath() string {
	return filepath.Join(newDir("test-tar-dir"), "test.tar")
}

func dumpTar(tarFile string) []testFile {
	tarredFiles := []testFile{}
	f, err := os.Open(tarFile)
	if err != nil {
		log.Fatalf("failed to dumpTar %s: %v", tarFile, err)
	}
	defer f.Close()
	tr := tar.NewReader(f)
	for {
		hdr, err := tr.Next()
		if err == io.EOF || hdr == nil {
			return tarredFiles
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			log.Fatalf("failed to dump tarred file %s: %v", hdr.Name, err)
		}
		tarredFiles = append(tarredFiles, testFile{path: hdr.Name, data: string(data)})
	}
}

func cleanup(dirs ...string) {
	for _, dir := range dirs {
		err := os.RemoveAll((dir))
		if err != nil {
			log.Fatalf("failed to cleanup %s: %v", dir, err)
		}
	}
}

var _ = Describe("Tar", func() {
	Context("directory", func() {
		It("tar all test memory files as expected", func() {
			tarFile := newTarFilePath()
			err := tarTestMemoryFiles(tarFile, testDirContents)
			Expect(err).ToNot(HaveOccurred())
			Expect(testDirContents).To(Equal(dumpTar(tarFile)))
			cleanup(filepath.Dir(tarFile))
		})

		It("tar all test fs files as expected", func() {
			tarFile := newTarFilePath()
			err := tarTestIOFiles(tarFile, testDirContents)
			Expect(err).ToNot(HaveOccurred())
			Expect(testDirContents).To(Equal(dumpTar(tarFile)))
			cleanup(filepath.Dir(tarFile))
		})
	})
})
