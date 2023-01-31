package sizedupload

import (
	"bytes"
	"context"
	"errors"
	"github.com/rolandhe/sizedupload/upctx"
	"io"
	"math"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"time"
)

var (
	errMessageTooLarge = errors.New("multipart: message too large")
	errInvalidWrite    = errors.New("invalid write result")
	errShortWrite      = errors.New("short write")
	err404             = errors.New("404")
	errExceedMax       = errors.New("exceed max file")
	errNoAuth          = errors.New("no auth")
)

type fileHeader struct {
	filename string
	header   textproto.MIMEHeader
	size     int64

	content []byte
	tmpfile string
}

type rawForm struct {
	value map[string][]string
	file  map[string][]*fileHeader
}

type ParsedFile struct {
	FormData map[string]string

	FileData *struct {
		OriginName string
		Size       int64
		Content    []byte
		filePath   string
	}
}

func (f *rawForm) removeAll() error {
	var err error
	for _, fhs := range f.file {
		for _, fh := range fhs {
			if fh.tmpfile != "" {
				e := os.Remove(fh.tmpfile)
				if e != nil && err == nil {
					err = e
				}
			}
		}
	}
	return err
}

func upload(context context.Context, req *http.Request, urlPath string) (_ *ParsedFile, userId int64, _ error) {
	ct := req.Header.Get("Content-Type")
	// RFC 7231, section 3.1.1.5 - empty type
	//   MAY be treated as application/octet-stream
	if ct == "" {
		ct = "application/octet-stream"
	}
	ct, _, _ = mime.ParseMediaType(ct)
	if ct == "multipart/form-data" && req.Method == "POST" {
		userId := UploadConfig.AuthFunc(req, context)
		if userId == 0 {
			return nil, 0, errNoAuth
		}

		mxLimit := UploadConfig.GetSize(urlPath, context)
		if mxLimit <= 0 {
			mxLimit = 4 * 1024 * 1024
		}
		myForm, err := parseStream(context, req, mxLimit, userId)
		defer func(f *rawForm) {
			if f != nil {
				f.removeAll()
			}
		}(myForm)

		if err != nil {
			return nil, userId, err
		}

		return convertRawForm(myForm), userId, nil
	}

	return nil, 0, err404
}

func convertRawForm(form *rawForm) *ParsedFile {
	var fh *fileHeader
	for _, v := range form.file {
		fh = v[0]
		break
	}

	formData := map[string]string{}
	for k, vs := range form.value {
		formData[k] = vs[0]
	}

	return &ParsedFile{
		FormData: formData,
		FileData: &struct {
			OriginName string
			Size       int64
			Content    []byte
			filePath   string
		}{OriginName: fh.filename, Size: fh.size, Content: fh.content, filePath: fh.tmpfile},
	}
}

func readForm(r *multipart.Reader, maxMemory int64, maxFileSize int64) (_ *rawForm, err error) {
	form := &rawForm{make(map[string][]string), make(map[string][]*fileHeader)}

	// Reserve an additional 10 KB for non-file parts.
	maxValueBytes := maxMemory + int64(10*1024)
	if maxValueBytes <= 0 {
		if maxMemory < 0 {
			maxValueBytes = 0
		} else {
			maxValueBytes = math.MaxInt64
		}
	}
	fileSize := 0
	for {
		p, err := r.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		fileSize++

		if fileSize > 1 {
			return nil, errors.New("just upload one file")
		}

		name := p.FormName()
		if name == "" {
			continue
		}
		filename := p.FileName()

		var b bytes.Buffer

		if filename == "" {
			// value, store as string in memory
			n, err := io.CopyN(&b, p, maxValueBytes+1)
			if err != nil && err != io.EOF {
				return nil, err
			}
			maxValueBytes -= n
			if maxValueBytes < 0 {
				return nil, errMessageTooLarge
			}
			form.value[name] = append(form.value[name], b.String())
			continue
		}

		// file, store in memory or on disk
		fh := &fileHeader{
			filename: filename,
			header:   p.Header,
		}
		n, err := io.CopyN(&b, p, maxMemory+1)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n > maxMemory {
			// too big, write to disk and flush buffer
			file, err := os.CreateTemp("", "multipart-")
			if err != nil {
				return nil, err
			}
			size, err := copyData(file, io.MultiReader(&b, p), maxFileSize)
			if cerr := file.Close(); err == nil {
				err = cerr
			}
			if err != nil {
				os.Remove(file.Name())
				return nil, err
			}
			fh.tmpfile = file.Name()
			fh.size = size
		} else {
			fh.content = b.Bytes()
			fh.size = int64(len(fh.content))
			maxMemory -= n
			maxValueBytes -= n
		}
		form.file[name] = append(form.file[name], fh)
	}

	return form, nil
}

func copyData(dst io.Writer, src io.Reader, maxFile int64) (written int64, err error) {
	size := 32 * 1024
	if l, ok := src.(*io.LimitedReader); ok && int64(size) > l.N {
		if l.N < 1 {
			size = 1
		} else {
			size = int(l.N)
		}
	}
	buf := make([]byte, size)

	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = errInvalidWrite
				}
			}
			written += int64(nw)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = errShortWrite
				break
			}
			if written > maxFile {
				err = errExceedMax
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}

func parseStream(ctx context.Context, req *http.Request, maxFileSize int64, userId int64) (*rawForm, error) {
	start := time.Now().UnixMilli()
	UploadConfig.LogInfo("tid=%s,uid:%d, call parseStream", upctx.GetTraceId(ctx), userId)
	var parseFormErr error
	if req.Form == nil {
		// Let errors in ParseForm fall through, and just
		// return it at the end.
		parseFormErr = req.ParseForm()
	}
	if req.MultipartForm != nil {
		return nil, nil
	}

	mr, err := req.MultipartReader()
	if err != nil {
		return nil, err
	}

	memMx := UploadConfig.MemoryLimit
	if memMx <= 0 {
		memMx = 10 * 1024
	}

	f, err := readForm(mr, memMx, maxFileSize)
	if err != nil {
		return nil, err
	}

	if req.PostForm == nil {
		req.PostForm = make(url.Values)
	}
	for k, v := range f.value {
		req.Form[k] = append(req.Form[k], v...)
		// r.PostForm should also be populated. See Issue 9305.
		req.PostForm[k] = append(req.PostForm[k], v...)
	}
	cost := time.Now().UnixMilli() - start
	UploadConfig.LogInfo("tid=%s, parse stream url: %s, uid=%d,cost=%d ms", upctx.GetTraceId(ctx), upctx.GetUrl(ctx), userId, cost)
	return f, parseFormErr
}
