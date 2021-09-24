package aliyundrive

import (
	"github.com/bwmarrin/snowflake"
	"github.com/jakeslee/aliyundrive/v1/http"
	"github.com/jakeslee/aliyundrive/v1/models"
)

type AliyunDrive struct {
	Credentials map[string]*Credential

	client    *http.Client
	snowflake *snowflake.Node
}

type Options struct {
	AutoRefresh bool
	Credential  []*Credential
}

func NewClient(options *Options) *AliyunDrive {
	node, _ := snowflake.NewNode(1)

	drive := &AliyunDrive{
		client:      http.NewClient(),
		Credentials: make(map[string]*Credential),
		snowflake:   node,
	}

	if len(options.Credential) > 0 {
		for _, credential := range options.Credential {
			_, _ = drive.AddCredential(credential)
		}
	}

	return drive
}

func (d *AliyunDrive) send(credential *Credential, r http.Request, response http.Response) error {
	if credential.AccessToken != "" {
		models.WithToken(r, credential.AccessToken)
	}

	err := d.client.Send(r, response)

	// 如果是 AliyunDriveError 需要检查是否需要刷新 Token
	if _, ok := err.(*http.AliyunDriveError); !ok && err != nil {
		return err
	}

	if value, ok := response.(*http.BaseResponse); ok {
		if value.Code == models.CodeAccessTokenInvalid {
			_, err := d.RefreshToken(credential)
			if err != nil {
				return err
			}

			return d.client.Send(r, response)
		}
	}

	return err
}
