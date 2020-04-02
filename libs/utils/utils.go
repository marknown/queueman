package utils

import "time"

// NowTimeStringCN get now time and return china time format
func NowTimeStringCN() string {
	var cstZone = time.FixedZone("CST", 8*3600) // UTC/GMT +08:00
	t := time.Now()
	return t.In(cstZone).Format("2006-01-02 15:04:05")
}
