package bailian

import (
	"context"
	"llm-util/ai/app/utils"
	"net/http"
)

// ChatCompletionRequest 百炼应用调用body结构
type ChatCompletionRequest struct {
	Input      *RequestInput      `json:"input"`
	Parameters *RequestParameters `json:"parameters,omitempty"`
}

func NewSimpleQuestion(question string) ChatCompletionRequest {
	return ChatCompletionRequest{
		Input: &RequestInput{
			Prompt: question,
		},
	}
}

type ChatCompletionResponse struct {
	AppResponse
	HttpHeader
}

// AppRequest 阿里百炼应用调用请求结构
// https://help.aliyun.com/zh/model-studio/developer-reference/call-application-through-api/?spm=a2c4g.11186623.help-menu-2400256.d_3_6.2f534823OTqNnO&scm=20140722.H_2846133._.OR_help-T_cn~zh-V_1
type AppRequest struct {
	// 应用的标识。
	// 在我的应用页面的应用卡片上可以获取应用ID。
	AppID string `json:"app_id"`
	//
	Input RequestInput `json:"input"`
	// 业务空间标识。
	// 调用子业务空间的应用时需传递workspace标识，调用默认业务空间的应用时无需传递workspace。
	// 在子业务空间里，点击我的应用页面的应用卡片上的调用，即可在应用API代码中获取子业务空间的workspace标识，具体请参考获取Workspace ID。
	// > 通过HTTP调用时，请指定Header中的 X-DashScope-WorkSpace。
	Workspace string `json:"workspace,omitempty"`
	// 是否流式输出回复。
	// 参数值：
	// false（默认值）：模型生成完所有内容后一次性返回结果。
	// true：边生成边输出，即每生成一部分内容就立即输出一个片段（chunk）。
	// 通过HTTP调用时，请指定Header中的 X-DashScope-SSE 为 enable。
	Stream bool `json:"stream"`
	//
	Parameters RequestParameters `json:"parameters,omitempty"`
}

type RequestInput struct {
	// 输入当前期望应用执行的指令prompt，用来指导应用生成回复。
	// 暂不支持传入文件。如果应用使用的是Qwen-Long模型，应用调用方法与其他模型一致。
	// 当您通过传入messages自己管理对话历史时，则无需传递prompt。
	Prompt string `json:"prompt"`
	// 历史对话的唯一标识。
	// 传入session_id时，请求将自动携带云端存储的对话历史。具体请参考多轮对话。
	// > 传入session_id时，prompt为必传。
	// > 若同时传入session_id和messages，则优先使用传入的messages。
	// > 目前仅智能体应用和对话型工作流应用支持多轮对话。
	// > 通过HTTP调用时，请将 session_id 放入 input 对象中。
	SessionID string `json:"session_id,omitempty"`
	// 由历史对话组成的消息列表。
	// 若您需要自行管理对话历史以实现多轮对话，可以通过 messages 传递历史对话信息。具体请参考多轮对话。
	// > 传入messages时，无需传入prompt，若二者同时传入，则prompt会被追加到messages列表的最后，作为补充信息。
	// > 若同时传入session_id和messages，则优先使用传入的messages。
	// > 目前仅智能体应用和对话型工作流应用支持多轮对话。
	// > 通过HTTP调用时，请将messages 放入 input 对象中。
	Messages []RequestMessage `json:"messages,omitempty"`
	// 工作流应用和智能体编排应用需要传递自定义参数时，通过该字段进行传递，示例如：biz_params = {"city": "杭州"}。
	// 智能体应用需要进行自定义插件的参数传递和用户级鉴权时，也通过该字段进行传递。
	// 通过HTTP调用时，请将 biz_params 放入 input 对象中。
	//  - NOTE: 百炼对于智能体应用以及工作流应用放入的结构不一致，智能体应用请将自定义参数放入user_prompt_params内
	BizParams any `json:"biz_params,omitempty"`
	// 长期记忆体ID。
	// 在百炼控制台应用中打开长期记忆开关并发布应用，通过指定 memory_id 调用应用时，系统依据用户偏好信息自动构建和保存长期记忆。后续使用同一 memory_id 调用时，系统会恢复这些长期记忆，并与最新的用户消息合并提供给模型处理。
	// memory_id的创建方法请参见CreateMemory。详细调用方法请参见应用调用长期记忆。
	// > 通过HTTP调用时，请将 memory_id 放入input 对象中。
	// > 目前仅智能体应用支持长期记忆。
	MemoryID string `json:"memory_id,omitempty"`
	// 图片链接列表。用于传递图片链接。
	// 支持以下两种使用场景：
	// - 图片检索：在智能体应用中，根据上传的图片链接，检索包含图片链接的结构化知识库。
	// - 图片理解：在通义千问VL模型的智能体应用中，还可以直接提问图片内容。
	// > 可以是多个，每个图片链接之间通过英文逗号分隔。
	// > 通过HTTP调用时，请将 image_list 放入 input 对象中。
	ImageList []string `json:"image_list,omitempty"`
}

