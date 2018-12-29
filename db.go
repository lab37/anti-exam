package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	//bolt是一个k-v数据库。很简单高效的一种。
	//bolt中的表叫bucket（桶），每次操作看为一个事务

	"github.com/boltdb/bolt"
)

//问题的一部分，答案和答案的更正时间
type QuestionCols struct {
	Answer string `json:"a"`
	Update int64  `json:"ts"`
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
		//如果不存在问题桶就创建一个
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
			v := newQuestionCols(question.CalData.TrueAnswer)
			err := b.Put([]byte(question.Data.Quiz), v.encodeQuestionCols())
			return err
		})
	}
	return nil
}

//查询一个问题的答案
func getAnswerInDb(question *Question) (str string) {
	memoryDb.View(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(QuestionBucket))
		v := b.Get([]byte(question.Data.Quiz))
		if len(v) == 0 {
			//必须反回nil，证明错误不是出在查询系统
			return nil
		}
		//得到问题结构的一部分，并更新问题的最后刷新时间
		q := decodeQuestionCols(v, time.Now().Unix())
		str = q.Answer
		return nil
	})
	return
}

//取得问题的最后更新时间，-1代表没有找到问题，现在时间表代表找到了
func FetchQuestionTime(quiz string) (res int64) {
	memoryDb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(QuestionBucket))
		v := b.Get([]byte(quiz))
		if len(v) == 0 {
			res = -1
			return nil
		}
		//取得答案部分并装更新时间刷新为现在
		q := decodeQuestionCols(v, time.Now().Unix())
		res = q.Update
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
					q := decodeQuestionCols(v, 0)
					//数据库中的时间
					if q.Update > FetchQuestionTime(string(k)) {
						i++
						b.Put(k, q.encodeQuestionCols())
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

//生成考题的答案部分
func newQuestionCols(answer string) *QuestionCols {
	return &QuestionCols{
		Answer: answer,
		Update: time.Now().Unix(),
	}
}

//解码一个考题的答案部分，并设置一下答案的更日期，能用json解就解，解不了直接转换为string算了。
func decodeQuestionCols(bs []byte, update int64) *QuestionCols {
	var q = &QuestionCols{}
	err := json.Unmarshal(bs, q)
	if err == nil {
		return q
	} else {
		q = newQuestionCols(string(bs))
		q.Update = update
	}
	return q
}

//json编码这个试题的答案部分
func (q *QuestionCols) encodeQuestionCols() []byte {
	bs, _ := json.Marshal(q)
	return bs
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
