package file

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// func GetFileMD5(filePath string) (string, error) {
// 	// 打开文件
// 	file, err := os.Open(filePath)
// 	if err != nil {
// 		return "", err
// 	}
// 	defer file.Close()
//
// 	// 创建一个md5哈希对象
// 	hash := md5.New()
//
// 	// 将文件内容拷贝到md5哈希对象
// 	_, err = io.Copy(hash, file)
// 	if err != nil {
// 		return "", err
// 	}
//
// 	// 获取最终的哈希值
// 	hashInBytes := hash.Sum(nil)
//
// 	// 返回MD5的16进制表示
// 	return fmt.Sprintf("%x", hashInBytes), nil
// }

func GetFileMD5(file *os.File) (md5Hash string, err error) {

	// 创建一个md5哈希对象
	hash := md5.New()

	// 将文件内容拷贝到md5哈希对象
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	// 获取最终的哈希值
	hashInBytes := hash.Sum(nil)

	// 返回MD5的16进制表示
	md5Hash = fmt.Sprintf("%x", hashInBytes)

	return
}

func CalculateMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func RemoveFileExtension(filename string) string {
	ext := path.Ext(filename)                // 获取文件扩展名
	return filename[:len(filename)-len(ext)] // 返回去除后缀的文件名
}

func GetFiles(directory string, ext string) ([]string, error) {
	files := make([]string, 0)

	// 处理扩展名：去除开头的点并转为小写
	ext = strings.ToLower(ext)
	ext = strings.TrimPrefix(ext, ".")

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 收集所有非目录文件，当 ext 为空时不限制扩展名
		if !info.IsDir() {
			if ext == "" || strings.ToLower(filepath.Ext(path)) == "."+ext {
				files = append(files, path)
			}
		}

		return nil
	})

	return files, err
}
