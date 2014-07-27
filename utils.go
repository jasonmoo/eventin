package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type (
	GzipResponseWriter struct {
		io.Writer
		http.ResponseWriter
	}
)

func (g GzipResponseWriter) Write(b []byte) (int, error) {
	return g.Writer.Write(b)
}

func NewGzipHandler(f func(w http.ResponseWriter, req *http.Request)) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		encoding := req.Header.Get("Accept-Encoding")

		if strings.Contains(encoding, "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			gz := gzip.NewWriter(w)
			defer gz.Close()
			f(GzipResponseWriter{gz, w}, req)
		} else {
			f(w, req)
		}

	})
}

func NewGzipFileHandler(path string, ignore_exts []string) http.HandlerFunc {

	type (
		CachedFile struct {
			Name        string
			Content     io.ReadSeeker
			GzipContent io.ReadSeeker
			Info        os.FileInfo
		}
	)

	var (
		CachedFiles      = make(map[string]*CachedFile)
		CachedFilesMutex sync.RWMutex
	)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		if req.URL.Path == "/" {
			req.URL.Path = "/index.html"
		}

		var (
			file_path = filepath.Join(path, req.URL.Path)
			ext       = filepath.Ext(file_path)
			encoding  = req.Header.Get("Accept-Encoding")
			mime_type = mime.TypeByExtension(ext)
		)

		CachedFilesMutex.RLock()
		file, exists := CachedFiles[file_path]
		CachedFilesMutex.RUnlock()

		if !exists {

			f, err := os.Open(file_path)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}

			info, err := f.Stat()
			if err != nil || info.IsDir() {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			content_bytes := new(bytes.Buffer)
			_, err = io.Copy(content_bytes, f)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			content := bytes.NewReader(content_bytes.Bytes())

			gzip_content_bytes := new(bytes.Buffer)
			gz := gzip.NewWriter(gzip_content_bytes)
			_, err = io.Copy(gz, content)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			gz.Close()
			gzip_content := bytes.NewReader(gzip_content_bytes.Bytes())

			file = &CachedFile{
				Name:        file_path,
				Content:     content,
				GzipContent: gzip_content,
				Info:        info,
			}

			// don't cache files in dev mode
			if !*dev {
				CachedFilesMutex.Lock()
				CachedFiles[file_path] = file
				CachedFilesMutex.Unlock()
			}

		}

		if strings.Contains(encoding, "gzip") && !inSlice(ignore_exts, ext) && mime_type != "" {
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Content-Type", mime_type)
			http.ServeContent(w, req, file.Info.Name(), file.Info.ModTime(), file.GzipContent)
		} else {
			http.ServeContent(w, req, file.Info.Name(), file.Info.ModTime(), file.Content)
		}

	})
}

func inSlice(haystack []string, needle string) bool {
	for _, v := range haystack {
		if strings.EqualFold(v, needle) {
			return true
		}
	}
	return false
}
