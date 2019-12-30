package dependentworker

import (
	"encoding/json"
	"fmt"
	"github.com/uxff/electio/dependentworker/repo"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"
)

const RegisterTimeoutSec = 5    // 已注册的超时检测
const RegisterIntervalSec = 2   // 作为worker或master注册间隔
const PingMasterIntervalSec = 2 // ping master 的时间间隔
const PingMasterErrLimit = 2    // ping master 超过这个次数会重新选举

type Worker struct {
	Id          string // from redis incr? // uniq in cluster, different from other nodes
	ClusterId   string
	ServiceAddr string // self addr of listen, must be addressable for other nodes
	//Port           string // self port of listen, must be addressable for other nodes
	MasterId       string
	LastRegistered int64 // timestamp

	ClusterMembers map[string]*Worker `json:"-"`

	repo               repo.Repo
	clusterAssist      *clusterHelper
	keepRegisterStatus int
}

func NewWorker(serviceAddr string, clusterId string) *Worker {
	w := &Worker{}
	w.ServiceAddr = serviceAddr
	w.ClusterId = clusterId

	w.clusterAssist = NewClusterHelper(w.ClusterId)
	w.ClusterMembers = make(map[string]*Worker, 0)
	// redis must be prepared already

	w.Id = w.clusterAssist.genMemberHash(w.ServiceAddr)

	return w
}

func (w *Worker) Start() error {
	log.Printf("worker %s will start", w.Id)

	go w.KeepRegistered() // todo with context

	// 等待别的worker注册成功
	time.Sleep(time.Millisecond * 20)

	errTimes := 0
	for {
		masterNode := w.FindMaster()

		if masterNode == nil {
			w.FindMembers()
			// 自己选举 按协约选举
			// 找到master后，将别人的master清空？
			masterNode = w.ElectMaster()
			log.Printf("master is elected:%v", masterNode)
		}

		var err error

		if masterNode == nil {
			log.Printf("worker(%s) cannot elect master", w.Id)
			//return fmt.Errorf("worker(%s) has no master")
			err = fmt.Errorf("worker(%s) cannot elect master", w.Id)
		} else {
			log.Printf("worker(%s) find a master:%s, then follow", w.Id, w.MasterId)

			w.Follow(masterNode)

			if w.Id == masterNode.Id {
				w.PerformMaster()
			}

			err = w.KeepPingMaster()
		}

		errTimes++

		log.Printf("master or members busy? keep smile and try again (%d times). %v", errTimes, err)

		if errTimes > PingMasterErrLimit {
			// 擦掉master立即重选 // todo alarm out
			w.EraseRegisteredMaster()
			log.Printf("master registered is incredible, erased, will elect new")
			errTimes = 0
			continue
			//break
		}

		time.Sleep(time.Millisecond * 500)
	}

	return fmt.Errorf("worker(%s) ping master(%s) error too many times", w.Id, w.MasterId)
}

// redis hashkey: /nota/clusterId.clusterSalt = [md5(addr:port/clusterId+salt):{workerInfo}]
func (w *Worker) Register() {
	// 从redis注册id
	w.LastRegistered = time.Now().Unix()
	w.repo.MapSet(w.clusterAssist.genMembersKey(), w.Id, w.ToString())

	// keep register master if one is master
	if w.IsMaster() {
		w.repo.Set(w.Id, w.ToString())
	}
}

func (w *Worker) KeepRegistered() {
	// 保持注册成功
	m := sync.Mutex{}
	m.Lock()
	defer m.Unlock()
	if w.keepRegisterStatus == 0 {
		w.keepRegisterStatus = 1
	} else {
		// its already running
		log.Printf("id:%s's keepRegistered is already running", w.Id)
		return
	}

	for {
		// register self

		w.Register()

		log.Printf("id:%s is registered, master:%s", w.Id, w.MasterId)

		// refresh load members
		w.FindMembers()

		time.Sleep(time.Second * RegisterIntervalSec)
	}
}

