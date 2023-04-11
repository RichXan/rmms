package rmms

import (
	"encoding/json"
	"log"
	"mms/config"
	"mms/response"
	"time"
)

var (
	CmdConn    string = "conn"
	CmdStart   string = "start"
	CmdStop    string = "stop"
	CmdDisconn string = "disconn"
)

const (
	Normal      int = 0
	Running     int = 1
	Stop        int = 2
	ConnectFail int = 3
	StartFail   int = 4
	OtherError  int = 99
)

// 对接收到的cmd指令，进行操作
func (r *RmmsClient) ActionCmdSub(cmd []byte) {
	var connectCmd config.ConnectCmd
	var replyTopic = r.config.StompTopic.CmdReply
	var statusTopic = r.config.StompTopic.StatusPush

	// 解析接收到的数据
	err := json.Unmarshal(cmd, &connectCmd)
	if err != nil {
		log.Println(err)
		r.Ws.Pubscribe(replyTopic, response.JsonUnmarshalError.MarshalToBytes(connectCmd.Seq))
		return
	}

	// 获取参数
	var nScanType = connectCmd.Payload.DeviceSysParams.Lidarparameter.NScanType
	var scanMode = connectCmd.Payload.DeviceSysParams.Lidarparameter.ScanMode
	var encoderFrequency = connectCmd.Payload.DeviceSysParams.Lidarparameter.EncoderFrequency
	var wheelCircumference = connectCmd.Payload.DeviceSysParams.Lidarparameter.WheelCircumference
	var projectName = connectCmd.Payload.ProjectName
	r.Param.Seq = connectCmd.Seq
	r.Param.ProjectPath = connectCmd.Payload.ProjectInfo.Path
	r.Param.TaskID = connectCmd.Payload.TaskID

	// 判断module_name是否为3DLidar
	if connectCmd.ModuleName != "3DLidar" {
		log.Println("module_name错误，不为3DLidar")
		r.Ws.Pubscribe(replyTopic, response.ModuleNameError.MarshalToBytes(connectCmd.Seq))
		return
	}

	// 项目保存路径
	r.Param.ProjectPath = connectCmd.Payload.ProjectInfo.Path

	switch connectCmd.Cmd {
	case CmdConn:
		// 判断rmms的状态是否为disconn
		if r.Param.Status != RmmsDisconn {
			r.Ws.Pubscribe(replyTopic, response.StatusConnError.MarshalToBytes(connectCmd.Seq))
			return
		}

		// 启动服务
		if err := r.Action1_StartServer(); err != nil {
			r.Ws.Pubscribe(replyTopic, err.MarshalToBytes(connectCmd.Seq))
			return
		}

		// 连接
		if err := r.Action2_Connect(nScanType, scanMode, encoderFrequency, wheelCircumference); err != nil {
			r.Ws.Pubscribe(replyTopic, response.StartServerError.MarshalToBytes(connectCmd.Seq))
			return
		}

		// 设置状态为conn,并发送回复
		r.Param.Status = RmmsConn
		r.Ws.Pubscribe(statusTopic, r.GenStatusResponse(Normal))
		r.Ws.Pubscribe(replyTopic, response.Success.MarshalToBytes(connectCmd.Seq))
	case CmdStart:
		// 判断rmms的状态是否为conn
		if r.Param.Status != RmmsConn {
			r.Ws.Pubscribe(replyTopic, response.StatusStartError.MarshalToBytes(connectCmd.Seq))
			return
		}

		// 新建工程
		if err := r.Action3_NewProject(projectName); err != nil {
			log.Println(err)
			r.Ws.Pubscribe(replyTopic, err.MarshalToBytes(connectCmd.Seq))
			return
		}

		// 开始测站扫描
		if err := r.Action4_StartStation(); err != nil {
			log.Println(err)
			r.Ws.Pubscribe(replyTopic, err.MarshalToBytes(connectCmd.Seq))
			return
		}

		// 设置状态为start,并发送回复
		r.Param.Status = RmmsStart
		r.Ws.Pubscribe(statusTopic, r.GenStatusResponse(Running))
		go func() {
			r.DataLoop()
		}()
		r.Ws.Pubscribe(replyTopic, response.Success.MarshalToBytes(connectCmd.Seq))
	case CmdStop:
		// 判断rmms的状态是否为start
		if r.Param.Status != RmmsStart {
			r.Ws.Pubscribe(replyTopic, response.StatusStopError.MarshalToBytes(connectCmd.Seq))
			return
		}

		// 停止测站扫描
		if err := r.Action5_StopStation(); err != nil {
			log.Println(err)
			r.Ws.Pubscribe(replyTopic, err.MarshalToBytes(connectCmd.Seq))
			return
		}

		// 保存工程
		if err := r.Action6_SaveProject(); err != nil {
			log.Println(err)
			r.Ws.Pubscribe(replyTopic, err.MarshalToBytes(connectCmd.Seq))
			return
		}

		// 设置状态为stop,并发送回复
		r.Param.Status = RmmsStop
		r.Ws.Pubscribe(statusTopic, r.GenStatusResponse(Stop))
		r.Ws.Pubscribe(replyTopic, response.Success.MarshalToBytes(connectCmd.Seq))
	case CmdDisconn:
		// 判断rmms的状态是否为stop
		if r.Param.Status != RmmsStop {
			r.Ws.Pubscribe(replyTopic, response.StatusDisconnError.MarshalToBytes(connectCmd.Seq))
			return
		}

		// 断开连接
		if err := r.Action7_CloseDevice(); err != nil {
			log.Println(err)
			r.Ws.Pubscribe(replyTopic, err.MarshalToBytes(connectCmd.Seq))
			return
		}
		r.Ws.Pubscribe(statusTopic, r.GenStatusResponse(Stop))
	default:
		log.Println("cmd错误")
		r.Ws.Pubscribe(replyTopic, response.CmdError.MarshalToBytes(connectCmd.Seq))
	}
}

