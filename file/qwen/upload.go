package qwen

import (
	"bytes"
	"encoding/json"
	"fmt"
	bailian20231229 "github.com/alibabacloud-go/bailian-20231229/v2/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"io/ioutil"
	"llm-util/conf"
	futil "llm-util/util/file"
	"llm-util/util/generic"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// https://api.aliyun.com/api/bailian/2023-12-29/ApplyFileUploadLease?spm=a2c4g.11186623.0.0.7ad7733a9WUhi3&RegionId=cn-beijing&tab=DEMO&lang=GO

func UploadFile(filePath string) (addFileResp *bailian20231229.AddFileResponse, err error) {

	fileInfo, md5, err := GetFileInfo(filePath)
	if err != nil {
		return
	}

	req := &ApplyFileUploadLeaseRequest{
		CategoryId:   "default",
		WorkspaceId:  conf.WORKSPACE_ID,
		FileName:     fileInfo.Name(),
		Md5:          md5,
		SizeInBytes:  strconv.FormatInt(fileInfo.Size(), 10),
		CategoryType: "SESSION_FILE",
	}
	// fmt.Println(req)

	leaseResp, err := req.Send()
	if err != nil {
		return
	}
	// fmt.Printf("leaseResp: %v\n", leaseResp)

	url := *leaseResp.Body.Data.Param.Url
	method := *leaseResp.Body.Data.Param.Method
	headers := leaseResp.Body.Data.Param.Headers.(map[string]interface{})
	xExtra := headers["X-bailian-extra"].(string)
	contentType := headers["Content-Type"].(string)

	err = uploadFile(filePath, url, method, xExtra, contentType)
	if err != nil {
		return
	}

	addFileResp, err = req.AddFile(leaseResp)
	if err != nil {
		return
	}
	// fmt.Printf("addFileResp: %v\n", addFileResp)

	// 查询文档状态
	for {
		if checkUploadFinish(conf.WORKSPACE_ID, *addFileResp.Body.Data.FileId) {
			break
		}
		time.Sleep(time.Second * 3)
	}
	return
}

func GetFileInfo(filePath string) (fileInfo os.FileInfo, md5 string, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	fileInfo, err = file.Stat()
	if err != nil {
		return
	}

	md5, err = futil.GetFileMD5(file)
	if err != nil {
		return
	}
	return
}

func (r *ApplyFileUploadLeaseRequest) AddFile(leaseResp *bailian20231229.ApplyFileUploadLeaseResponse) (addFileResp *bailian20231229.AddFileResponse, err error) {

	client, err := CreateClient()
	if err != nil {
		return nil, err
	}

	addFileRequest := &bailian20231229.AddFileRequest{
		CategoryId:   generic.PtrOf(r.CategoryId),
		CategoryType: generic.PtrOf(r.CategoryType),
		LeaseId:      leaseResp.Body.Data.FileUploadLeaseId,
		Parser:       generic.PtrOf("DASHSCOPE_DOCMIND"),
		Tags:         nil,
	}
	// fmt.Printf("addFileRequest: %v", addFileRequest)
	bailian20231229.NewClient(&openapi.Config{})

	runtime := &util.RuntimeOptions{}
	headers := make(map[string]*string)
	tryErr := func() (_e error) {
		defer func() {
			if r := tea.Recover(recover()); r != nil {
				_e = r
			}
		}()
		// 复制代码运行请自行打印 API 的返回值
		addFileResp, err = client.AddFileWithOptions(tea.String(r.WorkspaceId), addFileRequest, headers, runtime)
		if err != nil {
			return err
		}

		return nil
	}()

	if tryErr != nil {
		var error = &tea.SDKError{}
		if _t, ok := tryErr.(*tea.SDKError); ok {
			error = _t
		} else {
			error.Message = tea.String(tryErr.Error())
		}
		// 此处仅做打印展示，请谨慎对待异常处理，在工程项目中切勿直接忽略异常。
		// 错误 message
		fmt.Println(tea.StringValue(error.Message))
		// 诊断地址
		var data interface{}
		d := json.NewDecoder(strings.NewReader(tea.StringValue(error.Data)))
		d.Decode(&data)
		if m, ok := data.(map[string]interface{}); ok {
			recommend, _ := m["Recommend"]
			fmt.Println(recommend)
		}
		_, err = util.AssertAsString(error.Message)
		if err != nil {
			return nil, err
		}
	}

	return
}

// DescribeFileWithOptions 查看文档解析状态
// 调用频率限制：5次/秒
func DescribeFileWithOptions(workspaceId, fileId string) (descResp *bailian20231229.DescribeFileResponse, _err error) {
	client, _err := CreateClient()
	if _err != nil {
		return
	}

	runtime := &util.RuntimeOptions{}
	headers := make(map[string]*string)
	tryErr := func() (_e error) {
		defer func() {
			if r := tea.Recover(recover()); r != nil {
				_e = r
			}
		}()
		// 复制代码运行请自行打印 API 的返回值
		descResp, _err = client.DescribeFileWithOptions(tea.String(workspaceId), tea.String(fileId), headers, runtime)
		if _err != nil {
			return _err
		}

		return nil
	}()

	if tryErr != nil {
		var error = &tea.SDKError{}
		if _t, ok := tryErr.(*tea.SDKError); ok {
			error = _t
		} else {
			error.Message = tea.String(tryErr.Error())
		}
		// 此处仅做打印展示，请谨慎对待异常处理，在工程项目中切勿直接忽略异常。
		// 错误 message
		fmt.Println(tea.StringValue(error.Message))
		// 诊断地址
		var data interface{}
		d := json.NewDecoder(strings.NewReader(tea.StringValue(error.Data)))
		d.Decode(&data)
		if m, ok := data.(map[string]interface{}); ok {
			recommend, _ := m["Recommend"]
			fmt.Println(recommend)
		}
		_, _err = util.AssertAsString(error.Message)
		if _err != nil {
			return
		}
	}
	return
}

func checkUploadFinish(workspaceId, fileId string) bool {
	descResp, err := DescribeFileWithOptions(workspaceId, fileId)
	// fmt.Printf("descResp: %v", *descResp)
	if err != nil {
		fmt.Println(err)
		return true
	}

	if descResp == nil || descResp.Body == nil || descResp.Body.Data == nil || descResp.Body.Data.Status == nil {
		return false
	}

	if *descResp.Body.Status == "429" {
		// 频率限制 请求间隔设置15s
		time.Sleep(time.Second * 15)
	}

	switch *descResp.Body.Data.Status {
	case "INIT":
		fmt.Println("文档待解析")
	case "PARSING":
		fmt.Println("文档解析中...")
	case "PARSE_SUCCESS":
		fmt.Println("文档解析成功！请等待文档准备就绪")
	case "PARSE_FAILED":
		fmt.Println("文档解析失败")
		return true
	case "FILE_IS_READY":
		fmt.Println("文档已准备就绪")
		return true
	case "INDEX_BUILDING":
		fmt.Println("文档索引构建中...")
	default:
		fmt.Println("处理中...")
	}

	return false
}

func uploadFile(filePath, preSignedURL, method, xExtra, contentType string) error {

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 设置请求头
	headers := map[string]string{
		"X-bailian-extra": xExtra,
		"Content-Type":    contentType,
	}

	// 创建 PUT 请求
	req, err := http.NewRequest(method, preSignedURL, file)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	// fmt.Printf("uploadFile req: %v\n", *req)

	// 添加请求头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// fmt.Printf("uploadFile resp：%v\n", *resp)

	// 检查响应状态
	if resp.StatusCode == http.StatusOK {
		fmt.Println("File uploaded successfully.")
	} else {
		return fmt.Errorf("failed to upload the file. ResponseCode: %d", resp.StatusCode)
	}

	return nil
}

// uploadFileLink OSS 来源文档上传
func uploadFileLink(preSignedURL, sourceURL string) error {
	// 获取源文件
	resp, err := http.Get(sourceURL)
	if err != nil {
		return fmt.Errorf("failed to get source file: %v", err)
	}
	defer resp.Body.Close()

	// 检查源文件响应
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get source file. ResponseCode: %d", resp.StatusCode)
	}

	// 读取文件内容
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read source file content: %v", err)
	}

	// 设置请求头
	headers := map[string]string{
		"X-bailian-extra": "请替换为您在上一步中调用ApplyFileUploadLease接口实际返回的Data.Param.Headers中X-bailian-extra字段的值",
		"Content-Type":    "请替换为您在上一步中调用ApplyFileUploadLease接口实际返回的Data.Param.Headers中Content-Type字段的值",
	}

	// 创建 PUT 请求
	req, err := http.NewRequest("PUT", preSignedURL, bytes.NewReader(content))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// 添加请求头
	for key, value := range headers {
		req.Header.Add(key, value)
	}

	// 发送请求
	client := &http.Client{}
	respPut, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer respPut.Body.Close()

	// 检查响应状态
	if respPut.StatusCode == http.StatusOK {
		fmt.Println("File uploaded successfully.")
	} else {
		return fmt.Errorf("failed to upload the file. ResponseCode: %d", respPut.StatusCode)
	}

	return nil
}
