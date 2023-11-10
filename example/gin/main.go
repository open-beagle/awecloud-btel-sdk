package main

import (
	"fmt"
	"github.com/Shopify/sarama"
	"go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama"
	"go.opentelemetry.io/otel"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/open-beagle/awecloud-btel-sdk/btrace"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	if tracer := btrace.New(); tracer != nil {
		defer tracer.Shutdown()
	}
	router := gin.Default()
	router.Use(otelgin.Middleware(os.Getenv("BTEL_SERVICE_NAME")))
	router.GET("/user/:name", func(c *gin.Context) {
		name := c.Param("name")
		// kafka()
		c.String(http.StatusOK, "Hello %s", name)
	})
	router.Run(":8383")
}

func kafka() {
	config := sarama.NewConfig()
	//tailf包使用
	config.Producer.RequiredAcks = sarama.WaitForAll          //Producer生产者    发送完数据需要leader和follow都确认
	config.Producer.Partitioner = sarama.NewRandomPartitioner //Partitioner分区   新选出一个partition
	config.Producer.Return.Successes = true                   //成功交付的消息将在success channel返回
	config.Producer.MaxMessageBytes = 1

	producer, err := sarama.NewSyncProducer([]string{"127.0.0.1:9092"}, config)
	if err != nil {
		fmt.Println("Failed to start Sarama producer:", err)
		return
	}

	syncProducer := otelsarama.WrapSyncProducer(config, producer, otelsarama.WithTracerProvider(otel.GetTracerProvider()))

	//构造消息
	msg := &sarama.ProducerMessage{}
	msg.Topic = "web_log"
	msg.Value = sarama.StringEncoder("this is a test log 11111111111111111111111111111111111111111111111111111111111111111111111111111111111")

	//发送消息
	pid, offset, err := syncProducer.SendMessage(msg)
	if err != nil {
		fmt.Println("send msg failed, err:", err)
		return
	}
	fmt.Printf("pid:%v, offset:%v\n", pid, offset)
}
