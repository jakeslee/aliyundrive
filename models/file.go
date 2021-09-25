package models

import (
	"errors"
	"github.com/jakeslee/aliyundrive/http"
	http2 "net/http"
	"time"
)

// FolderFilesRequest 获取目录下文件列表
type FolderFilesRequest struct {
	http.BaseRequest

	DriveId               string `json:"drive_id"`
	Fields                string `json:"fields"`                  // 字段
	ImageThumbnailProcess string `json:"image_thumbnail_process"` // 图片缩略图处理器
	ImageUrlProcess       string `json:"image_url_process"`       // 图片处理器
	UrlExpireSec          string `json:"url_expire_sec"`          // Url 超时时间（秒）
	Limit                 int    `json:"limit"`                   // 单次拉取数量
	Marker                string `json:"marker"`                  // 分页拉新标记
	OrderBy               string `json:"order_by"`                // 排序字段
	OrderDirection        string `json:"order_direction"`         // 排序方向，DESC/ASC
	ParentFileId          string `json:"parent_file_id"`          // 目录 ID
	VideoThumbnailProcess string `json:"video_thumbnail_process"` // 视频缩略图处理器
}

const (
	OrderDirectionTypeDescend = "DESC"
	OrderDirectionTypeAscend  = "ASC"
)

const (
	ImageUrlProcessDefault       = "image/resize,w_1920/format,jpeg"
	ImageThumbnailProcessDefault = "image/resize,w_400/format,jpeg"
	VideoThumbnailProcessDefault = "video/snapshot,t_0,f_jpg,ar_auto,w_300"
	SearchResultOrderDefault     = "type ASC,updated_at DESC"
)

type FolderFilesResponse struct {
	http.BaseResponse
	Files
}

func NewFolderFilesRequest() *FolderFilesRequest {
	r := &FolderFilesRequest{
		Fields:                "*",
		Limit:                 100,
		ImageThumbnailProcess: ImageThumbnailProcessDefault,
		ImageUrlProcess:       ImageUrlProcessDefault,
		VideoThumbnailProcess: VideoThumbnailProcessDefault,
	}

	r.Init(AliyunDriveEndpoint).SetHttpMethod(http.Post).SetUrl("/v2/file/list")

	return r
}

type FileRequest struct {
	http.BaseRequest

	DriveId               string `json:"drive_id"`
	FileId                string `json:"file_id"`
	ImageThumbnailProcess string `json:"image_thumbnail_process"`
	VideoThumbnailProcess string `json:"video_thumbnail_process"`
}

type FileResponse struct {
	http.BaseResponse
	File
}

func NewFileRequest() *FileRequest {
	r := &FileRequest{
		ImageThumbnailProcess: ImageThumbnailProcessDefault,
		VideoThumbnailProcess: VideoThumbnailProcessDefault,
	}

	r.Init(AliyunDriveEndpoint).SetHttpMethod(http.Post).SetUrl("/v2/file/get")

	return r
}

type DownloadURLRequest struct {
	http.BaseRequest

	DriveId   string `json:"drive_id"`
	FileId    string `json:"file_id"`    // 文件 ID
	ExpireSec int    `json:"expire_sec"` // 下载链接超时时间（秒）
	FileName  string `json:"file_name"`  // 文件名
}

type DownloadURLResponse struct {
	http.BaseResponse

	Expiration string  `json:"expiration"`
	Method     string  `json:"method"`
	Size       int64   `json:"size"`
	Url        *string `json:"url"`
}

func NewDownloadURLRequest() *DownloadURLRequest {
	r := &DownloadURLRequest{
		ExpireSec: 14400, // 默认 4 小时
	}

	r.Init(AliyunDriveEndpoint).SetHttpMethod(http.Post).SetUrl("/v2/file/get_download_url")

	return r
}

type SearchRequest struct {
	http.BaseRequest

	DriveId               string `json:"drive_id"`
	Limit                 int    `json:"limit"`
	Marker                string `json:"marker"`
	OrderBy               string `json:"order_by"` // "type ASC,updated_at DESC"
	Query                 string `json:"query"`    // "name match '测试文件'"
	ImageThumbnailProcess string `json:"image_thumbnail_process"`
	ImageUrlProcess       string `json:"image_url_process"`
	VideoThumbnailProcess string `json:"video_thumbnail_process"`
}

