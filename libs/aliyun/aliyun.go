package aliyun

import (
	"bytes"
	"encoding/base64"
	"strconv"
	"strings"
	"time"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
)

const (
	ACCESS_FROM_USER = 0
	COLON            = ":"
)

// Config aliyun configure
type Config struct {
	AccessKey string
	AccessKeySecret string
	ResourceOwnerId uint64
}

// GetUserName Get username for amqp
func (c *Config) GetUserName() string {
	var buffer bytes.Buffer
	buffer.WriteString(strconv.Itoa(ACCESS_FROM_USER))
	buffer.WriteString(COLON)
	buffer.WriteString(strconv.FormatUint(c.ResourceOwnerId,10))
	buffer.WriteString(COLON)
	buffer.WriteString(c.AccessKey)
	return base64.StdEncoding.EncodeToString(buffer.Bytes())
}

// GetPassword Get password for amqp
func (c *Config) GetPassword() string {
	now := time.Now()
	currentMillis := strconv.FormatInt(now.UnixNano()/1000000,10)
	var buffer bytes.Buffer
	buffer.WriteString(strings.ToUpper(HmacSha1(currentMillis,c.AccessKeySecret)))
	buffer.WriteString(COLON)
	buffer.WriteString(currentMillis)
	return base64.StdEncoding.EncodeToString(buffer.Bytes())
}

// HmacSha1 encoder
func HmacSha1(keyStr string, message string) string {
	key := []byte(keyStr)
	mac := hmac.New(sha1.New, key)
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}