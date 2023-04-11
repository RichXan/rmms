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
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	// 读取yaml文件
	config, err := readYamltoStruct()
	if err != nil {
		fmt.Println("readYamltoStruct err:", err.Error())
		panic(err)
	}

	client := rmms.NewRmmsClient(&config)
	// 监听服务器发送的cmd指令
	client.Ws.AddSub(config.StompTopic.CmdSub, client.ActionCmdSub)

	// publish测试AddSub
	// client.Ws.AddSub(config.StompTopic.CmdReply, client.SubListen)

	client.Ws.WebsocketStart()

	// 读取cmd.json文件 转换成[]byte
	// cmd, err := ioutil.ReadFile("cmd.json")
	// if err != nil {
	// 	log.Println(err)
	// }
	// client.Ws.Pubscribe(config.StompTopic.CmdSub, cmd)

	for {
	}
}

// 四个cmd操作
// 1. 开始工程：接收连接的参数
// 2. 开始任务
// 3. 停止任务
// 4. 停止工程

// 监听topic

// err = client.Action2_Connect(0, 1, 3250, 3.25)
// err = client.Action3_NewProject("hanni")
// err = client.Action4_StartStation()

// str, err := client.QueryDAQCollectStatus()
// fmt.Println("DAQCollectStatus: ", str)

// str, err := client.QueryDAQFileSize()
// fmt.Println("DAQFileSize: ", str)

// str, err = client.QueryDAQCollectTime()
// fmt.Println("DAQCollectTime: ", str)

// str, err := client.QueryScannerCollectStatus()
// fmt.Println("ScannerCollectStatus: ", str)

// str, err := client.QueryFreeSpace()
// fmt.Println("FreeSpace: ", str)

// str, err := client.QueryLidarFileSizeMB()
// fmt.Println("LidarFileSizeMB: ", str)

// str1, str2, err := client.QueryGrayDepthImage()
// fmt.Println("GrayImage: ", str1)
// fmt.Println("DepthImage: ", str2)

// err = client.Action5_StopStation()
// err = client.Action6_SaveProject()
// err = client.Action7_CloseDevice()

// strTime := time.Now().Format("2006-01-02-15-04-05.000000")
// fmt.Println("strTIme", strTime)
// wg := sync.WaitGroup{} // 控制等待所有协程都执行完再退出程序
// rmmsClient, err := rmms.NewRmmsClient()
// if err != nil {
// 	fmt.Println("Failed to create RMMS client:", err)
// 	return
// }

// // rmmsClient.Test1()
// wg.Add(2)
// go func() {
// 	rmmsClient.Test1("test1")
// 	wg.Done()
// }()
// go func() {
// 	rmmsClient.Test1("test2")
// 	wg.Done()
// }()
// wg.Wait()
// go func() {
// 	rmmsClient.Test1("test2")
// 	wg.Done()
// }()
// go func() {
// 	rmmsClient.Test1("test3")
// 	wg.Done()
// }()
// go func() {
// 	rmmsClient.Test1("test4")
// 	wg.Done()
// }()
// go func() {
// 	rmmsClient.Test1("test5")
// 	wg.Done()
// }()
// wg.Wait()

// 读取yaml文件to struct
func readYamltoStruct() (config.GlobalConfig, error) {
	var config config.GlobalConfig
	file := "./config/global.yaml"

	// 读取文件
	f, err := ioutil.ReadFile(file)
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
