package main

import (
	inet "goim/libs/net"
	"goim/libs/net/xrpc"
	"goim/libs/proto"
	"strconv"
	"time"

	log "github.com/thinkboy/log4go"
)

var (
	logicRpcClient *xrpc.Clients
	logicRpcQuit   = make(chan struct{}, 1)

	logicService           = "RPC"
	logicServicePing       = "RPC.Ping"
	logicServiceConnect    = "RPC.Connect"
	logicServiceDisconnect = "RPC.Disconnect"
)

func InitLogicRpc(addrs []string) (err error) {
	var (
		bind          string
		network, addr string
		rpcOptions    []xrpc.ClientOptions
	)
	for _, bind = range addrs {
		if network, addr, err = inet.ParseNetwork(bind); err != nil {
			log.Error("inet.ParseNetwork() error(%v)", err)
			return
		}
		options := xrpc.ClientOptions{
			Proto: network,
			Addr:  addr,
		}
		rpcOptions = append(rpcOptions, options)
	}
	// rpc clients
	logicRpcClient = xrpc.Dials(rpcOptions)
	// ping & reconnect
	logicRpcClient.Ping(logicServicePing)
	guluLogger.Infof("init logic rpc: %v", rpcOptions)
	return
}

func connect(p *proto.Proto) (key string, rid int32, heartbeat time.Duration, err error) {
	var (
		arg   = proto.ConnArg{Token: string(p.Body), Server: Conf.ServerId}
		reply = proto.ConnReply{}
	)
	if err = logicRpcClient.Call(logicServiceConnect, &arg, &reply); err != nil {
		guluLogger.Errorf("c.Call(\"%s\", \"%v\", &ret) error(%v)", logicServiceConnect, arg, err)
		return
	}

	key = reply.Key
	rid = reply.RoomId
	guluLogger.Debug("connected! key is :" + key + "roomId is :" + strconv.Itoa(int(reply.RoomId)))
	//heartbeat = 1 * 60 * time.Second
	heartbeat = 24 * 60 * 60 * time.Second //TODO:心跳时间改成24小时
	return
}

func disconnect(key string, roomId int32) (has bool, err error) {
	var (
		arg   = proto.DisconnArg{Key: key, RoomId: roomId}
		reply = proto.DisconnReply{}
	)
	if err = logicRpcClient.Call(logicServiceDisconnect, &arg, &reply); err != nil {
		guluLogger.Errorf("c.Call(\"%s\", \"%v\", &ret) error(%v)", logicServiceConnect, arg, err)
		return
	}
	guluLogger.Debug("disconnect! key is :" + key + "roomId is :" + strconv.Itoa(int(roomId)))
	has = reply.Has
	return
}
