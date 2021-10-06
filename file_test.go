package aliyundrive

import (
	http2 "github.com/jakeslee/aliyundrive/http"
	"github.com/jakeslee/aliyundrive/models"
	"io"
	"net/http"
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

func TestGetByPath(t *testing.T) {
	cred, credential, err := GetClientAndCred()

	if err != nil {
		t.Fatalf("cred %v", err)
	}

	request := models.NewGetFileByPathRequest()
	request.DriveId = credential.DefaultDriveId
	request.FilePath = "aa/a"

	var resp models.FileResponse

	err = cred.send(credential, request, &resp)

	if err != nil {
		t.Fatalf("cred %v", err)
	}
}

func TestGetDownloadURL(t *testing.T) {
	drive := NewClient(&Options{
		AutoRefresh: true,
	})

	cred := NewCredential(&Credential{
		RefreshToken: refreshToken,
	})

	cred, _ = drive.AddCredential(cred)

	url, err := drive.GetDownloadURL(cred, "614ea15e32865fc2e8af4a4fb70b13d1103c70c0")
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
			ParentFileId: DefaultRootFileId,
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
	filePath := "/Volumes/Downloads/untitled folder/阿里小白羊版Mac v2.8.ccc.zip"
	stat, _ := os.Stat(filePath)

	name := stat.Name()

	t.Logf("name %s", name)

	f, _ := os.Open(filePath)

	drive, cred, err := GetClientAndCred()

	if err != nil {
		t.Fatalf("cred %v", err)
	}

	send := int64(0)

	file, err := drive.UploadFile(cred, &UploadFileOptions{
		Name:         name,
		Size:         stat.Size(),
		ParentFileId: DefaultRootFileId,
		Reader:       f,
		ProgressCallback: func(readCount int64) bool {
			send += readCount

			t.Logf("uploaded %d%%", (send*100)/stat.Size())

			return true
		},
	})
	if err != nil {
		t.Errorf("upload error %s", err)
	}

	t.Logf("file %+v", file)
}

func TestAliyunDrive_Download(t *testing.T) {
	drive, cred, err := GetClientAndCred()

	if err != nil {
		t.Fatalf("cred %v", err)
	}

	http.HandleFunc("/download", func(writer http.ResponseWriter, request *http.Request) {
		t.Logf("request method %s", request.Method)

		response, err := drive.Download(cred, "614ea15e32865fc2e8af4a4fb70b13d1103c70c0", request.Header.Get("range"))
		if err != nil {
			writer.Write([]byte(err.Error()))
			return
		}

		for key, values := range response.Header {
			for _, value := range values {
				writer.Header().Add(key, value)
			}
		}

		_, err = io.Copy(writer, response.Body)
		if err != nil {
			writer.Write([]byte(err.Error()))
			return
		}
	})

	t.Fatal(http.ListenAndServe(":18080", nil))
}

type ScanRequest struct {
	http2.BaseRequest

	DriveId string `json:"drive_id"`
	Limit   int    `json:"limit"`
	Marker  string `json:"marker"`
}

func (d *AliyunDrive) Test(credential *Credential) {
	request := &ScanRequest{
		DriveId: credential.DefaultDriveId,
		Limit:   1000,
	}

	request.Init(models.AliyunDriveEndpoint).
		SetHttpMethod(http2.Post).SetUrl("/v2/file/scan")

	var resp models.FolderFilesResponse

	_ = d.send(credential, request, &resp)
}

func TestTest(t *testing.T) {
	drive, cred, err := GetClientAndCred()

	if err != nil {
		t.Fatalf("cred %v", err)
	}

	aa, foundPath, err := drive.ResolvePathToFileId(cred, "/d/a/b/cc.gz")

	t.Log(aa, foundPath, err)
	//s := "a/b/c"
	//split := strings.Split(s, "/")
	//dir := filepath.Dir(filepath.Clean(s))
	//t.Log(split, dir)
}
