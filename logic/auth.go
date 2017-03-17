package main

import (
	"encoding/json"
	"goim/libs/define"
	"sync/atomic"
)

// developer could implement "Auth" interface for decide how get userId, or roomId
type Auther interface {
	Auth(token string) (userId int64, roomId int32)
}

type DefaultAuther struct {
}

func NewDefaultAuther() *DefaultAuther {
	return &DefaultAuther{}
}

//{"uid":1,"rid":1}
func (a *GuluAuther) Auth(token string) (userId int64, roomId int32) {
	// var err error
	// if userId, err = strconv.ParseInt(token, 10, 64); err != nil {
	// 	userId = 0
	// 	roomId = define.NoRoom
	// } else {
	// 	roomId = 1 // only for debug
	// }
	// return
	guluLogger.Info("token is " + token)
	var user GuLuAuthInfo
	if err := json.Unmarshal([]byte(token), &user); err != nil {
		guluLogger.Error("反序列化失败" + err.Error())
		userId = -1
		roomId = define.NoRoom
	} else {
		userId = user.UserId
		roomId = user.RoomId // only for debug
	}
	if userId <= 0 {
		// must positive
		// because router use userId for index it's session array
		// use increament uid for anonymous user
		userId = a.anonymousId()
	}

	guluLogger.Debug(user)
	return

}

type GuluAuther struct {
	nextAnonymousId int64
}

func NewGuluAuther() *GuluAuther {
	return &GuluAuther{}
}

type GuLuAuthInfo struct {
	RoomId int32 `json:"roomId"`
	UserId int64 `json:"userId,omitempty"`
}

func (a *GuluAuther) anonymousId() int64 {
	return atomic.AddInt64(&a.nextAnonymousId, 1)
}
