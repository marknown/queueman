package utils

import (
	"os"
	"time"
)

// NowTimeStringCN get now time and return china time format
func NowTimeStringCN() string {
	var cstZone = time.FixedZone("CST", 8*3600) // UTC/GMT +08:00
	t := time.Now()
	return t.In(cstZone).Format("2006-01-02 15:04:05")
}

// NowDateStringCN get now date and return china date format
func NowDateStringCN() string {
	var cstZone = time.FixedZone("CST", 8*3600) // UTC/GMT +08:00
	t := time.Now()
	return t.In(cstZone).Format("2006-01-02")
}

// Exists 判断所给路径文件/文件夹是否存在
func Exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

// IsDir 判断所给路径是否为文件夹
func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

// IsFile 判断所给路径是否为文件
func IsFile(path string) bool {
	return !IsDir(path)
}