// 通用的监听函数
func (r *RmmsClient) SubListen(data []byte) {
	var cmd map[string]interface{}

	// 解析接收到的数据
	err := json.Unmarshal(data, &cmd)
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("接收到的数据: %+v \r", cmd)
}

// 生成状态回复
func (r *RmmsClient) GenStatusResponse(status int) (data []byte) {
	var statusResponse = response.StatusResponse{
		Seq:        r.Param.Seq,
		ModuleName: "3DLidar",
		State: response.State{
			Status: status,
		},
	}
	data, err := json.Marshal(statusResponse)
	if err != nil {
		log.Println(err)
		return nil
	}
	return data
}

// 协程函数，用于定时发送数据，输入一个信号，用于控制协程函数的开关
func (r *RmmsClient) DataLoop() {
	// 定时发送数据
	topic := r.config.StompTopic.DataPush
	for {
		if r.Param.Status == RmmsStart || r.Param.Status == RmmsStop {
			r.Ws.Pubscribe(topic, r.GenDataResponse())
			time.Sleep(1 * time.Second)
		} else {
			return
		}
	}

}

// 生成数据回复
func (r *RmmsClient) GenDataResponse() (data []byte) {
	var DAQCollectStatus, _ = r.QueryDAQCollectStatus()
	var DAQFileSize, _ = r.QueryDAQFileSize()
	var DAQCollectTime, _ = r.QueryDAQCollectTime()
	var ScannerCollectStatus, _ = r.QueryScannerCollectStatus()
	var FreeSpace, _ = r.QueryFreeSpace()
	var LidarFileSizeMB, _ = r.QueryLidarFileSizeMB()
	var GrayImage, DepthImage, _ = r.QueryGrayDepthImage()

	var dataResponse = response.DataResponse{
		Seq:        r.Param.Seq,
		ModuleName: "3DLidar",
		Data: response.Data{
			DevicesValue: response.DevicesValue{
				DAQCollectStatus:     DAQCollectStatus,
				DAQFileSize:          DAQFileSize,
				DAQCollectTime:       DAQCollectTime,
				ScannerCollectStatus: ScannerCollectStatus,
				FreeSpace:            FreeSpace,
				LidarFileSizeMB:      LidarFileSizeMB,
				GrayImage:            GrayImage,
				DepthImage:           DepthImage,
			},
		},
	}
	data, err := json.Marshal(dataResponse)
	if err != nil {
		log.Println(err)
		return nil
	}
	return data
}
