package rmms

import (
	"fmt"
	"mms/response"
	"strconv"
	"strings"
	"time"

	"github.com/axgle/mahonia" //编码转换
)

// 向设备发送命令并返回结果
func (r *RmmsClient) sendCommand(port int, cmd string) ([]byte, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	time.Sleep(200 * time.Millisecond)

	// TODO: 需要发送的什么内容？
	if cmd == "" {
		return nil, nil
	}

	bytes := []byte(cmd)
	response, err := r.tc.Send2Port(port, bytes)
	if err != nil {
		return nil, err
	}

	if string(response) == "$Err" {
		DErrorLog.Println("设备端回复内容为：" + string(response))
		return nil, fmt.Errorf("发送命令失败:response=$Err")
	}

	// TODO: check response
	// fmt.Println("response:", string(response))

	return response, nil
}

// 关闭连接
func (r *RmmsClient) close() error {
	return r.tc.CloseAllPortConn()
}

// 启动采集操控服务程序
func (r *RmmsClient) action1StartServer() error {
	InfoLog.Println("发送启动采集操控服务程序指令成功，启动采集操控服务程序...")
	// 发送消息
	resp, err := r.sendCommand(tcp_port_rmms, "$SCT,RMMSSERVER")
	if err != nil {
		// gps服务未启动，gps端口连接失败。
		DErrorLog.Println("发送启动采集操控服务程序指令失败" + ",$SCT,RMMSSERVER" + ":" + err.Error())
		return err
	}

	// 校验返回值
	if string(resp) != "$OK" {
		err := fmt.Errorf("发送启动采集操控服务程序指令失败" + ",$SCT,RMMSSERVER" + ":" + string(resp))
		DErrorLog.Println(err.Error())
		return err
	}

	// 等待15秒，等待服务程序启动，连接子设备
	r.mutex.Lock()
	defer r.mutex.Unlock()
	InfoLog.Println("发送启动采集操控服务程序指令成功，启动采集操控服务程序...")
	for i := 0; i+5 < 15; i++ {
		// 上报设备状态
		r.Ws.Pubscribe(r.config.StompTopic.CmdReply,
			response.WaitForConnReply.MarshalToCMDReplyBytes(r.Param.Seq, 15-i))
		InfoLog.Printf("waitting %d seconds...\n", 15-i)
		time.Sleep(1 * time.Second)
	}
	SuccessLog.Println("启动采集操控服务程序成功！")

	return nil
}

func (r *RmmsClient) connAllTcpServer() error {
	// 初始化连接tcp_port_daq端口
	err := r.tc.InitConnPort(tcp_ip, tcp_port_daq)
	if err != nil {
		DErrorLog.Println("初始化连接", tcp_port_daq, "端口失败！")
		return err
	}
	SuccessLog.Println("daq服务已连接成功！")

	// 初始化连接tcp_port_sync端口
	err = r.tc.InitConnPort(tcp_ip, tcp_port_sync)
	if err != nil {
		DErrorLog.Println("初始化连接", tcp_port_sync, "端口失败！")
		return err
	}
	SuccessLog.Println("sync服务已连接成功！")

	// 初始化连接tcp_port_scanner端口
	err = r.tc.InitConnPort(tcp_ip, tcp_port_scanner)
	if err != nil {
		DErrorLog.Println("初始化连接", tcp_port_scanner, "端口失败！")
		return err
	}
	SuccessLog.Println("scanner服务已连接成功！")

	// 初始化连接tcp_port_gps端口
	err = r.tc.InitConnPort(tcp_ip, tcp_port_gps)
	if err != nil {
		DErrorLog.Println("初始化连接", tcp_port_gps, "端口失败！")
		return err
	}
	SuccessLog.Println("gps服务已连接成功！")

	for i := 10; i < 15; i++ {
		// 上报设备状态
		r.Ws.Pubscribe(r.config.StompTopic.CmdReply,
			response.WaitForConnReply.MarshalToCMDReplyBytes(r.Param.Seq, 15-i))
		InfoLog.Printf("waitting %d seconds...\n", 15-i)
		time.Sleep(1 * time.Second)
	}
	SuccessLog.Println("连接子tcp服务成功！")

	return nil
}

