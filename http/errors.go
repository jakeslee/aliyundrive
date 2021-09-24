package http

import "fmt"

type AliyunDriveError struct {
	Code    string
	Message string
}

func (p *AliyunDriveError) Error() string {
	return fmt.Sprintf("[AliyunDriveError] Code=%s, Message=%s", p.Code, p.Message)
}

func NewAliyunDriveError(code, message string) error {
	return &AliyunDriveError{
		Code:    code,
		Message: message,
	}
}
