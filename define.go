package sizedupload

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

type GetUploadUser func(req *http.Request, ctx context.Context) int64

type SizeProvider interface {
	GetSize(url string, ctx context.Context) int64
}

type ProcessFileResult struct {
	Id             int64  `json:"id"`
	TargetFileName string `json:"targetFileName"`
}

type ResultOutput interface {
	OutputResult(w http.ResponseWriter, ctx context.Context, result *ProcessFileResult)
	OutputExceed(w http.ResponseWriter, ctx context.Context)
	OutputNoAuth(w http.ResponseWriter, ctx context.Context)
	OutputErr(w http.ResponseWriter, ctx context.Context, err error)
}

type Logger interface {
	LogInfo(format string, args ...any)
	LogError(format string, args ...any)
}

type defaultLogger struct {
}

func (logger *defaultLogger) LogInfo(format string, args ...any) {
	log.Println("[Info]", fmt.Sprintf(format, args...))
}

func (logger *defaultLogger) LogError(format string, args ...any) {
	log.Println("[Error]", fmt.Sprintf(format, args...))
}
