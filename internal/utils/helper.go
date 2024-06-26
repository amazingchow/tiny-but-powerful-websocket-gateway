package utils

import (
	"fmt"
	"regexp"
	"runtime"
)

var (
	API_VERSION_REGEX = regexp.MustCompile(`^\d+(\.\d+){2}$`)
)

// CheckApiVersion 检查API版本是否合法
func CheckApiVersion(version string) bool {
	return API_VERSION_REGEX.MatchString(version)
}

// PrintMemUsage 打印内存使用情况
func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 将字节转换为MiB
	fmt.Println("------------------------------------------------------------")
	fmt.Println("Memory Usage:")
	fmt.Printf("\tAlloc = %v MiB\n", m.Alloc/1024/1024)
	fmt.Printf("\tTotalAlloc = %v MiB\n", m.TotalAlloc/1024/1024)
	fmt.Printf("\tSys = %v MiB\n", m.Sys/1024/1024)
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
	fmt.Println("------------------------------------------------------------")
}
