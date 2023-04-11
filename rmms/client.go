package rmms

import (
	"fmt"
	"log"
	"sync"
	"time"

	"mms/config"
	"mms/response"
	"mms/tcp"
	"mms/websocket"
)

// 设备的tcp服务器配置
const (
	//设备ip
	tcp_ip = "192.168.1.92"
	//总控端口
	tcp_port_rmms = 8000
	//惯导
	tcp_port_daq = 8500
	//同步
	tcp_port_sync = 8300
	//扫描仪
	tcp_port_scanner = 8400
	//GPS，唐源未用
	tcp_port_gps = 8600
)

type RmmsStatus int

const (
	RmmsDisconn RmmsStatus = iota
	RmmsConn
	RmmsStart
	RmmsStop
)

type RmmsParam struct {
	Status      RmmsStatus // 状态
	TaskID      string     // 任务ID
	ProjectPath string     // 项目路径
}

type RmmsClient struct {
	tc     *tcp.TcpClient
	mutex  sync.Mutex
	Ws     *websocket.StompWS
	Param  *RmmsParam
	config *config.GlobalConfig
}

// 创建一个rmms客户端
func NewRmmsClient(config *config.GlobalConfig) *RmmsClient {
	tc := tcp.NewTcpClient()
	ws := websocket.NewStompWs()
	ws.WebsocketInit(config.StompConfig)
	return &RmmsClient{
		tc:     tc,
		Ws:     ws,
		config: config,
		Param: &RmmsParam{
			Status: RmmsDisconn,
		},
	}
}

// 启动服务，通过总控启动扫描采集服务程序
func (r *RmmsClient) Action1_StartServer() *response.ReplyResponse {
	if err := r.action1StartServer(); err != nil {
		log.Println(err)
		return response.StartServerError
	}

	// 等待十秒
	time.Sleep(10 * time.Second)
	return nil
}

// 连接各服务
func (r *RmmsClient) Action2_Connect(nScanType, scanMode,
	encoderFrequency int, wheelCircumference float64) *response.ReplyResponse {
	if nScanType < 0 || nScanType > 3 {
		return response.NScanTypeError
	}

	// 测试各服务状态
	if err := r.actionTestAllServer(); err != nil {
		log.Println(err)
		response.ConnectServerError.Msg = err.Error()
		return response.ConnectServerError
	}

	//2.同步，打开gps
	if err := r.action2OpenGPS(); err != nil {
		log.Println(err)
		response.ConnectServerError.Msg = err.Error()
		return response.ConnectServerError
	}

	//3.同步，设置无gps模式
	if err := r.action2NoGPSMode(); err != nil {
		log.Println(err)
		response.ConnectServerError.Msg = err.Error()
		return response.ConnectServerError
	}

	//4.扫描：设置扫描频率
	if err := r.action2ScanType(nScanType); err != nil {
		log.Println(err)
		response.ConnectServerError.Msg = err.Error()
		return response.ConnectServerError
	}

	// 5.设置扫描模式  %d=0表示为拉行，1表示为推行
	if err := r.action2SetScanMode(scanMode); err != nil {
		log.Println(err)
		response.ConnectServerError.Msg = err.Error()
		return response.ConnectServerError
	}

	// 6.设置编码器、车轮周长参数，其中%d表示对应的编码器频率，%lf表示车轮周长(小数点后三位)，单位为米。
	if err := r.action2SetEncoderWheelCircumference(encoderFrequency,
		wheelCircumference); err != nil {
		log.Println(err)
		response.ConnectServerError.Msg = err.Error()
		return response.ConnectServerError
	}
	return nil
}

