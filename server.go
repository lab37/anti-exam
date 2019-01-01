package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"time"

	"github.com/coreos/goproxy"
)

//初始化并启动代理
func startProxyServer(port string) {

	proxy := goproxy.NewProxyHttpServer()
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)

	proxy.OnRequest().DoFunc(requestHandleChain)
	proxy.OnResponse().DoFunc(responseHandleChain)

	log.Println("代理服务器运行端口:" + port)
	log.Fatal(http.ListenAndServe(":"+port, proxy))
}

//处理请求的函数链
func requestHandleChain(oldReq *http.Request, ctx *goproxy.ProxyCtx) (newReq *http.Request, resp *http.Response) {
	newReq = oldReq
	if ctx.Req.URL.Path == `/question/bat/findQuiz` || ctx.Req.URL.Path == `/question/fight/findQuiz` {
		bs, _ := ioutil.ReadAll(newReq.Body)                //读取Body，注意因为body是buffer.reader，故读取后off会在最后，导致无法再读
		newReq.Body = ioutil.NopCloser(bytes.NewReader(bs)) //重新封装
		setRoomIdByRequest(bs)
	} else if ctx.Req.URL.Host == `abc.com` {
		resp = new(http.Response)
		resp.StatusCode = 200
		resp.Header = make(http.Header)
		resp.Header.Add("Content-Disposition", "attachment; filename=ca.crt")
		resp.Header.Add("Content-Type", "application/octet-stream")
		resp.Body = ioutil.NopCloser(bytes.NewReader(goproxy.CA_CERT))
	}
	return
}

//处理响应的函数链
func responseHandleChain(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	if resp == nil {
		return resp
	}
	//返回的是试题
	if ctx.Req.URL.Path == "/question/bat/findQuiz" || ctx.Req.URL.Path == "/question/fight/findQuiz" {
		bs, _ := ioutil.ReadAll(resp.Body)
		bsNew, ansPos := getAndInsertAnswerIntoResponse(bs)
		resp.Body = ioutil.NopCloser(bytes.NewReader(bsNew))
		if autoMatic == 1 {
			go autoProcess(ansPos) // adb自动点击答案
		}
		//返回的是单题的PK结果
	} else if ctx.Req.URL.Path == "/question/bat/choose" || ctx.Req.URL.Path == "/question/fight/choose" {
		bs, _ := ioutil.ReadAll(resp.Body)
		resp.Body = ioutil.NopCloser(bytes.NewReader(bs))
		go saveQuestionAndAnswerOfPkResult(bs)
		//返回的是比赛结果
	} else if ctx.Req.URL.Path == "/question/bat/fightResult" || ctx.Req.URL.Path == "/question/fight/fightResult" {
		if autoMatic == 1 {
			go autoProcess(-1)
		}
	}
	return resp
}

//关闭代理
func closeProxyServer() {
	memoryDb.Close()
}

//自动模式
func autoProcess(ansPos int) {
	var screanCenterX = 550    // center of screen
	var firstItemY = 1280      // center of first item (y)
	var qualifyingItemY = 2000 // 排位列表最后一项 y 坐标
	if ansPos >= 0 {
		log.Printf("【点击】正在点击选项：%d", ansPos)
		time.Sleep(time.Millisecond * 3800)                                //延迟
		go clickScreenByAdbShell(screanCenterX, firstItemY+200*(ansPos-1)) // process click
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
