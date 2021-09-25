package aliyundrive

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/jakeslee/aliyundrive/v1/http"
	"github.com/jakeslee/aliyundrive/v1/models"
	"github.com/sirupsen/logrus"
	"io"
	"math/big"
	http2 "net/http"
	"os"
	"strconv"
	"strings"
)

const (
	// ThunkSizeDefault 默认 10MB 大小
	ThunkSizeDefault = 1024 * 1024 * 10
)

type FolderFilesOptions struct {
	FolderFileId   string
	OrderBy        string
	OrderDirection string
	Marker         string
}

// GetFolderFiles 获取指定目录下的文件列表
func (d *AliyunDrive) GetFolderFiles(credential *Credential, options *FolderFilesOptions) (*models.FolderFilesResponse, error) {
	request := models.NewFolderFilesRequest()

	request.DriveId = credential.DefaultDriveId
	request.ParentFileId = options.FolderFileId
	request.OrderBy = options.OrderBy
	request.OrderDirection = options.OrderDirection
	request.Marker = options.Marker

	var resp models.FolderFilesResponse

	err := d.send(credential, request, &resp)

	return &resp, err
}

// GetFile 获取文件信息
func (d *AliyunDrive) GetFile(credential *Credential, fileId string) (*models.FileResponse, error) {
	request := models.NewFileRequest()

	request.DriveId = credential.DefaultDriveId
	request.FileId = fileId

	var resp models.FileResponse

	err := d.send(credential, request, &resp)

	return &resp, err
}

// GetDownloadURL 获取下载路经
// https://www.aliyundrive.com 获取的 RefreshToken 得到的 URL 需要带 Referrer 下载
// 移动端 Web 或手机端获取的 RefreshToken 得到的 URL可以直链下载
func (d *AliyunDrive) GetDownloadURL(credential *Credential, fileId string) (*models.DownloadURLResponse, error) {
	request := models.NewDownloadURLRequest()

	request.DriveId = credential.DefaultDriveId
	request.FileId = fileId

	var resp models.DownloadURLResponse

	err := d.send(credential, request, &resp)

	return &resp, err
}

