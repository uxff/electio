package worker

import (
	"crypto/md5"
	"encoding/hex"
)

const regPreKey = "/myapp/"
const masterKey = "master"
const membersKey = "workers"

type clusterHelper struct {
	clusterId   string
	clusterSalt string
}

func NewClusterHelper(clusterId string, clusterSalt string) *clusterHelper {
	return &clusterHelper{
		clusterId,
		clusterSalt,
	}
}

func (c *clusterHelper) genClusterKeyBase() string {
	return regPreKey + c.clusterId + "/"
}

// 生成一个字符串key(比如redis key) 用于保存 worders
func (c *clusterHelper) genMasterIdKey() string {
	return c.genClusterKeyBase() + masterKey
}

// 生成一个字符串key(比如redis key) 用于保存 worders
func (c *clusterHelper) genMembersKey() string {
	return c.genClusterKeyBase() + membersKey
}

func (c *clusterHelper) genMemberHash(memberId string) string {
	s := memberId + "/" + c.clusterSalt
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
