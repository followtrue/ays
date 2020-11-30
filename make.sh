#!/bin/bash
work_dir=$(dirname $(readlink -f $0))
cd ${work_dir}
mkdir -p /usr/local/lib64/rocketmq/
cp config/rocketmq_cpp/librocketmq.a /usr/local/lib64/rocketmq/librocketmq.a
cp config/rocketmq_cpp/librocketmq.so /usr/local/lib64/rocketmq/librocketmq.so
ln -s /usr/local/lib64/rocketmq/librocketmq.a /lib64/librocketmq.a
ln -s /usr/local/lib64/rocketmq/librocketmq.so /lib64/librocketmq.so
mkdir -p /usr/local/include
cp -r config/rocketmq_cpp/rocketmq /usr/local/include/
echo "/usr/local/lib64/rocketmq" > /etc/ld.so.conf.d/rocketmq.conf
ldconfig
make