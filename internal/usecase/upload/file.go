package upload

import (
	"mime"
	"net/http"
	"path"
	"strings"
)

const (
	// _sniffLen is how much content http.DetectContentType looks at.
	_sniffLen = 512
	// _maxNameLen matches upload_files.name VARCHAR(255).
	_maxNameLen = 255
	// _maxExtLen matches entity.UploadFile's extension limit, dot included.
	_maxExtLen = 16
)

// baseName keeps only the last path element of a client-supplied file name; some browsers send
// a full path such as C:\fakepath\cover.png.
func baseName(name string) string {
	name = path.Base(strings.ReplaceAll(name, `\`, "/"))
	if name == "." || name == "/" {
		return ""
	}

	return strings.TrimSpace(name)
}

// normalizeExt returns the lowercased extension of name, or "" when it is missing or not plain
// ASCII letters and digits — the file is still stored, just without an extension.
func normalizeExt(name string) string {
	ext := strings.ToLower(path.Ext(name))
	if len(ext) < 2 || len(ext) > _maxExtLen {
		return ""
	}

	for _, c := range ext[1:] {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			return ""
		}
	}

	return ext
}

// detectMime prefers the type implied by the extension, which is also the Content-Type the file
// is served with, and falls back to sniffing the content. The client's declared Content-Type is
// never trusted. Parameters such as "; charset=utf-8" are dropped.
func detectMime(ext string, head []byte) string {
	t := mime.TypeByExtension(ext)
	if ext == "" || t == "" {
		t = http.DetectContentType(head)
	}

	if mediaType, _, err := mime.ParseMediaType(t); err == nil {
		return mediaType
	}

	return "application/octet-stream"
}
