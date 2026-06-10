package file

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"llm-util/conf"
	futil "llm-util/util/file"
	"llm-util/util/generic"

	bailian "github.com/alibabacloud-go/bailian-20231229/v2/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
)

const (
	EndpointURL = "bailian.cn-beijing.aliyuncs.com"

	// 轮询间隔与限流等待
	pollInterval    = 3 * time.Second
	throttleWait    = 15 * time.Second
	maxPollAttempts = 120 // 最长等待约 6 分钟
)

// 可通过 -ldflags -X 在编译时注入，为空时回退到环境变量
var (
	AccessKeyId     string
	AccessKeySecret string
)

// CreateClient 使用 AK/SK 初始化阿里云百炼 SDK Client
func CreateClient() (*bailian.Client, error) {
	akId := AccessKeyId
	if akId == "" {
		akId = os.Getenv("ALIBABA_CLOUD_ACCESS_KEY_ID")
	}
	akSecret := AccessKeySecret
	if akSecret == "" {
		akSecret = os.Getenv("ALIBABA_CLOUD_ACCESS_KEY_SECRET")
	}
	config := &openapi.Config{
		AccessKeyId:     tea.String(akId),
		AccessKeySecret: tea.String(akSecret),
		Endpoint:        tea.String(EndpointURL),
	}
	return bailian.NewClient(config)
}

// GetFileInfo 获取文件信息和 MD5
func GetFileInfo(filePath string) (os.FileInfo, string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, "", fmt.Errorf("获取文件信息失败: %w", err)
	}

	md5, err := futil.GetFileMD5(f)
	if err != nil {
		return nil, "", fmt.Errorf("计算文件MD5失败: %w", err)
	}

	return info, md5, nil
}

// UploadFile 完整的文件上传流程，返回 fileId
//
// 流程：
//  1. 计算文件 MD5 和大小
//  2. ApplyFileUploadLease 申请上传租约
//  3. 上传文件二进制到预签名 URL
//  4. AddFile 注册文件到百炼
//  5. 轮询等待文件就绪
func UploadFile(filePath string) (string, error) {
	// Step 1: 文件信息
	fileInfo, md5, err := GetFileInfo(filePath)
	if err != nil {
		return "", err
	}

	fileName := fileInfo.Name()
	sizeInBytes := strconv.FormatInt(fileInfo.Size(), 10)

	// Step 2: 申请上传租约
	leaseResp, err := applyLease(fileName, md5, sizeInBytes)
	if err != nil {
		return "", fmt.Errorf("申请上传租约失败: %w", err)
	}

	// Step 3: 上传文件到 OSS
	if err := uploadToOSS(filePath, leaseResp.Data.Param); err != nil {
		return "", fmt.Errorf("上传文件到OSS失败: %w", err)
	}

	// Step 4: 注册文件
	fileId, err := registerFile(leaseResp.Data.FileUploadLeaseId)
	if err != nil {
		return "", fmt.Errorf("注册文件失败: %w", err)
	}

	// Step 5: 轮询等待文件就绪
	if err := waitReady(fileId); err != nil {
		return "", fmt.Errorf("等待文件就绪失败: %w", err)
	}

	return fileId, nil
}

// applyLease 申请文件上传租约
func applyLease(fileName, md5, sizeInBytes string) (*ApplyLeaseResponse, error) {
	client, err := CreateClient()
	if err != nil {
		return nil, fmt.Errorf("创建SDK客户端失败: %w", err)
	}

	req := &bailian.ApplyFileUploadLeaseRequest{
		CategoryType: tea.String(CategoryTypeSessionFile),
		FileName:     tea.String(fileName),
		Md5:          tea.String(md5),
		SizeInBytes:  tea.String(sizeInBytes),
	}

	runtime := &util.RuntimeOptions{}
	headers := make(map[string]*string)

	resp, err := client.ApplyFileUploadLeaseWithOptions(
		tea.String("default"),
		tea.String(conf.WORKSPACE_ID),
		req, headers, runtime,
	)
	if err != nil {
		return nil, err
	}

	if resp.Body == nil || resp.Body.Data == nil {
		msg := ""
		if resp.Body != nil && resp.Body.Message != nil {
			msg = *resp.Body.Message
		}
		return nil, fmt.Errorf("申请租约返回数据为空: %s", msg)
	}

	if resp.Body.Data.Param == nil {
		return nil, fmt.Errorf("申请租约返回Param为空")
	}

	// 转换为自定义类型（避免后续代码依赖 SDK 类型）
	param := resp.Body.Data.Param
	uploadHeaders := make(map[string]string)
	if h, ok := param.Headers.(map[string]interface{}); ok {
		for k, v := range h {
			if s, ok := v.(string); ok {
				uploadHeaders[k] = s
			}
		}
	}

	return &ApplyLeaseResponse{
		Code:    tea.StringValue(resp.Body.Code),
		Message: tea.StringValue(resp.Body.Message),
		Success: tea.BoolValue(resp.Body.Success),
		Data: &ApplyLeaseData{
			FileUploadLeaseId: tea.StringValue(resp.Body.Data.FileUploadLeaseId),
			Type:              tea.StringValue(resp.Body.Data.Type),
			Param: &UploadParam{
				Headers: uploadHeaders,
				Method:  tea.StringValue(param.Method),
				Url:     tea.StringValue(param.Url),
			},
		},
	}, nil
}