// TODO 未完成
func (r *RmmsClient) actionTestAllServer() error {
	_, err := r.sendCommand(tcp_port_daq, "")
	if err != nil {
		// daq服务未启动，daq端口连接失败。
		return err
	}

	_, err = r.sendCommand(tcp_port_sync, "")
	if err != nil {
		// sync服务未启动，sync端口连接失败。
		return err
	}

	_, err = r.sendCommand(tcp_port_scanner, "")
	if err != nil {
		// scanner服务未启动，scanner端口连接失败。
		return err
	}

	_, err = r.sendCommand(tcp_port_gps, "")
	if err != nil {
		// gps服务未启动，gps端口连接失败。
		return err
	}
	return nil
}

// 控制GPS脉冲，发送开关，打开指令
func (r *RmmsClient) action2OpenGPS() error {
	// 打开gps脉冲
	response, err := r.sendCommand(tcp_port_sync, "%SGS47\r\n")
	if err != nil {
		// gps服务未启动，gps端口连接失败。
		DErrorLog.Println("打开GPS失败" + ",%SGS47" + ":" + err.Error())
		return err
	}

	// 校验返回值
	if string(response) != "$OK" {
		errstr := fmt.Sprintf("打开GPS失败" + ",%SGS47" + ":" + string(response))
		DErrorLog.Println(errstr)
		return fmt.Errorf(errstr)
	}

	SuccessLog.Println("1.成功打开GPS脉冲")
	return nil
}

// 设置采集模式为无 GPS 模式（即隧道模式）
func (r *RmmsClient) action2NoGPSMode() error {
	// 打开gps
	response, err := r.sendCommand(tcp_port_sync, "%SMT4A\r\n")
	if err != nil {
		// gps服务未启动，gps端口连接失败。
		DErrorLog.Println("设置采集模式为无GPS模式失败" + ",%SMT4A" + ":" + err.Error())
		return err
	}

	// 校验返回值
	if string(response) != "$OK" {
		return fmt.Errorf("设置采集模式为无GPS模式失败")
	}
	SuccessLog.Println("2.成功设置采集模式为无GPS模式")
	return nil
}

// 设置扫描频率
func (r *RmmsClient) action2ScanType(scanType int) error {
	// 设置扫描频率
	// 第一个参数表示是否使用该扫描头扫描，为 1 表示使用，为  0 表示不使用（默认为 1 开启）
	// 第二个参数表示设置扫描模式，分为 4 档
	// 0 表示 50hz 20000 点频，1 表示 50hz 10000 点频，2 表示 100hz 10000 点频，3 表示 200hz 5000 点频；
	cmd := fmt.Sprintf("$LFTSC,1,%d", scanType)
	response, err := r.sendCommand(tcp_port_scanner, cmd)
	if err != nil {
		// gps服务未启动，gps端口连接失败。
		DErrorLog.Println("设置扫描频率失败" + ",$LFTSC,1," + strconv.Itoa(scanType) + ":" + err.Error())
		return err
	}

	// 校验返回值
	if string(response) != "$OK" {
		err := fmt.Errorf("设置扫描频率失败" + ",$LFTSC,1," + strconv.Itoa(scanType) + ":" + string(response))
		DErrorLog.Println(err.Error())
		return err
	}
	//无校验
	SuccessLog.Printf("3.成功设置扫描频率为：%d \n", scanType)
	return nil
}

