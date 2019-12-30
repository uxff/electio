package redisrepo

import (
	"github.com/garyburd/redigo/redis"
	"github.com/uxff/electio/log"
	"time"
)

type RedisRepo struct {
	storeDsn string
	serverAddr string
	serverPort string
	password string
	dbnumber string

	maxConn int

	redisPool map[string]*redis.Pool
	redisConn *redis.Conn
}

func (r *RedisRepo) SetStoreUrl(dsn string) error {
	r.storeDsn = dsn

	r.redisPool[dsn] = &redis.Pool{
		MaxIdle:     500,
		MaxActive:   r.maxConn,
		Wait:        true,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			tStart := time.Now()
			c, err := redis.Dial("tcp", r.serverAddr)
			if err != nil {
				return nil, err
			}
			if r.password != "" {
				if _, err := c.Do("AUTH", r.password); err != nil {
					c.Close()
					return nil, err
				}
			}

			if _, err := c.Do("SELECT", r.dbnumber); err != nil {
				c.Close()
				return nil, err
			}

			if time.Now().Sub(tStart).Seconds() > 1 {
				log.Warnf("Dial Redis too long lookup?? %v", time.Now().Sub(tStart))
			}

			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	return nil
}

func (r *RedisRepo) Get(key string) string {
	// todo expire
	conn := r.redisPool[r.storeDsn].Get()
	v, err := redis.String(conn.Do("GET", key))
	if err != nil {
		log.Errorf("todo ")
		return ""
	}

	return v
}
func (r *RedisRepo) Set(key, val string) {
	// todo expire
	conn := r.redisPool[r.storeDsn].Get()
	v, err := redis.String(conn.Do("SET", key, val))
	if err != nil {
		log.Errorf("todo ")
		//return ""
	}

	//return v
}
