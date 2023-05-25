package rmms

import (
	"encoding/json"
	"log"
	"mms/config"
	"mms/response"
	"runtime"
	"time"
)

// 接收到的指令类型
var (
	CmdConn    string = "conn"
	CmdStart   string = "start"
	CmdStop    string = "stop"
	CmdDisconn string = "disconn"
)

// 向前端发送的状态码
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
	// 捕获错误，防止宕机
	defer func() {
		// 发生宕机时，获取panic传递的上下文并打印
		err := recover()
		if err != nil {
			switch err.(type) {
			case runtime.Error: // 运行时错误
				log.Println("runtime error:", err)
			default: // 非运行时错误
				log.Println("error:", err)
			}
		}
	}()

	// string 为json格式
	log.Println("revicesd datas: ", string(cmd))
	var nScanType, scanMode, encoderFrequency int
	var wheelCircumference float64
	var projectName = "hanni"
	var connectCmd config.ConnectCMD
	var startCmd config.StartCMD
	var replyTopic = r.config.StompTopic.CmdReply
	var statusTopic = r.config.StompTopic.StatusPush

	// 解析接收到的数据
	err := json.Unmarshal(cmd, &connectCmd)
	if err != nil {
		log.Println(err)
		r.Ws.Pubscribe(replyTopic, response.JsonUnmarshalError.MarshalToBytes(connectCmd.Seq))
		return
	}

	if connectCmd.Cmd == CmdConn {
		err := json.Unmarshal(cmd, &connectCmd)
		if err != nil {
			log.Println(err)
			r.Ws.Pubscribe(replyTopic, response.JsonUnmarshalError.MarshalToBytes(connectCmd.Seq))
			return
		}
		nScanType = connectCmd.Payload.DeviceInfo.Lidar01.Property.Lidarparameter.NScanType
		scanMode = connectCmd.Payload.DeviceInfo.Lidar01.Property.Lidarparameter.ScanMode
		encoderFrequency = connectCmd.Payload.DeviceInfo.Lidar01.Property.Lidarparameter.EncoderFrequency
		wheelCircumference = connectCmd.Payload.DeviceInfo.Lidar01.Property.Lidarparameter.WheelCircumference
		r.Param.Seq = connectCmd.Seq
		r.Param.ProjectPath = connectCmd.Payload.ProjectInfo.Path
	} else if connectCmd.Cmd == CmdStart {
		err := json.Unmarshal(cmd, &startCmd)
		if err != nil {
			log.Println(err)
			r.Ws.Pubscribe(replyTopic, response.JsonUnmarshalError.MarshalToBytes(connectCmd.Seq))
			return
		}
		r.Param.Seq = connectCmd.Seq
		r.Param.TaskID = connectCmd.Payload.TaskID
	} else if connectCmd.Cmd == CmdStop {
		err := json.Unmarshal(cmd, &startCmd)
		if err != nil {
			log.Println(err)
			r.Ws.Pubscribe(replyTopic, response.JsonUnmarshalError.MarshalToBytes(connectCmd.Seq))
			return
		}
		r.Param.Seq = connectCmd.Seq
		r.Param.TaskID = connectCmd.Payload.TaskID
	} else if connectCmd.Cmd == CmdDisconn {
		err := json.Unmarshal(cmd, &startCmd)
		if err != nil {
			log.Println(err)
			r.Ws.Pubscribe(replyTopic, response.JsonUnmarshalError.MarshalToBytes(connectCmd.Seq))
			return
		}
		r.Param.Seq = connectCmd.Seq
		r.Param.ModuleName = connectCmd.ModuleName
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
		// 设置状态为stop,并发送回复
		r.Param.Status = RmmsDisconn
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
	diseaseTopic := r.config.StompTopic.DiseasePush
	for {
		if r.Param.Status == RmmsStart {
			r.Ws.Pubscribe(topic, r.GenDataResponse())
			r.Ws.Pubscribe(diseaseTopic, r.GenDiseaseRequestData())
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
			},
		},
	}
	data, err := json.Marshal(dataResponse)
	if err != nil {
		log.Println(err)
		return nil
	}

	log.Printf("发送的数据: %+v \n", dataResponse)
	return data
}

// 生成病害程序发送图片识别数据内容
func (r *RmmsClient) GenDiseaseRequestData() (data []byte) {
	var GrayImage, DepthImage, _ = r.QueryGrayDepthImage()
	requestData := map[string]interface{}{
		"seq":        r.Param.Seq,
		"moduleName": "3DLidar",
		"cmd":        "disease_seg",
		"data": map[string]interface{}{
			"project_path": r.Param.ProjectPath,
			"taskID":       r.Param.TaskID,
			"devicesvalue": map[string]interface{}{
				"GrayImage":  GrayImage,
				"DpethImage": DepthImage,
			},
		},
	}
	data, err := json.Marshal(requestData)
	if err != nil {
		log.Println(err)
		return nil
	}

	log.Printf("发送的数据: %+v \n", requestData)
	return data
}