// 设置扫描模式  %d=0表示为拉行，1表示为推行
func (r *RmmsClient) action2SetScanMode(mode int) error {
	response, err := r.sendCommand(tcp_port_scanner, "$LSMOD,"+strconv.Itoa(mode))
	if err != nil {
		DErrorLog.Println("设置扫描模式失败" + ",$LSMOD," + strconv.Itoa(mode) + ":" + err.Error())
		return err
	}

	// 校验返回值
	if string(response) != "$OK" {
		err := fmt.Errorf("设置扫描模式失败" + ",$LSMOD," + strconv.Itoa(mode) + ":" + string(response))
		DErrorLog.Println(err.Error())
		return err
	}
	SuccessLog.Printf("4.成功设置扫描模式为：%d \n", mode)
	return nil
}

// 设置编码器、车轮周长参数，其中%d表示对应的编码器频率，%lf表示车轮周长，单位为米。
func (r *RmmsClient) action2SetEncoderWheelCircumference(encoderFrequency int, wheelCircumference float64) error {
	response, err := r.sendCommand(tcp_port_scanner, "$DMIV,"+strconv.Itoa(encoderFrequency)+","+strconv.FormatFloat(wheelCircumference, 'f', 2, 64))
	if err != nil {
		DErrorLog.Printf("设置编码器、车轮周长参数失败,$DMIV,%d,%.2f : %s \n", encoderFrequency, wheelCircumference, err.Error())
		return err
	}

	// 校验返回值
	if string(response) != "$OK" {
		err := fmt.Errorf("设置编码器、车轮周长参数失败" + ",$DMIV," + strconv.Itoa(encoderFrequency) + "," + strconv.FormatFloat(wheelCircumference, 'f', 2, 64) + ":" + string(response))
		DErrorLog.Println(err.Error())
		return err
	}

	SuccessLog.Printf("5.成功设置编码器、车轮周长参数为：%d, %f \n", encoderFrequency, wheelCircumference)
	return nil
}

// 设置惯导工程名
func (r *RmmsClient) action3SetDAQProjectName(projectName string) error {
	response, err := r.sendCommand(tcp_port_daq, "$SPN,"+projectName)
	if err != nil {
		DErrorLog.Println("设置惯导工程名失败" + ",$SPN," + projectName + ":" + err.Error())
		return err
	}

	// 校验返回值
	if string(response) != "$OK" {
		err := fmt.Errorf("设置惯导工程名失败" + ",$SPN," + projectName + ":" + string(response))
		DErrorLog.Println(err.Error())
		return err
	}

	SuccessLog.Printf("成功设置惯导工程名为：%s \n", projectName)
	return nil
}

// 设置扫描工程名
func (r *RmmsClient) action3SetScanProjectName(projectName string) error {
	response, err := r.sendCommand(tcp_port_scanner, "$SPN,"+projectName)
	if err != nil {
		DErrorLog.Println("设置扫描工程名失败" + ",$SPN," + projectName + ":" + err.Error())
		return err
	}

	// 校验返回值
	if string(response) != "$OK" {
		err := fmt.Errorf("设置扫描工程名失败" + ",$SPN," + projectName + ":" + string(response))
		DErrorLog.Println(err.Error())
		return err
	}

	SuccessLog.Println("成功设置扫描工程名为：", projectName)
	return nil
}

// 设置同步工程名
func (r *RmmsClient) action3SetSyncProjectName(projectName string) error {
	response, err := r.sendCommand(tcp_port_sync, "$SPN,"+projectName)
	if err != nil {
		DErrorLog.Println("设置同步工程名失败" + ",$SPN," + projectName + ":" + err.Error())
		return err
	}

	// 校验返回值
	if string(response) != "$OK" {
		err := fmt.Errorf("设置同步工程名失败" + ",$SPN," + projectName + ":" + string(response))
		DErrorLog.Println(err.Error())
		return err
	}


	SuccessLog.Println("成功设置同步工程名为：", projectName)
	return nil
}

