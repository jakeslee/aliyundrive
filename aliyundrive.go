package aliyundrive

import (
	"crypto/tls"
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/jakeslee/aliyundrive/http"
	"github.com/jakeslee/aliyundrive/models"
	"github.com/sirupsen/logrus"
	gohttp "net/http"
	"reflect"
	"time"
)

const (
	DefaultRootFileId = "root"
)

func init() {
	logrus.SetFormatter(&nested.Formatter{
		HideKeys: true,
	})
}

type AliyunDrive struct {
	Credentials map[string]*Credential

	client    *http.Client
	rawClient *gohttp.Client
	cache     *bigCache
}

type Options struct {
	AutoRefresh bool
	Credential  []*Credential
}

func NewClient(options *Options) *AliyunDrive {
	drive := &AliyunDrive{
		Credentials: make(map[string]*Credential),
		client:      http.NewClient(),
		rawClient: &gohttp.Client{
			Transport: &gohttp.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
	}

	drive.cache, _ = newBigCache(&bigCacheOptions{
		ttl:       5 * time.Minute,
		size:      0,
		cleanFreq: time.Minute,
	})

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

	baseValue := reflect.ValueOf(response).Elem().FieldByName("BaseResponse")

	if baseValue.IsValid() {
		if value, ok := baseValue.Addr().Interface().(*http.BaseResponse); ok {
			if value.Code == models.CodeAccessTokenInvalid {
				_, err := d.RefreshToken(credential)
				if err != nil {
					return err
				}

				return d.client.Send(r, response)
			}
		}
	}

	return err
}

// EvictCacheWithPrefix 失效 Key 前缀为 keyPrefix 的缓存
func (d *AliyunDrive) EvictCacheWithPrefix(keyPrefix string) int {
	return d.cache.RemoveWithPrefix(keyPrefix)
}
