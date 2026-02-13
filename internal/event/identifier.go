package event

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"

	"github.com/denisbrodbeck/machineid"
)

// distinctId 用于存储唯一标识符
var distinctId string

const (
	// hashKey 用于生成哈希值的密钥
	hashKey = "charm"
	// fallbackId 当无法获取设备ID时的回退标识符
	fallbackId = "unknown"
)

// getDistinctId 获取设备的唯一标识符
// 首先尝试使用机器ID，如果失败则尝试获取MAC地址并哈希，最后使用回退值
func getDistinctId() string {
	// 尝试获取受保护的机器ID
	if id, err := machineid.ProtectedID(hashKey); err == nil {
		return id
	}
	// 如果获取机器ID失败，尝试获取MAC地址并哈希
	if macAddr, err := getMacAddr(); err == nil {
		return hashString(macAddr)
	}
	// 如果所有方法都失败，返回回退标识符
	return fallbackId
}

// getMacAddr 获取本机活动网络接口的MAC地址
// 返回第一个非回环且已启用的网络接口的MAC地址
func getMacAddr() (string, error) {
	// 获取所有网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	// 遍历网络接口，查找符合条件的接口
	for _, iface := range interfaces {
		// 检查接口是否已启用、非回环接口，且具有硬件地址
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 && len(iface.HardwareAddr) > 0 {
			// 检查接口是否有地址分配
			if addrs, err := iface.Addrs(); err == nil && len(addrs) > 0 {
				return iface.HardwareAddr.String(), nil
			}
		}
	}
	// 未找到符合条件的网络接口
	return "", fmt.Errorf("未找到具有MAC地址的活动网络接口")
}

// hashString 使用HMAC-SHA256对字符串进行哈希
// 参数 str 为需要哈希的原始字符串
// 返回十六进制编码的哈希字符串
func hashString(str string) string {
	// 创建HMAC-SHA256哈希器
	hash := hmac.New(sha256.New, []byte(str))
	// 写入密钥数据
	hash.Write([]byte(hashKey))
	// 返回十六进制编码的哈希结果
	return hex.EncodeToString(hash.Sum(nil))
}
