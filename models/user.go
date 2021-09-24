package models

import "github.com/jakeslee/aliyundrive/v1/http"

type UserInfo struct {
	http.BaseResponse
	DomainId       string                 `json:"domain_id"`
	UserId         string                 `json:"user_id"`
	Avatar         string                 `json:"avatar"`
	CreatedAt      int64                  `json:"created_at"`
	UpdatedAt      int64                  `json:"updated_at"`
	Email          string                 `json:"email"`
	NickName       string                 `json:"nick_name"`
	Phone          string                 `json:"phone"`
	Role           string                 `json:"role"`
	Status         string                 `json:"status"`
	UserName       string                 `json:"user_name"`
	Description    string                 `json:"description"`
	DefaultDriveId string                 `json:"default_drive_id"`
	UserData       map[string]interface{} `json:"user_data"`
}

type UserInfoRequest struct {
	http.BaseRequest
}

func NewUserInfoRequest() *UserInfoRequest {
	u := &UserInfoRequest{}

	u.Init(AliyunDriveEndpoint).SetHttpMethod(http.Post).SetUrl("/v2/user/get")

	return u
}
