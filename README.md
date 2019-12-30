# electio

This project implements a distributed election algorithm.
Without dependency, each node conversations with mates by http protocol.
Distributed election algorithm, no runtime dependence.

### Work steps:

- 1.Start the goroutine 1 to Ping the peer node regularly.
- 2.Start goroutine 2 to prepare to perform master function. If the current node is not a master, the goroutine is idle.
- 3. The master goroutine checks the survival status of other nodes in the cluster. After more than half of the nodes are started:
- 3.1. check whether there is a master in the existing cluster, and follow if there is;
- 3.2.if there is no master after step 3.1, select a master according to the agreement and follow it. In this step, a master must be selected;
- 3.3.if the master is himself after step 3.2, perform the master role and ask his peers to follow him;
- 3.4.if the master node selected in step 3.2 follows another master 2, then forward following master 2 directly, and so on;
- 4.The timeout can be set for the mutual Ping of goroutine 1. After the timeout, the peer is marked as inactive.
- 5.If master timeout is found in goroutine 1, the master goroutine will be notified to initiate election.


### Communication protocol:

Request content:
```
GET /ping?fromId=xxxx&masterId=xxxx
```

Respond contents:
```
type PingRes struct {
    Code int        // codes
    Msg string      // message
    WorkerId string // workerId of responder
    MasterId string // masterId of responder
    Members map [string] * worker / / map [masterid] * matched by worker responder
}

```


### Constants

```
//Registered nodes, detect their timeout. After the normal node times out, it is marked as inactive. After the master time out, it will start the election.
const RegisterTimeoutSec = 2

//Time interval for periodic registration as worker to peer
const RegisterIntervalSec = 1
```