type RequestParameters struct {
	// 在流式输出模式下是否开启增量输出。
	// 通过HTTP调用时，请将incremental_output放入parameters对象中。
	IncrementalOutput bool `json:"incremental_output"`
	// 工作流应用的流式输出模式。具体使用方法请参考流式输出。
	// 参数值及使用方法如下：
	// `full_thoughts`（默认值）：
	// 描述：所有节点的流式结果在thoughts字段中输出。
	// 要求：同时必须要设置has_thoughts为True。
	// `agent_format`：
	// - 描述：使用与智能体应用相同的输出模式。
	// - 效果：在控制台应用中，可选择打开指定节点的结果返回开关，则该节点的流式结果将在output的text字段中输出。
	// - 场景：适合只关心中间指定节点输出的场景。
	// > 结果返回开关当前仅支持文本转换节点、大模型节点以及结束节点（结束节点默认打开）。
	// > 在并行节点中同时开启结果返回开关，会导致内容混杂。因此，开启开关的节点需要有明确的输出先后顺序。
	// 通过HTTP调用时，请将flow_stream_mode放入parameters对象中。
	FlowStreamMode string `json:"flow_stream_mode,omitempty"`
	// 是否输出插件调用或知识检索的过程信息，默认值False。调用时设置此参数为True，则可在thoughts字段中返回过程信息。
	// 调用智能体应用实现Prompt样例库时，需要将此参数设置为True。
	// > 通过HTTP调用时，请将 has_thoughts 放入 parameters 对象中。
	HasThoughts bool `json:"has_thoughts,omitempty"`
	// 用于配置与RAG相关的参数。包括但不限于对指定的知识库或文档进行检索。详细用法和规则请参见检索知识库。
	// 通过HTTP调用时，请将 rag_options 放入 parameters 对象中。
	RagOptions []RequestInputRagOptions `json:"rag_options,omitempty"`
}