// 从内存种发现
func (w *Worker) FindMaster() *Worker {
	val := w.repo.Get(w.Id)

	if len(val) == 0 {
		return nil
	}

	target := &Worker{}
	err := json.Unmarshal([]byte(val), target)
	if err != nil {
		log.Printf("master from redis unmarshall error:%v", err)
		return nil
	}

	regTimeDiff := time.Now().Unix() - target.LastRegistered
	if regTimeDiff > RegisterTimeoutSec {
		log.Printf("master(%s) has time out for %d sec", target.Id, regTimeDiff)
		return nil
	}

	return target
}

func (w *Worker) Follow(target *Worker) {
	if target.MasterId != "" && target.MasterId != target.Id {
		// 跟随主人的主人
		//return w.Follow(target.MasterId)
		log.Printf("will follow master(%s)'s master(%s)?", target.Id, target.MasterId)
	}

	w.MasterId = target.Id
	w.Register()
	// as same as PerformFollower
}

func (w *Worker) FindMembers() {
	matesInRedis := w.repo.MapGetAll(w.clusterAssist.genMembersKey())

	w.ClusterMembers = make(map[string]*Worker, 0)

	now := time.Now().Unix()

	for makeId, mateValInRedis := range matesInRedis {
		mate := new(Worker)
		json.Unmarshal([]byte(mateValInRedis), mate)

		regTimeDiff := now - mate.LastRegistered
		if regTimeDiff > RegisterTimeoutSec {
			log.Printf("mate(%s) register has time out for %d sec, ignore", mate.Id, regTimeDiff)
			if regTimeDiff > RegisterTimeoutSec*2 {
				w.EraseRegisteredWorker(mate.Id)
			}
			continue
		}

		w.ClusterMembers[makeId] = mate
	}
}

func (w *Worker) ElectMaster() *Worker {
	if len(w.ClusterMembers) == 0 {
		// must use self
		w.MasterId = w.Id
		return w
	}

	allMateIds := make([]string, len(w.ClusterMembers))
	idx := 0
	for mateId := range w.ClusterMembers {
		allMateIds[idx] = mateId
		idx++
	}

	sort.Strings(allMateIds)

	expectedMasterId := allMateIds[0]

	//w.MasterId = expectedMasterId
	log.Printf("w(%v) elected master:%v", w.Id, expectedMasterId)

	return w.ClusterMembers[expectedMasterId]
}

func (w *Worker) PerformMaster() {
	if !w.IsMaster() {
		return
	}

	log.Printf("worker %s will perform master", w.Id)

	// register to master
	w.repo.Set(w.Id, w.ToString())

	// ping every mates
	// order to every mates
}

func (w *Worker) EraseRegisteredMaster() {

	// register to master
	w.repo.Set(w.Id, "")
}

func (w *Worker) EraseRegisteredWorker(workerId string) {

	// register to master
	w.repo.MapSet(w.clusterAssist.genMembersKey(), workerId, "")
}

func (w *Worker) KeepPingMaster() error {
	for {

		err := w.PingNode(w.ClusterMembers[w.MasterId])
		if err != nil {
			return err
		}

		log.Printf("worker(%s) ping master(%s) ok", w.Id, w.MasterId)

		time.Sleep(time.Second * RegisterIntervalSec)
	}
}

func (w *Worker) PingNode(target *Worker) error {

	if target == nil {
		return fmt.Errorf("worker(%s) has no target when pingNode", w.Id)
	}

	targetUrl := target.ServiceAddr
	if len(targetUrl) <= 4 {
		return fmt.Errorf("worker(%s) ping target(%s)'s serviceAddr illegal", w.Id, target.Id)
	}
	if targetUrl[:4] != "http" {
		targetUrl = "http://" + targetUrl
	}
	_, err := http.Get(targetUrl)
	return err
}

//
func (w *Worker) ToString() string {
	buf, _ := json.Marshal(w)
	return string(buf)
}

func (w *Worker) IsMaster() bool {
	return w.Id == w.MasterId
}