// 新建工程，开始惯导检测，Action4必须至少在此接口后5分钟才能调用
func (r *RmmsClient) Action3_NewProject(projectName string) *response.ReplyResponse {
	// 测试各服务状态
	if err := r.actionTestAllServer(); err != nil {
		log.Println(err)
		response.NewProjectError.Msg = err.Error()
		return response.NewProjectError
	}

	// 判断是否已经在惯导采集状态是否正在采集
	DAQStatus, err := r.queryDAQCollectStatus()
	if err != nil {
		log.Println(err)
		response.NewProjectError.Msg = err.Error()
		return response.NewProjectError
	}

	// 判断激光雷达采集状态是否正在采集
	scannerStatus, err := r.queryScannerCollectStatus()
	if err != nil {
		log.Println(err)
		response.NewProjectError.Msg = err.Error()
		return response.NewProjectError
	}

	if DAQStatus != "0" {
		if scannerStatus == "1" {
			return response.LidarCollectingNewError
		}
		return response.DAQIsCollectingNewError
	}

	//2.惯导：设置工程名
	var nameStr string
	if projectName == "" {
		nameStr = "HD099" + "_" + time.Now().Format("20060102150405")
	} else {
		nameStr = projectName + "_" + time.Now().Format("20060102150405")
	}

	log.Println("工程名为：", nameStr)
	if err := r.action3SetDAQProjectName(nameStr); err != nil {
		log.Println(err)
		return response.NewProjectError
	}
	if err := r.action3SetScanProjectName(nameStr); err != nil {
		log.Println(err)
		return response.NewProjectError
	}
	if err := r.action3SetSyncProjectName(nameStr); err != nil {
		log.Println(err)
		return response.NewProjectError
	}
	if err := r.action3StartDqa(); err != nil {
		log.Println(err)
		return response.NewProjectError
	}
	if err := r.action3StartSyn(); err != nil {
		log.Println(err)
		return response.NewProjectError
	}

	// 占用锁，等待五分钟
	r.mutex.Lock()
	defer r.mutex.Unlock()
	fmt.Println("正在启动惯导检测，等待5分钟...")
	time.Sleep(5 * time.Minute)
	fmt.Println("惯导检测完毕，可以开始测站")
	return nil
}

// 开始测站扫描，90秒后才能达到正常转速
func (r *RmmsClient) Action4_StartStation() *response.ReplyResponse {
	// 测试各服务状态
	if err := r.actionTestAllServer(); err != nil {
		log.Println(err)
		return response.StartStationError
	}

	// 判断激光雷达采集状态是否正在采集
	DAQStatus, err := r.queryDAQCollectStatus()
	if err != nil {
		log.Println(err)
		return response.StartStationError
	}

	// 判断激光雷达采集状态是否正在采集
	scannerStatus, err := r.queryScannerCollectStatus()
	if err != nil {
		log.Println(err)
		return response.StartStationError
	}

	if DAQStatus != "1" {
		if scannerStatus == "1" {
			return response.LidarCollectingStartError
		}
		return response.DAQIsCollectingStartError
	}

	strTime := time.Now().Format("2006-01-02-15-04-05.000000")
	// 取出.后的字符串 000000 并修改为 000-000格式的字符串
	nanoSecond := strTime[:len(strTime)-3] + "-" + strTime[len(strTime)-3:]
	// 将strTime的最后六位替换成nanoSecond
	strTime = strTime[:len(strTime)-6] + nanoSecond

	if err := r.action4StartScanner(strTime); err != nil {
		log.Println("err:", err)
		return response.StartScannerError
	}

	// 占用锁，等待90秒
	fmt.Println("正在启动测站扫描，等待90秒")
	r.mutex.Lock()
	defer r.mutex.Unlock()
	time.Sleep(90 * time.Second)
	return nil
}

// 停止测站扫描，45秒后才能完全停止
func (r *RmmsClient) Action5_StopStation() *response.ReplyResponse {
	// 测试各服务状态
	if err := r.actionTestAllServer(); err != nil {
		log.Println("err:", err)
		return response.StopStationError
	}

	if err := r.action5StopScan(); err != nil {
		log.Println("err:", err)
		return response.StopStationError
	}

	// 占用锁，等待五分钟
	r.mutex.Lock()
	defer r.mutex.Unlock()
	fmt.Println("正在停止测站扫描，等待五分钟...")
	time.Sleep(5 * time.Minute)
	fmt.Println("测站扫描停止完毕")
	return nil
}

