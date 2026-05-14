package qwen

// 申请文档上传租约

const (
	CategoryType_UNSTRUCTURED = "UNSTRUCTURED"
	CategoryType_SESSION_FILE = "SESSION_FILE"
)

type ApplyFileUploadLeaseRequest struct {
	// 上传文档所属类目 ID，即 AddCategory 接口返回的CategoryId。您也可以在数据管理-非结构化数据页面，单击类目名称旁的 ID 图标获取。此处允许传入 default，即使用系统创建的“默认类目”。
	// 当 CategoryType 为 SESSION_FILE，此值传“default”即可，系统会自动创建或者匹配默认类目，后续会开放动态文件类目接口及控制台管理页面。
	// 示例值:
	// cate_cdd11b1b79a74e8bbd675c356a91ee35xxxxxxxx
	CategoryId string
	// 上传文档所属的业务空间 ID。在百炼的控制台首页，单击页面左上角业务空间详情图标获取。
	//
	// 示例值:
	// llm-3z7uw7fwz0vexxxx
	WorkspaceId string

	// 上传文档的名称，注意后缀需要带上文档格式类型。支持格式：pdf、docx、doc、txt、md、pptx、ppt、xlsx、xls、png、jpg、jpeg、bmp、gif。 文档名称长度限制 4-128 个字符。
	//
	// 示例值:
	// XXXX产品清单.pdf
	FileName string
	// 上传文档的 MD5 值，服务端会验证该字段（当前暂未开启），请正确填写。
	//
	// 示例值:
	// 19657c391f6c70bcea63c154d8606bb3
	// 字符长度 <= 64
	// 字符长度 >= 1
	Md5 string
	// 上传文档的大小，单位字节，服务端会验证该字段（当前暂未开启），请正确填写。取值范围：1B-100M。
	//
	// 示例值:
	// 1000
	SizeInBytes string
	// 类目类型，可选，默认值为 UNSTRUCTURED，取值范围：
	//
	// UNSTRUCTURED：非结构化数据，用于构建知识库场景。
	// SESSION_FILE：动态会话文件，用于在智能体应用中上传文档进行分析，此类文档仅在当前会话有效，过期会被自动清理。
	// 示例值:
	// UNSTRUCTURED
	// 枚举值:
	// UNSTRUCTURED
	// SESSION_FILE
	CategoryType string
}

type ApplyFileUploadLeaseResponse struct {
	// 错误状态码。
	//
	// 示例值:
	// DataCenter.FileTooLarge
	Code string
	// 接口业务数据字段。
	Data *ApplyFileUploadLeaseResponseData
	// 错误信息。
	//
	// 示例值:
	// User not authorized to operate on the specified resource
	Message string
	// 请求 ID。
	//
	// 示例值:
	// 778C0B3B-xxxx-5FC1-A947-36EDD13606AB
	RequestId string
	// 接口返回的状态码。
	//
	// 示例值:
	// 200
	Status string
	// 接口调用是否成功，可能值为：
	//
	// true：成功。
	// false：失败。
	// 示例值:
	// true
	Success bool
}

// ApplyFileUploadLeaseResponseData
// 接口业务数据字段。
type ApplyFileUploadLeaseResponseData struct {
	// 租约唯一 ID，后续调用 AddFile 接口时，需要使用该参数。
	//
	// 示例值:
	// 1e6a159107384782be5e45ac4759b247.1719325231035
	FileUploadLeaseId string
	Param             *ApplyFileUploadLeaseResponseDataParam
	// 文档的上传方式，可能值为：
	//
	// OSS.PreSignedURL
	// HTTP
	// 示例值:
	// HTTP
	Type string
}

// ApplyFileUploadLeaseResponseDataParam
// 用于上传文档的 HTTP 请求参数。
type ApplyFileUploadLeaseResponseDataParam struct {
	// 需要放到 Header 中的 K-V 字段，K 和 V 均为字符串。
	//
	// 示例值:
	// "X-bailian-extra":"MTAwNTQyNjQ5NTE2OTE3OA==", "Content-Type":"application/pdf"
	Headers map[string]string
	// HTTP 调用方法，可能值为：
	//
	// PUT
	// POST
	// 示例值:
	// PUT
	Method string
	// 文档的上传 URL 地址。
	//
	// 示例值:
	// https://bailian-datahub-data-origin-prod.oss-cn-hangzhou.aliyuncs.com/1005426495169178/10024405/68abd1dea7b6404d8f7d7b9f7fbd332d.1716698936847.pdf?Expires=1716699536&OSSAccessKeyId=TestID&Signature=HfwPUZo4pR6DatSDym0zFKVh9Wg%3D
	Url string
}