// 开始DQA采集
func (r *RmmsClient) action3StartDqa() error {
	response, err := r.sendCommand(tcp_port_daq, "$SSG")
	if err != nil {
		DErrorLog.Println("开始DQA采集失败" + ",$SSG" + ":" + err.Error())
		return err
	}

	// 校验返回值
	if string(response) != "$OK" {
		err := fmt.Errorf("开始DQA采集失败" + ",$SSG" + ":" + string(response))
		DErrorLog.Println(err.Error())
		return err
	}

	SuccessLog.Println("成功开始DQA采集")
	return nil
}

// 开始syn采集
func (r *RmmsClient) action3StartSyn() error {
	response, err := r.sendCommand(tcp_port_sync, "$SSG")
	if err != nil {
		DErrorLog.Println("开始syn采集失败" + ",$SSG" + ":" + err.Error())
		return err
	}

	// 校验返回值
	if string(response) != "$OK" {
		err := fmt.Errorf("开始syn采集失败" + ",$SSG" + ":" + string(response))
		DErrorLog.Println(err.Error())
		return err
	}

	return nil
}

// 开始扫描
func (r *RmmsClient) action4StartScanner(time string) error {
	response, err := r.sendCommand(tcp_port_scanner, "$SSG,"+time)
	if err != nil {
		DErrorLog.Println("开始扫描失败" + ",$SSG," + time + ":" + err.Error())
		return err
	}

	// 校验返回值
	if string(response) != "$OK" {
		err := fmt.Errorf("开始扫描失败" + ",$SSG," + time + ":" + string(response))
		DErrorLog.Println(err.Error())
		return err
	}

	SuccessLog.Println("成功开始扫描:", time)
	return nil
}

// 停止扫描
func (r *RmmsClient) action6StopScan() error {
	response, err := r.sendCommand(tcp_port_scanner, "$STG")
	if err != nil {
		DErrorLog.Println("停止扫描失败" + ",$STG" + ":" + err.Error())
		return err
	}

	// 校验返回值
	if string(response) != "$OK" {
		err := fmt.Errorf("停止扫描失败" + ",$STG" + ":" + string(response))
		DErrorLog.Println(err.Error())
		return err
	}

	SuccessLog.Println("成功停止扫描")
	return nil
}

// 停止惯导 工程采集
func (r *RmmsClient) action5StopCollect() error {
	response, err := r.sendCommand(tcp_port_daq, "$STG")
	if err != nil {
		DErrorLog.Println("停止惯导采集失败" + ",$STG" + ":" + err.Error())
		return err
	}

	// 校验返回值
	if string(response) != "$OK" {
		err := fmt.Errorf("停止惯导采集失败" + ",$STG" + ":" + string(response))
		DErrorLog.Println(err.Error())
		return err
	}

	SuccessLog.Println("成功停止惯导采集")
	return nil
}

// 停止同步 工程采集
func (r *RmmsClient) action6StopSynCollect() error {
	response, err := r.sendCommand(tcp_port_sync, "$STG")
	if err != nil {
		DErrorLog.Println("停止同步采集失败" + ",$STG" + ":" + err.Error())
		return err
	}

	// 校验返回值
	if string(response) != "$OK" {
		err := fmt.Errorf("停止同步采集失败" + ",$STG" + ":" + string(response))
		DErrorLog.Println(err.Error())
		return err
	}

	SuccessLog.Println("成功停止同步")
	return nil
}

// 关闭同步GPS
func (r *RmmsClient) action7CloseGPS() error {
	response, err := r.sendCommand(tcp_port_sync, "%SGE51\r\n")
	if err != nil {
		DErrorLog.Println("关闭GPS失败" + ",%SGE51" + ":" + err.Error())
		return err
	}

	// 校验返回值
	if string(response) != "$OK" {
		err := fmt.Errorf("关闭GPS失败" + ",%SGE51" + ":" + string(response))
		DErrorLog.Println(err.Error())
		return err
	}

	SuccessLog.Println("成功关闭GPS")
	return nil
}

