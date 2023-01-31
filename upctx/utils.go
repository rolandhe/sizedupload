package upctx

import (
	"context"
	"github.com/google/uuid"
	"net/http"
)

func BuildContext(req *http.Request, urlPath string) context.Context {
	return context.WithValue(context.Background(), ctxLocalData, &localData{
		traceId: getOrGenTraceId(req),
		urlPath: urlPath,
	})
}

func SetGoStdHeader(header http.Header, name string, value string) {
	header.Set(name, value)
}

func SetNormalHeader(header http.Header, name string, value string) {
	header[name] = []string{value}
}

func GetTraceId(ctx context.Context) string {
	value := ctx.Value(ctxLocalData)
	if value == nil {
		return ""
	}
	data, ok := value.(*localData)
	if !ok || data == nil {
		return ""
	}
	return data.traceId
}

func GetUrl(ctx context.Context) string {
	value := ctx.Value(ctxLocalData)
	if value == nil {
		return ""
	}
	data, ok := value.(*localData)
	if !ok || data == nil {
		return ""
	}
	return data.urlPath
}

type localData struct {
	traceId string
	urlPath string
}

func getOrGenTraceId(req *http.Request) string {
	traceId := req.Header.Get(GOTraceId)
	if traceId == "" {
		traceId = uuid.New().String()
	}
	return traceId
}
