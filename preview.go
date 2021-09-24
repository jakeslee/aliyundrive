package aliyundrive

import "github.com/jakeslee/aliyundrive/v1/models"

// GetVideoPreviewUrl 获取视频预览 URL
func (d *AliyunDrive) GetVideoPreviewUrl(credential *Credential, fileId string) (*models.VideoPreviewUrlResponse, error) {
	request := models.NewVideoPreviewUrlRequest()

	request.DriveId = credential.DefaultDriveId
	request.FileId = fileId

	var resp models.VideoPreviewUrlResponse

	err := d.send(credential, request, &resp)

	return &resp, err
}

// GetVideoPreviewPlayInfo 获取视频播放信息
func (d *AliyunDrive) GetVideoPreviewPlayInfo(credential *Credential, fileId string) (*models.VideoPreviewPlayInfoResponse, error) {
	request := models.NewVideoPreviewPlayInfoRequest()

	request.DriveId = credential.DefaultDriveId
	request.FileId = fileId

	var resp models.VideoPreviewPlayInfoResponse

	err := d.send(credential, request, &resp)

	return &resp, err
}