type SearchResponse struct {
	http.BaseResponse

	Files
}

func NewSearchRequest() *SearchRequest {
	r := &SearchRequest{
		ImageUrlProcess:       ImageUrlProcessDefault,
		ImageThumbnailProcess: ImageThumbnailProcessDefault,
		VideoThumbnailProcess: VideoThumbnailProcessDefault,
		OrderBy:               SearchResultOrderDefault,
		Limit:                 100,
	}

	r.Init(AliyunDriveEndpoint).SetHttpMethod(http.Post).SetUrl("/v2/file/search")

	return r
}

type CreateFileRequest struct {
	http.BaseRequest

	DriveId       string `json:"drive_id"`
	ParentFileId  string `json:"parent_file_id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Size          int64  `json:"size"`
	CheckNameMode string `json:"check_name_mode"`
}

// GetUploadUrlRequest 获取文件分片上传路径
type GetUploadUrlRequest struct {
	http.BaseRequest

	DriveId      string      `json:"drive_id"`
	FileId       string      `json:"file_id"`        // 文件 ID
	ParentFileId string      `json:"parent_file_id"` // 父文件 ID
	Type         string      `json:"type"`
	Name         string      `json:"name"`           // 文件名字
	ContentType  string      `json:"content_type"`   // 内容类型
	PartInfoList []*PartInfo `json:"part_info_list"` // 文件上传分片信息
}

type GetUploadUrlResponse struct {
	http.BaseResponse

	PartInfoList []*PartInfo `json:"part_info_list"` // 文件上传分片信息
}

type PartInfo struct {
	PartSize          int64   `json:"part_size"`           // 分片大小
	PartNumber        int8    `json:"part_number"`         // 分片序号
	ContentType       string  `json:"content_type"`        // 内容类型
	InternalUploadUrl *string `json:"internal_upload_url"` // 内部上传地址
	UploadUrl         *string `json:"upload_url"`          // 分片上传路径
	Id                int     // 分片 ID
	StartOffset       int64   // 当前分片开始位置
	EndOffset         int64   // 分片结束位置
	IsUploaded        bool    // 是否已经上传
}

// NewPartInfoList 基于文件大小和分片大小创建分片信息
func NewPartInfoList(size, partSize int64) ([]*PartInfo, error) {
	if size < 0 {
		return nil, errors.New("size cannot be -1")
	}

	var result []*PartInfo
	var count int64

	for (count + partSize) < size {
		result = append(result, &PartInfo{
			Id:          len(result),
			StartOffset: count,
			EndOffset:   count + partSize - 1,
			PartNumber:  int8(len(result) + 1),
			PartSize:    partSize,
		})

		count += partSize
	}

	var endOffset int64

	if size > 0 {
		endOffset = size - 1
	}

	result = append(result, &PartInfo{
		Id:          len(result),
		StartOffset: count,
		EndOffset:   endOffset,
		PartNumber:  int8(len(result) + 1),
		PartSize:    partSize,
	})

	return result, nil
}

// CreateWithFoldersPreHashRequest 使用 PreHash 进行 Proof 前校验
type CreateWithFoldersPreHashRequest struct {
	http.BaseRequest
	CreateWithFolders

	PreHash string `json:"pre_hash"`
}

type CreateWithFoldersPreHashResponse struct {
	http.BaseResponse
	CreateWithFoldersWithProofResponse

	PreHash string `json:"pre_hash"`
}

func NewCreateWithFoldersPreHashRequest() *CreateWithFoldersPreHashRequest {
	r := &CreateWithFoldersPreHashRequest{
		CreateWithFolders: CreateWithFolders{
			Type:          FileTypeFile,
			CheckNameMode: CheckNameModeAutoRename,
		},
	}

	r.Init(AliyunDriveEndpoint).
		SetHttpMethod(http.Post).
		SetUrl("/adrive/v2/file/createWithFolders")

	return r
}

// CreateWithFoldersWithProofRequest 带 Proof 信息上传，可用于秒传
type CreateWithFoldersWithProofRequest struct {
	http.BaseRequest
	CreateWithFolders

	ContentHash     string `json:"content_hash"`
	ContentHashName string `json:"content_hash_name"`
	ProofCode       string `json:"proof_code"`
	ProofVersion    string `json:"proof_version"`
}

type CreateWithFoldersWithProofResponse struct {
	http.BaseResponse

	DriveId      string      `json:"drive_id"`
	DomainId     string      `json:"domain_id"`
	EncryptMode  string      `json:"encrypt_mode"`   // 加密模式
	FileId       string      `json:"file_id"`        // 文件 ID
	FileName     string      `json:"file_name"`      // 文件名
	Location     string      `json:"location"`       // 存储区域
	ParentFileId string      `json:"parent_file_id"` // 父文件 ID
	RapidUpload  bool        `json:"rapid_upload"`   // 是否秒传
	Type         string      `json:"type"`           // 操作类型
	UploadId     string      `json:"upload_id"`      // 上传 ID
	PartInfoList []*PartInfo `json:"part_info_list"` // 文件上传分片信息
}

func NewCreateWithFoldersWithProofRequest() *CreateWithFoldersWithProofRequest {
	r := &CreateWithFoldersWithProofRequest{
		CreateWithFolders: CreateWithFolders{
			Type:          FileTypeFile,
			CheckNameMode: CheckNameModeAutoRename,
		},
		ProofVersion:    "v1",
		ContentHashName: "sha1",
	}

	r.Init(AliyunDriveEndpoint).
		SetHttpMethod(http.Post).
		SetUrl("/adrive/v2/file/createWithFolders")

	return r
}

type CompleteFileUploadRequest struct {
	http.BaseRequest

	DriveId  string `json:"drive_id"`
	UploadId string `json:"upload_id"` // 上传 ID
	FileId   string `json:"file_id"`   // 文件 ID
}

type CompleteFileUploadResponse struct {
	http.BaseResponse
	File

	UploadId string `json:"upload_id"` // 上传 ID
}

func NewCompleteFileUploadRequest() *CompleteFileUploadRequest {
	r := &CompleteFileUploadRequest{}

	r.Init(AliyunDriveEndpoint).
		SetHttpMethod(http.Post).
		SetUrl("/v2/file/complete")

	return r
}

type RemoveFileRequest struct {
	http.BaseRequest

	DriveId string `json:"drive_id"`
	FileId  string `json:"file_id"` // 文件 ID
}

// NewRemoveFileRequest 创建删除文件请求
func NewRemoveFileRequest() *RemoveFileRequest {
	r := &RemoveFileRequest{}

	r.Init(AliyunDriveEndpoint).
		SetHttpMethod(http.Post).
		SetUrl("/v2/recyclebin/trash")

	return r
}

type RenameFileRequest struct {
	http.BaseRequest

	DriveId       string        `json:"drive_id"`
	FileId        string        `json:"file_id"`
	CheckNameMode CheckNameMode `json:"check_name_mode"`
	Name          string        `json:"name"`
}

type RenameFileResponse struct {
	http.BaseResponse

	File
}

// NewRenameFileRequest 创建重命名请求
func NewRenameFileRequest() *RenameFileRequest {
	r := &RenameFileRequest{
		CheckNameMode: CheckNameModeRefuse,
	}

	r.Init(AliyunDriveEndpoint).
		SetHttpMethod(http.Post).
		SetUrl("/v2/file/update")

	return r
}

type MoveFileRequest struct {
	http.BaseRequest

	DriveId        string `json:"drive_id"`
	FileId         string `json:"file_id"`
	ToDriveId      string `json:"to_drive_id"`
	ToParentFileId string `json:"to_parent_file_id"`
}

func NewMoveFileRequest() *MoveFileRequest {
	r := &MoveFileRequest{}

	r.Init(AliyunDriveEndpoint).
		SetHttpMethod(http.Post).
		SetUrl("/v2/file/move")

	return r
}

type BatchRequest struct {
	http.BaseRequest

	Requests []*BatchRequestItem `json:"requests"`
	Resource string              `json:"resource"`
}

type BatchRequestItem struct {
	Body    http.Request      `json:"body"`
	Headers map[string]string `json:"headers"`
	Id      string            `json:"id"`
	Method  string            `json:"method"`
	Url     string            `json:"url"`
}

type BatchResponse struct {
	http.BaseResponse

	Responses []*struct {
		Body   map[string]interface{} `json:"body"`
		Id     string                 `json:"id"`
		Status int                    `json:"status"`
	} `json:"responses"`
}

// NewBatchRequest 创建批处理操作请求
func NewBatchRequest() *BatchRequest {
	r := &BatchRequest{
		Resource: string(FileTypeFile),
	}

	r.Init(AliyunDriveEndpoint).
		SetHttpMethod(http.Post).
		SetUrl("/v3/batch")

	return r
}

// NewBatchMoveRequest 创建批量移动文件请求
func NewBatchMoveRequest(requests []*MoveFileRequest) *BatchRequest {
	batchRequest := NewBatchRequest()

	for _, request := range requests {
		batchRequest.Requests = append(batchRequest.Requests, &BatchRequestItem{
			Id:     request.FileId,
			Method: http2.MethodPost,
			Url:    "/file/move",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: request,
		})
	}

	return batchRequest
}

type GetPathRequest struct {
	http.BaseRequest

	DriveId string `json:"drive_id"`
	FileId  string `json:"file_id"`
}

type GetPathResponse struct {
	http.BaseResponse

	Items []*File `json:"items"`
}

// NewGetPathRequest 创建查询路径请求，返回从当前 FileId 到根路径
func NewGetPathRequest() *GetPathRequest {
	r := &GetPathRequest{}

	r.Init(AliyunDriveEndpoint).
		SetHttpMethod(http.Post).
		SetUrl("/adrive/v1/file/get_path")

	return r
}

type CheckNameMode string

const (
	CheckNameModeAutoRename CheckNameMode = "auto_rename"
	CheckNameModeRefuse     CheckNameMode = "refuse"
)

type CreateWithFolders struct {
	DriveId       string        `json:"drive_id"`
	Name          string        `json:"name"`
	CheckNameMode CheckNameMode `json:"check_name_mode"`
	ParentFileId  string        `json:"parent_file_id"`
	Type          FileType      `json:"type"`
	Size          int64         `json:"size"`
	PartInfoList  []*PartInfo   `json:"part_info_list"`
}

type Files struct {
	Items      []File `json:"items"`       // 文件列表
	NextMarker string `json:"next_marker"` // 分页标记
	Paths      []Path `json:"paths"`
}

type FileType string

const (
	FileTypeFile   FileType = "file"
	FileTypeFolder FileType = "folder"
)

// File 文件对象
type File struct {
	http.BaseResponse
	DriveId      string     `json:"drive_id"`
	Name         string     `json:"name"`
	Type         FileType   `json:"type"`
	DomainId     string     `json:"domain_id"`
	EncryptMode  string     `json:"encrypt_mode"`
	FileId       string     `json:"file_id"`
	Hidden       bool       `json:"hidden"`
	ParentFileId string     `json:"parent_file_id"`
	Starred      bool       `json:"starred"`
	Status       string     `json:"status"`
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
	Paths        []Path     `json:"paths"`

	FileItem
}

const (
	FileStatusAvailable = "available"
)

// Path 路经信息
type Path struct {
	Name   string `json:"name"`
	FileId string `json:"file_id"`
}

// FileItem 文件项信息
type FileItem struct {
	Category             string                 `json:"category"`               // 文件类型
	ContentHash          string                 `json:"content_hash"`           // 内容 HASH
	ContentHashName      string                 `json:"content_hash_name"`      // HASH 方法
	Crc64Hash            string                 `json:"crc64_hash"`             // CRC64 HASH
	ContentType          string                 `json:"content_type"`           // 内容类型
	MimeType             string                 `json:"mime_type"`              // MIME Type
	MimeExtension        string                 `json:"mime_extension"`         // MIME 后缀
	FileExtension        string                 `json:"file_extension"`         // 文件后缀
	Size                 int64                  `json:"size"`                   // 大小
	DownloadUrl          *string                `json:"download_url"`           // 下载地址
	Thumbnail            *string                `json:"thumbnail"`              // 缩略图地址
	Url                  *string                `json:"url"`                    // URL
	ImageMediaMetadata   map[string]interface{} `json:"image_media_metadata"`   // 图像媒体元信息
	VideoMediaMetadata   map[string]interface{} `json:"video_media_metadata"`   // 视频媒体元信息
	VideoPreviewMetadata map[string]interface{} `json:"video_preview_metadata"` // 视频预览元信息
	Labels               []*string              `json:"labels"`                 // 标签
	PunishFlag           int64                  `json:"punish_flag"`
}
