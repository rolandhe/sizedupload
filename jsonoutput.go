package sizedupload

import (
	"context"
	"encoding/json"
	"net/http"
)

type Result struct {
	Success bool   `json:"success"`
	Code    int    `json:"code,omitempty"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func NewJsonResultOutput() ResultOutput {
	return &jsonOutput{}
}

type jsonOutput struct {
}

func (*jsonOutput) OutputResult(w http.ResponseWriter, ctx context.Context, result *ProcessFileResult) {
	r := &Result{
		Success: true,
		Code:    200,
		Data:    result,
	}
	writeJson(w, r)
}
func (*jsonOutput) OutputExceed(w http.ResponseWriter, ctx context.Context) {
	r := &Result{Code: 5002, Message: "exceed max file"}
	writeJson(w, r)
}
func (*jsonOutput) OutputNoAuth(w http.ResponseWriter, ctx context.Context) {
	r := &Result{Code: 4009, Message: "Permission denied"}
	writeJson(w, r)
}
func (*jsonOutput) OutputErr(w http.ResponseWriter, ctx context.Context, err error) {
	r := &Result{Code: 5001, Message: "The system is out of order, please try again later"}
	writeJson(w, r)
}

func writeJson(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json, _ := json.Marshal(data)
	w.Write(json)
}
