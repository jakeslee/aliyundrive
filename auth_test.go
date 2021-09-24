package aliyundrive

import (
	"os"
	"testing"
)

var (
	refreshToken = os.Getenv("refreshToken")
)

func TestRefreshToken(t *testing.T) {
	drive := NewClient(&Options{
		AutoRefresh: true,
	})

	cred := NewCredential(&Credential{
		RefreshToken: refreshToken,
	})

	cred.RegisterChangeEvent(func(credential *Credential) {
		t.Logf("credential change: %+v", *credential)
	})

	c, err := drive.AddCredential(cred)

	if err != nil {
		t.Error(err)
	}

	t.Log(c)
}

func TestGetUserInfo(t *testing.T) {
	cred := NewCredential(&Credential{
		RefreshToken: refreshToken,
	})

	cred.RegisterChangeEvent(func(credential *Credential) {
		t.Logf("credential change: %+v", *credential)
	})

	drive := NewClient(&Options{
		AutoRefresh: true,
		Credential: []*Credential{
			cred,
		},
	})

	info, err := drive.GetUserInfo(cred)

	if err != nil {
		t.Error(err)
	}

	t.Logf("get user info %+v", info)
}
