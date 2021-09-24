package aliyundrive

import (
	"bufio"
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
)

func ToMD5(content string) string {
	hash := md5.New()

	hash.Write([]byte(content))

	return fmt.Sprintf("%x", hash.Sum(nil))
}

func ToSHA1WithReader(content io.Reader) (string, error) {
	hash := sha1.New()

	_, err := io.Copy(hash, content)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func Min(lhs, rhs int64) int64 {
	if lhs <= rhs {
		return lhs
	}
	return rhs
}

func ChecksumFileSha1(file *os.File) (string, error) {
	_, err := file.Seek(0, 0)
	if err != nil {
		return "", err
	}

	reader := bufio.NewReader(file)
	hash := sha1.New()

	buf := make([]byte, 65536)

	for {
		switch n, err := reader.Read(buf); err {
		case nil:
			hash.Write(buf[:n])
		case io.EOF:
			return fmt.Sprintf("%x", hash.Sum(nil)), nil
		default:
			return "", err
		}
	}
}
