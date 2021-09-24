package aliyundrive

import (
	"os"
	"testing"
)

func GetClientAndCred() (*AliyunDrive, *Credential, error) {
	drive := NewClient(&Options{
		AutoRefresh: true,
	})

	cred := NewCredential(&Credential{
		RefreshToken: refreshToken,
	})

	cred, err := drive.AddCredential(cred)

	return drive, cred, err
}

func TestGetDownloadURL(t *testing.T) {
	drive := NewClient(&Options{
		AutoRefresh: true,
	})

	cred := NewCredential(&Credential{
		RefreshToken: refreshToken,
	})

	cred, _ = drive.AddCredential(cred)

	url, err := drive.GetDownloadURL(cred, "61448987147d665483664d9fb126ffd7e8ac4661")
	if err != nil {
		t.Errorf("get download url error %v", err)
	}

	t.Logf("download url: %s", *url.Url)
}

func TestAliyunDrive_UploadFileRapid(t *testing.T) {
	filePath := "/Volumes/Downloads/视频资源/小林家的龙女仆/第02季/小林家的龙女仆.第二季.日语中字.2021.HD1080P.X264.AAC-YYDS/S02E11.mp4"
	stat, _ := os.Stat(filePath)
	name := stat.Name()

	f, _ := os.Open(filePath)

	drive, cred, err := GetClientAndCred()

	if err != nil {
		t.Fatalf("cred %v", err)
	}

	rapid, err := drive.UploadFileRapid(cred, &UploadFileRapidOptions{
		UploadFileOptions{
			Name:         name,
			Size:         stat.Size(),
			ParentFileId: "root",
			ProgressCallback: func(readCount int64) bool {
				t.Logf("uploaded %d bytes", readCount)

				return true
			},
		},
		f,
	})

	if err != nil {
		t.Errorf("upload error %s", err)
	}

	t.Logf("file %+v", rapid)
}

func TestAliyunDrive_UploadFile(t *testing.T) {
	filePath := "/Volumes/Downloads/embyserver.txt"
	stat, _ := os.Stat(filePath)

	name := stat.Name()

	t.Logf("name %s", name)

	f, _ := os.Open(filePath)

	drive, cred, err := GetClientAndCred()

	if err != nil {
		t.Fatalf("cred %v", err)
	}

	file, err := drive.UploadFile(cred, &UploadFileOptions{
		Name:         name,
		Size:         stat.Size(),
		ParentFileId: "root",
		reader:       f,
		ProgressCallback: func(readCount int64) bool {
			t.Logf("uploaded %d bytes", readCount)

			return true
		},
	})
	if err != nil {
		t.Errorf("upload error %s", err)
	}

	t.Logf("file %+v", file)
}