type RequestInputRagOptions struct {
	// 知识库ID，传入该参数将对指定知识库内所有文档进行检索。使用步骤如下：
	// 在百炼控制台智能体应用中打开知识库检索增强开关；
	// > RAG应用过此步骤。
	// 两种方式配置知识库
	// 在应用内单击配置知识库，添加指定知识库，并发布应用。
	// 直接发布应用，调用时通过此参数传递知识库ID配置指定知识库。
	// API调用。具体用法请参考检索知识库。
	// 在知识索引页面可以获取知识库ID，也可以使用CreateIndex接口返回的Data.Id。
	// > 知识库ID上限5个，每个ID之间用英文逗号分隔，例如["知识库ID1", "知识库ID2"]。如果知识库ID传入多于5个，只生效前5个。
	PipelineIDs []string `json:"pipeline_id,omitempty"`
	// 非结构化文档ID，传入该参数将对指定非结构化文档进行检索。
	// > 传入文档ID时，还需要传入文档所属的知识库ID才会生效。
	// 在数据管理页面的文档列表中可以获取文档ID，也可以使用AddFile接口返回的文档ID。
	// > 文档ID上限100个，每个ID之间用英文逗号分隔，例如["文档ID1", "文档ID2"]。
	FileIDs []string `json:"file_id,omitempty"`
	// 非结构化文档的元数据，传入该参数将对具备该元数据的非结构化文档进行检索。
	// 在知识索引页面，进入某个知识库后可以查看非结构化文档的元数据（Meta信息）。在创建非结构化知识库时可以设置元数据。调用ListChunks接口可获取指定文档的所有文本切片的详细信息。
	// > 传入元数据时，还需要传入所属的知识库ID才会生效。
	MetaDataFilter interface{} `json:"meta_data_filter,omitempty"`
	// 非结构化文档的标签，传入该参数将对具备该标签的非结构化文档进行检索。
	// 在数据管理页面，可以查看非结构化文档的标签。也可以通过DescribeFile接口获取文档标签。
	// > 可以是多个tag，每个tag之间用英文逗号分隔，例如["标签1", "标签2"]。
	Tags []string `json:"tags,omitempty"`
	// 结构化文档的列名和值，键值对形式。
	// 传入该参数，将对结构化文档里符合条件的内容进行检索。
	// 调用ListChunks接口可获取所有文本切片的详细信息。
	// > 可以是多个，每对键值对之间用英文逗号分隔。
	StructuredFilter interface{} `json:"structured_filter,omitempty"`
	// 用于在智能体应用的当前请求中上传文件ID。
	// 基于这些文件，应用可临时扩充知识库、回答文件内容等。详细调用方法请参考文件交互。
	// > 文件ID以“file_session”开头。
	// 获取方式：根据API上传文档中的步骤 1、2 和 3 完成文件上传后获取。
	// https://help.aliyun.com/zh/model-studio/developer-reference/upload-files-by-calling-api?spm=a2c4g.11186623.0.0.54fc4823QedxG4
	// > 步骤中需注意必须设置CategoryType为SESSION_FILE和CategoryId为default，否则获取的ID无效。
	// > 每个文件ID之间用英文逗号分隔，例如["文件ID1", "文件ID2"]。 。
	// 支持上传的文件上限10个。支持上传本地的文档、图片、视频或音频，格式要求为：
	// 文档（单文件不超过100MB）：.doc,.docx,.wps,.ppt,.pptx,.xls,.xlsx,.md,.txt,.pdf；
	// 图片（单文件不超过20MB）：.png,.jpg,.jpeg,.bmp,.gif；
	// > 目前仅支持上传包含文字内容的本地图片。
	// 视频（单文件不超过512MB）：.mp4,.mkv,.avi,.mov,.wmv；
	// 音频（单文件不超过512MB）：.aac,.amr,.flac,.flv,.m4a,.mp3,.mpeg,.ogg,.opus,.wav,.webm,.wma；
	SessionFileIDs []string `json:"session_file_ids,omitempty"`
}

type RequestInputBizParams struct {
	// 表示自定义提示词变量参数信息。用于传递在提示词中插入配置的变量。
	UserPromptParams map[string]interface{} `json:"user_prompt_params"`
	// 表示自定义插件参数信息。
	UserDefinedParams RequestInputBizParamsUserDefinedParams `json:"user_defined_params"`
	// 表示自定义插件的用户级鉴权信息。
	UserDefinedTokens RequestInputBizParamsUserDefinedTokens `json:"user_defined_tokens"`
}

type RequestInputBizParamsUserDefinedParams struct {
	// 表示插件ID，your_plugin_code，依据具体的插件变化。
	PluginID string `json:"plugin_id"`
	// 对象最内侧包含的多个键值对。每个键值对表示用户自定义的待传递参数名及其指定值。
	PluginParams map[string]interface{} `json:"plugin_params"`
}

type RequestInputBizParamsUserDefinedTokens struct {
	// 插件 ID，可在插件卡片中获取。通过your_plugin_code字段传递。
	PluginID string `json:"plugin_id"`
	// 传递该插件需要的用户鉴权信息，如实际DASHSCOPE_API_KEY的值。
	UserToken string `json:"user_token"`
}

type RequestMessage struct {
	// 模型的目标或角色。如果设置系统消息，请放在messages列表的第一位。
	Content string `json:"content"`
	// 用户发送给模型的消息。
	Role string `json:"role"`
}

// AppResponse 阿里百炼应用调用返回结构
// https://help.aliyun.com/zh/model-studio/developer-reference/call-application-through-api/?spm=a2c4g.11186623.help-menu-2400256.d_3_6.2f534823OTqNnO&scm=20140722.H_2846133._.OR_help-T_cn~zh-V_1
type AppResponse struct {
	// 返回的状态码。
	// 200表示请求成功，否则表示请求失败。可以通过code获取错误码，通过message字段获取错误详细信息。
	StatusCode int `json:"status_code"`
	// 当前的请求ID。
	RequestID string `json:"request_id,omitempty"`
	// 表示错误码，调用成功时为空值。
	// 错误码详情请参见错误码。https://help.aliyun.com/zh/model-studio/developer-reference/error-code?spm=a2c4g.11186623.0.0.54fc4823QedxG4
	Code string `json:"code"`
	// 表示失败详细信息，成功忽略。
	Message string `json:"message,omitempty"`
	// 表示调用结果信息。
	Output ResponseOutput `json:"output,omitempty"`
	// 表示本次请求使用的数据信息。
	Usage ResponseUsage `json:"usage,omitempty"`
}

