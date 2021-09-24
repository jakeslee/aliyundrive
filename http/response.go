package http

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
)

type Response interface {
	ParseErrorFromHTTPResponse(body []byte) error
}

type BaseResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (b *BaseResponse) ParseErrorFromHTTPResponse(body []byte) error {
	if b.Code != "" {
		return NewAliyunDriveError(b.Code, b.Message)
	}

	return nil
}

func parseFromHTTPResponse(response *resty.Response, out Response) error {
	body := response.Body()

	err := json.Unmarshal(body, &out)

	if err != nil {
		msg := fmt.Sprintf("Fail to parse json content: %s, because: %s", body, err)

		return NewAliyunDriveError("ClientError.ParseJsonError", msg)
	}

	err = out.ParseErrorFromHTTPResponse(body)

	if err != nil {
		return err
	}

	return nil
}
