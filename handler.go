package main

import (
	"bytes"
	"encoding/json"
	"log"
	"math/rand"
	"net/url"
	"strconv"
	"time"
)

var (
	roomID string
)

type Question struct {
	//题干部分
	Data struct {
		Quiz        string   `json:"quiz"`
		Options     []string `json:"options"`
		Num         int      `json:"num"`
		School      string   `json:"school"`
		Type        string   `json:"type"`
		Contributor string   `json:"contributor"`
		EndTime     int      `json:"endTime"`
		CurTime     int      `json:"curTime"`
	} `json:"data"`
	Errcode int `json:"errcode"`
	//答案部分
	CalData struct {
		RoomID     string
		quizNum    string
		Answer     string //选择的答案
		TrueAnswer string //正确答案
	} `json:"-"`
}

type PKResult struct {
	Data struct {
		UID         int  `json:"uid"`
		Num         int  `json:"num"`
		Answer      int  `json:"answer"`
		Option      int  `json:"option"`
		Yes         bool `json:"yes"`
		Score       int  `json:"score"`
		TotalScore  int  `json:"totalScore"`
		RowNum      int  `json:"rowNum"`
		RowMult     int  `json:"rowMult"`
		CostTime    int  `json:"costTime"`
		RoomID      int  `json:"roomId"`
		EnemyScore  int  `json:"enemyScore"`
		EnemyAnswer int  `json:"enemyAnswer"`
	} `json:"data"`
	Errcode int `json:"errcode"`
}

//根据http请求的Url中解析出roomID，并设置全局变量roomID
func setRoomIdByRequest(bs []byte) {
	values, _ := url.ParseQuery(string(bs))
	roomID = values.Get("roomID")
}

//搜索答案并把答案插入response中，返回修改后的影响和答案所在屏幕位置
func getAndInsertAnswerIntoResponse(bs []byte) (bsNew []byte, answerPosition int) {
	bsNew = bs
	originalQuestion := &Question{}
	json.Unmarshal(bs, originalQuestion)
	originalQuestion.CalData.RoomID = roomID
	originalQuestion.CalData.quizNum = strconv.Itoa(originalQuestion.Data.Num)

	var optionsWithCounting map[string]int
	//优先从数据库中取得这个问题的答案，如果库中没有，则用百度搜
	answer := getAnswerFromDb(originalQuestion)
	if answer == "" {
		tx := time.Now()
		optionsWithCounting = getAnswerFromBaidu(originalQuestion.Data.Quiz, originalQuestion.Data.Options)
		tx2 := time.Now()
		//输出搜索消耗的时间
		log.Printf("Cost time %d ms\n", tx2.Sub(tx).Nanoseconds()/1e6)
	}

	originalQuestion.CalData.TrueAnswer = answer
	originalQuestion.CalData.Answer = answer

	//缓存考题
	putQuestionInCache(originalQuestion)

	answerPosition = 0

	//生成带答案的试题
	cheatedQuestion := &Question{}
	json.Unmarshal(bs, cheatedQuestion)

	if originalQuestion.CalData.TrueAnswer != "" {
		//若此题在题库中
		for i, option := range cheatedQuestion.Data.Options {
			if option == originalQuestion.CalData.TrueAnswer {
				//在问题的答案中正确选项后面加上4个字
				cheatedQuestion.Data.Options[i] = option + "[标准答案]"
				answerPosition = i + 1
				break
			}
		}
	} else {
		//若此题不在题库中
		var max int = 0
		for i, option := range cheatedQuestion.Data.Options {
			if optionsWithCounting[option] > 0 {
				//在问题的答案的每个选项后面加上这个选项在百度中出现的次数（频率）
				cheatedQuestion.Data.Options[i] = option + "[" + strconv.Itoa(optionsWithCounting[option]) + "]"
				if optionsWithCounting[option] > max {
					max = optionsWithCounting[option]
					answerPosition = i + 1
				}
			}
		}
	}

	//重新封好这个修改好的题目
	bsNew, _ = json.Marshal(cheatedQuestion)
	var out bytes.Buffer
	json.Indent(&out, bsNew, "", " ")

	var answerItem string = "不知道"

	//题目找到了答案
	if answerPosition != 0 {
		answerItem = cheatedQuestion.Data.Options[answerPosition-1]
	} else {
		//题不在题库并且百度也没搜到，则生成一个随机位置
		answerPosition = rand.Intn(4) + 1
	}

	//在电脑上显示答案是什么
	log.Printf("Question answer predict =>\n 【题目】 %v\n 【正确答案】%v\n", cheatedQuestion.Data.Quiz, answerItem)

	if mode == 0 {
		//返回修改后的考题
		return out.Bytes(), answerPosition
	} else {
		//返回原题和答案的位置
		return bs, answerPosition
	}
}

//根据pk结果把问题和正确答案存储到本地题库
func saveQuestionAndAnswerOfPkResult(bs []byte) {
	pkResult := &PKResult{}
	json.Unmarshal(bs, pkResult)

	//log.Println("response choose", roomID, pkResult.Data.Num, string(bs))
	question := getQuestionFromCache(roomID, strconv.Itoa(pkResult.Data.Num))
	if question == nil {
		log.Println("error in get question", pkResult.Data.RoomID, pkResult.Data.Num)
		return
	}
	question.CalData.TrueAnswer = question.Data.Options[pkResult.Data.Answer-1]
	if pkResult.Data.Yes {
		question.CalData.TrueAnswer = question.Data.Options[pkResult.Data.Option-1]
	}
	log.Printf("【保存数据】  %s, %s", question.Data.Quiz, question.CalData.TrueAnswer)
	storeQuestionToDb(question)
}

//roomID=476376430&quizNum=4&option=4&uid=26394007&t=1515326786076&sign=3592b9d28d045f3465206b4147ea872b
