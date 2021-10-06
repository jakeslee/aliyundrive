package aliyundrive

import (
	"errors"
	"github.com/asaskevich/EventBus"
	"github.com/jakeslee/aliyundrive/models"
	"github.com/jinzhu/copier"
	"github.com/sirupsen/logrus"
	"math/rand"
	"strconv"
)

type Credential struct {
	UserId         string
	Name           string
	AccessToken    string
	RefreshToken   string
	RootFolder     string
	DefaultDriveId string
	eventbus       EventBus.Bus
}

const (
	eventTokenChange = "token:change"
)

func NewCredential(credential *Credential) *Credential {
	c := &Credential{
		RootFolder: DefaultRootFileId,
		eventbus:   EventBus.New(),
	}

	_ = copier.CopyWithOption(c, credential, copier.Option{
		IgnoreEmpty: true,
	})

	return c
}

func (c *Credential) RegisterChangeEvent(fn func(credential *Credential)) *Credential {
	_ = c.eventbus.Subscribe(eventTokenChange, fn)

	return c
}

// RefreshToken 刷新 RefreshToken，更新 AccessToken 和 Credential 里的相关信息
func (d *AliyunDrive) RefreshToken(credential *Credential) (*models.RefreshTokenResponse, error) {
	refreshTokenRequest := models.NewRefreshTokenRequest()
	refreshTokenRequest.RefreshToken = credential.RefreshToken
	var token models.RefreshTokenResponse

	err := d.send(credential, refreshTokenRequest, &token)

	if token.Code != "" {
		logrus.Errorf("refresh token error: %s", token.Message)
		return &token, errors.New(token.Message)
	}

	credential.RefreshToken = token.RefreshToken
	credential.AccessToken = token.AccessToken
	credential.Name = token.NickName
	credential.DefaultDriveId = token.DefaultDriveId

	credential.eventbus.Publish(eventTokenChange, credential)

	_, ok := d.Credentials[token.UserId]

	if !ok || credential.UserId != token.UserId {
		delete(d.Credentials, credential.UserId)

		d.Credentials[token.UserId] = credential
		credential.UserId = token.UserId
	}

	return &token, err
}

// AddCredential 增加新的 Credential，同时刷新 RefreshToken
func (d *AliyunDrive) AddCredential(credential *Credential) (*Credential, error) {
	credential.UserId = strconv.Itoa(rand.Intn(100000))

	d.Credentials[credential.UserId] = credential
	_, err := d.RefreshToken(credential)

	return credential, err
}

// GetCredentialFromUserId 通过 UserId 取 Credential
func (d *AliyunDrive) GetCredentialFromUserId(userId string) *Credential {
	return d.Credentials[userId]
}

// GetUserInfo get user information
func (d *AliyunDrive) GetUserInfo(credential *Credential) (*models.UserInfo, error) {
	userInfoRequest := models.NewUserInfoRequest()

	var user models.UserInfo

	err := d.send(credential, userInfoRequest, &user)

	return &user, err
}
