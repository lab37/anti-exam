package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	//bolt是一个k-v数据库。很简单高效的一种。
	//bolt中的表叫bucket（桶），每次操作看为一个事务

	"github.com/PuerkitoBio/goquery"
	"github.com/boltdb/bolt"
)

//正确答案和答案的更正时间
type CorrectAnswer struct {
	Answer string `json:"a"`
	Update int64  `json:"ts"`
}

//生成考题的答案部分
func newCorrectAnswer(answer string, update int64) *CorrectAnswer {
	return &CorrectAnswer{
		Answer: answer,
		Update: update,
	}
}

//json编码这个试题的答案部分
func (q *CorrectAnswer) unmarshal(bs []byte) {
	json.Unmarshal(bs, q)
}

//json编码这个试题的答案部分
func (q *CorrectAnswer) marshal() []byte {
	bs, _ := json.Marshal(q)
	return bs
}

var (
	memoryDb       *bolt.DB
	QuestionBucket = "Question"
)

//打开键值数据库，并创建题库桶
func init() {
	var err error
	memoryDb, err = bolt.Open("questions.data", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	memoryDb.Update(func(tx *bolt.Tx) error {
		//如果不存在题库桶就创建一个
		_, err := tx.CreateBucketIfNotExists([]byte(QuestionBucket))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
}

//存储问题到本地桶中
func storeQuestionToDb(question *Question) error {
	if question.CalData.TrueAnswer != "" {
		//tx就是一个事务，代表开始一个事务
		return memoryDb.Update(func(tx *bolt.Tx) error {
			//打开一个要操作的桶
			b := tx.Bucket([]byte(QuestionBucket))
			v := newCorrectAnswer(question.CalData.TrueAnswer, time.Now().Unix())
			//题库中只存储题干的题目和正确答案以及答案的最新正确时间。
			err := b.Put([]byte(question.Data.Quiz), v.marshal())
			return err
		})
	}
	return nil
}

//从题库中查询一个问题的答案
func getAnswerFromDb(question *Question) (answerStr string) {
	//从库中找题的答案部分，查到了为答案，没查到答案为空
	memoryDb.View(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(QuestionBucket))
		v := b.Get([]byte(question.Data.Quiz))
		if len(v) == 0 { //没查到
			//必须反回nil，证明错误不是出在查询系统
			return nil
		}
		q := newCorrectAnswer("", time.Now().Unix())
		q.unmarshal(v)
		answerStr = q.Answer
		return nil
	})
	return
}

//从百度搜索结果中统计出每个答案选项出现的次数，以这个出现次数做为比较哪个答案对的依据.
func getAnswerFromBaidu(quiz string, options []string) map[string]int {
	values := url.Values{}
	values.Add("wd", quiz)
	req, _ := http.NewRequest("GET", "http://www.baidu.com/s?"+values.Encode(), nil)
	ans := make(map[string]int, len(options))
	for _, option := range options {
		ans[option] = 0
	}
	resp, _ := http.DefaultClient.Do(req)
	if resp == nil {
		return ans
	}
	//解析返回的文档，变成jquery那样的文档对象模型
	doc, _ := goquery.NewDocumentFromReader(resp.Body)
	defer resp.Body.Close()
	str := doc.Find("#content_left .result").Text()
	//统计答案在搜索结果中出现的个数，得到一个每个选项的可能性大小
	for _, option := range options {
		ans[option] = strings.Count(str, option)
	}
	return ans
}

//取得问题的最后更新时间，-1代表没有找到问题
func getUpdateTimeOfAnswer(quiz string) (updateTime int64) {
	memoryDb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(QuestionBucket))
		v := b.Get([]byte(quiz))
		if len(v) == 0 {
			updateTime = -1
			return nil
		}
		q := newCorrectAnswer("", time.Now().Unix())
		q.unmarshal(v)
		updateTime = q.Update
		return nil
	})
	return
}

//取得所有考题
func showAllQuestions() {
	var kv = map[string]string{}
	memoryDb.View(func(tx *bolt.Tx) error {
		// 假设问题桶存在并且有key
		b := tx.Bucket([]byte(QuestionBucket))
		//获取所有键值对
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			fmt.Printf("key=%s, value=%s\n", k, v)
			//建立所有映射
			kv[string(k)] = string(v)
		}
		return nil
	})

}

//计算桶里有多少考题
func countQuestions() int {
	var i int
	memoryDb.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := tx.Bucket([]byte(QuestionBucket))
		c := b.Cursor()

		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			i++
		}
		return nil
	})
	return i
}

//合并第三方的数据库到题库,即导入外来题库（多个文件可以)
func mergeQuestions(fs ...string) {
	var i int
	for _, f := range fs {
		thirdDb, err := bolt.Open(f, 0600, nil)
		defer thirdDb.Close()
		if err != nil {
			log.Println("error in merge file db "+f, err.Error())
			continue
		}
		thirdDb.View(func(thirdTx *bolt.Tx) error {
			// Assume bucket exists and has keys
			b := thirdTx.Bucket([]byte(QuestionBucket))
			c := b.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				memoryDb.Update(func(tx *bolt.Tx) error {
					b := tx.Bucket([]byte(QuestionBucket))
					//三方包的时间
					q := newCorrectAnswer("", 0)
					q.unmarshal(v)
					//数据库中的时间
					if q.Update > getUpdateTimeOfAnswer(string(k)) {
						i++
						b.Put(k, q.marshal())
					}
					return nil
				})
			}
			log.Println("merged file", f)
			return nil
		})
	}
	log.Println("merged", i, "questions")
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
