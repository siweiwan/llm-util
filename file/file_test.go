package file

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGetFileInfo 纯本地测试：验证文件信息获取（MD5、大小）
func TestGetFileInfo(t *testing.T) {
	// 创建临时测试文件
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test_document.txt")
	content := "Hello, this is a test file for MD5 calculation."
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	info, md5, err := GetFileInfo(testFile)
	if err != nil {
		t.Fatalf("GetFileInfo 返回错误: %v", err)
	}

	// 验证文件名
	if info.Name() != "test_document.txt" {
		t.Errorf("文件名不匹配: got %q", info.Name())
	}

	// 验证文件大小
	if info.Size() != int64(len(content)) {
		t.Errorf("文件大小不匹配: got %d, want %d", info.Size(), len(content))
	}

	// 验证 MD5 非空且为 32 字符（hex 编码）
	if len(md5) != 32 {
		t.Errorf("MD5 长度异常: got %d, want 32, md5=%q", len(md5), md5)
	}

	t.Logf("文件: %s, 大小: %d bytes, MD5: %s", info.Name(), info.Size(), md5)
}

// TestGetFileInfo_NotExist 测试文件不存在时的错误处理
func TestGetFileInfo_NotExist(t *testing.T) {
	_, _, err := GetFileInfo("/nonexistent/path/file.txt")
	if err == nil {
		t.Fatal("预期文件不存在时应返回错误")
	}
	t.Logf("预期错误: %v", err)
}

// TestApplyLease 测试申请上传租约（需要真实 AK/SK）
func TestApplyLease(t *testing.T) {
	if os.Getenv("ALIBABA_CLOUD_ACCESS_KEY_ID") == "" {
		t.Skip("跳过: 未设置 ALIBABA_CLOUD_ACCESS_KEY_ID")
	}

	// 创建临时文件以获取真实的 MD5 和大小
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test_upload.pdf")
	if err := os.WriteFile(testFile, make([]byte, 1024), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	_, md5, err := GetFileInfo(testFile)
	if err != nil {
		t.Fatalf("GetFileInfo 失败: %v", err)
	}

	resp, err := applyLease("test_upload.pdf", md5, "1024")
	if err != nil {
		t.Fatalf("applyLease 失败: %v", err)
	}

	if resp.Data == nil || resp.Data.FileUploadLeaseId == "" {
		t.Fatal("返回的 FileUploadLeaseId 为空")
	}
	if resp.Data.Param == nil || resp.Data.Param.Url == "" {
		t.Fatal("返回的上传 URL 为空")
	}

	t.Logf("LeaseId: %s", resp.Data.FileUploadLeaseId)
	t.Logf("Upload URL: %s", resp.Data.Param.Url)
	t.Logf("Method: %s", resp.Data.Param.Method)
	t.Logf("Headers: %v", resp.Data.Param.Headers)
}

// TestUploadFile 完整上传流程测试（需要真实 AK/SK + 真实文件）
func TestUploadFile(t *testing.T) {
	if os.Getenv("ALIBABA_CLOUD_ACCESS_KEY_ID") == "" {
		t.Skip("跳过: 未设置 ALIBABA_CLOUD_ACCESS_KEY_ID")
	}

	testFilePath := os.Getenv("TEST_UPLOAD_FILE")
	if testFilePath == "" {
		t.Skip("跳过: 未设置 TEST_UPLOAD_FILE 环境变量（指定要上传的文件路径）")
	}

	fileId, err := UploadFile(testFilePath)
	if err != nil {
		t.Fatalf("UploadFile 失败: %v", err)
	}

	if fileId == "" {
		t.Fatal("返回的 fileId 为空")
	}

	t.Logf("上传成功! FileId: %s", fileId)
}

// TestDescribeFileStatus 测试查询文件状态（需要真实 AK/SK）
func TestDescribeFileStatus(t *testing.T) {
	if os.Getenv("ALIBABA_CLOUD_ACCESS_KEY_ID") == "" {
		t.Skip("跳过: 未设置 ALIBABA_CLOUD_ACCESS_KEY_ID")
	}

	fileId := os.Getenv("TEST_FILE_ID")
	if fileId == "" {
		t.Skip("跳过: 未设置 TEST_FILE_ID 环境变量")
	}

	status, err := describeFileStatus(fileId)
	if err != nil {
		t.Fatalf("describeFileStatus 失败: %v", err)
	}

	t.Logf("FileId: %s, Status: %s", fileId, status)
}
