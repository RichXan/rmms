import threading
import time,os
import stomp,json
from stitch import Stitch
from stitchV import MyStichingV
from concurrent.futures import ThreadPoolExecutor
from queue import Queue




def connect(body):
    global conn
    '''从list中取出单链表中第一个为不为空的data（带锁取）并保存data
        取出数据后将list.data清空
    '''
    # 接收到的json数据解析成python对象
    print("----------------enter connect-------------")
    try: 
        text = json.loads(body)
        text['code'] = 0
        param = text.get("payload").get("param")
        # 接收到主题发送来的消息，不显示接收到的具体信息
        print('received a message  from  "/topic/cmd.picture.stitch"  seq:"%s"' %
        (text.get("seq")))
        # 水平操作
        if param.get("direction") == "horizontal":
            horizontal(param, text)
        # 垂直拼接
        elif param.get("direction") == "vertical":
            vertical(param, text)
    except json.decoder.JSONDecodeError: 
        # 发送的数据的格式错误或者不是json格式
        text["code"] = 503
        conn.send(topic=json.dumps(text), destination=pubTopic)
    # except: 
    #     # 其他错误
    #     msg["code"] = 505
    #     conn.send(topic=json.dumps(msg), destination=pubTopic)

def horizontal(param, text):
    if (len(param['filename']) == 2 and param['filename'][0] == param['filename'][1]):
        # 两张照片相同
        text["code"] = 504
        conn.send(body=json.dumps(text), destination=pubTopic)
    else:
        if (len(param['filename']) == 2 or len(param['filename']) == 3 or len(param['filename']) == 4) :
            ui = Stitch()
            ui.filename = param.get("filename")
            ui.resultImgDirectory = param.get("resultImgDirectory","./ouput")
            if not os.path.exists(ui.resultImgDirectory):
                    os.mkdir(ui.resultImgDirectory)      
            ui.resultImgFilename = param.get("resultImgFilename","result.jpg")
            ui.radioButton_Dis = param.get("algorithm",True)
            ui.imgnum = len(param['filename'])
            ui.change_gray = param.get("change_gray",True)
            param["resultPath"] = ui.resultImgDirectory + '/' +  ui.resultImgFilename
            try: 
                ui.rd_img()
                ui.auto_stich_pro()
                text["code"] = 0
            except json.decoder.JSONDecodeError: 
                # 拼接失败
                text["code"] = 502
            # 拼接成功
            conn.send(body=json.dumps(text), destination=pubTopic)
            
        else:
            # 传入的照片数量异常
            text["code"] = 501
            conn.send(body=json.dumps(text), destination=pubTopic)

def vertical(param, text):
    if (len(param['filename']) == 2) :
        ui = MyStichingV()
        ui.filename = param.get("filename")
        ui.resultImgDirectory = param.get("resultImgDirectory","./ouput")
        if not os.path.exists(ui.resultImgDirectory):
                os.mkdir(ui.resultImgDirectory)      
        ui.resultImgFilename = param.get("resultImgFilename","result.jpg")
        # ui.radioButton_Dis = param.get("algorithm",True)
        ui.imgnum = len(param['filename'])
        # ui.change_gray = param.get("change_gray",True)
        text["resultPath"] = ui.resultImgDirectory + '/' + ui.resultImgFilename
        try: 
            ui.biaoding()
            text["code"] = 0
        except: 
            # 拼接失败
            text["code"] = 501
        conn.send(body=json.dumps(text), destination=pubTopic)
      Lmp.ConnectionListener):
    global conn
    global json_queue
    def on_error(self, frame):
        print('received an error "%s"' % frame.body)
    
    def on_message(self, frame):
        if frame.headers['subscription'] == "/topic/cmd.picture.stitch":
            '''将数据存入单链表：list.append(data)'''
            json_queue.put(frame.body)
            pool.submit(connect,args=(frame.body))
            # 通过submit函数提交执行的函数到线程池中，submit函数立即返回，不阻塞
            # task = pool.submit(connect, (frame.body))
            # task.result()
            # pool.shutdown()


if __name__ == "__main__":
    host_and_ports=[("139.9.44.106","21113")]
    subTopic = "/topic/cmd.picture.stitch"  # 接收json订阅的主题
    pubTopic = "/topic/cmd.stitch.result"  # 发布的主题
    heartTopic = "/topic/cmd.stitch.heart" # 心跳主题

    conn = stomp.Connection(host_and_ports)
    conn.set_listener('', MyListener())
    conn.connect("admin", "sealan")
    conn.subscribe(destination=subTopic, id=subTopic, ack='auto') # 订阅心跳主题
    conn.subscribe(destination=pubTopic, id=pubTopic, ack='auto') # 订阅心跳主题
    conn.subscribe(destination=heartTopic, id=heartTopic, ack='auto') # 订阅心跳主题
    # 水平测试
    jsonTest = '{"seq":123,"cmd":"synthetize","payload":{"param":{"filename":["./data/0.jpg","./data/1.jpg", "./data/2.jpg" ],"direction": "horizontal","resultImgDirectory":"./output","resultImgFilename":"result.jpg","ratioLow":0.8,"algorithm":true,"showmatches":true,"fusion":true,"gray":true}}}'

    # 垂直测试
    # jsonTest = '{"seq":123,"cmd":"synthetize","payload":{"param":{"filename":["E:/sealan/SZEDU/connect/test/result.jpg","E:/sealan/SZEDU/connect/test/result1.jpg"],"direction": "vertical","resultImgDirectory":"./output","resultImgFilename":"result555.jpg","ratioLow":0.8,"algorithm":true,"showmatches":true,"fusion":true,"gray":true}}}'
                        
    conn.send(body=jsonTest,destination=subTopic)

    json_queue = Queue()
    pool = ThreadPoolExecutor(max_workers=50)
    # pool.submit(connect,args=(frame.body))
    lock = threading.Lock()
    while True:
        seq = int(time.time())
        # 发送心跳数据z
        heartMsg={"seq":seq,"cmd":"stitchHeart"}
        # conn.send(body=json.dumps(jsonTest),destination=subTopic)
        conn.send(body=json.dumps(heartMsg),destination=heartTopic)
        # 测试给subTopic发送多组数据有没有顺序错误
        for i in range(10):
            subTest = {"seq":1,"cmd":"synthetize","payload":{"param":{"filename":["./data/0.jpg","./data/1.jpg", "./data/2.jpg" ],"direction": "horizontal","resultImgDirectory":"./output","resultImgFilename":"result.jpg","ratioLow":0.8,"algorithm":True,"showmatches":True,"fusion":True,"gray":True}}}
            subTest["seq"] = subTest["seq"] + i
            conn.send(body=json.dumps(subTest),destination=subTopic)
        print("not  ",not json_queue.empty())
        if not json_queue.empty():
            # lock.acquire()
            print("nononono")
            json_data = json_queue.get()
            print("json_queue.get()",json_queue.get())
        # print("*"*30)
        # print(body)
            pool.submit(connect,args=(json_data))
            # lock.release()
        time.sleep(5)
    conn.disconnect()