// 关闭扫描 (TODO: dangous)
func (r *RmmsClient) action7CloseScanner() error {
	response, err := r.sendCommand(tcp_port_scanner, "$SHTD")
	if err != nil {
		DErrorLog.Println("关闭扫描失败" + ",$SHTD" + ":" + err.Error())
		return err
	}

	// 校验返回值
	if string(response) != "$OK" {
		err := fmt.Errorf("关闭扫描失败" + ",$SHTD" + ":" + string(response))
		DErrorLog.Println(err.Error())
		return err
	}

	SuccessLog.Println("成功关闭扫描")
	return nil
}

// 查询DAQ采集状态，1-正在采集，0-未采集，其他-异常
func (r *RmmsClient) queryDAQCollectStatus() (string, error) {
	response, err := r.sendCommand(tcp_port_daq, "$GDS")
	if err != nil {
		DErrorLog.Println("查询DAQ采集状态失败" + ",$GDS" + ":" + err.Error())
		return "", err
	}

	// 校验返回值，是否为前四个字符是否为 $DS, 且大于4个字符
	if (string(response)[0:5] != "$DS,0" && string(response)[0:5] != "$DS,1") ||
		len(string(response)) < 5 {
		DErrorLog.Println("查询DAQ采集状态失败" + ",$GDS" + ":" + string(response))
		return "", fmt.Errorf("返回值错误")
	}

	SuccessLog.Println("DAQ采集状态：", string(response)[4:])
	return string(response)[4:], nil
}

// 查询DAQ文件大小，单位：MB
func (r *RmmsClient) queryDAQFileSize() (string, error) {
	response, err := r.sendCommand(tcp_port_daq, "$GFS")
	if err != nil {
		DErrorLog.Println("查询DAQ文件大小失败" + ",$GFS" + ":" + err.Error())
		return "", err
	}

	// 校验返回值，是否为前四个字符是否为 $FS,
	if string(response)[0:4] != "$FS," || len(string(response)) < 5 {
		DErrorLog.Println("查询DAQ文件大小失败" + ",$GFS" + ":" + string(response))
		return "", fmt.Errorf("返回值错误")
	}

	// 将返回值转换为float64
	fileSize, err := strconv.ParseFloat(string(response)[4:], 64)
	if err != nil {
		SErrorLog.Println("转换失败")
		return "", err
	}
	size := fmt.Sprintf("%.2f", fileSize/1024/1024)
	SuccessLog.Printf("DAQ文件大小：%s MB \n", size)
	return size, nil
}

// 查询DAQ采集时长，单位：秒
func (r *RmmsClient) queryDAQCollectDurationS() (string, error) {
	response, err := r.sendCommand(tcp_port_daq, "$GST")
	if err != nil {
		DErrorLog.Println("查询DAQ采集时长失败" + ",$GST" + ":" + err.Error())
		return "", err
	}

	// 校验返回值，是否为前四个字符是否为 $ST,
	if string(response)[0:4] != "$ST," || len(string(response)) < 5 {
		DErrorLog.Println("查询DAQ采集时长失败" + ",$GST" + ":" + string(response))
		return "", fmt.Errorf("返回值错误")
	}

	// 返回第五个字符及之后的字符串
	SuccessLog.Printf("DAQ采集时长：%s s \n", string(response)[4:])
	return string(response)[4:], nil
}

// 查询激光当前采集状态，1-正在采集，0-未采集，其他-异常
func (r *RmmsClient) queryScannerCollectStatus() (string, error) {
	response, err := r.sendCommand(tcp_port_scanner, "$GDS,1")
	if err != nil {
		DErrorLog.Println("查询激光采集状态失败" + ",$GDS,1" + ":" + err.Error())
		return "", err
	}

	// 校验返回值，是否为前7个字符是否为 $DS,1,0, 且大于4个字符
	if (string(response)[0:7] != "$DS,1,0" && string(response)[0:7] != "$DS,1,1") ||
		len(string(response)) < 7 {
		DErrorLog.Println("查询激光采集状态失败" + ",$GDS,1" + ":" + string(response))
		return "", fmt.Errorf("返回值错误")
	}

	// 返回第五个字符及之后的字符串
	SuccessLog.Println("激光采集状态：", string(response)[6:])
	return string(response)[6:], nil
}

