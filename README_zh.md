# electio

分布式选举算法，没有运行时依赖。

### 工作步骤：

- 1、启动协程1定时ping同伴节点。
- 2、启动协程2预备执行master职能。当前节点不是master，则协程空闲。
- 3、主协程检查集群中其他节点的存活状态。有一半以上节点启动后：
 - 3.1 检查已存在集群中是否已经有master,有则follow；
 - 3.2 如果3.1后无master，则按照约定选举出一个master，并follow，本步骤一定会出一个master；
 - 3.3 如果3.2后master是自己，则履行master职能，并要求同伴们follow自己；
 - 3.4 如果3.2选出的master节点还follow其他master2，则直接follow master2，以此类推；
- 4、协程1相互ping可以设置超时时间，超时后同伴被标记为Inactive状态。
- 5、协程1中发现master超时，则通知主协程发起选举。

### 通信协议：
请求内容：
```
GET /ping?fromId=xxxx&masterId=xxxx
```
响应内容：
```
type PingRes struct {
	Code int            // 
	Msg string          // 
	WorkerId string     //
	MasterId string     // 
	Members map[string]*Worker  // map[masterId] *Worker 响应者的mated
}

```

### 常量配置
```
// 已注册的节点，检测其超时时间。普通节点超时后标记为Inactive,master超时后会出发选举。
const RegisterTimeoutSec = 2  

// 作为worker向同伴周期性注册的时间间隔
const RegisterIntervalSec = 1 
```
