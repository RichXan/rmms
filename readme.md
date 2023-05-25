1. conn
   - 启动服务
   - 连接
   - 设置状态为 conn
2. start
   - 新建工程
   - 开始测站扫描
   - 设置当前状态为 start
   - 一秒钟上报一次设备 data 数据

> 发送拼接数据给 disease 程序，json 数据格式：

```json
{
  "seq": 123,
  "cmd": "disease_seg",
  "module_name": "3DLidar",
  "data": {
    "project_path": "path",
    "taskID": "id",
    "devicesvalue": {
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
