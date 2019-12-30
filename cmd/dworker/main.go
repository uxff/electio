package main

import (
	"flag"
	worker "github.com/uxff/electio/dependentworker"
	"github.com/uxff/electio/dependentworker/redisrepo"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

const (
	DefaultHttpServerAddr = "127.0.0.1:8001"
)

func main() {
	//log.Printf("Hello Tester")

	httpAddr := DefaultHttpServerAddr
	clusterId := "c1"
	clusterMembers := ""
	dsn := ""

	flag.StringVar(&httpAddr, "httpaddr", httpAddr, "http addr to serve")
	flag.StringVar(&clusterId, "clusterId", clusterId, "cluster id")
	flag.StringVar(&clusterMembers, "members", clusterMembers, "cluster members, comma splited, like: 1.1.1.1:8005,1.1.1.2:8006")
	flag.StringVar(&dsn, "dsn", dsn, "store dsn url")
	flag.Parse()

	log.Printf("node httpaddr=%s clusterId=%s members=%s", httpAddr, clusterId, clusterMembers)

	repo := &redisrepo.RedisRepo{}
	repo.SetStoreUrl(dsn)

	workerNode := worker.NewWorker(httpAddr, clusterId, repo)
	workerNode.AddMates(strings.Split(clusterMembers, ","))

	// 准备启动服务
	serveErrorChan := make(chan error, 1)

	// start http server
	go func() {
		log.Printf("http server will start at %v", httpAddr)
		serveErrorChan <- workerNode.ServePingable()
	}()

	// start cluster node
	go func() {
		log.Printf("worker server will start ")
		serveErrorChan <- workerNode.Start()
	}()

	// 监听信号，先关闭rpc服务，再关闭消息队列
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL, syscall.SIGHUP, syscall.SIGQUIT)

	select {
	case sig := <-ch:
		log.Printf("receive signal '%v', server will exit", sig)
		workerNode.Quit()
	}

	os.Exit(1)
}
