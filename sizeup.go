package sizedupload

import (
	"context"
	"github.com/rolandhe/sizedupload/upctx"
	"net/http"
	"time"
)

var UploadConfig = &upConfig{
	MemoryLimit: int64(512 * 1024),
	Logger:      &defaultLogger{},
}

type ProcessFileHandler func(file *ParsedFile, userId int64, context context.Context) (*ProcessFileResult, error)

type upConfig struct {
	SizeProvider
	ResultOutput
	Logger
	AuthFunc    GetUploadUser
	MemoryLimit int64
}

func SizeUploadHandler(req *http.Request, urlPath string, w http.ResponseWriter, fileHandler ProcessFileHandler) {
	if urlPath == "" {
		urlPath = req.URL.Path
	}
	ctx := upctx.BuildContext(req, urlPath)

	file, userId, err := upload(ctx, req, urlPath)
	if err == nil {
		result, err := handlerFileWithLog(fileHandler, file, userId, ctx)
		if err != nil {
			UploadConfig.OutputErr(w, ctx, err)
			return
		}
		UploadConfig.OutputResult(w, ctx, result)
		return
	}

	if err == errNoAuth {
		UploadConfig.OutputNoAuth(w, ctx)
		return
	}
	if err == errExceedMax {
		UploadConfig.OutputExceed(w, ctx)
		return
	}
	if err == err404 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	UploadConfig.OutputErr(w, ctx, err)
}

func handlerFileWithLog(fileHandler ProcessFileHandler, file *ParsedFile, userId int64, ctx context.Context) (*ProcessFileResult, error) {
	start := time.Now().UnixMilli()
	r, err := fileHandler(file, userId, ctx)
	cost := time.Now().UnixMilli() - start

	if err != nil {
		UploadConfig.LogError("tid=%s, url: %s, uid=%d,handle file cost=%d ms, err: %v", upctx.GetTraceId(ctx), upctx.GetUrl(ctx), userId, cost, err)
	} else {
		UploadConfig.LogInfo("tid=%s,  url: %s, uid=%d,handle file cost=%d ms", upctx.GetTraceId(ctx), upctx.GetUrl(ctx), userId, cost)
	}
	return r, err
}
