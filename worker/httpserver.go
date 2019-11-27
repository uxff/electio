package worker

import (
	"github.com/gin-gonic/gin"
	"log"
	"strings"
)

func (w *Worker) ServePingable() error {

	router := gin.Default()

	router.GET("/", func(c *gin.Context) {
		w.jsonOk(c, nil)
	})

	// 接收对方的ping 表示良好
	router.GET("/ping", func(c *gin.Context) {
		fromId := c.Query("fromId")
		if fromId == "" {
			w.jsonError(c, "fromId must no be empty", nil)
			return
		}

		if _, ok := w.ClusterMembers[fromId]; !ok {
			w.jsonError(c, "fromId:"+fromId+" not exist", nil)
			return
		}

		masterId := c.Query("masterId")
		//if masterId == "" {
		//	w.jsonError(c, "masterId must no be empty", nil)
		//	return
		//}

		w.RegisterIn(fromId, masterId)

		w.jsonOk(c, nil)
	})

	// 增加节点 支持批量添加
	// @param nodes=http://127.0.0.1:10010,http://127.0.0.1:10011
	router.GET("/add", func(c *gin.Context) {
		nodesStr := c.Query("nodes")
		if nodesStr == "" {
			w.jsonError(c, "nodes must not be empty", nil)
			return
		}

		nodesArr := strings.Split(nodesStr, ",")
		// todo 通知别人add
		w.AddMates(nodesArr)

		w.jsonOk(c, nil)
	})

	// 删除节点
	router.GET("/remove", func(c *gin.Context) {
		nodeId := c.Query("nodeId")
		if nodeId == "" {
			w.jsonError(c, "nodeId must no be empty", nil)
			return
		}

		if nodeId == w.Id {
			w.Quit()
			w.jsonOk(c, nil)
			return
		}

		delete(w.ClusterMembers, nodeId)
		w.jsonOk(c, nil)
	})

	// 被命令跟随某个master
	router.GET("/follow", func(c *gin.Context) {
		fromId := c.Query("fromId")
		if fromId == "" {
			w.jsonError(c, "fromId must no be empty", nil)
			return
		}

		masterId := c.Query("masterId")
		if masterId == "" {
			w.jsonError(c, "masterId must no be empty", nil)
			return
		}

		if _, ok := w.ClusterMembers[masterId]; !ok {
			w.jsonError(c, "masterId:"+masterId+" not exist", nil)
			return
		}

		if err := w.PingNode(masterId); err != nil {
			w.jsonError(c, "node ping error:"+err.Error(), nil)
			return
		}

		w.Follow(masterId)
		log.Printf("%s demand me follow: %s", fromId, masterId)
		w.jsonOk(c, nil)

	})

	// 删除master 重新选举
	router.GET("/erasemaster", func(c *gin.Context) {
		masterId := c.Query("masterId")
		if masterId == "" {
			w.jsonError(c, "node must no be empty", nil)
			return
		}

		w.MasterId = ""
		//w.masterGoneChan <- true
		log.Printf("erasemaster: %s", masterId)
		w.jsonOk(c, nil)
	})

	// 其他节点向本节点提交其投票
	//router.GET("/collectvotedmaster", func(c *gin.Context) {
	//	fromId := c.Query("fromId")
	//	if fromId == "" {
	//		w.jsonError(c, "fromId must no be empty", nil)
	//		return
	//	}
	//
	//	voteId := c.Query("voteId")
	//	if voteId == "" {
	//		w.jsonError(c, "voteId must no be empty", nil)
	//		return
	//	}
	//
	//	if _, ok := w.ClusterMembers[fromId]; !ok {
	//		w.jsonError(c, "fromId:"+fromId+" not exist", nil)
	//		return
	//	}
	//
	//	w.ClusterMembers[fromId].VotedMasterId = voteId
	//	log.Printf("collect from %s voted master %s", fromId, voteId)
	//	w.jsonOk(c, nil)
	//})

	return router.Run(w.ServiceAddr)

}

func (w *Worker) jsonError(c *gin.Context, msg string, data interface{}) {
	c.JSON(200, gin.H{
		"code":     1,
		"msg":      msg,
		"workerId": w.Id,
		"members":  w.ClusterMembers,
		"data":     data,
		"masterId": w.MasterId,
	})
}
func (w *Worker) jsonOk(c *gin.Context, data interface{}) {
	c.JSON(200, gin.H{
		"code":     0,
		"msg":      "ok",
		"workerId": w.Id,
		"members":  w.ClusterMembers,
		"data":     data,
		"masterId": w.MasterId,
	})
}
