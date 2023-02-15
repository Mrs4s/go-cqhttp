package onebot

// Self 机器人自身标识
//
// https://12.onebot.dev/connect/data-protocol/basic-types/#_10
type Self struct {
	Platform string `json:"platform"`
	UserID   string `json:"user_id"`
}

// Request 动作请求是应用端为了主动向 OneBot 实现请求服务而发送的数据
//
// https://12.onebot.dev/connect/data-protocol/action-request/
type Request struct {
	Action string // 动作名称
	Params any    // 动作参数
	Echo   any    // 每次请求的唯一标识
}

// Response 动作响应是 OneBot 实现收到应用端的动作请求并处理完毕后，发回应用端的数据
//
// https://12.onebot.dev/connect/data-protocol/action-response/
type Response struct {
	Status  string `json:"status"`  // 执行状态，必须是 ok、failed 中的一个
	Code    int64  `json:"retcode"` // 返回码
	Data    any    `json:"data"`    // 响应数据
	Message string `json:"message"` // 错误信息
	Echo    any    `json:"echo"`    // 动作请求中的 echo 字段值
}

// Event 事件
//
// https://12.onebot.dev/connect/data-protocol/event/
type Event struct {
	ID         string
	Time       int64
	Type       string
	DetailType string
	SubType    string
	Self       *Self
}
