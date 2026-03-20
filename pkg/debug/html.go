package debug

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

type HTMLProvider interface {
	HTML() (string, error)
}

func WritePageHTMLToFile(page HTMLProvider, filename string) {
	if page == nil || filename == "" {
		return
	}

	html, err := page.HTML()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"file": filename,
		}).WithError(err).Warn("get page html failed, skip writing file")
		return
	}

	absPath, absErr := filepath.Abs(filename)
	if absErr != nil {
		absPath = filename
	}

	if writeErr := os.WriteFile(filename, []byte(html), 0o644); writeErr != nil {
		logrus.WithFields(logrus.Fields{
			"file": filename,
			"path": absPath,
			"size": len(html),
		}).WithError(writeErr).Warn("write page html to file failed")
		return
	}

	logrus.WithFields(logrus.Fields{
		"file": filename,
		"path": absPath,
		"size": len(html),
	}).Debug("page html written to file")
}
