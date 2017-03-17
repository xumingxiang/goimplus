package main

import (
	"flag"
	"goim/libs/perf"
	"infrastructure/loggingclient"
	"runtime"

	log "github.com/thinkboy/log4go"
)

var (
	DefaultStat *Stat
	guluLogger  loggingclient.ILog
)

func init() {
	guluLogger = loggingclient.GetLogger("goim.job")
}

func main() {
	flag.Parse()
	if err := InitConfig(); err != nil {
		panic(err)
	}
	log.LoadConfiguration(Conf.Log)
	runtime.GOMAXPROCS(runtime.NumCPU())
	perf.Init(Conf.PprofAddrs)
	DefaultStat = NewStat()
	// comet
	err := InitComet(Conf.Comets,
		CometOptions{
			RoutineSize: Conf.RoutineSize,
			RoutineChan: Conf.RoutineChan,
		})
	if err != nil {
		guluLogger.Warn("comet rpc current can't connect, retry")
	}
	// start monitor
	if Conf.MonitorOpen {
		InitMonitor(Conf.MonitorAddrs)
	}
	// round
	round := NewRound(RoundOptions{
		Timer:     Conf.Timer,
		TimerSize: Conf.TimerSize,
	})
	// room
	InitRoomBucket(round,
		RoomOptions{
			BatchNum:   Conf.RoomBatch,
			SignalTime: Conf.RoomSignal,
			IdleTime:   Conf.RoomIdle,
		})
	//room info
	MergeRoomServers()
	go SyncRoomServers()
	InitPush()
	if err := InitKafka(); err != nil {
		panic(err)
	}
	// block until a signal is received.
	InitSignal()
}