// 获取磁盘可用空间，单位：MB
func (r *RmmsClient) queryFreeSpaceMB() (string, error) {
	response, err := r.sendCommand(tcp_port_scanner, "$DKLT,1")
	if err != nil {
		DErrorLog.Println("获取磁盘可用空间失败" + ",$DKLT,1" + ":" + err.Error())
		return "", err
	}

	// 校验返回值，是否为前6个字符是否为 $DK,1,
	if string(response)[0:6] != "$DK,1," || len(string(response)) < 7 {
		DErrorLog.Println("获取磁盘可用空间失败" + ",$DKLT,1" + ":" + string(response))
		return "", fmt.Errorf("返回值错误")
	}

	// 将返回值转换为float64
	fileSize, err := strconv.ParseFloat(string(response)[6:], 64)
	if err != nil {
		SErrorLog.Println("转换失败")
		return "", err
	}
	size := fmt.Sprintf("%.2f", fileSize*1024)
	// 返回第五个字符及之后的字符串
	SuccessLog.Printf("磁盘可用空间：%s MB \n", size)
	return size, nil
}

// 查询激光数据文件大小，单位：MB
func (r *RmmsClient) queryScannerFileSize() (string, error) {
	response, err := r.sendCommand(tcp_port_scanner, "$GLDT,1")
	if err != nil {
		DErrorLog.Println("查询激光数据文件大小失败" + ",$GLDT,1" + ":" + err.Error())
		return "", err
	}

	// 校验返回值，是否为前5个字符是否为 $GLL,
	if string(response)[0:5] != "$GLL," || len(string(response)) < 6 {
		DErrorLog.Println("查询激光数据文件大小失败" + ",$GLDT,1" + ":" + string(response))
		return "", fmt.Errorf("返回值错误")
	}

	// 将返回值转换为float64
	fileSize, err := strconv.ParseFloat(string(response)[5:], 64)
	if err != nil {
		SErrorLog.Println("转换失败")
		return "", err
	}

	// 返回第五个字符及之后的字符串
	size := fmt.Sprintf("%.2f", fileSize*1024)
	SuccessLog.Printf("激光数据文件大小：%s MB \n", size)
	return size, nil
}

// 灰度影像生成
// 实时查询生成的灰度、深度影像数据
func (r *RmmsClient) queryGrayDepthImage() (string, string, error) {
	response, err := r.sendCommand(tcp_port_scanner, "$ZIMG,1")
	if err != nil {
		DErrorLog.Println("查询灰度影像失败" + ",$ZIMG,1" + ":" + err.Error())
		return "", "", err
	}

	// 校验返回值，是否为前4个字符是否为 $LD,
	if string(response) == "$IMG,1,NoImg,NoImg," {
		SuccessLog.Println("灰度影像：NoImg" + " , 深度影像：NoImg")
		return "NoImg", "NoImg", nil
	}

	// 校验返回值，是否为前7个字符是否为 $IMG,1,
	if string(response)[0:7] != "$IMG,1," || len(string(response)) < 7 {
		DErrorLog.Println("查询灰度影像失败" + ",$ZIMG,1" + ":" + string(response))
		return "", "", fmt.Errorf("返回值错误")
	}

	// 返回第7个字符及之后的字符串
	// 用,分割字符串
	strs := strings.Split(string(response)[7:], ",")
	var enc = mahonia.NewDecoder("gbk")
	// 将前两个字符替换成 \\192.168.1.92
	strs[0] = strings.Replace(enc.ConvertString(strs[0]), "D:", "\\\\192.168.1.92", 1)
	strs[1] = strings.Replace(enc.ConvertString(strs[1]), "D:", "\\\\192.168.1.92", 1)
	SuccessLog.Println("灰度影像：" + strs[0] + " , 深度影像：" + strs[1])
	return strs[0], strs[1], nil
}
