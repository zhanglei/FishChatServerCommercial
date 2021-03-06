package base

import (
	"goProject/libnet"
	"goProject/protocol"
)

type ChannelMap map[string]*ChannelState
type SessionMap map[string]*libnet.Session

// type WebSessionMap map[string]*

type AckMap map[string]*AckFrequency

const COMM_PREFIX = "IM"

var ChannleList []string

func init() {
	ChannleList = []string{
		protocol.SYSCTRL_CLIENT_STATUS,
		protocol.SYSCTRL_TOPIC_STATUS,
		protocol.SYSCTRL_TOPIC_SYNC,
		protocol.SYSCTRL_SEND,
		protocol.SYSCTRL_MONITOR,
		protocol.STORE_CLIENT_INFO,
		protocol.STORE_TOPIC_INFO}
}

type ChannelState struct {
	ChannelName  string
	Channel      *libnet.Channel
	ClientIDlist []string
}

type AckFrequency struct {
	LastTime  int64
	Frequency byte
	// ClientID  string
}

func NewChannelState(channelName string, channel *libnet.Channel) *ChannelState {
	return &ChannelState{
		ChannelName:  channelName,
		Channel:      channel,
		ClientIDlist: make([]string, 0),
	}
}

type SessionState struct {
	ClientID   string
	LastPing int64
}

func NewSessionState(cid string, lastPing int64) *SessionState {
	return &SessionState{
		ClientID:   cid,
		LastPing:   lastPing,
	}
}

type Config interface {
	LoadConfig(configfile string) (*Config, error)
}
