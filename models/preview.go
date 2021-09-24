package models

import "github.com/jakeslee/aliyundrive/v1/http"

type VideoPreviewUrlRequest struct {
	http.BaseRequest

	DriveId   string `json:"drive_id"`
	FileId    string `json:"file_id"`
	ExpireSec int    `json:"expire_sec"`
}

type VideoPreviewUrlResponse struct {
	http.BaseResponse

	TemplateList []struct {
		TemplateId string `json:"template_id"`
		Status     string `json:"status"`
		Url        string `json:"url"`
	} `json:"template_list"`
}

func NewVideoPreviewUrlRequest() *VideoPreviewUrlRequest {
	r := &VideoPreviewUrlRequest{
		ExpireSec: 14400,
	}

	r.Init(AliyunDriveEndpoint).SetHttpMethod(http.Post).SetUrl("/v2/databox/get_video_play_info")

	return r
}

type VideoPreviewPlayInfoRequest struct {
	http.BaseRequest

	Category   string `json:"category"`
	DriveId    string `json:"drive_id"`
	FileId     string `json:"file_id"`
	TemplateId string `json:"template_id"`
}

type VideoPreviewPlayInfoResponse struct {
	http.BaseResponse

	VideoPreviewPlayInfo struct {
		LiveTranscodingTaskList []struct {
			TemplateId string `json:"template_id"`
			Status     string `json:"status"`
			Url        string `json:"url"`
			Stage      string `json:"stage"`
		} `json:"live_transcoding_task_list"`
		Meta struct {
			Duration            float64                `json:"duration"`
			Height              int                    `json:"height"`
			Width               int                    `json:"width"`
			LiveTranscodingMeta map[string]interface{} `json:"live_transcoding_meta"`
		} `json:"meta"`
	} `json:"video_preview_play_info"`
}

const PreviewCategoryDefault = "live_transcoding"

func NewVideoPreviewPlayInfoRequest() *VideoPreviewPlayInfoRequest {
	r := &VideoPreviewPlayInfoRequest{
		Category: PreviewCategoryDefault,
	}

	r.Init(AliyunDriveEndpoint).SetHttpMethod(http.Post).SetUrl("/v2/file/get_video_preview_play_info")

	return r
}

type OfficePreviewUrlRequest struct {
	http.BaseRequest

	AccessToken string `json:"access_token"`
	DriveId     string `json:"drive_id"`
	FileId      string `json:"file_id"`
}
