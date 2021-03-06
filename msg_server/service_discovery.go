package main

import (
	"encoding/json"
	"github.com/coreos/etcd/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/coreos/etcd/client"
	"goProject/log"
	"runtime"
	"time"
)

type Worker struct {
	Name    string
	IP      string
	KeysAPI client.KeysAPI
	Server  *MsgServer
}

// workerInfo is the service register information to etcd
type WorkerInfo struct {
	Name       string
	IP         string
	CPU        int
	SessionNum uint64
}

func NewWorker(name, IP string, endpoints []string, server *MsgServer) *Worker {
	cfg := client.Config{
		Endpoints:               endpoints,
		Transport:               client.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second,
	}

	etcdClient, err := client.New(cfg)
	if err != nil {
		log.Fatal("Error: cannot connec to etcd:", err)
	}

	w := &Worker{
		Name:    name,
		IP:      IP,
		KeysAPI: client.NewKeysAPI(etcdClient),
		Server:  server,
	}
	go w.HeartBeat()
	return w
}

func (w *Worker) HeartBeat() {
	api := w.KeysAPI

	for {
		info := &WorkerInfo{
			Name:       w.Name,
			IP:         w.IP,
			CPU:        runtime.NumCPU(),
			SessionNum: (uint64)(len(w.Server.sessions)),
		}

		key := "workers/" + w.Name
		value, _ := json.Marshal(info)

		_, err := api.Set(context.Background(), key, string(value), &client.SetOptions{
			TTL: time.Second * 10,
		})
		if err != nil {
			log.Info("Error update workerInfo:", err)
		}
		time.Sleep(time.Second * 3)
	}
}
