# stdcs-3dlidar
三维激光雷达设备服务项目
> 该项目是接收为了接收前端发送的请求对设备做相应的操作流程（连接、启动、停止、关闭）以及设备的心跳，状态，生成相应的数据，深度图、灰度图原始图返回给前端。 

1. conn
   - 启动服务
   - 连接
   - 设置状态为 conn
   - 新建工程
   - 启动扫描仪，开始转动测站扫描
2. start
   - 设置当前状态为 start
   - 一秒钟上报一次设备 data 数据

> 发送图片数据给 disease 程序，json 数据格式：

```json
{
  "seq": 123,
  "cmd": "disease_seg",
  "module_name": "3DLidar",
  "data": {
    "project_path": "path",
    "taskID": "id",
    "devicesvalue": {
      "DAQCollectStatus": "0",
      "DAQFileSize": "100M",
      "DAQCollectTime": "100s",
      "ScannerCollectStatus": "0",
      "FreeSpace": "1G",
      "LidarFileSizeMB": "1024M",
      "GrayImage": "path",
      "DepthImage": "path"
    }
  }
}
```

disease 程序回复 json 格式：

```json
{
  "seq": 123,
  "module_name": "3DLidar",
  "data": {
    "devicesvalue": {
      "GrayImage": "path",
      "DepthImage": "path"
    }
  }
}
```

3. stop
   - 停止测站扫描
   - 保存工程
4. disconn
   - 断开连接
