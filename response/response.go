package response

import (
	"encoding/json"
	"fmt"
)

// 回复的json格式
type ReplyResponse struct {
	Seq  int    `json:"seq"`
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

var codes = map[int]string{}

// 创建接口标准返回错误
func NewResponse(code int, message string) *ReplyResponse {
	if _, ok := codes[code]; ok {
		panic(fmt.Sprintf("错误码 %d 已经存在，请更换一个", code))
	}
	codes[code] = message
	return &ReplyResponse{
		Seq:  0,
		Code: code,
		Msg:  message,
	}
}

func (r *ReplyResponse) SetSeq(seq int) {
	r.Seq = seq
}

// 序列化
func (r *ReplyResponse) MarshalToBytes(seq int) []byte {
	r.SetSeq(seq)
	msg, err := json.Marshal(r)
	if err != nil {
		JsonMarshalError.SetSeq(seq)
		msg, _ = json.Marshal(JsonMarshalError)
		return msg
	}
	return msg
}

var (
	Success = NewResponse(0, "成功")

	// 后端错误
	ModuleNameError = NewResponse(90001, "module_name错误，不为3DLidar")
	CmdError        = NewResponse(90002, "cmd错误，不为conn、start、stop、disconn")

	// 执行conn操作时的错误
	StatusConnError    = NewResponse(90010, "当前的状态不允许执行连接操作")
	StartServerError   = NewResponse(90011, "启动扫描采集服务程序失败")
	ConnectServerError = NewResponse(90012, "连接服务程序失败")
	NScanTypeError     = NewResponse(90013, "nScanType错误，不为0、1、2、3")

	// 执行start操作时的错误
	StatusStartError          = NewResponse(90020, "当前的状态不允许执行启动操作")
	NewProjectError           = NewResponse(90021, "新建工程失败")
	StartStationError         = NewResponse(90022, "开始测站扫描失败")
	LidarCollectingNewError   = NewResponse(90023, "激光雷达采集状态正在采集,无法新建工程")
	DAQIsCollectingNewError   = NewResponse(90024, "DAQ采集状态正在采集,无法新建工程")
	LidarCollectingStartError = NewResponse(90025, "激光雷达采集状态正在采集,无法开始测站")
	DAQIsCollectingStartError = NewResponse(90026, "DAQ采集状态未在采集,无法开始测站")
	StartScannerError         = NewResponse(90027, "启动扫描仪失败")

	// 执行stop操作时的错误
	StatusStopError  = NewResponse(90030, "当前的状态不允许执行停止操作")
	StopStationError = NewResponse(90031, "停止测站扫描失败")
	SaveProjectError = NewResponse(90032, "保存工程失败")

	// 执行disconnect操作时的错误
	StatusDisconnError = NewResponse(90040, "当前的状态不允许执行断开操作")
	CloseDeviceError = NewResponse(90041, "关闭设备失败")
	// 服务器错误
	JsonMarshalError   = NewResponse(95001, "json序列化错误")
	JsonUnmarshalError = NewResponse(95002, "json反序列化错误")
)
