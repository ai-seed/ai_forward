package services

// stringValue 获取字符串指针的值
func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// floatValue 获取浮点数指针的值
func floatValue(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

// int64Ptr 返回int64指针
func int64Ptr(i int64) *int64 {
	return &i
}

// int64Value 获取int64指针的值
func int64Value(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

// boolPtr 返回bool指针
func boolPtr(b bool) *bool {
	return &b
}

// boolValue 获取bool指针的值
func boolValue(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
