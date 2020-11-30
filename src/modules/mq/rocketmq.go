package mq

import (
	"github.com/apache/rocketmq-client-go/core"
	"ays/src/modules/app"
	"ays/src/modules/consul"
	"fmt"
	"errors"
	"ays/src/modules/logger"
	"strconv"
	"time"
	"hash/crc32"
)

//延迟级别 1s 5s 10s 30s 1m 2m 3m 4m 5m 6m 7m 8m 9m 10m 20m 30m 1h 2h | 1->
var (
	clientConfig rocketmq.ClientConfig
	pullConsumer rocketmq.PullConsumer
	producer rocketmq.Producer 
)

type Msg rocketmq.Message

type mqClient rocketmq.ClientConfig

type queueSelectorByOrderID struct{}


// 顺序队列保证
func (s queueSelectorByOrderID) Select(size int, m *rocketmq.Message, arg interface{}) int {
    return arg.(int) % size
}


// 初始化客户端参数
func NewMqClient ( ) mqClient {
	
	return mqClient{
		GroupID:    app.Config.MQ.Group,
		NameServer: app.Config.MQ.Namesrv,
		LogC: &rocketmq.LogConfig{
			Path:     app.Config.MQ.LogPath,
			FileSize: int64(app.Config.MQ.LogFilesize),
			FileNum:  app.Config.MQ.LogFilenum,
			Level:    rocketmq.LogLevelDebug,
		},
	}
}



// 发送消息无序
func (cli mqClient) SendMessageSync (msg Msg) (error) {
	
	if err := cli.setProducerConf();err != nil {
		return err
	}

	rocketMsg := rocketmq.Message(msg)

	if _ , err := producer.SendMessageSync(&rocketMsg); err != nil {
		return err
	}

	return nil
}


// 只发送请求不等待应答
// 适用于某些耗时非常短，但对可靠性要求并不高的场景，例如日志收集
func (cli mqClient) SendMessageOneway (msg Msg) (error) {
	if err := cli.setProducerConf(); err != nil {
		return err
	}

	rocketMsg := rocketmq.Message(msg)

	if err := producer.SendMessageOneway(&rocketMsg); err != nil {
		return err
	}
	return nil
}


// 顺序发消息
// 使用Message.Keys保证分发到一个队列
func (cli mqClient) SendMessageOrderly(msg Msg) (error) {

	if err := cli.setProducerConf(); err != nil {
		return err
	}

	selector 	:= queueSelectorByOrderID{}
	rocketMsg 	:= rocketmq.Message(msg)

	if _ , err := producer.SendMessageOrderly( &rocketMsg, selector, topicHash(msg.Keys) , 1); err != nil {
		return err
	}

	return nil 
} 



// 
func (cli mqClient) setProducerConf () (error) {

  	if producer != nil {
    	return nil
	}

	config := &rocketmq.ProducerConfig{ClientConfig: rocketmq.ClientConfig(cli) }
	p , err := rocketmq.NewProducer(config)
	if err != nil {
		return err
	}
	producer = p

	if err = producer.Start(); err != nil {
		return err
	}
	return nil
}



// 推送消息
func (cli mqClient) ConsumeWithPush (topic string) (*rocketmq.MessageExt, error) {

	var msg *rocketmq.MessageExt
	var sign = 1;	
	if topic == "" {
		return msg , errors.New("topic cannot by empty")
	}

	config := &rocketmq.PushConsumerConfig{ClientConfig: rocketmq.ClientConfig(cli) }
	ch := make(chan interface{})

	consumer, err := rocketmq.NewPushConsumer(config)
	if err != nil {
		return msg,err
	}
	
	// MUST subscribe topic before consumer started.
	consumer.Subscribe(topic, "*", func(message *rocketmq.MessageExt) rocketmq.ConsumeStatus {
		fmt.Printf("-----%s-----\n", message.Body)
		msg = message
		if sign == 1 {
			ch <- "quit"
			sign = 2
		}
		return rocketmq.ConsumeSuccess
	})

	err = consumer.Start()
	if err != nil {
		return msg,err
	}
	
	fmt.Printf("consumer: %s started...\n", consumer)

	<- ch

	err = consumer.Shutdown()

	fmt.Printf("++++%s+++++\n", msg.Body)

	if err != nil {
		fmt.Printf("shutdown error...\n")
		return msg,err
	}

	fmt.Printf("func end\n")
	return msg,nil	
}


// 生成消费者
func (cli mqClient) NewPullConsume () error {
	config := &rocketmq.PullConsumerConfig{ClientConfig: rocketmq.ClientConfig(cli) }
	consumer,err := rocketmq.NewPullConsumer(config)
	pullConsumer = consumer
	if err = consumer.Start() ;err != nil {
		return err
	}
	return nil
}


// 拉数据
func (cli mqClient) ConsumeWithPull(topic string) (*rocketmq.Message,error) {
	
	if topic == "" {
		return nil , errors.New("topic must can not empty\n")
	}

	if pullConsumer == nil {
		if err := cli.NewPullConsume();err != nil {
			return nil , err 
		}
	}

	var messageExts []*rocketmq.MessageExt
	pullMaxNum 	:= 1
	msgNum 		:= 0
	
	mqs := pullConsumer.FetchSubscriptionMessageQueues(topic)

	fmt.Printf("Topic: %s   fetch subscription mqs:%+v\n", topic, mqs)

	PULL:	
	for _, mq := range mqs {
        
		pr := pullConsumer.Pull(mq, "*", getOffset(topic), pullMaxNum)

		if msgNum = len(pr.Messages);msgNum == 0 {
			continue
		}
		
		if pr.Status == rocketmq.PullNoNewMsg {
			break PULL	
		} 

		if pr.Status == rocketmq.PullBrokerTimeout {
			return nil , errors.New("broker timeout occur")
		}
	
		// 设置offset
		setOffset(topic,pr.NextBeginOffset) 
		messageExts = pr.Messages
	}
	if len(messageExts) > 0 {
		return &(messageExts[0].Message),nil
	}else {
		return nil,nil
	}
}


// 循环拉取数据
type HandleFunc func ( *rocketmq.Message )
func (cli mqClient) LoopPull(topic string ,handleFunc HandleFunc) (error) {

    for {
		msg,err := cli.ConsumeWithPull(topic)
		logger.IfError(err)
		if err != nil {
			return err
		}
		if msg == nil {
			time.Sleep(time.Duration(5)*time.Second)
		}else{
			handleFunc(msg)
		}
	}
	return nil
}


func setOffset(topic string , offset int64) (bool,error) {

    k := "mq_"+topic
    v := fmt.Sprintf("%d",offset)     
    return consul.KvSet(k,v) 
}


func getOffset(topic string) int64 {

    k := "mq_"+topic
	v , _ := strconv.ParseInt(consul.KvGet(k),10,64)
    return v
}


func (cli mqClient ) Clear () {
	if pullConsumer != nil {
		pullConsumer.Shutdown()
	}

	if  producer != nil {
		producer.Shutdown()
	}
}



func topicHash (s string) int {
	v := int(crc32.ChecksumIEEE([]byte(s)))
	if v >= 0 {
		return v
	}
	if -v >= 0 {
		return -v
	}
	return 0
}

