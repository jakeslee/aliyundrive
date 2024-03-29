package aliyundrive

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/jakeslee/aliyundrive/http"
	"github.com/jakeslee/aliyundrive/models"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"io"
	"math/big"
	http2 "net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	// ThunkSizeDefault 默认 10MB 大小
	ThunkSizeDefault = 1024 * 1024 * 10
	timeLayout       = "2006-01-02T15:04:05.000Z"
)

type FolderFilesOptions struct {
	FolderFileId   string
	OrderBy        string
	OrderDirection string
	Marker         string
}

// GetFolderFiles 获取指定目录下的文件列表
func (d *AliyunDrive) GetFolderFiles(credential *Credential, options *FolderFilesOptions) (*models.FolderFilesResponse, error) {
	cacheKey := fmt.Sprintf("%s:%s", options.FolderFileId, options.Marker)

	var resp models.FolderFilesResponse

	if cached, err := d.cache.Get(cacheKey); err == nil {
		return cached.(*models.FolderFilesResponse), nil
	}

	request := models.NewFolderFilesRequest()

	request.DriveId = credential.DefaultDriveId
	request.ParentFileId = options.FolderFileId
	request.OrderBy = options.OrderBy
	request.OrderDirection = options.OrderDirection
	request.Marker = options.Marker

	err := d.send(credential, request, &resp)

	if err == nil {
		_ = d.cache.Set(cacheKey, &resp)
		//d.cacheMap.Store(cacheKey, &resp)

		go d.cacheFiles(resp.Items)
	}

	return &resp, err
}

// GetByPath 通过 Path 取得文件信息，不存在则错误
func (d *AliyunDrive) GetByPath(credential *Credential, fullPath string) (*models.FileResponse, error) {
	fullPath = PrefixSlash(filepath.Clean(fullPath))

	request := models.NewGetFileByPathRequest()
	request.DriveId = credential.DefaultDriveId
	request.FilePath = fullPath

	var resp models.FileResponse

	err := d.send(credential, request, &resp)

	if err == nil {
		_ = d.cache.Set(resp.FileId, &resp)
	}

	return &resp, err
}

var ErrPartialFoundPath = errors.New("partial found: only found partial parent")

// ResolvePathToFileId 通过路径查找 fileId
// 当查询出现错误时，返回 "","", err
// 当查找到路径前一部分时，返回 fileId, prefix, ErrPartialFoundPath
// 当全部找到时，返回 fileId, fullpath, nil
func (d *AliyunDrive) ResolvePathToFileId(credential *Credential, fullpath string) (string, string, error) {
	path := PrefixSlash(filepath.Clean(fullpath))

	foundPath := "/"

	if path == "/" {
		go func() {
			_, _ = d.GetFile(credential, DefaultRootFileId)
		}()
		return DefaultRootFileId, foundPath, nil
	}

	splitFolders := strings.Split(path, "/")

	fileId := DefaultRootFileId

	for i := 0; i < len(splitFolders)-1; i++ {
		matched := false
		marker := ""

		for !matched {
			folderFiles, err := d.GetFolderFiles(credential, &FolderFilesOptions{
				OrderBy:        "updated_at",
				OrderDirection: models.OrderDirectionTypeDescend,
				FolderFileId:   fileId,
				Marker:         marker,
			})

			if err != nil {
				return "", "", err
			}

			for _, item := range folderFiles.Items {
				if item.Name == splitFolders[i+1] {
					fileId = item.FileId
					foundPath = filepath.Join(foundPath, item.Name)
					matched = true
					break
				}
			}

			if matched {
				break
			}

			if folderFiles.NextMarker == "" {
				return fileId, foundPath, ErrPartialFoundPath
			}

			marker = folderFiles.NextMarker
		}
	}

	return fileId, foundPath, nil
}

// cacheFiles 缓存 FileId 对应的 File 信息
func (d *AliyunDrive) cacheFiles(files []*models.File) {
	for _, file := range files {
		_ = d.cache.Set(file.FileId, &models.FileResponse{
			File: file,
		})
	}
}

// GetFile 获取文件信息
func (d *AliyunDrive) GetFile(credential *Credential, fileId string) (*models.FileResponse, error) {
	if v, err := d.cache.Get(fileId); err == nil {
		return v.(*models.FileResponse), nil
	}

	request := models.NewFileRequest()

	request.DriveId = credential.DefaultDriveId
	request.FileId = fileId

	var resp models.FileResponse

	err := d.send(credential, request, &resp)

	if err == nil {
		_ = d.cache.Set(fileId, &resp)
	}

	return &resp, err
}

