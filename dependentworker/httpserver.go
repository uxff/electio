package dependentworker

import (
	"github.com/gin-gonic/gin"
	"strings"
)

func (w *Worker) ServePingable() error {

	router := gin.Default()

	router.GET("/", func(c *gin.Context) {
		w.jsonOk(c, nil)
	})

	// 接收对方的ping 表示良好
	router.GET("/ping", func(c *gin.Context) {
		w.jsonOk(c, nil)
	})

	// 增加节点 支持批量添加
	// @param nodes=http://127.0.0.1:10010,http://127.0.0.1:10011
	router.GET("/add", func(c *gin.Context) {
		nodesStr := c.GetString("nodes")
		if nodesStr == "" {
			w.jsonError(c, "nodes must not be empty", nil)
			return
		}

		nodesArr := strings.Split(nodesStr, ",")
		for _, node := range nodesArr {
			mate := NewWorker(node, w.ClusterId)
			w.ClusterMembers[mate.Id] = mate
		}
	})

	// 删除节点
	router.GET("/remove", func(c *gin.Context) {
		node := c.GetString("node")
		if node == "" {
			w.jsonError(c, "node must no be empty", nil)
			return
		}

		delete(w.ClusterMembers, node)
		w.jsonOk(c, nil)
	})

	return router.Run(w.ServiceAddr)

}

func (w *Worker) jsonError(c *gin.Context, msg string, data interface{}) {
	c.JSON(200, gin.H{
		"code":     1,
		"msg":      msg,
		"workerId": w.Id,
		"members":  w.ClusterMembers,
		"data":     data,
	})
}
func (w *Worker) jsonOk(c *gin.Context, data interface{}) {
	c.JSON(200, gin.H{
		"code":     0,
		"msg":      "ok",
		"workerId": w.Id,
		"members":  w.ClusterMembers,
		"data":     data,
	})
}
