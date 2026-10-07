// Package cbz writes a volume's pages into a .cbz file - a plain zip
// archive of images, understood by comic/manga readers. Mangabind never
// resizes, recompresses, or otherwise touches image content (see the
// README's "Out of scope" section); pages are stored as-is.
package cbz

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"

	"github.com/gustavommcv/mangabind/internal/grouper"
)

// Write creates a .cbz at outPath containing pages, in the given order.
// It creates outPath's parent directory if needed, and replaces an existing
// file at outPath, but only once the new one is complete: see WriteWithProgress.
func Write(outPath string, pages []grouper.Page) error {
	return WriteWithProgress(context.Background(), outPath, pages, nil)
}

// WriteWithProgress writes the same archive as Write and calls onPage after
// each page has been copied.
//
// The archive is built under another name, ".<name>.<random>.part" in the same
// folder, and moved onto outPath only when it is complete and has been flushed
// to disk, so outPath is always either what it was before or the whole new
// volume - never half of one. Before, outPath was truncated the moment the
// write began: a run that failed or was interrupted halfway left a volume that
// looked finished and was not (or, worse, was a valid archive with the pages
// after the failure missing), and took with it the good volume it replaced.
// On any failure the part file is removed.
//
// ctx is looked at before each page. When it is done the write stops, the part
// file is removed, and the error wraps ctx's: errors.Is(err, context.Canceled)
// is how a caller tells an interruption from a failure.
func WriteWithProgress(ctx context.Context, outPath string, pages []grouper.Page, onPage func(completed int)) (err error) {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	f, err := createPart(outPath)
	if err != nil {
		return err
	}
	part := f.Name()
	defer func() {
		if err != nil {
			f.Close()
			_ = os.Remove(part) // best effort: the error that matters is the one being returned
		}
	}()

	sources := &sourceArchives{}
	defer sources.close()

	zw := zip.NewWriter(f)
	for index, p := range pages {
		if err := ctx.Err(); err != nil {
			zw.Close()
			return err
		}
		if err := addPage(zw, sources, p); err != nil {
			zw.Close()
			return fmt.Errorf("adding %s: %w", p.SourcePath, err)
		}
		if onPage != nil {
			onPage(index + 1)
		}
	}
	// Every step that can fail on the way to a complete file is checked: the
	// central directory, the flush to disk and the close, where a network or
	// quota error that was held back until then is finally reported.
	if err := zw.Close(); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(part, outPath)
}

// createPart creates the file a volume is built in, next to outPath so that
// moving it onto outPath is a rename within one folder. The name begins with a
// dot and ends in ".part", so it is never mistaken for a volume, and it is
// created with the permissions an ordinary new file gets, which is what the
// volume will have.
func createPart(outPath string) (*os.File, error) {
	dir, name := filepath.Split(outPath)
	for attempt := 0; ; attempt++ {
		part := filepath.Join(dir, fmt.Sprintf(".%s.%06d.part", name, rand.IntN(1_000_000)))
		f, err := os.OpenFile(part, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
		if err == nil {
			return f, nil
		}
		if !errors.Is(err, os.ErrExist) || attempt >= 100 {
			return nil, err
		}
	}
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
	return fmt.Sprintf("%s - Vol.%s.cbz", manga, grouper.VolumeLabel(volume))
}

// SeriesFileName builds the output filename for a whole series combined into
// one volume, e.g. "Chainsaw Man.cbz" - see
// docs/adr/0012-combine-series-into-one-volume.md.
func SeriesFileName(manga string) string {
	return fmt.Sprintf("%s.cbz", manga)
}