// uploadToOSS 将文件二进制上传到预签名 URL
func uploadToOSS(filePath string, param *UploadParam) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	req, err := http.NewRequest(param.Method, param.Url, f)
	if err != nil {
		return fmt.Errorf("创建上传请求失败: %w", err)
	}
	for k, v := range param.Headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送上传请求失败: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) // drain body

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("上传文件HTTP状态码异常: %d", resp.StatusCode)
	}
	return nil
}

// registerFile 调用 AddFile 将上传的文件注册到百炼数据中心
func registerFile(leaseId string) (string, error) {
	client, err := CreateClient()
	if err != nil {
		return "", fmt.Errorf("创建SDK客户端失败: %w", err)
	}

	req := &bailian.AddFileRequest{
		CategoryId:   generic.PtrOf("default"),
		CategoryType: generic.PtrOf(CategoryTypeSessionFile),
		LeaseId:      generic.PtrOf(leaseId),
		Parser:       generic.PtrOf(ParserAutoSelect),
	}

	runtime := &util.RuntimeOptions{}
	headers := make(map[string]*string)

	resp, err := client.AddFileWithOptions(
		tea.String(conf.WORKSPACE_ID),
		req, headers, runtime,
	)
	if err != nil {
		return "", err
	}

	if resp.Body == nil || resp.Body.Data == nil || resp.Body.Data.FileId == nil {
		msg := ""
		if resp.Body != nil && resp.Body.Message != nil {
			msg = *resp.Body.Message
		}
		return "", fmt.Errorf("注册文件返回数据为空: %s", msg)
	}

	return *resp.Body.Data.FileId, nil
}

// waitReady 轮询文件状态直到就绪或失败
func waitReady(fileId string) error {
	for i := 0; i < maxPollAttempts; i++ {
		status, err := DescribeFileStatus(fileId)
		if err != nil {
			return err
		}

		switch status {
		case StatusFileReady, StatusParseSuccess:
			return nil
		case StatusParseFailed, StatusSafeCheckFailed, StatusIndexBuildFailed:
			return fmt.Errorf("文件处理失败，状态: %s", status)
		case StatusFileExpired:
			return fmt.Errorf("文件已过期")
		}

		time.Sleep(pollInterval)
	}
	return fmt.Errorf("等待文件就绪超时")
}

// DescribeFileStatus 查询文件当前状态
func DescribeFileStatus(fileId string) (string, error) {
	client, err := CreateClient()
	if err != nil {
		return "", fmt.Errorf("创建SDK客户端失败: %w", err)
	}

	runtime := &util.RuntimeOptions{}
	headers := make(map[string]*string)

	resp, err := client.DescribeFileWithOptions(
		tea.String(conf.WORKSPACE_ID),
		tea.String(fileId),
		headers, runtime,
	)
	if err != nil {
		return "", err
	}

	if resp.Body == nil {
		return "", fmt.Errorf("查询文件状态返回Body为空")
	}

	// 限流处理：状态码 429 时等待后重试
	if resp.Body.Status != nil && *resp.Body.Status == "429" {
		time.Sleep(throttleWait)
		return DescribeFileStatus(fileId)
	}

	if resp.Body.Data == nil || resp.Body.Data.Status == nil {
		return StatusInit, nil
	}

	return *resp.Body.Data.Status, nil
}
