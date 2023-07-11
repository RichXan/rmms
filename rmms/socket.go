package rmms

import (
	"encoding/json"
	"fmt"
	"log"
	"mms/config"
	// "strings"
	"runtime"
	"mms/response"
	// "runtime"
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
				buf := make([]byte, 1<<16)
				runtime.Stack(buf, true)
				fmt.Println("buf", string(buf))
				log.Println("runtime error:", err)
			default: // 非运行时错误
				buf := make([]byte, 1<<16)
				runtime.Stack(buf, true)
				fmt.Println("buf", string(buf))
				log.Println("error:", err)
			}
		}
	}()

	// string 为json格式
	log.Println("start #################################")
	log.Println("revicesd datas: ", string(cmd))
	log.Printf("r.Param: %+v\n", r.Param)
	log.Println("end #################################")
	var nScanType, scanMode, encoderFrequency int
	var wheelCircumference float64
	var projectName = "hanni"
	var connectCmd config.ConnectCMD
	var startCmd config.StartCMD
	var replyTopic = r.config.StompTopic.CmdReply
	var statusTopic = r.config.StompTopic.StatusPush
	seq := int(r.Param.Seq)
	r.Param.Seq = seq
	// int64 to int32

	// 解析接收到的数据
	err := json.Unmarshal(cmd, &connectCmd)
	if err != nil {
		log.Println(err)
		r.Ws.Pubscribe(replyTopic, response.JsonUnmarshalError.MarshalToStatusBytes(seq, 0))
		return
	}

	if connectCmd.Cmd == CmdConn {
		err := json.Unmarshal(cmd, &connectCmd)
		if err != nil {
			log.Println(err)
			r.Ws.Pubscribe(replyTopic, response.JsonUnmarshalError.MarshalToStatusBytes(seq, 0))
			return
		}
		nScanType = connectCmd.Payload.DeviceInfo.Lidar01.Property.Lidarparameter.NScanType
		scanMode = connectCmd.Payload.DeviceInfo.Lidar01.Property.Lidarparameter.ScanMode
		encoderFrequency = connectCmd.Payload.DeviceInfo.Lidar01.Property.Lidarparameter.EncoderFrequency
		wheelCircumference = connectCmd.Payload.DeviceInfo.Lidar01.Property.Lidarparameter.WheelCircumference
		// 项目保存路径
		r.Param.ProjectPath = connectCmd.Payload.ProjectInfo.Path
	} else if connectCmd.Cmd == CmdStart {
		err := json.Unmarshal(cmd, &startCmd)
		if err != nil {
			log.Println(err)
			r.Ws.Pubscribe(replyTopic, response.JsonUnmarshalError.MarshalToStatusBytes(seq, 0))
			return
		}
		taskID := connectCmd.Payload.TaskID
		if taskID != "" {
			r.Param.TaskID = taskID
		}
	} else if connectCmd.Cmd == CmdStop {
		err := json.Unmarshal(cmd, &startCmd)
		if err != nil {
			log.Println(err)
			r.Ws.Pubscribe(replyTopic, response.JsonUnmarshalError.MarshalToStatusBytes(seq, 0))
			return
		}
		taskID := connectCmd.Payload.TaskID
		if taskID != "" {
			r.Param.TaskID = taskID
		}
	} else if connectCmd.Cmd == CmdDisconn {
		err := json.Unmarshal(cmd, &startCmd)
		if err != nil {
			log.Println(err)
			r.Ws.Pubscribe(replyTopic, response.JsonUnmarshalError.MarshalToStatusBytes(seq, 0))
			return
		}
		r.Param.ModuleName = connectCmd.ModuleName
	}

	switch connectCmd.Cmd {
	case CmdConn:
		// 判断rmms的状态是否为disconn
		if r.Param.Status != RmmsDisconn {
			log.Println("当前的状态为："+ r.Param.Status + "不允许执行连接操作")
			r.Ws.Pubscribe(replyTopic, response.StatusConnError.MarshalToStatusBytes(seq, 0))
			return
		}

		// 连接
		if err := r.Action2_Connect(nScanType, scanMode, encoderFrequency, wheelCircumference); err != nil {
			r.Ws.Pubscribe(replyTopic, response.StartServerError.MarshalToStatusBytes(seq, 0))
			return
		}

		// 新建工程
		if err := r.Action3_NewProject(projectName); err != nil {
			log.Println(err)
			r.Ws.Pubscribe(replyTopic, err.MarshalToStatusBytes(seq, 0))
			return
		}

		daqStatus, _ := r.queryDAQCollectStatus()
		scannerStatus, _ := queryScannerCollectStatus()

		// 判断是否需要启动扫描采集服务程序
		if daqStatus != "1" || scannerStatus != "1" {
			// 开始测站扫描
			if err := r.Action4_StartStation(); err != nil {
				log.Println(err)
				r.Ws.Pubscribe(replyTopic, err.MarshalToStatusBytes(seq, 0))
				return
			}
		}

		// 设置状态为conn,并发送回复
		r.Param.Status = RmmsConn
		r.Ws.Pubscribe(statusTopic, r.GenStatusResponse(Normal))
		r.Ws.Pubscribe(replyTopic, response.Success.MarshalToStatusBytes(seq, 0))
	case CmdStart:
		// 判断rmms的状态是否为conn,stop
		if !(r.Param.Status == RmmsConn || r.Param.Status == RmmsStop) {
			log.Println("当前的状态为："+ r.Param.Status + "不允许执行启动操作")
			r.Ws.Pubscribe(replyTopic, response.StatusStartError.MarshalToStatusBytes(seq, 0))
			return
		}

		// 设置状态为start,并发送回复
		r.Param.Status = RmmsStart
		r.Ws.Pubscribe(statusTopic, r.GenStatusResponse(Running))
		go func() {
			r.DataLoop()
		}()
		r.Ws.Pubscribe(replyTopic, response.Success.MarshalToStatusBytes(seq, 0))
	case CmdStop:
		// 判断rmms的状态是否为start
		if r.Param.Status != RmmsStart {
			log.Println("当前的状态为："+ r.Param.Status + "不允许执行停止操作")
			r.Ws.Pubscribe(replyTopic, response.StatusStopError.MarshalToStatusBytes(seq, 0))
			return
		}

		// 设置状态为stop,并发送回复
		r.Param.Status = RmmsStop
		r.Ws.Pubscribe(statusTopic, r.GenStatusResponse(Stop))
		r.Ws.Pubscribe(replyTopic, response.Success.MarshalToStatusBytes(seq, 0))
	case CmdDisconn:
		// 判断rmms的状态是否为conn或者stop
		if r.Param.Status != RmmsConn && r.Param.Status != RmmsStop {
			log.Println("当前的状态为："+ r.Param.Status + "不允许执行断开连接操作")
			r.Ws.Pubscribe(replyTopic, response.StatusDisconnError.MarshalToStatusBytes(seq, 0))
			return
		}

		// 停止测站扫描
		if err := r.Action5_StopStation(); err != nil {
			log.Println(err)
			r.Ws.Pubscribe(replyTopic, err.MarshalToStatusBytes(seq, 0))
			return
		}

		// 保存工程
		if err := r.Action6_SaveProject(); err != nil {
			log.Println(err)
			r.Ws.Pubscribe(replyTopic, err.MarshalToStatusBytes(seq, 0))
			return
		}

		// 断开连接
		if err := r.Action7_CloseDevice(); err != nil {
			log.Println(err)
			r.Ws.Pubscribe(replyTopic, err.MarshalToStatusBytes(seq, 0))
			return
		}

		fmt.Printf("startCmd: %+v\n", startCmd)
		fmt.Printf("r.Param: %+v\n", r.Param)
		// 设置状态为stop,并发送回复
		r.Param.Status = RmmsDisconn
		r.Ws.Pubscribe(statusTopic, r.GenStatusResponse(Stop))
	default:
		log.Println("cmd错误")
		r.Ws.Pubscribe(replyTopic, response.CmdError.MarshalToStatusBytes(seq, 0))
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

			// 若有生成的图片数据，则发送
			disease_data := r.GenDiseaseRequestData()
			if disease_data != nil {
				r.Ws.Pubscribe(diseaseTopic, disease_data)
			}
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
	if GrayImage == "NoImg" || DepthImage == "NoImg"{
		return nil
	}

	if GrayImage == r.Param.LastGrayImage || DepthImage == r.Param.LastDepthImage {
		return nil
	}
	r.Param.LastGrayImage = GrayImage
	r.Param.LastDepthImage = DepthImage

	// 修改GrayImage和DepthImage的路径
	// GrayImage = strings.Replace(GrayImage, "\\192.168.1.92\\data", "Z:\\", 1)
	// DepthImage = strings.Replace(DepthImage, "\\192.168.1.92\\data", "Z:\\", 1)
	// GrayImage = strings.Replace(GrayImage, "\\", "\\\\", -1)
	// DepthImage = strings.Replace(DepthImage, "\\", "\\\\", -1)

	requestData := map[string]interface{}{
		"seq":        r.Param.Seq,
		"module_name": "3DLidar",
		"cmd":        "disease_seg",
		"data": map[string]interface{}{
			"project_path": r.Param.ProjectPath,
			"taskID":       r.Param.TaskID,
			"devicesvalue": map[string]interface{}{
				"GrayImage":  GrayImage,
				"DepthImage": DepthImage,
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
