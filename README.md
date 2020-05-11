# Queueman

Queueman 是一个适用于 RabbitMQ、Redis 队列的高性能分发中间件。支持延时队列、并发控制、失败自动重试。

1. 简单的并发控制
2. 简单配置就可以自动失败后重试
3. 不用再写命令行代码就可以消费队列了

**测试理论速度：单机 1-3 万条/秒**

**配置用例1:**
- 服务器: 阿里云 ecs.sn1ne.large 2 vCPU 4 GiB (I/O优化) CentOS 7.2 64位
- Redis: 阿里云 Redis 4.0 1G集群版 最大连接数： 20,000
- RabbitMQ: 阿里云 消息队列 AMQP 协议版本：0-9-1 TPS峰值流量：1000
- 网络：阿里云内网连接

队列类型 | 数据量 | 处理时间（秒） | 处理速度 | 备注
------------ | ------------- | ------------- | ------------- | -------------
RabbitMQ | 1000000 | 76.785453 | 13023/s | 阿里云内网连接、开启 auto ack、不开启 DispatchURL 转发
Redis | 1000000 | 31.523070 | 31722/s | 阿里云内网连接、不开启 DispatchURL 转发
同时跑 RabbitMQ & Redis 各一个队列| 各测试1000000 | 126.724109 49.977177 | 11318/s | 内网、开启 auto ack、不开启 DispatchURL 转发


**配置用例2:**
- 服务器: MacBook Pro (Retina, 15-inch, Mid 2015) 2.2 GHz 四核Intel Core i7 macOS Catalina 10.15.3
- Redis: 阿里云 Redis 4.0 1G集群版 最大连接数： 20,000
- RabbitMQ: 阿里云 消息队列 AMQP 协议版本：0-9-1 TPS峰值流量：1000
- 网络：公网连接

队列类型 | 数据量 | 处理时间（秒） | 处理速度 | 备注
------------ | ------------- | ------------- | ------------- | -------------
本机 | RabbitMQ | 1000000 | 79.617749254 | 12560/s | 公网连接、开启 auto ack、不开启 DispatchURL 转发
本机 | Redis | 1000000 | 78.276917 | 12775/s | 公网连接、不开启 DispatchURL 转发
本机 | 同时跑 RabbitMQ & Redis 各一个队列| 各测试1000000 | 97.412025 89.785312 | 10752/s | 公网连接、开启 auto ack、不开启 DispatchURL 转发

## 内容列表

