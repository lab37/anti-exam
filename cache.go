package main

import (
	"time"
	//go-cache是一个k-v对的内存缓存，用于单机系统
	//本质是一个带有效期的map[string]interface{}
	//优点是这些映射是线程安全的
	//任何对像在给定的时间内（也可以永久）可以安全的存储并且安全地用于多个协程
	cache "github.com/patrickmn/go-cache"
)

// 创建一个默认过期时间为5分钟的缓存，每10分钟清除一次过期项目
var questionInCache = cache.New(5*time.Minute, 10*time.Minute)

//通过房间号和考试号从缓存中取得考题
func getQuestionFromCache(roomID, quizNum string) *Question {
	key := roomID + "_" + quizNum
	if entity, ok := questionInCache.Get(key); ok {
		//类型断言，检查一个接口对象entity的动态类型是否和断言的类型Question匹配
		//如果这个检查成功，则检查entity的动态类型和动态值不变，但是类型被转换为接口类型Question
		return entity.(*Question)
	}
	return nil
}

//以房间号和考试号为key把考题存入缓存中
func putQuestionInCache(question *Question) {
	key := question.CalData.RoomID + "_" + question.CalData.quizNum
	questionInCache.Set(key, question, cache.DefaultExpiration)
}
