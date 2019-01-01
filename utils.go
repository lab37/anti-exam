package main

import (
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

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

//自动模式
func autoProcess(answerPosition int) {
	var screanCenterX = 550    // center of screen
	var firstItemY = 1280      // center of first item (y)
	var qualifyingItemY = 2000 // 排位列表最后一项 y 坐标
	if answerPosition >= 0 {
		log.Printf("【点击】正在点击选项：%d", answerPosition)
		time.Sleep(time.Millisecond * 3800)                                        //延迟
		go clickScreenByAdbShell(screanCenterX, firstItemY+200*(answerPosition-1)) // process click
	} else {
		// go to next match
		log.Printf("【点击】将点击继续挑战按钮...")
		time.Sleep(time.Millisecond * 7500)
		go clickScreenByAdbShell(screanCenterX, firstItemY+400) // 继续挑战 按钮在第三个item处
		log.Printf("【点击】将点击排位列表底部一项，进行比赛匹配...")
		time.Sleep(time.Millisecond * 2000)
		go clickScreenByAdbShell(screanCenterX, qualifyingItemY)
	}
}

//使用adb命令模拟手机屏幕点击答案
func clickScreenByAdbShell(posX int, posY int) {
	var err error
	touchX, touchY := strconv.Itoa(posX), strconv.Itoa(posY)
	_, err = exec.Command("adb", "shell", "input", "swipe", touchX, touchY, touchX, touchY).Output()
	if err != nil {
		log.Fatal("error: check adb connection.")
	}
}
