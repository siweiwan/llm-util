package util

// GetFirstXChars 获取字符串的前x个字符
func GetFirstXChars(str string, x int) string {
	if x <= 0 {
		return ""
	}

	runes := []rune(str) // 将字符串转换为rune切片
	if x > len(runes) {
		x = len(runes) // 防止越界
	}
	return string(runes[:x]) // 获取前x个字符
}