type ResponseOutput struct {
	// 模型生成的回复内容。
	Text string `json:"text,omitempty"`
	// 完成原因。
	// 正在生成时为null，生成结束时如果由于停止token导致则为stop。
	FinishReason string `json:"finish_reason,omitempty"`
	// 当前对话的唯一标识。
	// 在后续请求中传入，可携带历史对话记录。
	SessionID string `json:"session_id,omitempty"`
	// 模型的思考过程信息。
	// 调用时设置has_thoughts参数为True，则可在thoughts中查看插件调用或知识检索的过程信息
	Thoughts      []ResponseOutputThought      `json:"thoughts,omitempty"`
	DocReferences []ResponseOutputDocReference `json:"doc_references,omitempty"`
}

type ResponseOutputThought struct {
	// 模型的思考结果。
	// 如果您在智能体应用中选择了深度思考模型，并设置了has_thoughts参数为True，则调用时在此字段返回模型的思考过程。
	Thought string `json:"thought,omitempty"`
	// 模型的思考过程。
	// 如果您在工作流应用中选择了深度思考模型，并设置了has_thoughts参数为True，则调用时在此字段返回模型的思考过程。
	ReasoningContent string `json:"reasoning_content,omitempty"`
	// 大模型返回的执行步骤类型。如api，表示执行API插件。
	ActionType string `json:"action_type,omitempty"`
	// 模型调用返回的结果。
	Response any `json:"response,omitempty"`
	// 执行的action名称，如文档检索、API插件。
	ActionName string `json:"action_name,omitempty"`
	// 执行的步骤。
	Action string `json:"action,omitempty"`
	// 入参的流式结果。
	ActionInputStream string `json:"action_input_stream,omitempty"`
	// 插件的输入参数。
	ActionInput string `json:"action_input,omitempty"`
	// 检索或插件的返回结果。
	Observation string `json:"observation,omitempty"`
}

type ResponseOutputDocReference struct {
	// 模型引用的召回文档索引，如[1]。
	IndexID string `json:"index_id,omitempty"`
	// 模型引用的文本切片标题。
	Title string `json:"title,omitempty"`
	// 模型引用的文档ID。
	DocID string `json:"doc_id,omitempty"`
	// 模型引用的文档名。
	DocName string `json:"doc_name,omitempty"`
	// 模型引用的具体文本内容。
	Text string `json:"text,omitempty"`
	// 模型引用的图片URL列表。
	Images []string `json:"images,omitempty"`
	// 模型引用的文本切片的页码。
	PageNumber []int `json:"page_number,omitempty"`
}

type ResponseUsage struct {
	// 本次应用调用到的模型信息。
	Models []ResponseUsageModel `json:"models,omitempty"`
}

type ResponseUsageModel struct {
	// 本次应用调用到的模型 ID。
	ModelID string `json:"model_id,omitempty"`
	// 用户输入文本转换成Token后的长度。
	InputTokens int `json:"input_tokens,omitempty"`
	// 模型生成回复转换为Token后的长度。
	OutputTokens int `json:"output_tokens,omitempty"`
}

func (c *Client) CreateChatCompletion(ctx context.Context, request ChatCompletionRequest,
) (response ChatCompletionResponse, err error) {

	if request.Parameters == nil {
		request.Parameters = &RequestParameters{IncrementalOutput: false}
	}

	err = utils.Retry(
		ctx,
		utils.RetryPolicy{
			MaxAttempts:    c.config.RetryTimes,
			InitialBackoff: ErrorRetryBaseDelay,
			MaxBackoff:     ErrorRetryMaxDelay,
		},
		func() bool { return true },
		func() error {
			req, inErr := c.newRequest(
				ctx,
				http.MethodPost,
				c.fullURL(c.config.AppID),
				withBody(request),
			)
			if inErr != nil {
				return inErr
			}

			return c.sendRequest(req, &response)
		},
		nil,
		needRetryError,
	)

	return
}
