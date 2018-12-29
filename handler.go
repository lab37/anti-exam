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

	CalData struct {
		RoomID     string
		quizNum    string
		Answer     string
		TrueAnswer string
	} `json:"-"`
}

type ChooseResp struct {
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

//在http请求的Url中解析出roomID
func handleQuestionReq(bs []byte) {
	values, _ := url.ParseQuery(string(bs))
	roomID = values.Get("roomID")
}

//根据服务器返回的题目,进行答案搜索，
func handleQuestionResp(bs []byte) (bsNew []byte, ansPos int) {
	bsNew = bs
	question := &Question{}
	json.Unmarshal(bs, question)
	question.CalData.RoomID = roomID
	question.CalData.quizNum = strconv.Itoa(question.Data.Num)

	//从数据库中取得这个问题的答案
	answer := getAnswerInDb(question)
	var ret map[string]int
	//如果库中没有，则用百度搜
	if answer == "" {
		tx := time.Now()
		ret = getAnswerFromBaidu(question.Data.Quiz, question.Data.Options)
		tx2 := time.Now()
		//输出搜索消耗的时间
		log.Printf("Cost time %d ms\n", tx2.Sub(tx).Nanoseconds()/1e6)
	}
	question.CalData.TrueAnswer = answer
	question.CalData.Answer = answer
	//缓存考题
	setQuestionToCache(question)

	ansPos = 0
	//重新解析一下返回的题目到一个新变量中
	respQuestion := &Question{}
	json.Unmarshal(bs, respQuestion)

	if question.CalData.TrueAnswer != "" {
		//若此题在题库中
		for i, option := range respQuestion.Data.Options {
			if option == question.CalData.TrueAnswer {
				//在问题的答案中正确选项后面加上4个字
				respQuestion.Data.Options[i] = option + "[标准答案]"
				ansPos = i + 1
				break
			}
		}
	} else {
		//若此题不在题库中
		var max int = 0
		for i, option := range respQuestion.Data.Options {
			if ret[option] > 0 {
				//在问题的答案的每个选项后面加上这个选项在百度中出现的次数（频率）
				respQuestion.Data.Options[i] = option + "[" + strconv.Itoa(ret[option]) + "]"
				if ret[option] > max {
					max = ret[option]
					ansPos = i + 1
				}
			}
		}
	}
	//重新封好这个修改好的题目
	bsNew, _ = json.Marshal(respQuestion)

	var out bytes.Buffer
	json.Indent(&out, bsNew, "", " ")
	var answerItem string = "不知道"
	if ansPos != 0 {
		answerItem = respQuestion.Data.Options[ansPos-1]
	} else {
		//随机点击
		ansPos = rand.Intn(4) + 1
	}
	log.Printf("Question answer predict =>\n 【题目】 %v\n 【正确答案】%v\n", respQuestion.Data.Quiz, answerItem)

	//直接将答案返回在客户端,可能导致封号,所以只在服务端显示
	if Mode == 0 {
		//返回修改后的答案
		return out.Bytes(), ansPos
	} else {
		//返回答案
		return bs, ansPos
	}
}

//hijack 提交请求
func handleChooseReq(bs []byte) (newBs []byte) {
	newBs = bs
	values, _ := url.ParseQuery(string(bs))
	quizNum := values.Get("quizNum")
	question := getQuestionFromCache(roomID, quizNum)
	if question == nil {
		return
	}

	return
}

//把问题存储到本地题库
func handleChooseResponse(bs []byte) {
	chooseResp := &ChooseResp{}
	json.Unmarshal(bs, chooseResp)

	//log.Println("response choose", roomID, chooseResp.Data.Num, string(bs))
	question := getQuestionFromCache(roomID, strconv.Itoa(chooseResp.Data.Num))
	if question == nil {
		log.Println("error in get question", chooseResp.Data.RoomID, chooseResp.Data.Num)
		return
	}
	question.CalData.TrueAnswer = question.Data.Options[chooseResp.Data.Answer-1]
	if chooseResp.Data.Yes {
		question.CalData.TrueAnswer = question.Data.Options[chooseResp.Data.Option-1]
	}
	log.Printf("【保存数据】  %s, %s", question.Data.Quiz, question.CalData.TrueAnswer)
	storeQuestionToDb(question)
}

//roomID=476376430&quizNum=4&option=4&uid=26394007&t=1515326786076&sign=3592b9d28d045f3465206b4147ea872b