// Download 下载文件
func (d *AliyunDrive) Download(credential *Credential, fileId, requestRange string) (*http2.Response, error) {
	urlResponse, err := d.GetDownloadURL(credential, fileId)

	if err != nil {
		return nil, err
	}

	logrus.Infof("download file %s, url: %s", fileId, *urlResponse.Url)

	request, err := http2.NewRequest(http2.MethodGet, *urlResponse.Url, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("referer", "https://www.aliyundrive.com/")

	size := urlResponse.Size

	if requestRange != "" {
		splitRange := strings.Split(requestRange, "-")

		if len(splitRange) == 2 {
			if end, err := strconv.ParseInt(splitRange[1], 10, 64); err != nil && end >= size {
				leftLen := len(splitRange[0])

				requestRange = requestRange[:leftLen+1]
			}
		}

		request.Header.Set("range", requestRange)
		logrus.Infof("request %s range: %s", fileId, requestRange)
	}

	res, err := d.rawClient.Do(request)

	logrus.Infof("request %s finished", fileId)

	if err != nil {
		return nil, err
	}

	return res, nil
}

// Search 查找文件
func (d *AliyunDrive) Search(credential *Credential, keyword, marker string) (*models.SearchResponse, error) {
	request := models.NewSearchRequest()

	request.DriveId = credential.DefaultDriveId
	request.Query = fmt.Sprintf("name match '%s'", keyword)
	request.Marker = marker

	var resp models.SearchResponse

	err := d.send(credential, request, &resp)

	return &resp, err
}

// SearchNameInFolder 在目录下查找文件名
func (d *AliyunDrive) SearchNameInFolder(credential *Credential, name, parentFileId string) (*models.SearchResponse, error) {
	request := models.NewSearchRequest()

	request.DriveId = credential.DefaultDriveId
	request.Query = fmt.Sprintf("parent_file_id = \"%s\" and (name = \"%s\")", parentFileId, name)

	var resp models.SearchResponse

	err := d.send(credential, request, &resp)

	return &resp, err
}

// ComputeProofCodeV1 计算上传的 proof_code, version=v1
// proof code 是文件内容的部分片段，主要用于实现秒传
// 算法：使用 AccessToken MD5 值的前 16 位 HEX 值转换为十进制数并对文件大小取模，
// 结果作为 proof code 的获取起始位置，取文件内容 8 位 byte 并用 Base64 编码
func (d *AliyunDrive) ComputeProofCodeV1(credential *Credential, file *os.File, size int64) (string, error) {
	hashed := ToMD5(credential.AccessToken)[0:16]
	hashedInt, _ := new(big.Int).SetString(hashed, 16)

	start := hashedInt.Mod(hashedInt, big.NewInt(size)).Int64()
	end := Min(start+8, size)

	n := end - start

	proof := make([]byte, n)

	_, err := file.Seek(start, 0)
	if err != nil {
		return "", err
	}

	read, err := file.Read(proof)
	if err != nil || int64(read) != n {
		return "", errors.New(fmt.Sprintf("read proof_code, read: %d, n: %d error %s", read, n, err))
	}

	return base64.StdEncoding.EncodeToString(proof), nil
}

// ComputePreHash 计算文件 PreHash，只计算前 1KB 的 SHA1
func (d *AliyunDrive) ComputePreHash(content io.Reader) (string, error) {
	reader := io.LimitReader(content, 1024)

	preHash, err := ToSHA1WithReader(reader)
	if err != nil {
		return "", err
	}

	return preHash, nil
}

type CreateWithFoldersOptions struct {
	Name         string
	ParentFileId string // 父路径
	Size         int64
	PreHash      string
	ContentHash  string
	ProofCode    string
}

// CreateWithFolders 在目录下创建文件，如果非秒传，接下来需要分片上传
// 响应内容中，如果需要上传，则从 PartInfoList 中获取分片上传地址信息
func (d *AliyunDrive) CreateWithFolders(credential *Credential, options *CreateWithFoldersOptions) (http.Response, error) {
	var request *models.CreateWithFolders
	var r http.Request
	var resp http.Response

	// 如果没提供 proof code 则使用 PreHash 创建文件
	if options.ProofCode == "" {
		preHashRequest := models.NewCreateWithFoldersPreHashRequest()

		preHashRequest.PreHash = options.PreHash

		request = &preHashRequest.CreateWithFolders
		r = preHashRequest
		resp = &models.CreateWithFoldersPreHashResponse{}
	} else {
		proofRequest := models.NewCreateWithFoldersWithProofRequest()

		proofRequest.ProofCode = options.ProofCode
		proofRequest.ContentHash = options.ContentHash
		proofRequest.ProofCode = options.ProofCode

		request = &proofRequest.CreateWithFolders
		r = proofRequest
		resp = &models.CreateWithFoldersWithProofResponse{}
	}

	request.Name = options.Name
	request.Size = options.Size
	request.DriveId = credential.DefaultDriveId
	request.ParentFileId = options.ParentFileId

	var err error

	request.PartInfoList, err = models.NewPartInfoList(options.Size, ThunkSizeDefault)

	if err != nil {
		return nil, err
	}

	err = d.send(credential, r, resp)

	return resp, err
}

// CompleteUpload 完成分片上传后，通过此接口结束上传（合并分片）
func (d *AliyunDrive) CompleteUpload(credential *Credential, fileId, uploadId string) (*models.CompleteFileUploadResponse, error) {
	request := models.NewCompleteFileUploadRequest()

	request.UploadId = uploadId
	request.FileId = fileId
	request.DriveId = credential.DefaultDriveId

	var resp models.CompleteFileUploadResponse

	err := d.send(credential, request, &resp)

	return &resp, err
}

// PartUpload 分片数据上传
// 因服务端使用流式计算 SHA1 值，单个文件的分片需要串行上传，不支持多个分片平行上传
func (d *AliyunDrive) PartUpload(credential *Credential, uploadUrl string, content []byte, callback ProgressCallback) error {
	p := &progressReader{
		bytes.NewReader(content),
		callback,
	}

	request, err := http2.NewRequest("PUT", uploadUrl, p)
	if err != nil {
		return err
	}

	request.Header.Set("Expect", "100-continue")

	response, err := d.rawClient.Do(request)
	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logrus.Errorf("close file error %s", err)
		}
	}(response.Body)

	return nil
}

type UploadFileRapidOptions struct {
	UploadFileOptions

	File *os.File
}

// UploadFileRapid 上传文件（秒传）
// 当文件较大（如1GB以上）时，计算整个文件的 sha1 将花费较大的资源。先执行预秒传匹配到可能的数据才执行秒传。
func (d *AliyunDrive) UploadFileRapid(credential *Credential, options *UploadFileRapidOptions) (*models.File, error) {
	// 执行预秒传
	preHash, err := d.ComputePreHash(options.File)
	if err != nil {
		return nil, err
	}

	response, err := d.CreateWithFolders(credential, &CreateWithFoldersOptions{
		ParentFileId: options.ParentFileId,
		Name:         options.Name,
		Size:         options.Size,
		PreHash:      preHash,
	})

	var preHashMatch bool

	if err, ok := err.(*http.AliyunDriveError); ok {
		if err.Code != models.CodePreHashMatched {
			return nil, err
		} else {
			preHashMatch = true
		}
	} else if err != nil {
		return nil, err
	}

	// 返回 rapid_upload=false，则表明预秒传没有匹配到对应的数据，直接上传数据
	if !preHashMatch {
		if preHashResp, ok := response.(*models.CreateWithFoldersPreHashResponse); ok && !preHashResp.RapidUpload {
			return d.uploadParts(credential, &uploadPartsOptions{
				fileId:       preHashResp.FileId,
				uploadId:     preHashResp.UploadId,
				partInfoList: preHashResp.PartInfoList,
				reader:       options.File,
				callback:     options.ProgressCallback,
			})
		}
	}

	// 表明预秒传匹配到可能的数据，再次调用秒传流程
	proofCodeV1, err := d.ComputeProofCodeV1(credential, options.File, options.Size)
	if err != nil {
		return nil, err
	}

	contentSha1, err := ChecksumFileSha1(options.File)
	if err != nil {
		return nil, err
	}

	response, err = d.CreateWithFolders(credential, &CreateWithFoldersOptions{
		ParentFileId: options.ParentFileId,
		Name:         options.Name,
		Size:         options.Size,
		ProofCode:    proofCodeV1,
		ContentHash:  strings.ToUpper(contentSha1),
	})
	if err != nil {
		return nil, err
	}

	proofResp := response.(*models.CreateWithFoldersWithProofResponse)

	if proofResp.RapidUpload {
		file, err := d.GetFile(credential, proofResp.FileId)
		if err != nil {
			return nil, err
		}

		return &file.File, nil
	}

	// 最后如果秒传还是失败，说明预秒传 HASH 碰撞了，直接上传
	return d.uploadParts(credential, &uploadPartsOptions{
		fileId:       proofResp.FileId,
		uploadId:     proofResp.UploadId,
		partInfoList: proofResp.PartInfoList,
		reader:       options.File,
		callback:     options.ProgressCallback,
	})
}