- [背景](#背景)
- [如何安装](#如何安装)
	- [克隆到本地并预处理](#克隆到本地并预处理)
  	- [各种环境运行](#各种环境运行)
- [详细介绍](#详细介绍)
	- [完整流程](#完整流程)
	- [开发者做两件事就够了](#开发者做两件事就够了)
	- [开发者推送数据到队列](#开发者推送数据到队列)
	- [指定地址接收数据](#指定地址接收数据)
	- [指定地址的响应](#指定地址的响应)
	- [并发控制](#并发控制)
	- [失败后处理](#失败后处理)
  	- [命令行工具](#命令行工具)
	- [查看统计信息](#查看统计信息)
	- [配置文件详解](#配置文件详解)
- [维护者](#维护者)
- [如何贡献](#如何贡献)
- [使用许可](#使用许可)

## 背景

1. 队列越来越多，消费脚本也越来越多，通过多进程来消费队列，开销也比较大。
2. 程序员既要写服务端代码，也要写命令行代码，还要对命令行代码进行部署，容易出错。
3. 正常业务要延时处理，有没有比较简单的方式来实现自动延时，不用写正常业务代码，还要写延时业务代码。

于是，是否可以有一种有新的轻量模式来取代这种传统模式，让开发人员更关注实现业务本身？让开发人员方便快捷的完成如下流程：

1. 开发人员写 web 代码 push 数据到队列
2. 队列中间件取出数据，转发到指定 URL 地址
3. 开发人员写 web 代码接收并处理

## 如何安装

### 克隆到本地并预处理
```sh
# git clone https://github.com/marknown/queueman.git
# cd queueman
# # 配置你的 queueman.json 文件 #
# cp bin/queueman_linux /usr/local/bin/queueman_linux
# cp queueman.json /etc/queueman.json
# mkdir -p /var/log/queueman
```

### 各种环境运行

#### Linux
```
# /usr/bin/nohup /usr/local/bin/queueman_linux -c /etc/queueman.json > /var/log/queueman/error.log 2>&1 &
```

#### MacOS

``` sh
# sudo ./bin/queueman_macos -c ./queueman.json &
```

#### Systemctl
```sh
# cp queueman.service /usr/lib/systemd/system
# systemctl enable queueman.service
# systemctl start queueman.service
```

#### Supervisor
复制文件到 Supervisor 配制目录
```
cp supervisor.conf /etc/supervisor.d/queueman.conf
```

配置文件示例
```
[program:queueman]
process_name=%(program_name)s_%(process_num)02d
directory = /usr/local/bin/queueman_linux
command=/usr/local/bin/queueman_linux -c /etc/queueman.json
autostart=true
autorestart=true
user=root
numprocs=1
redirect_stderr=true
stdout_logfile_maxbytes = 100MB
stdout_logfile_backups = 20
stdout_logfile=/var/log/queueman/error.log
```

1. 把 Supervisor 配置文件放入 supervisor.d 目录
2. 使用如下命令启动
```
supervisorctl update
```
3. 状态查看
```
supervisorctl status queueman:
```
1. 启动与停止
```
supervisorctl start queueman:
supervisorctl stop queueman:
```

## 详细介绍

### 完整流程
1. 开发者推送数据到指定队列
2. Queuman根据配置取出数据
3. 通过 http(s) 分发至指定 URL 地址
4. 开发者在指定 URL 地址进行业务处理，并返回成功或者失败信息
5. 开发者返回成功（流程结束）
6. 开发者返回失败，Queueman会接收到失败响应，然后推送数据到失败延时重试队列
7. 失败延时重试队列到指定时间点，继续走 2 到 6 步

### 开发者做两件事就够了
1. 推送数据到指定队列
2. 在指定 URL 地址写业务处理代码，返回成功或者失败

### 开发者推送数据到队列
#### Redis 正常队列
```
LPUSH queue:test1 value1
```

#### Redis 延时队列
使用 zset 实现，要使用特定格式并 json encode 成字符串
```
ZADD queue:test2 NX 指定触发的Unix时间戳 {"uuid":"cada1c8d-503e-4a82-b463-2b7ce2d2816e","time":1585817621,"data":"redis delay data"}

示例
ZADD queue:test2 NX 1585817681 "{\"uuid\":\"cada1c8d-503e-4a82-b463-2b7ce2d2816e\",\"time\":1585817621,\"data\":\"redis delay data\"}"
```

Redis 标准 DelayData 格式
名称 | 说明 | 示例
------------ | ------------- | -------------
uuid | 唯一id(zset消息不允许重复) | "cada1c8d-503e-4a82-b463-2b7ce2d2816e"
time | 生成当前信息的 Unix 时间戳 | 1585817621
data | 实际要处理的值，指定 URL 只会收到这个内容，不含 uuid、time | "redis delay data"

#### RabbitMQ 正常队列
```
ch.Publish(
	exchangeName, // exchange
	routingKey,   // routing key
	false,        // mandatory
	false,        // immediate
	amqp.Publishing{
		ContentType:  "text/plain",
		Body:         []byte(body),
		DeliveryMode: deliveryMode,
	})
```

#### RabbitMQ 延时队列
```
header := amqp.Table{"x-delay": 10 * i * 1000}
if "aliyun" == strings.ToLower(sourceType) {
	header = amqp.Table{"delay": 10 * i * 1000}
}

ch.Publish(
	exchangeName, // exchange
	routingKey,   // routing key
	false,        // mandatory
	false,        // immediate
	amqp.Publishing{
		Headers:      header,
		ContentType:  "text/plain",
		Body:         []byte(body),
		DeliveryMode: deliveryMode,
	})
```

### 指定地址接收数据

Queueman 会使用 `POST` 方式把数据以 `application/x-www-form-urlencoded` 格式推送到 `queueman.json` 里每个队列指定的 `DispatchURL`

名称 | 说明 | 示例
------------ | ------------- | -------------
queueName | 队列名称 | queue:test
delayName | 正常队列失败后的延时处理队列名称 | queue:test:delay
queueData | 取出的队列数据 | {"name":"jhon", "age":"16"}

`UserAgent` 是 `QueueMan 版本号` 示例 `QueueMan V1.0.0` 注意版本号会升级

### 指定地址的响应
名称 | 说明 | 示例
------------ | ------------- | -------------
code | if success `1` else `0` | 1
message | if success `ok` else `fail` | ok

#### 成功请返回如下格式
```json
{"code":1,"message":"ok"}
```

#### 失败请返回如下格式
```json
{"code":0,"message":"fail"}
```

### 并发控制
`Concurency`      Queueman转发并发数，请根据服务器性能来设置
`DelayConcurency` 失败后的延时队列转发并发数，请根据服务器性能来设置

### 失败后处理
如果第一次处理失败，如有 DelayOnFailure 设置且长度大于0，则按配置依次重试，直到成功或者最后一次尝试失败。最后一次尝试失败后，丢弃队列数据
```
"DelayOnFailure": [60, 300]
```
如上配置表示第一次失败后，投递消息到失败延时处理队列，60秒（一分钟）后触发，如果再一次失败则`300-60=240`秒后触发（即第五分钟），再次失败则丢弃。

## 命令行工具
`-c` 指定配置文件路径
```
./queueman_linux -c ./queueman.json
```

`-t` 测试指定的配置文件是否正常（可以结合 `-c`一起使用）
```
./queueman_linux -c ./queueman.json -t
```

`-s` 在命令行下显示统计信息（可以结合 `-c`一起使用）
```
./queueman_linux -c ./queueman.json -s
```

`-h` 查看帮助信息
```
./queueman_linux -h

----------------------------------------------
	Welcome to Use QueueMan V0.0.1
----------------------------------------------

usage:
  -c string
    	the configure file path (default "./queueman.json")
  -h	show help information
  -s	show statistics information
  -t	test configure in "queueman.json" file
```

## 查看统计信息
### 命令行下查看
```
./queueman_linux -c ./queueman.json -s
```

### Web方式查看
返回 html 格式
```
curl "http://127.0.0.1:8080/statistic?format=html"
```

返回 json 格式
```
curl "http://127.0.0.1:8080/statistic?format=json"
```

## 配置文件详解
```
{
    "App": {								   # Queueman app 级别配置
        "IsDebug" : false,                     # true （会输出 info 级别信息） false （只输出 warn 级别及以上信息）
        "PIDFile" : "/var/run/queueman.pid",   # pid文件位置
        "LogFormatter": "json",                # 日志格式 text json 
        "LogDir"  : "/var/log/queueman/"       # 日志文件目录，为空表示输出到 stdout
    },
    "Statistic": {                             # 统计配置
        "HTTPPort": 8080,                      # Web 查看端口
        "SourceType": "Redis",                 # 统计到 Redis
        "RedisSource" : {
            "Network"       : "tcp",           # Redis 超时时间
            "Host"          : "127.0.0.1",     # Redis 地址
            "Port"          : 6379,            # Redis 端口
            "Password"      : "",              # Redis 密码，没有设置请置空
            "DB"            : 0,               # Redis DB 值
            "Timeout"       : 5,               # Redis 超时时间
            "MaxActive"     : 1000,
            "MaxIdle"       : 200,
            "MaxIdleTimeout": 10,
            "Wait"          : true
        }
    },
    "Redis": [
        {
            "Config" : {                       # 第一个 Redis 的连接配置，为同一节点下的 Queues 服务
                "Network"       : "tcp",       # Redis 超时时间
                "Host"          : "127.0.0.1", # Redis 地址
                "Port"          : 6379,        # Redis 端口
                "Password"      : "",          # Redis 密码，没有设置请置空
                "DB"            : 0,           # Redis DB 值
                "Timeout"       : 5,           # Redis 超时时间
                "MaxActive"     : 1000,
                "MaxIdle"       : 200,
                "MaxIdleTimeout": 10,
                "Wait"          : true
            },
            "Queues": [
                {
                    "IsEnabled": true,                         # 是否启用，若为 false 则不 Queueman 不处理
                    "IsDelayQueue": false,			           # 是否延时队列 true 是 false 否
                    "IsDelayRaw": false,                       # 延时队列是否初次读取时返回原始值，只有当延时队列zset格式不是Redis 标准 DelayData 格式时（比如自定义格式时）true 是 false 否
                    "QueueName": "queue:test1",                # 队列名称
                    "DispatchURL": "http://127.0.0.1/receive", # 接收队列数据的 http(s) 地址
                    "DispatchTimeout": 30,                     # Queueman转发超时时间
                    "Concurency": 5,                           # Queueman转发并发数，请根据服务器性能来设置
                    "DelayConcurency": 3,                      # 失败后的延时队列转发并发数，请根据服务器性能来设置
                    "DelayOnFailure": [60, 120]                # 单位（秒）失败后第N秒尝试，多个值则尝试多次。第一次失败后的秒数
                },
                {
                    "IsEnabled": true,
                    "IsDelayQueue": true,
                    "IsDelayRaw": false,
                    "QueueName": "queue:test2",
                    "DispatchURL": "http://127.0.0.1/receive",
                    "DispatchTimeout": 30,
                    "Concurency": 5,
                    "DelayConcurency": 3,
                    "DelayOnFailure": [60, 120]
                }
            ]
        },
		{
            "Config" : {                       # 第二个 Redis 的连接配置，为同一节点下的 Queues 服务
                "Network"       : "tcp",
                "Host"          : "127.0.0.1",
                "Port"          : 6379,
                "Password"      : "",
                "DB"            : 0,
                "Timeout"       : 5,
                "MaxActive"     : 1000,
                "MaxIdle"       : 200,
                "MaxIdleTimeout": 10,
                "Wait"          : true
            },
            "Queues": [
                {
                    "IsEnabled": true,
                    "IsDelayQueue": false,
                    "IsDelayRaw": false,
                    "QueueName": "queue:test3",
                    "DispatchURL": "http://127.0.0.1/receive",
                    "DispatchTimeout": 30,
                    "Concurency": 5,
                    "DelayConcurency": 3,
                    "DelayOnFailure": [60, 120]
                },
                {
                    "IsEnabled": true,
                    "IsDelayQueue": true,
                    "IsDelayRaw": false,
                    "QueueName": "queue:test4",
                    "DispatchURL": "http://127.0.0.1/receive",
                    "DispatchTimeout": 30,
                    "Concurency": 5,
                    "DelayConcurency": 3,
                    "DelayOnFailure": [60, 120]
                }
            ]
        }
    ],
    "RabbitMQ": [
        {
            "Config" : {											   # 第一个 RabbitMQ 连接配置，为同一节点下的 Queues 服务
                "Scheme"       : "amqp",                               # RabbitMQ协议 amqp、amqps
                "Host"          : "Replace your Host",                 # RabbitMQ 地址
                "Port"          : 5672,                                # RabbitMQ 端口
                "User"          : "",                                  # RabbitMQ 用户名（Type为 aliyun 里保持空）
                "Password"      : "",                                  # RabbitMQ 密码（Type为 aliyun 里保持空）
                "Vhost"         : "Replace your Vhost",                # RabbitMQ Vhost
                "Type"          : "Aliyun",                            # RabbitMQ 类型 为 空 或者 Aliyun
                "AliyunParams"  : {									   # RabbitMQ Aliyun 的参数配置，如非阿时云，这一行可以删除
                    "AccessKey"      : "Replace your AccessKey",       # RabbitMQ Aliyun 的 AccessKey
                    "AccessKeySecret": "Replace your AccessKeySecret", # RabbitMQ Aliyun 的 AccessKeySecret
                    "ResourceOwnerId": 1000000000000000                # RabbitMQ Aliyun 的 ResourceOwnerId
                }
            },
            "Queues" : [
                {
                    "IsEnabled": true,                                 # 是否启用，若为 false 则不 Queueman 不处理
                    "IsDelayQueue": false,                             # 是否延时队列 true 是 false 否
                    "IsDurable": true,                                 # 是否持久化
                    "ExchangeName": "test.exchange.direct1",           # 交换机名称
                    "ExchangeType": "direct",                          # 交换机类型
                    "QueueName": "test.queue.direct1",                 # 队列名称
                    "RoutingKey": "test.route.direct1",                # RoutingKey
                    "ConsumerTag": "test.consumer.direct1",            # 消费Tag
                    "IsAutoAck": false,                                # 是否自动确认 true (自动) false (Queueman接收到成功响应后触发
                    "DispatchURL": "http://127.0.0.1/receive",         # 接收队列数据的 http(s) 地址
                    "DispatchTimeout": 30,                             # Queueman转发超时时间
                    "Concurency": 5,                                   # Queueman转发并发数，请根据服务器性能来设置
                    "DelayConcurency": 3,                              # 失败后的延时队列转发并发数，请根据服务器性能来设置
                    "DelayOnFailure": [60, 120]                        # 单位（秒）失败后第N秒尝试，多个值则尝试多次。第一次失败后的秒数
                },
                {
                    "IsEnabled": true,
                    "IsDelayQueue": true,
                    "IsDurable": true,
                    "ExchangeName": "test.exchange.direct2",
                    "ExchangeType": "direct",
                    "QueueName": "test.queue.direct2",
                    "RoutingKey": "test.route.direct2",
                    "ConsumerTag": "test.consumer.direct2",
                    "IsAutoAck": false,
                    "DispatchURL": "http://127.0.0.1/receive",
                    "DispatchTimeout": 30,
                    "Concurency": 5,
                    "DelayConcurency": 3,
                    "DelayOnFailure": [60, 120]
                }
            ]
        },
        {
            "Config" : {											   # 第二个 RabbitMQ 标准连接配置，为同一节点下的 Queues 服务
                "Scheme"       : "amqp",                               # RabbitMQ协议 amqp、amqps
                "Host"          : "Replace your Host",                 # RabbitMQ 地址
                "Port"          : 5672,                                # RabbitMQ 端口
                "User"          : "Replace your User",                 # RabbitMQ 用户名（Type为 aliyun 里保持空）
                "Password"      : "Replace your Password",             # RabbitMQ 密码（Type为 aliyun 里保持空）
                "Vhost"         : "Replace your Vhost",                # RabbitMQ Vhost
                "Type"          : ""                                   # RabbitMQ 类型 为 空 或者 Aliyun
            },
            "Queues" : [
                {
                    "IsEnabled": true,
                    "IsDelayQueue": false,
                    "IsDurable": true,
                    "ExchangeName": "test.exchange.direct3",
                    "ExchangeType": "direct",
                    "QueueName": "test.queue.direct3",
                    "RoutingKey": "test.route.direct3",
                    "ConsumerTag": "test.consumer.direct3",
                    "IsAutoAck": false,
                    "DispatchURL": "http://127.0.0.1/receive",
                    "DispatchTimeout": 30,
                    "Concurency": 5,
                    "DelayConcurency": 3,
                    "DelayOnFailure": [60, 120]
                },
                {
                    "IsEnabled": true,
                    "IsDelayQueue": true,
                    "IsDurable": true,
                    "ExchangeName": "test.exchange.direct4",
                    "ExchangeType": "direct",
                    "QueueName": "test.queue.direct4",
                    "RoutingKey": "test.route.direct4",
                    "ConsumerTag": "test.consumer.direct4",
                    "IsAutoAck": false,
                    "DispatchURL": "http://127.0.0.1/receive",
                    "DispatchTimeout": 30,
                    "Concurency": 5,
                    "DelayConcurency": 3,
                    "DelayOnFailure": [60, 120]
                }
            ]
        }
    ]
}
```

## 维护者

[@marknown](https://github.com/marknown)

## 如何贡献

非常欢迎你的加入! [提一个Issue](https://github.com/marknown/queueman/issues/new) 或者提交一个 Pull Request.

## 使用许可

[MIT](LICENSE) © marknown
