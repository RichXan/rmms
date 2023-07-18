package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"mms/config"
	rmms "mms/rmms"

	"gopkg.in/yaml.v2"
)

func main() {
	log.Println("3DLidar服务启动")
	// 读取yaml文件
	filePath := "./config/rmms.yaml"
	config, err := readYamltoStruct(filePath)
	if err != nil {
		fmt.Println("readYamltoStruct err:", err.Error())
		panic(err)
	}

	client := rmms.NewRmmsClient(&config)

	// 启动服务
	if respErr := client.Action1_StartServer(); respErr != nil {
		client.Ws.Pubscribe(config.StompTopic.CmdReply, respErr.MarshalToCMDReplyBytes(0, 0))
		fmt.Println("启动服务失败: ", respErr.Msg)
	}
	// 监听服务器发送的cmd指令
	client.Ws.AddSub(config.StompTopic.CmdSub, client.ActionCmdSub)

	// disease
	// client.Ws.AddSub("/topic/data.disease.error", print_sub_msg_error)
	// client.Ws.AddSub("/topic/data.push", print_sub_msg_data_push)
	// client.Ws.AddSub(config.StompTopic.DiseasePush, print_sub_msg_disease)

	client.Ws.WebsocketStart()

	for {
	}
}

// 读取yaml文件to struct
func readYamltoStruct(filePath string) (config.GlobalConfig, error) {
	var config config.GlobalConfig

	// 读取文件
	f, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println(err)
		return config, err
	}

	// 转换成Struct
	err = yaml.Unmarshal(f, &config)
	if err != nil {
		fmt.Printf("%v\n", err.Error())
		return config, err
	}
	return config, nil
}
