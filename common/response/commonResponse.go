package response

type ErrorResponse struct {
	// 错误码
	Code int `json:"code"`
	// 错误信息
	Message string `json:"message"`
	// 错误状态
	Status string `json:"status"`
}

type Response struct {
	// 操作成功结果。成功时有此结构
	Result interface{} `json:"result,omitempty"`
	// 操作失败结果。失败时有此结构
	Error *ErrorResponse `json:"error,omitempty"`
	// 请求traceId
	// required: true
	RequestId string `json:"requestId"`
}

type CommonResponse struct {
	// 操作是否成功 [true/false]
	// required: true
	Success bool `json:"success"`
}
