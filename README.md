# aliyundrive
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fjakeslee%2Faliyundrive.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fjakeslee%2Faliyundrive?ref=badge_shield)


本项目是基于阿里云盘网页接口封装的 SDK 工具包，可以用于扩展开发其它功能，包含以下特性：

- 文件下载 URL 获取
- 文件分片上传
- 秒传（基于 proof code v1 秒传）
- 文件移动、重命名、删除等操作
- 文件批量操作（移动）
- 文件上传限速

## 使用

```shell
go get github.com/jakeslee/aliyundrive
```
安装后使用以下方式使用：

```go
package main

import (
	"github.com/jakeslee/aliyundrive"
	"log"
	"os"
)

func main() {
	drive := aliyundrive.NewClient(&aliyundrive.Options{
		AutoRefresh: true,
		UploadRate:  2 * 1024 * 1024, // 限速 2MBps
	})

	cred, err := drive.AddCredential(aliyundrive.NewCredential(&aliyundrive.Credential{
		RefreshToken: "aliyundrive refresh token",
	}))

	file, err := os.OpenFile("/tmp/demo", os.O_RDONLY, 0)
	if err != nil {
		log.Fatal(err)
	}

	fileRapid, rapid, err := drive.UploadFileRapid(cred, &aliyundrive.UploadFileRapidOptions{
		UploadFileOptions: aliyundrive.UploadFileOptions{
			Name:         "name",
			Size:         1000,
			ParentFileId: aliyundrive.DefaultRootFileId,
		},
		File: file,
	})

	log.Printf("file: %v, rapid: %v", fileRapid, rapid)
	// ...
}
```

## 感谢

本项目开发过程中大量参考了以下优秀开源项目代码，感谢大佬们的贡献！

- [liupan1890/aliyunpan](https://github.com/liupan1890/aliyunpan)
- [Xhofe/alist](https://github.com/Xhofe/alist)
- [zxbu/webdav-aliyundriver](https://github.com/zxbu/webdav-aliyundriver)
- https://www.aliyundrive.com
- [阿里网盘开发者文档](https://help.aliyun.com/document_detail/213019.html)


## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fjakeslee%2Faliyundrive.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fjakeslee%2Faliyundrive?ref=badge_large)