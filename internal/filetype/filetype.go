package filetype

import (
	"fmt"
	"path/filepath"
	"strings"
)

// PDFSources are Graph-supported extensions for content?format=pdf.
// See https://learn.microsoft.com/en-us/graph/api/driveitem-get-content-format
var PDFSources = map[string]struct{}{
	"doc": {}, "docx": {}, "dot": {}, "dotx": {}, "dotm": {},
	"xls": {}, "xlsx": {}, "xlsm": {},
	"ppt": {}, "pptx": {}, "pps": {}, "ppsx": {},
	"rtf": {},
	"odt": {}, "ods": {}, "odp": {},
	"html": {}, "htm": {},
	"md": {}, "markdown": {},
	"eml": {}, "msg": {},
	"epub": {},
	"tif": {}, "tiff": {},
	"dwg": {},
}

// Resolve returns the Graph upload extension for path, optionally overridden by typ.
func Resolve(path, typ string) (string, error) {
	ext := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(typ)), ".")
	if ext == "" {
		ext = strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	}
	if ext == "" {
		return "", fmt.Errorf("no file type: give a supported extension or --type")
	}
	if _, ok := PDFSources[ext]; !ok {
		return "", fmt.Errorf("unsupported type %q; Graph PDF sources: %s", ext, SupportedList())
	}
	return ext, nil
}

// DefaultOutput is <dir>/<basename>.pdf next to the input.
func DefaultOutput(input string) string {
	dir := filepath.Dir(input)
	base := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
	if base == "" || base == "." {
		base = "output"
	}
	return filepath.Join(dir, base+".pdf")
}

// SupportedList is a stable, comma-separated extension list for errors.
func SupportedList() string {
	names := []string{
		"doc", "docx", "dot", "dotx", "dotm",
		"xls", "xlsx", "xlsm",
		"ppt", "pptx", "pps", "ppsx",
		"rtf", "odt", "ods", "odp",
		"html", "htm", "md", "markdown",
		"eml", "msg", "epub", "tif", "tiff", "dwg",
	}
	return strings.Join(names, ", ")
}