// GetDownloadURL 获取下载路经
// https://www.aliyundrive.com 获取的 RefreshToken 得到的 URL 需要带 Referrer 下载
// 移动端 Web 或手机端获取的 RefreshToken 得到的 URL可以直链下载
func (d *AliyunDrive) GetDownloadURL(credential *Credential, fileId string) (*models.DownloadURLResponse, error) {
	var resp models.DownloadURLResponse

	key := fileId + ":url"

	if cached, err := d.cache.Get(key); err == nil {
		response := cached.(*models.DownloadURLResponse)

		urlExp, err := time.Parse(timeLayout, response.Expiration)

		if err == nil {
			if time.Now().Sub(urlExp) > time.Hour {
				return response, nil
			}
		}
	}

	request := models.NewDownloadURLRequest()

	request.DriveId = credential.DefaultDriveId
	request.FileId = fileId

	err := d.send(credential, request, &resp)

	if err == nil {
		_ = d.cache.Set(key, &resp)
	}

	return &resp, err
}

// Download 下载文件
func (d *AliyunDrive) Download(credential *Credential, fileId, requestRange string) (*http2.Response, error) {
	urlResponse, err := d.GetDownloadURL(credential, fileId)

	if err != nil {
		return nil, err
	}

	logrus.Debugf("download file %s, url: %s", fileId, *urlResponse.Url)

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
	}

	res, err := d.rawClient.Do(request)

	logrus.Debugf("request %s finished", fileId)

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
	Name          string
	ParentFileId  string // 父路径
	Size          int64
	CheckNameMode models.CheckNameMode
	PreHash       string
	ContentHash   string
	ProofCode     string
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

	if options.CheckNameMode != "" {
		request.CheckNameMode = options.CheckNameMode
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
func (d *AliyunDrive) PartUpload(credential *Credential, uploadUrl string, reader io.Reader, callback ProgressCallback) error {
	var p io.Reader

	p = &progressReader{
		reader,
		callback,
	}

	if d.uploadLimitEnable {
		p = &RateLimiterReader{
			limiter: d.uploadRateLimiter,
			Reader:  p,
		}
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

	File        *os.File
	ContentHash string
}

// UploadFileRapid 上传文件（秒传）
// 当文件较大（如1GB以上）时，计算整个文件的 sha1 将花费较大的资源。先执行预秒传匹配到可能的数据才执行秒传。
func (d *AliyunDrive) UploadFileRapid(credential *Credential, options *UploadFileRapidOptions) (file *models.File, rapid bool, err error) {
	// 重置文件读取位置
	_, err = options.File.Seek(0, 0)
	if err != nil {
		return nil, false, err
	}

	// 执行预秒传
	preHash, err := d.ComputePreHash(options.File)
	if err != nil {
		return nil, false, err
	}

	// 对于空文件，回退到普通上传
	if options.Size <= 0 {
		preHash = ""
	}

	response, err := d.CreateWithFolders(credential, &CreateWithFoldersOptions{
		ParentFileId: options.ParentFileId,
		Name:         options.Name,
		Size:         options.Size,
		PreHash:      preHash,
	})

	doneFn := func(info *ProgressInfo) {
		d.EvictCacheWithPrefix(options.ParentFileId)
		if options.ProgressDone != nil {
			options.ProgressDone(info)
		}
	}

	var preHashMatch bool

	if err, ok := err.(*http.AliyunDriveError); ok {
		if err.Code != models.CodePreHashMatched {
			return nil, false, err
		} else {
			preHashMatch = true
		}
	} else if err != nil {
		return nil, false, err
	}

	// 返回 rapid_upload=false，则表明预秒传没有匹配到对应的数据，直接上传数据
	if !preHashMatch {
		if preHashResp, ok := response.(*models.CreateWithFoldersPreHashResponse); ok && !preHashResp.RapidUpload {
			file, err = d.uploadParts(credential, &uploadPartsOptions{
				fileId:           preHashResp.FileId,
				uploadId:         preHashResp.UploadId,
				partInfoList:     preHashResp.PartInfoList,
				reader:           options.File,
				progressCallback: options.ProgressCallback,
				progressDone:     doneFn,
			})

			return file, false, err
		}
	}

	// 表明预秒传匹配到可能的数据，再次调用秒传流程
	proofCodeV1, err := d.ComputeProofCodeV1(credential, options.File, options.Size)
	if err != nil {
		return nil, false, err
	}

	// 如果已经提供内容 HASH，不用重复计算
	contentSha1 := options.ContentHash
	if options.ContentHash == "" {
		contentSha1, err = ChecksumFileSha1(options.File)
		if err != nil {
			return nil, false, err
		}
	}

	response, err = d.CreateWithFolders(credential, &CreateWithFoldersOptions{
		ParentFileId: options.ParentFileId,
		Name:         options.Name,
		Size:         options.Size,
		ProofCode:    proofCodeV1,
		ContentHash:  strings.ToUpper(contentSha1),
	})
	if err != nil {
		return nil, false, err
	}

	proofResp := response.(*models.CreateWithFoldersWithProofResponse)

	if proofResp.RapidUpload {
		file, err := d.GetFile(credential, proofResp.FileId)
		if err != nil {
			return nil, false, err
		}

		doneFn(&ProgressInfo{
			FileId:       file.FileId,
			UploadId:     proofResp.UploadId,
			PartInfoList: proofResp.PartInfoList,
		})

		return file.File, true, nil
	}

	// 最后如果秒传还是失败，说明预秒传 HASH 碰撞了，直接上传
	file, err = d.uploadParts(credential, &uploadPartsOptions{
		fileId:           proofResp.FileId,
		uploadId:         proofResp.UploadId,
		partInfoList:     proofResp.PartInfoList,
		reader:           options.File,
		progressCallback: options.ProgressCallback,
		progressDone:     doneFn,
	})

	return file, false, err
}

type UploadFileOptions struct {
	Name             string
	Size             int64
	ParentFileId     string
	ProgressStart    func(info *ProgressInfo)
	ProgressCallback ProgressCallback
	ProgressDone     func(info *ProgressInfo)
	Reader           io.Reader
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

	if options.ProgressStart != nil {
		options.ProgressStart(&ProgressInfo{
			FileId:       preHashResp.FileId,
			UploadId:     preHashResp.UploadId,
			PartInfoList: preHashResp.PartInfoList,
		})
	}

	return d.uploadParts(credential, &uploadPartsOptions{
		fileId:           preHashResp.FileId,
		uploadId:         preHashResp.UploadId,
		partInfoList:     preHashResp.PartInfoList,
		reader:           options.Reader,
		progressCallback: options.ProgressCallback,
		progressDone: func(info *ProgressInfo) {
			// 更新目录缓存
			d.EvictCacheWithPrefix(options.ParentFileId)

			if options.ProgressDone != nil {
				options.ProgressDone(info)
			}
		},
	})
}

type uploadPartsOptions struct {
	reader           io.Reader
	partInfoList     []*models.PartInfo
	fileId           string
	uploadId         string
	progressCallback ProgressCallback
	progressDone     func(info *ProgressInfo)
}

// uploadParts 上传分片并合并文件
func (d *AliyunDrive) uploadParts(credential *Credential, options *uploadPartsOptions) (*models.File, error) {
	if file, ok := options.reader.(*os.File); ok {
		_, err := file.Seek(0, 0)
		if err != nil {
			return nil, err
		}
	}

	bufferSize := int64(ThunkSizeDefault)

	if len(options.partInfoList) > 0 {
		info := options.partInfoList[0]
		bufferSize = info.PartSize
	}

	for _, info := range options.partInfoList {
		if info.PartSize == 0 {
			continue
		}

		err := d.PartUpload(credential, *info.UploadUrl, io.LimitReader(options.reader, bufferSize), options.progressCallback)
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

	if options.progressDone != nil {
		options.progressDone(&ProgressInfo{
			FileId:       options.fileId,
			UploadId:     options.uploadId,
			PartInfoList: options.partInfoList,
		})
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

	if err == nil {
		file, _ := d.GetFile(credential, fileId)

		d.EvictCacheWithPrefix(fileId)
		d.EvictCacheWithPrefix(file.ParentFileId)
	}

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

	if err == nil {
		file, _ := d.GetFile(credential, fileId)

		d.EvictCacheWithPrefix(fileId)
		d.EvictCacheWithPrefix(file.ParentFileId)
		d.EvictCacheWithPrefix(toParentFileId)
	}

	return &resp, err
}

// RemoveFile 删除文件
func (d *AliyunDrive) RemoveFile(credential *Credential, fileId string) (*http.BaseResponse, error) {
	request := models.NewRemoveFileRequest()

	request.DriveId = credential.DefaultDriveId
	request.FileId = fileId

	var resp http.BaseResponse

	err := d.send(credential, request, &resp)

	if err == nil {
		file, _ := d.GetFile(credential, fileId)

		d.EvictCacheWithPrefix(fileId)
		d.EvictCacheWithPrefix(file.ParentFileId)
	}

	return &resp, err
}

// CreateDirectory 创建目录
func (d *AliyunDrive) CreateDirectory(credential *Credential, parentFileId, name string) (*models.File, error) {
	request := models.NewCreateWithFoldersPreHashRequest()

	request.DriveId = credential.DefaultDriveId
	request.CheckNameMode = models.CheckNameModeRefuse
	request.Type = models.FileTypeFolder
	request.ParentFileId = parentFileId
	request.Name = name

	var resp models.File

	err := d.send(credential, request, &resp)

	d.EvictCacheWithPrefix(parentFileId)

	return &resp, err
}

type ProgressInfo struct {
	FileId       string
	UploadId     string
	PartInfoList []*models.PartInfo
}

type RateLimiterReader struct {
	io.Reader
	limiter *rate.Limiter
}

func (p *RateLimiterReader) Read(buf []byte) (n int, err error) {
	err = p.limiter.WaitN(context.TODO(), len(buf))
	if err != nil {
		return 0, err
	}

	n, err = p.Reader.Read(buf)
	return n, err
}

type ProgressCallback func(readCount int64) bool

type progressReader struct {
	io.Reader
	Callback ProgressCallback
}

func (p *progressReader) Read(buf []byte) (n int, err error) {
	n, err = p.Reader.Read(buf)

	if p.Callback != nil && !p.Callback(int64(n)) {
		return 0, errors.New("user stop")
	}

	return n, err
}
