package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"

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
		//读取Body，注意因为body是buffer.reader，故读取后off会在最后，导致无法再读, 只能重封
		bs, _ := ioutil.ReadAll(newReq.Body)
		newReq.Body = ioutil.NopCloser(bytes.NewReader(bs))
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
		bsNew, answerPosition := getAndInsertAnswerIntoResponse(bs)
		resp.Body = ioutil.NopCloser(bytes.NewReader(bsNew))
		if autoMatic == 1 {
			go autoProcess(answerPosition) // adb自动点击答案
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