// 保存工程，Action5之后至少间隔5分钟才能调用此接口，此接口主要内容为确认各采集停止后再停止惯导，停止同步
func (r *RmmsClient) Action6_SaveProject() *response.ReplyResponse {
	// 测试各服务状态
	if err := r.actionTestAllServer(); err != nil {
		log.Println("err:", err)
		return response.SaveProjectError
	}

	if err := r.action6StopCollect(); err != nil {
		log.Println("err:", err)
		return response.SaveProjectError
	}

	if err := r.action6StopSynCollect(); err != nil {
		log.Println("err:", err)
		return response.SaveProjectError
	}

	return nil
}

// 系统关机，包括断开各服务
func (r *RmmsClient) Action7_CloseDevice() *response.ReplyResponse {
	// 测试各服务状态
	if err := r.actionTestAllServer(); err != nil {
		log.Println("err:", err)
		return response.CloseDeviceError
	}

	if err := r.action7CloseGPS(); err != nil {
		log.Println("err:", err)
		return response.CloseDeviceError
	}

	if err := r.action7CloseScanner(); err != nil {
		log.Println("err:", err)
		return response.CloseDeviceError
	}

	return nil
}

// 查询DAQ采集状态，1-正在采集，0-未采集，其他-异常
func (r *RmmsClient) QueryDAQCollectStatus() (string, error) {
	// 测试各服务状态
	if err := r.actionTestAllServer(); err != nil {
		return "", err
	}

	result, err := r.queryDAQCollectStatus()
	if err != nil {
		return "", err
	}

	return result, nil
}

// 查询DAQ文件大小，单位：MB
func (r *RmmsClient) QueryDAQFileSize() (string, error) {
	// 测试各服务状态
	if err := r.actionTestAllServer(); err != nil {
		return "", err
	}

	// 查询文件大小
	result, err := r.queryDAQFileSize()
	if err != nil {
		return "", err
	}

	return result, nil
}

// 查询DAQ采集时长，单位：秒
func (r *RmmsClient) QueryDAQCollectTime() (string, error) {
	// 测试各服务状态
	if err := r.actionTestAllServer(); err != nil {
		return "", err
	}

	// 查询DAQ采集时长
	result, err := r.queryDAQCollectDurationS()
	if err != nil {
		return "", err
	}

	return result, nil
}

// 查询激光当前采集状态，1-正在采集，0-未采集，其他-异常
func (r *RmmsClient) QueryScannerCollectStatus() (string, error) {
	// 测试各服务状态
	if err := r.actionTestAllServer(); err != nil {
		return "", err
	}

	// 查询激光当前采集状态
	result, err := r.queryScannerCollectStatus()
	if err != nil {
		return "", err
	}

	return result, nil
}

// 获取磁盘可用空间，单位：MB
func (r *RmmsClient) QueryFreeSpace() (string, error) {
	// 测试各服务状态
	if err := r.actionTestAllServer(); err != nil {
		return "", err
	}

	// 获取磁盘可用空间
	result, err := r.queryFreeSpaceMB()
	if err != nil {
		return "", err
	}

	return result, nil
}

// 查询激光数据文件大小，单位：MB
func (r *RmmsClient) QueryLidarFileSizeMB() (string, error) {
	// 测试各服务状态
	if err := r.actionTestAllServer(); err != nil {
		return "", err
	}

	// 获取磁盘可用空间
	result, err := r.queryScannerFileSize()
	if err != nil {
		return "", err
	}

	return result, nil
}

// 实时查询生成的灰度、深度影像数据
func (r *RmmsClient) QueryGrayDepthImage() (string, string, error) {
	// 测试各服务状态
	if err := r.actionTestAllServer(); err != nil {
		return "", "", err
	}

	gray, depth, err := r.queryGrayDepthImage()
	if err != nil {
		return "", "", err
	}
	return gray, depth, nil
}
