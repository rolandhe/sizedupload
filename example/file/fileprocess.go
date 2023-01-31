package file

import (
	"context"
	up "github.com/rolandhe/sizedupload"
)

func DoProcessFile(file *up.ParsedFile, userId int64, context context.Context) (*up.ProcessFileResult, error) {

	return &up.ProcessFileResult{
		Id:             123,
		TargetFileName: "hello world",
	}, nil
}
