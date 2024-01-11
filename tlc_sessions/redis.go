package main

import (
	"context"
	"encoding/json"

	"github.com/fetristan/tlc_sessions/sessionsservice"
	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()
var rdb *redis.Client

func connectToRedisDatabase(host string, port string, password string, db int) {
	rdb = redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,
		Password: password,
		DB:       db,
	})
}

/*func setRedisDatabaseSession(session sessionsservice.Session) {
	jsonStr, _ := json.Marshal(session)
	err := rdb.Set(ctx, session.CallerUid+session.CalleeUid, jsonStr, 0).Err()
	if err != nil {
		panic(err)
	}
	log.Debugf("Redis : SET : %s", jsonStr)
}*/

func setRedisDatabaseSessions(sessions []sessionsservice.Session) {
	_, err := rdb.Del(ctx, "tlc_sessions").Result()
	if err != nil {
		panic(err)
	}
	jsonStr, _ := json.Marshal(sessions)
	err = rdb.Set(ctx, "tlc_sessions", jsonStr, 0).Err()
	if err != nil {
		panic(err)
	}
}

func getRedisDatabaseSessions() ([]sessionsservice.Session, bool) {
	val, err := rdb.Get(ctx, "tlc_sessions").Result()
	var redisSessions []sessionsservice.Session
	if err == redis.Nil {
		log.Debugf("Redis : SESSIONS NOT FOUND")
	} else if err != nil {
		panic(err)
	} else {
		log.Debugf("Redis : FOUND : %s", val)
	}
	json.Unmarshal([]byte(val), &redisSessions)
	return redisSessions, (err != redis.Nil && err != nil)
}

/*func getRedisDatabaseSession(session sessionsservice.Session) (sessionsservice.Session, bool) {
	val, err := rdb.Get(ctx, session.CallerUid+session.CalleeUid).Result()
	var redisSession sessionsservice.Session
	if err == redis.Nil {
		log.Debugf("Redis : NOT FOUND %s", session.CallerUid+session.CalleeUid)
	} else if err != nil {
		panic(err)
	} else {
		log.Debugf("Redis : FOUND : %s", val)
	}
	json.Unmarshal([]byte(val), &redisSession)
	return redisSession, err != redis.Nil
}

func delRedisDatabaseSession(session sessionsservice.Session) {
	val, err := rdb.Del(ctx, session.CallerUid+session.CalleeUid).Result()
	if err != nil {
		panic(err)
	} else {
		log.Debugf("Redis : DELETE : %s", val)
	}
}*/
