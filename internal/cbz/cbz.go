// Package cbz writes a volume's pages into a .cbz file - a plain zip
// archive of images, understood by comic/manga readers. Mangabind never
// resizes, recompresses, or otherwise touches image content (see the
// README's "Out of scope" section); pages are stored as-is.
package cbz

import (
	"archive/zip"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gustavommcv/Mangabind/internal/grouper"
)

// Write creates a .cbz at outPath containing pages, in the given order.
// It creates outPath's parent directory if needed, and overwrites an
// existing file at outPath.
func Write(outPath string, pages []grouper.Page) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	for _, p := range pages {
		if err := addPage(zw, p); err != nil {
			zw.Close()
			return fmt.Errorf("adding %s: %w", p.SourcePath, err)
		}
	}
	return zw.Close()
}

func addPage(zw *zip.Writer, p grouper.Page) error {
	src, err := os.Open(p.SourcePath)
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
