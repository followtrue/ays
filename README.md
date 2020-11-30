# Ays 调度平台

#### Tips
###### 1.首次拉取，需要安装依赖包
```shell
# 安装govendor
go get -u github.com/kardianos/govendor
# 拉取依赖
govendor sync
```
###### 2.通过数据库表结构生成model文件
```shell
xorm reverse mysql root:123456@(127.0.0.1:3306)/ays?charset=utf8mb4 ./config/xorm_templates ./src/models
```
###### 3.更新go-bind文件
修改config中配置后必须执行
```shell
./go-bindata -o src/libs/env/bindata.go config/...
sed -i 's/package main/package env/g' src/libs/env/bindata.go
```
当报错./go-bindata: Permission denied时，执行`chmod +x ./go-bindata`

###### 4.注册中心逻辑（node部分）：
- （1）注册中心可手动操作增加节点--kv中的nodelist增加一条记录
- （2）执行器node启动时-kv中nodelist增加一条记录（有去重）
- （3）执行器node启动时-用ip+port去consul注册自己
- （4）consul根据watch配置监听kv变更，有变更会通知注册中心。注册中心发现有新增时会增加一个node健康状态的监听（使用WatchNodeList方法）
- （5）node健康状态有变化时，监听会产生触发回调，修改数据库状态

###### 5.定时任务逻辑（列表对比机制）：
- （1）概念：go定时任务列表；ays_timer_list定时任务记录中间表；全部任务列表
- （2）定时触发：对比 go定时任务列表和ays_timer_list，互相剔除无效的
- （3）定时触发：对比全部任务列表和ays_timer_list中间表，从而新增和更新定时任务

5.打包和安装
- （1）打包
```shell
（1）本地有go环境，在项目根目录下执行
make
（2）使用docker
docker run -it --rm -v ${ays_path}:/go/src/gitlab.keda-digital.com/kedadigital/ays golang /bin/bash /go/src/gitlab.keda-digital.com/kedadigital/ays/make.sh
${ays_path} 指ays项目根目录
# 将会在{项目根目录}/cmd/bin/目录下生成各个环境打包文件
```
- （2）安装

`以uat环境安装为例，先将uat_ays.tar.gz复制到目标主机并解压`

manager 安装
```shell
# 以下命令需要使用root用户执行或者直接用sudo

# 如果是安装调度中心
./installConfig manager -e uat -p 172.21.16.81 -P 18080 # 安装连接rocketmq必须类库
./aysManager install -p 172.21.16.81 -P 18080 # 安装调度器
./aysManager start | stop | status | restart # install后可直接用service命令，也可用此文件执行
./aysManager debug -p 172.21.16.81 -P 18080 # 调试模式

# 调度器可使用service命令管理
service aysmanager start | stop | status | restart | reload（不中断服务重启）
```
node 安装
```shell
# 以下命令需要使用root用户执行或者直接用sudo

# 如果是安装执行器
./installConfig node -e uat # 安装连接rocketmq必须类库
./aysNode install -g node_group_ksa -p 172.21.16.121 -P 18181 # 安装执行器,-g参数用于指定执行器组
./aysNode start | stop | status | restart # install后可直接用service命令，也可用此文件执行
./aysNode debug -g node_group_ksa -p 172.21.16.140 -P 18181 # 调试模式

# 执行器可使用service命令管理
service aysnode start | stop | status | restart | reload（不中断服务重启）
```

修改环境配置-MySQL、RocketMQ、Consul
- 新增环境时，需要在cmd/installConfig/installConfig.go中的envMap增加文件名
- 修改配置，修改config/env中的环境对应文件
- [更新go-bind文件](#3.更新go-bind文件)

升级
```shell
# （1）将可执行文件替换现运行的文件
# （2）若需要不中断当前服务重启，使用service aysmanager | aysnode reload
# （3）若代码未变更就需要重新执行install，在service aysmanager | aysnode restart
```