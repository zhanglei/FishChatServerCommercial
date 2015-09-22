package mongo_store

import (
	"goProject/log"
	// "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//群组信息表
type TopicStoreData struct {
	TopicID       string   `bson:"TopicID"`       //群组ID
	MsgServerAddr string   `bson:"MsgServerAddr"` //群组所在服务器
	FounderID     string   `bson:"FounderID"`     //创建者
	ClientsID     []string `bson:"ClientsID"`     //成员[u1, u2, u3]
	Alive         bool     `bson:"Alive"`         //群组是否是激活状态
}

// //群组
// type TopicStoreData struct {
// 	ClientID      string `bson:"ClientID"`
// 	ClientAddr    string `bson:"ClientAddr"`
// 	MsgServerAddr string `bson:"MsgServerAddr"`
// 	Alive         bool   `bson:"Alive"`
// }

// 新建群组
func (self *MongoStore) CreateTopic(db string, c string, data TopicStoreData) error {
	return self.Upsert(db, c, data)
}

//查询群组
func (self *MongoStore) GetTopicFromTopicID(db string, c string, topicID string) *TopicStoreData {
	log.Info("Get topic from topicID")

	self.rwMutex.Lock()
	defer self.rwMutex.Unlock()

	op := self.session.DB(db).C(c)
	var result *TopicStoreData

	op.Find(bson.M{"TopicID": topicID}).One(&result)
	defer func() {
		if err := recover(); err != nil {
			log.Error(err)
		}
	}()

	return result
}

//根据用户ID查询用户所在群组
func (self *MongoStore) GetTopicsFromClientID(db string, c string, clientID string) []*TopicStoreData {
	log.Info("Get topic from clientID")

	self.rwMutex.Lock()
	defer self.rwMutex.Unlock()

	op := self.session.DB(db).C(c)
	var result []*TopicStoreData

	// regex := `,` + clientID + `,`
	op.Find(bson.M{"ClientsID": bson.M{"$eq": clientID}}).All(&result)
	defer func() {
		if err := recover(); err != nil {
			log.Error(err)
		}
	}()

	return result
}

// 修改群组
// func (self *MongoStore) UpdateTopicFromTopicID(db string, c string, topicID string) error {
// 	log.Info("Update topic from topicID")

// 	var err error
// 	var result *TopicStoreData

// 	self.rwMutex.Lock()
// 	defer self.rwMutex.Unlock()

// 	op := self.session.DB(db).C(c)
// }