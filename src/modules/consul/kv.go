package consul

import (
	"encoding/json"
	"github.com/hashicorp/consul/api"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/logger"
)

func KvSetObj(key string, obj interface{}) (bool, error) {
	listJsonBytes, err := json.Marshal(obj)
	logger.IfError(err)

	if err != nil {
		return false, err
	} else {
		return KvSetJson(key, listJsonBytes)
	}
}

// 设置json类型值
func KvSetJson(key string, json []byte) (bool, error) {
	listJson := string(json[:])
	return KvSet(key, listJson)
}

// 设置kv
func KvSet(key string, val string) (bool, error) {
	kvPair := CreateKvPair(key, val)

	return SetKVPair(kvPair)
}

// 获取val
func KvGet(key string) string {
	kvPair, _, err := ConsulClient.KV().Get(key, &api.QueryOptions{})
	logger.IfError(err)

	if err != nil {
		return ""
	}

	if kvPair != nil {
		return string(kvPair.Value[:])
	} else {
		return ""
	}
}

// 删除kv
func KvDel(key string) (bool, error) {
	_, err := ConsulClient.KV().Delete(key, &api.WriteOptions{})
	if err != nil {
		return false, err
	} else {
		return true, nil
	}
}

// 获取kvPair
func GetKvPair(key string) *api.KVPair {
	kvPair, _, err := ConsulClient.KV().Get(key, &api.QueryOptions{})
	logger.IfError(err)
	return kvPair
}

// 构造kvPair
func CreateKvPair(key string, val string) *api.KVPair {
	return &api.KVPair{
		Key: key,
		Value: []byte(val),
		Flags: 0,
	}

}

// 通过前缀查询列表
func PreSearch(prefix string) []string {
	kvPairs, _, err := ConsulClient.KV().List(prefix, &api.QueryOptions{})
	logger.IfError(err)

	var kvs []string
	for _, kvPair := range kvPairs {
		kvs = append(kvs, string(kvPair.Value[:]))
	}

	return kvs
}

// 通过前缀查询key列表
func PreSearchKeys(prefix string) []string {
	kvPairs, _, err := ConsulClient.KV().List(prefix, &api.QueryOptions{})
	logger.IfError(err)

	keys := []string{}
	for _, kvPair := range kvPairs {
		keys = append(keys, string(kvPair.Key))
	}

	return keys
}

// 前缀查询列表KVPairs
func PreList(prefix string) api.KVPairs {
	kvPairs, _, err := ConsulClient.KV().List(prefix, &api.QueryOptions{})
	logger.IfError(err)
	return kvPairs
}

// 锁定并赋值KVPair
func LockKVPair(kvPair *api.KVPair) bool {
	res, _, err := ConsulClient.KV().Acquire(kvPair, &api.WriteOptions{})
	logger.IfError(err)
	return res
}

// 释放锁定并赋值KVPair
func ReleaseKVPair(kvPair *api.KVPair) bool {
	res, _, err := ConsulClient.KV().Release(kvPair, &api.WriteOptions{})
	logger.IfError(err)
	return res
}

// 设置值KVPair
func SetKVPair(kvPair *api.KVPair) (bool, error) {
	var status bool
	_, err := ConsulClient.KV().Put(kvPair, &api.WriteOptions{})
	logger.IfError(err)
	if err != nil {
		status = false
	} else {
		status = true
	}

	return status, err
}

// 检测并设置kv
func KVPairCheckAndSet(kvPair *api.KVPair, val string) (bool, error) {
	kvPair.Value = []byte(val)

	status, _, err := ConsulClient.KV().CAS(kvPair, &api.WriteOptions{})
	logger.IfError(err)

	return status, err
}