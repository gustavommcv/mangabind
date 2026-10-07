// Package cbz writes a volume's pages into a .cbz file - a plain zip
// archive of images, understood by comic/manga readers. Mangabind never
// resizes, recompresses, or otherwise touches image content (see the
// README's "Out of scope" section); pages are stored as-is.
package cbz

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gustavommcv/mangabind/internal/grouper"
)

// Write creates a .cbz at outPath containing pages, in the given order.
// It creates outPath's parent directory if needed, and overwrites an
// existing file at outPath.
func Write(outPath string, pages []grouper.Page) error {
	return WriteWithProgress(outPath, pages, nil)
}

// WriteWithProgress writes the same archive as Write and calls onPage after
// each page has been copied. The archive is not complete until this returns.
func WriteWithProgress(outPath string, pages []grouper.Page, onPage func(completed int)) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	sources := &sourceArchives{}
	defer sources.close()

	zw := zip.NewWriter(f)
	for index, p := range pages {
		if err := addPage(zw, sources, p); err != nil {
			zw.Close()
			return fmt.Errorf("adding %s: %w", p.SourcePath, err)
		}
		if onPage != nil {
			onPage(index + 1)
		}
	}
	return zw.Close()
}

func addPage(zw *zip.Writer, sources *sourceArchives, p grouper.Page) error {
	src, err := sources.open(p)
	if err != nil {
		return err
	}
	defer src.Close()

	w, err := zw.CreateHeader(&zip.FileHeader{
		Name:   p.ArchiveName,
		Method: zip.Store, // images are already compressed; don't re-deflate them
	})
	if err != nil {
		return err
	}

	_, err = io.Copy(w, src)
	return err
}

// openArchive opens a source .cbz. It is a variable so that a test can count
// how often it is called.
var openArchive = zip.OpenReader

// sourceArchives opens the pages of a volume: a plain file, or - when
// SourceInArchive is set - an entry inside a source .cbz.
//
// A source archive is opened once and read from for as long as pages keep
// coming from it, which is how they come: a chapter's pages are consecutive.
// It used to be opened again for every page, which meant reading its whole
// directory again each time, so the work grew with the square of the number of
// entries (8,000 entries took two minutes). Only the archive being read is
// kept open, so a volume of hundreds of .cbz chapters never holds hundreds of
// files open at once.
//
// An entry is found through the *zip.File the archive lists, not through
// zip.Reader.Open, which follows io/fs rules: it refuses names such as "./a.jpg"
// or "../a.jpg" that archives made by other tools hold, and with two entries of
// the same name it always returns the first. The scanner lists every entry, in
// order, so the nth page asking for a name gets the nth entry of that name.
type sourceArchives struct {
	path    string                 // the archive that is open, "" for none
	reader  *zip.ReadCloser        // that archive
	entries map[string][]*zip.File // its entries by name, in archive order
	handed  map[string]int         // entries handed out, by archive and name; kept when the archive is closed
}

func (s *sourceArchives) open(p grouper.Page) (io.ReadCloser, error) {
	if p.SourceInArchive == "" {
		return os.Open(p.SourcePath)
	}
	if s.path != p.SourcePath {
		if err := s.use(p.SourcePath); err != nil {
			return nil, err
		}
	}

	if s.handed == nil {
		s.handed = map[string]int{}
	}
	key := p.SourcePath + "\x00" + p.SourceInArchive
	files := s.entries[p.SourceInArchive]
	if s.handed[key] >= len(files) {
		return nil, fmt.Errorf("%s has no entry %q", p.SourcePath, p.SourceInArchive)
	}
	file := files[s.handed[key]]
	s.handed[key]++
	return file.Open()
}

// use closes the archive that is open, if any, and opens the one at path.
func (s *sourceArchives) use(path string) error {
	s.close()
	reader, err := openArchive(path)
	// An archive with a name that is not local ("../a.jpg") is still readable:
	// the name is only ever used to find the entry, never to make a file.
	if err != nil && (!errors.Is(err, zip.ErrInsecurePath) || reader == nil) {
		return err
	}
	entries := make(map[string][]*zip.File, len(reader.File))
	for _, file := range reader.File {
		entries[file.Name] = append(entries[file.Name], file)
	}
	s.path, s.reader, s.entries = path, reader, entries
	return nil
}

func (s *sourceArchives) close() {
	if s.reader != nil {
		s.reader.Close()
	}
	s.path, s.reader, s.entries = "", nil, nil
}

// VolumeFileName builds the output filename for a manga volume, e.g.
// "Chainsaw Man - Vol.01.cbz".
func VolumeFileName(manga string, volume float64) string {
	var numPart string
	if volume == math.Trunc(volume) {
		numPart = fmt.Sprintf("%02d", int(volume))
	} else {
		numPart = strconv.FormatFloat(volume, 'f', -1, 64)
	}
	return fmt.Sprintf("%s - Vol.%s.cbz", manga, numPart)
}

// SeriesFileName builds the output filename for a whole series combined into
// one volume, e.g. "Chainsaw Man.cbz" - see
// docs/adr/0012-combine-series-into-one-volume.md.
func SeriesFileName(manga string) string {
	return fmt.Sprintf("%s.cbz", manga)
}
