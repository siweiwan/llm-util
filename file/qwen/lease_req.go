package qwen

import (
	"encoding/json"
	"fmt"
	bailian20231229 "github.com/alibabacloud-go/bailian-20231229/v2/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"llm-util/conf"
	futil "llm-util/util/file"
	"os"
	"strconv"
	"strings"
)

const (
	ENDPOINT_URL = "bailian.cn-beijing.aliyuncs.com"
)

// 可通过 -ldflags -X 在编译时注入，为空时回退到环境变量
var (
	AccessKeyId     string
	AccessKeySecret string
)

var Client *bailian20231229.Client

func init() {
	var err error
	Client, err = CreateClient()
	if err != nil {
		panic(fmt.Sprintf("init client failed, err:%s", err))
	}
}

func CreateUploadRequest(filePath string) (req *ApplyFileUploadLeaseRequest, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	md5, err := futil.GetFileMD5(file)
	if err != nil {
		return nil, err
	}

	req = &ApplyFileUploadLeaseRequest{
		CategoryId:   "default",
		WorkspaceId:  conf.WORKSPACE_ID,
		FileName:     file.Name(),
		Md5:          md5,
		SizeInBytes:  strconv.FormatInt(fileInfo.Size(), 10),
		CategoryType: "SESSION_FILE",
	}

	return
}

// CreateClient
// 使用AK&SK初始化账号Client
func CreateClient() (result *bailian20231229.Client, err error) {
	// 工程代码泄露可能会导致 AccessKey 泄露，并威胁账号下所有资源的安全性。以下代码示例仅供参考。
	// 建议使用更安全的 STS 方式，更多鉴权访问方式请参见：https://help.aliyun.com/document_detail/378661.html。
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
	}
	// Endpoint 请参考 https://api.aliyun.com/product/bailian
	config.Endpoint = tea.String(ENDPOINT_URL)
	result = &bailian20231229.Client{}
	result, err = bailian20231229.NewClient(config)
	return result, err
}

// 申请文档上传租约
func (r *ApplyFileUploadLeaseRequest) Send() (res *bailian20231229.ApplyFileUploadLeaseResponse, err error) {

	client, err := CreateClient()
	if err != nil {
		return nil, err
	}

	applyFileUploadLeaseRequest := &bailian20231229.ApplyFileUploadLeaseRequest{
		CategoryType: tea.String(r.CategoryType),
		FileName:     tea.String(r.FileName),
		Md5:          tea.String(r.Md5),
		SizeInBytes:  tea.String(r.SizeInBytes),
	}

	runtime := &util.RuntimeOptions{}
	headers := make(map[string]*string)

	tryErr := func() (e error) {
		defer func() {
			if r := tea.Recover(recover()); r != nil {
				e = r
			}
		}()
		// 复制代码运行请自行打印 API 的返回值
		res, err = client.ApplyFileUploadLeaseWithOptions(tea.String(r.CategoryId), tea.String(r.WorkspaceId), applyFileUploadLeaseRequest, headers, runtime)
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
			return res, err
		}
	}
	return res, err
}
