package models

import (
	"github.com/jakeslee/aliyundrive/v1/http"
	"time"
)

const (
	AliyunDriveEndpoint     = "https://api.aliyundrive.com"
	AliyunDriveAuthEndpoint = "https://auth.aliyundrive.com"
	CodeAccessTokenInvalid  = "AccessTokenInvalid"
	CodePreHashMatched      = "PreHashMatched"
)

type RefreshTokenRequest struct {
	http.BaseRequest

	RefreshToken string `json:"refresh_token"`
	GrantType    string `json:"grant_type"`
}

func WithToken(request http.Request, token string) {
	request.GetHeaders()["authorization"] = "Bearer " + token
}

// NewRefreshTokenRequest create RefreshTokenRequest with api
func NewRefreshTokenRequest() *RefreshTokenRequest {
	request := &RefreshTokenRequest{
		GrantType: "refresh_token",
	}

	request.Init(AliyunDriveAuthEndpoint).
		SetHttpMethod(http.Post).
		SetUrl("/v2/account/token")

	return request
}

type RefreshTokenResponse struct {
	http.BaseResponse
	UserInfo

	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`

	DefaultSboxDriveId string        `json:"default_sbox_drive_id"`
	ExpireTime         *time.Time    `json:"expire_time"`
	State              string        `json:"state"`
	ExistLink          []interface{} `json:"exist_link"`
	NeedLink           bool          `json:"need_link"`
	PinSetup           bool          `json:"pin_setup"`
	IsFirstLogin       bool          `json:"is_first_login"`
	NeedRpVerify       bool          `json:"need_rp_verify"`
	DeviceId           string        `json:"device_id"`
}
