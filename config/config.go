package config

import "mms/websocket"

// 全局配置
type GlobalConfig struct {
	StompTopic  *Topic                   `yaml:"topic" mapstructure:"topic"`
	StompConfig *websocket.StompWSConfig `yaml:"stomp" mapstructure:"stomp"`
}

// 后端服务器websocket通信相关配置
type Topic struct {
	ModuleName string `yaml:"moduleName" mapstructure:"moduleName"`
	CmdSub     string `yaml:"cmdSub" mapstructure:"cmdSub"`
	CmdReply   string `yaml:"cmdReply" mapstructure:"cmdReply"`
	DataPush   string `yaml:"dataPush" mapstructure:"dataPush"`
	StatusPush string `yaml:"statusPush" mapstructure:"statusPush"`
}

// 后端服务器发送的指令json格式
type ConnectCmd struct {
	Seq        int    `json:"seq"`
	Cmd        string `json:"cmd"`
	ModuleName string `json:"module_name"`
	Payload    struct {
		ProjectInfo struct {
			Path string `json:"path"`
		} `json:"projectInfo"`
		DeviceInfo struct {
		} `json:"deviceInfo"`
		DeviceSysParams struct {
			Lidarparameter struct {
				NScanType          int     `json:"nScanType"`
				ScanMode           int     `json:"scanMode"`
				EncoderFrequency   int     `json:"encoderFrequency"`
				WheelCircumference float64 `json:"wheelCircumference"`
			} `json:"lidarparameter"`
		} `json:"deviceSysParams"`
		TaskID      string `json:"taskID"`
		ProjectName string `json:"projectName"`
	} `json:"payload"`
}
