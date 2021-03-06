package main

import (
	"encoding/json"
	"goProject/base"
	"goProject/common"
	"goProject/info"
	"goProject/libnet"
	"goProject/log"
	"goProject/protocol"
	"goProject/storage/mongo_store"
	// "gopkg.in/mgo.v2/json"
	"strconv"
	"time"
)

/*********************************************************************************
TODU LIST:
1、MUTUAL ACK
*********************************************************************************/

//------------------------------------------------------------------------------
// 请求总入口
//------------------------------------------------------------------------------
/**
	发送Ask请求 Type对应处理机制

	::addFriend
	procAskAddFriend()
	[addFriend, friendName]
	发送请求信息给好友

	::addTopic
	procAskAddTopic()
	[addTopic, topicName]
	根据群名称找到创始人发送信息给创始人

	::inviteTopic
	procAskInviteTopic()
	[inviteTopic, topicName, friendId]
	发送邀请进群组的请求
**/
func (self *ProtoProc) procAsk(cmd protocol.Cmd, session *libnet.Session) error {
	var err error

	if len(cmd.GetArgs()) < protocol.SEND_ASK_CMD_ARGS_NUM+1 {
		log.Info(info.NOT_ENOUGH_ARGUMENTS)
		return nil
	}

	clientId := cmd.GetArgs()[0]
	msgType := cmd.GetArgs()[1]
	friendId := cmd.GetArgs()[2]
	send2Time := time.Now().Unix()
	uuid := common.NewV4().String()

	//保存消息到mongodb中
	data := mongo_store.MutualRecordMessageData{clientId, friendId, msgType, send2Time, uuid, false}
	err = self.msgServer.mongoStore.Upsert(mongo_store.DATA_BASE_NAME, mongo_store.RECORD_MUTUAL_MESSAGE_COLLECTION, &data)
	if err != nil {
		log.Error("error:", err)
		return err
	}

	switch msgType {
	case protocol.SEND_ASK_CMD_TYPE_ADD_FRIEND:
		err = self.procAskAddFriend(cmd, session, data)
		if err != nil {
			log.Error("error:", err)
			return err
		}

	case protocol.SEND_ASK_CMD_TYPE_ADD_TOPIC:
	case protocol.SEND_ASK_CMD_TYPE_INVITE_TOPIC:
	default:
		log.Info("the ask type is undefined.\n", cmd)
	}

	return err
}

//添加好友
func (self *ProtoProc) procAskAddFriend(cmd protocol.Cmd, session *libnet.Session, data mongo_store.MutualRecordMessageData) error {
	log.Info("procAskAddFriend")
	var err error

	//定义返回用户请求信息
	resp := protocol.NewCmdResponse(protocol.RESP_ASK_CMD)
	resp.Repo = cmd.GetReport()

	storeSession, err := self.msgServer.mongoStore.GetClientFromId(mongo_store.DATA_BASE_NAME,
		mongo_store.CLIENT_INFO_COLLECTION, data.FromID)
	if err != nil {
		log.Error("error:", err)
		return err
	}

	if common.InArray(storeSession.Friends, data.ToID) {
		resp.Ok = false
		resp.Message = info.THE_ID_IS_ALREADY_YOUR_FRIEND
	} else {
		if self.msgServer.sessions[data.ToID] != nil {
			log.Info("In the same server")

			receive := protocol.NewCmdResponse(protocol.RECEIVE_ASK_CMD)
			receive.AddArg(data.Type)
			receive.AddArg(data.FromID)
			receive.AddArg(strconv.FormatInt(data.Time, 10))
			receive.AddArg(data.UUID)

			if self.msgServer.sessions[data.ToID] != nil {
				self.msgServer.sessions[data.ToID].Send(receive)
				if err != nil {
					log.Error(err.Error())
					return err
				}

				//储存ACK，用来验证
				ack := new(base.AckFrequency)
				ack.Frequency = 1
				ack.LastTime = data.Time
				self.msgServer.mutualAckMap[data.UUID] = ack
			}
		} else {
			log.Info("Not in the same server")
			if self.msgServer.channels[protocol.SYSCTRL_SEND] != nil {

				rcmd := protocol.NewCmdSimple(protocol.ROUTE_ASK_CMD)
				rcmd.AddArg(data.Type)
				rcmd.AddArg(data.FromID)
				rcmd.AddArg(data.ToID)
				rcmd.AddArg(strconv.FormatInt(data.Time, 10))
				rcmd.AddArg(data.UUID)

				temp, err := json.Marshal(rcmd)
				if err != nil {
					log.Error(err.Error())
					return err
				}

				routerMsg := protocol.NewCmdSimple(protocol.ROUTE_MSG_CMD)
				routerMsg.AddArg(storeSession.MsgServerAddr)
				routerMsg.AddArg(string(temp))

				err = self.msgServer.channels[protocol.SYSCTRL_SEND].Channel.Broadcast(routerMsg)
				if err != nil {
					log.Error(err.Error())
					return err
				}
			}
			// }
		}
	}

	//返回用户请求
	err = session.Send(resp)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	return err
}

