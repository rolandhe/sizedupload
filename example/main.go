package main

import (
	"context"
	"fmt"
	"github.com/julienschmidt/httprouter"
	up "github.com/rolandhe/sizedupload"
	"github.com/rolandhe/sizedupload/example/file"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"log"
	"net/http"
	"os"
	"time"
)

// confSizeUpload 初始化相关依赖
func confSizeUpload() {
	var err error
	// 可以调用NewConfSizeProvider方法，它从./conf/sizeconfig.yml读取
	up.UploadConfig.SizeProvider, err = up.NewConfSizeProviderByConfFile("./example/conf/sizeconfig.yml")
	if err != nil {
		log.Println(err)
		panic(err)
	}
	up.UploadConfig.AuthFunc = func(req *http.Request, ctx context.Context) int64 {
		return 1004
	}
	up.UploadConfig.ResultOutput = up.NewJsonResultOutput()
	// 配置日志输出，如果不配置，会使用缺省日志输出
	//sizedupload.UploadConfig.Logger =

	up.UploadConfig.MemoryLimit = 1024 * 1024
}

func init() {
	confSizeUpload()
}

func main() {

	router := httprouter.New()
	router.POST("/fgw/upload/*path", func(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
		up.SizeUploadHandler(req, params.ByName("path"), w, file.DoProcessFile)
	})

	h2s := &http2.Server{}

	host := fmt.Sprintf("0.0.0.0:%d", 8080)

	server := &http.Server{
		Addr:        host,
		Handler:     h2c.NewHandler(router, h2s),
		IdleTimeout: time.Minute * 30,
	}

	log.Printf("Listening [0.0.0.0:%d]...\n", 8080)

	checkErr(server.ListenAndServe(), "while listening")
}

func checkErr(err error, msg string) {
	if err == nil {
		return
	}
	log.Printf("ERROR: %s: %s\n", msg, err)
	os.Exit(1)
}
