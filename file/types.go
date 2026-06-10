package file

// CategoryType 类目类型
const (
	CategoryTypeUnstructured = "UNSTRUCTURED" // 非结构化数据，用于构建知识库
	CategoryTypeSessionFile  = "SESSION_FILE" // 会话文件，用于智能体应用会话交互
)

// 文件状态常量
const (
	StatusInit              = "INIT"                  // 待解析
	StatusInParseQueue      = "IN_PARSE_QUEUE"        // 解析队列排队中
	StatusParsing           = "PARSING"               // 解析中
	StatusParseSuccess      = "PARSE_SUCCESS"         // 解析完成（UNSTRUCTURED 终态）
	StatusParseFailed       = "PARSE_FAILED"          // 解析失败
	StatusSafeChecking      = "SAFE_CHECKING"         // 安全检测中
	StatusSafeCheckFailed   = "SAFE_CHECK_FAILED"     // 安全检测失败
	StatusIndexBuilding     = "INDEX_BUILDING"        // 索引构建中
	StatusIndexBuildSuccess = "INDEX_BUILD_SUCCESS"   // 索引构建成功
	StatusIndexBuildFailed  = "INDEX_BUILDING_FAILED" // 索引构建失败
	StatusIndexDeleted      = "INDEX_DELETED"         // 文件索引已删除
	StatusFileReady         = "FILE_IS_READY"         // 文件准备完毕（SESSION_FILE 终态）
	StatusFileExpired       = "FILE_EXPIRED"          // 文件过期
)

// Parser 解析器类型
const (
	ParserAutoSelect     = "AUTO_SELECT"         // 自动选择解析器
	ParserDocmind        = "DOCMIND"             // 智能文档解析
	ParserDocmindDigital = "DOCMIND_DIGITAL"     // 电子文档解析
	ParserDocmindLLM     = "DOCMIND_LLM_VERSION" // 大模型文档解析
	ParserQwenVL         = "DASH_QWEN_VL_PARSER" // Qwen VL 解析
)

// --- ApplyFileUploadLease 申请文件上传租约 ---

// ApplyLeaseRequest 申请上传租约请求
type ApplyLeaseRequest struct {
	WorkspaceId  string // 业务空间 ID
	CategoryId   string // 类目 ID，SESSION_FILE 时传 "default"
	FileName     string // 文件名（含扩展名），4-128 字符
	Md5          string // 文件 MD5
	SizeInBytes  string // 文件大小（字节），1B-100M
	CategoryType string // UNSTRUCTURED | SESSION_FILE
}

// ApplyLeaseResponse 申请上传租约响应
type ApplyLeaseResponse struct {
	Code      string          `json:"Code"`
	Data      *ApplyLeaseData `json:"Data"`
	Message   string          `json:"Message"`
	RequestId string          `json:"RequestId"`
	Status    string          `json:"Status"`
	Success   bool            `json:"Success"`
}

// ApplyLeaseData 租约数据
type ApplyLeaseData struct {
	FileUploadLeaseId string       `json:"FileUploadLeaseId"` // 租约唯一 ID
	Param             *UploadParam `json:"Param"`             // 上传参数
	Type              string       `json:"Type"`              // OSS.PreSignedURL | HTTP
}

// UploadParam 文件上传 HTTP 请求参数
type UploadParam struct {
	Headers map[string]string `json:"Headers"` // 上传请求头（含 X-bailian-extra、Content-Type）
	Method  string            `json:"Method"`  // PUT | POST
	Url     string            `json:"Url"`     // 预签名上传 URL
}

// --- AddFile 添加文件 ---

// AddFileRequest 添加文件请求
type AddFileRequest struct {
	WorkspaceId  string // 业务空间 ID
	LeaseId      string // 上传租约 ID（ApplyLeaseResponse.Data.FileUploadLeaseId）
	Parser       string // 解析器类型，默认 AUTO_SELECT
	CategoryId   string // 类目 ID
	CategoryType string // UNSTRUCTURED | SESSION_FILE
}

// AddFileResponse 添加文件响应
type AddFileResponse struct {
	Code      string       `json:"Code"`
	Data      *AddFileData `json:"Data"`
	Message   string       `json:"Message"`
	RequestId string       `json:"RequestId"`
	Status    string       `json:"Status"`
	Success   bool         `json:"Success"`
}

// AddFileData 文件注册数据
type AddFileData struct {
	FileId string `json:"FileId"` // 文件 ID，后续操作的核心标识
	Parser string `json:"Parser"` // 实际使用的解析器
}

// --- DescribeFile 查询文件状态 ---

// DescribeFileResponse 查询文件状态响应
type DescribeFileResponse struct {
	Code      string            `json:"Code"`
	Data      *DescribeFileData `json:"Data"`
	Message   string            `json:"Message"`
	RequestId string            `json:"RequestId"`
	Status    string            `json:"Status"`
	Success   bool              `json:"Success"`
}

// DescribeFileData 文件详情数据
type DescribeFileData struct {
	CategoryId  string   `json:"CategoryId"`
	CreateTime  string   `json:"CreateTime"`
	FileId      string   `json:"FileId"`
	FileName    string   `json:"FileName"`
	FileType    string   `json:"FileType"` // pdf, docx, doc, txt, md, ...
	Parser      string   `json:"Parser"`
	SizeInBytes int64    `json:"SizeInBytes"`
	Status      string   `json:"Status"` // 文件状态（见常量）
	Tags        []string `json:"Tags"`
}