//获取用户P2P未读信息
func (self *ProtoProc) procAskOfflineMsg(session *libnet.Session, cid string) error {
	var err error

	//从mongo读取信息
	recordData, err := self.msgServer.mongoStore.ReadMutualRecordMessage(mongo_store.DATA_BASE_NAME, mongo_store.RECORD_P2P_MESSAGE_COLLECTION, cid)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	//把从数据库中取出的数据发送给Client
	for _, v := range recordData {

		receive := protocol.NewCmdResponse(protocol.RECEIVE_ASK_CMD)
		receive.AddArg(v.Type)
		receive.AddArg(v.FromID)
		receive.AddArg(strconv.FormatInt(v.Time, 10))
		receive.AddArg(v.UUID)

		// //缓存uuid,等待ack
		// ack := new(base.AckFrequency)
		// ack.Frequency = 1
		// ack.LastTime = time.Now().Unix()
		// self.msgServer.p2pAckMap[v.UUID] = ack

		time.Sleep(100)

		if self.msgServer.sessions[cid] != nil {
			err = self.msgServer.sessions[cid].Send(receive)
			if err != nil {
				log.Error(err.Error())
				return err
			}
		}
	}

	return err
}

//------------------------------------------------------------------------------
// 回应请求总入口
//------------------------------------------------------------------------------
/**
	回应对方的请求 Type对应处理机制

	::addFriend
	procReactAddFriend()
	[addFriend, friendName]
	发送请求信息给好友

	::addTopic
	procReactAddTopic()
	[addTopic, topicName]
	根据群名称找到创始人发送信息给创始人

	::inviteTopic
	procReactInviteTopic()
	[inviteTopic, topicName, friendId]
	发送邀请进群组的请求
**/
func (self *ProtoProc) procReact(cmd protocol.Cmd, session *libnet.Session) error {
	var err error

	if len(cmd.GetArgs()) < protocol.SEND_REACT_CMD_ARGS_NUM+1 {
		log.Info(info.NOT_ENOUGH_ARGUMENTS)
		return nil
	}

	// //定义返回用户请求信息
	// resp := protocol.NewCmdResponse(protocol.RESP_REACT_CMD)
	// resp.Repo = cmd.GetReport()

	reactType := cmd.GetArgs()[0]
	uuid := cmd.GetArgs()[1]

	result := self.msgServer.mongoStore.ReadMutualRecordMessageFromUuid(mongo_store.DATA_BASE_NAME, mongo_store.RECORD_MUTUAL_MESSAGE_COLLECTION, uuid)
	if result == nil {
		log.Info("No data.")
		// resp.Ok = false
		// resp.Message = info.NO_INITIATE_THIS_REQUEST
	} else {

		if reactType == protocol.SEND_REACT_CMD_AGREE {
			switch result.Type {
			case protocol.SEND_REACT_CMD_TYPE_ADD_FRIEND:
				err = self.procReactAddFriend(cmd, session, *result)
				if err != nil {
					log.Error("error:", err)
					return err
				}

			case protocol.SEND_REACT_CMD_TYPE_ADD_TOPIC:
			case protocol.SEND_REACT_CMD_TYPE_INVITE_TOPIC:
			default:
				log.Info("the react type is undefined.\n", cmd)
			}
		}

		err = self.msgServer.mongoStore.RemoveMutualRecordMessageFromUuid(mongo_store.DATA_BASE_NAME, mongo_store.RECORD_MUTUAL_MESSAGE_COLLECTION, uuid)
		if err != nil {
			log.Error(err.Error())
			return err
		}
	}

	// //返回用户请求
	// err = session.Send(resp)
	// if err != nil {
	// 	log.Error(err.Error())
	// 	return err
	// }

	return err
}

//添加好友
func (self *ProtoProc) procReactAddFriend(cmd protocol.Cmd, session *libnet.Session, data mongo_store.MutualRecordMessageData) error {
	log.Info("procReactAddFriend")
	var err error

	//定义返回用户请求信息
	resp := protocol.NewCmdResponse(protocol.RESP_REACT_CMD)
	resp.Repo = cmd.GetReport()

	clientInfo := self.msgServer.mongoStore.GetClientsFromIds(mongo_store.DATA_BASE_NAME, mongo_store.CLIENT_INFO_COLLECTION, []string{data.FromID, data.ToID})
	if clientInfo == nil {
		log.Error(err.Error())
		resp.Message = info.NO_CLIENT_INFO
		resp.Ok = false
	} else if len(clientInfo) == 2 {

		var my, myFriend mongo_store.SessionStoreData
		if data.ToID == clientInfo[0].ClientID {
			my = *clientInfo[0]
			myFriend = *clientInfo[1]
		} else {
			my = *clientInfo[1]
			myFriend = *clientInfo[0]
		}

		if !common.InArray(my.Friends, myFriend.ClientID) {
			err = self.msgServer.mongoStore.UpdateFriendsFromId(mongo_store.DATA_BASE_NAME, mongo_store.CLIENT_INFO_COLLECTION, my.ClientID, append(my.Friends, myFriend.ClientID))
			if err != nil {
				log.Error(err.Error())
				resp.Message = info.ERROR
				resp.Ok = false
			}
		}
		if !common.InArray(myFriend.Friends, my.ClientID) {
			err = self.msgServer.mongoStore.UpdateFriendsFromId(mongo_store.DATA_BASE_NAME, mongo_store.CLIENT_INFO_COLLECTION, myFriend.ClientID, append(myFriend.Friends, my.ClientID))
			if err != nil {
				log.Error(err.Error())
				resp.Message = info.ERROR
				resp.Ok = false
			}
		}

		if resp.Ok == true {
			//通知好友
			// SEND_MESSAGE_P2P_CMD
			sendTF := protocol.NewCmdSimple(protocol.SEND_MESSAGE_P2P_CMD)
			sendTF.AddArg(myFriend.ClientID + " 已经同意您的好友请求!")
			sendTF.AddArg(myFriend.ClientID)
			self.procSendMessageP2P(sendTF, session)
		}

		resp.AddArg(myFriend.ClientID)
	} else {
		resp.Message = info.NO_CLIENT_INFO
		resp.Ok = false
	}

	err = session.Send(resp)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	return err
}