type UploadFileOptions struct {
	Name             string
	Size             int64
	ParentFileId     string
	ProgressCallback ProgressCallback
	reader           io.Reader
}

// UploadFile 同步上传文件（非秒传）
func (d *AliyunDrive) UploadFile(credential *Credential, options *UploadFileOptions) (*models.File, error) {
	response, err := d.CreateWithFolders(credential, &CreateWithFoldersOptions{
		ParentFileId: options.ParentFileId,
		Name:         options.Name,
		Size:         options.Size,
	})

	if err != nil {
		return nil, err
	}

	// 没有传 PreHash，此处肯定是非秒传
	preHashResp := response.(*models.CreateWithFoldersPreHashResponse)

	return d.uploadParts(credential, &uploadPartsOptions{
		fileId:       preHashResp.FileId,
		uploadId:     preHashResp.UploadId,
		partInfoList: preHashResp.PartInfoList,
		reader:       options.reader,
		callback:     options.ProgressCallback,
	})
}

type uploadPartsOptions struct {
	reader       io.Reader
	partInfoList []*models.PartInfo
	fileId       string
	uploadId     string
	callback     ProgressCallback
}

// uploadParts 上传分片并合并文件
func (d *AliyunDrive) uploadParts(credential *Credential, options *uploadPartsOptions) (*models.File, error) {
	if file, ok := options.reader.(*os.File); ok {
		_, err := file.Seek(0, 0)
		if err != nil {
			return nil, err
		}
	}

	buffer := make([]byte, ThunkSizeDefault)

	for _, info := range options.partInfoList {
		read, err := options.reader.Read(buffer)
		if err != nil {
			return nil, err
		}

		if read < ThunkSizeDefault {
			buffer = buffer[:read]
		}

		err = d.PartUpload(credential, *info.UploadUrl, buffer, options.callback)

		if err != nil {
			return nil, err
		}
	}

	uploadResp, err := d.CompleteUpload(credential, options.fileId, options.uploadId)
	if err != nil {
		return nil, err
	}

	if uploadResp.Code != "" || uploadResp.Status != models.FileStatusAvailable {
		logrus.Errorf("upload file error %v", uploadResp)

		return nil, errors.New(fmt.Sprintf("upload file id: %s, error: %s", uploadResp.FileId, uploadResp.Message))
	}

	return &uploadResp.File, nil
}

// RenameFile 重命名文件
func (d *AliyunDrive) RenameFile(credential *Credential, fileId, name string) (*models.RenameFileResponse, error) {
	request := models.NewRenameFileRequest()

	request.DriveId = credential.DefaultDriveId
	request.FileId = fileId
	request.Name = name

	var resp models.RenameFileResponse

	err := d.send(credential, request, &resp)

	return &resp, err
}

// MoveFile 移动单个文件
func (d *AliyunDrive) MoveFile(credential *Credential, fileId, toParentFileId string) (*http.BaseResponse, error) {
	request := models.NewMoveFileRequest()

	request.DriveId = credential.DefaultDriveId
	request.ToDriveId = credential.DefaultDriveId
	request.FileId = fileId
	request.ToParentFileId = toParentFileId

	var resp http.BaseResponse

	err := d.send(credential, request, &resp)

	return &resp, err
}

// RemoveFile 删除文件
func (d *AliyunDrive) RemoveFile(credential *Credential, fileId string) (*http.BaseResponse, error) {
	request := models.NewRemoveFileRequest()

	request.DriveId = credential.DefaultDriveId
	request.FileId = fileId

	var resp http.BaseResponse

	err := d.send(credential, request, &resp)

	return &resp, err
}

type ProgressCallback func(readCount int64) bool

type progressReader struct {
	*bytes.Reader
	Callback ProgressCallback
}

func (p *progressReader) Read(buf []byte) (n int, err error) {
	n, err = p.Reader.Read(buf)

	if !p.Callback(int64(n)) {
		return 0, errors.New("user stop")
	}

	return n, err
}
