package main

import (
	"flag"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// 模式一: 默认模式, 修改了服务端返回的数据, 更加友好地提示正确答案, 运行方式如上所述: ./brain 或者源码下执行 go run main.go
// 模式二: 隐身模式, 严格返回原始数据, 该模式可以防止作弊检测(客户端检查返回题目和服务端的对比（散列）,模式一很容易被侦测出使用了作弊, 模式二避免了修改返回的题目),
//         但该模式的缺点是降低了用户的体验,题目答案的提示只能显示在PC电脑上，用户要对着电脑上的答案在手机上答题, 运行方式如上所述 ./brain -m 1 或者源码下执行 go run main.go -m 1
// 模式三：自动模式 ** 注意此模式不同手机点击可能不稳定, 谨慎使用 ** 安卓机的自动刷题模式，需要将手机连接到电脑，并安装adb，且需要在开发者模式中
//        打开usb调试，使用前请根据自身手机分辨率，调整spider文件clickProcess中的相应参数：手机屏幕中心x坐标，第一个选项中心y坐标，排位列表中最后一项中
//        心y坐标。运行方式如上所述 ./brain -a 1 -m 1 或者源码下执行 go run main.go -a 1 -m 1
var (
	mode      int
	automatic int
)

func init() {
	flag.IntVar(&mode, "m", 0, "run mode 0 : default mode, easy to be detected of cheating; 1 : invisible mode")
	flag.IntVar(&automatic, "a", 0, "run automatic  0 : manual  1 : automatic")
	flag.Parse()
}

func main() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		run("8998", mode, automatic)
	}()
	<-c
	close()
}

var (
	baidu_url = "http://www.baidu.com/s?"
)

//从百度搜索结果中统计出每个答案选项出现的次数，以这个出现次数做为比较哪个答案对的依据.
func getAnswerFromBaidu(quiz string, options []string) map[string]int {
	values := url.Values{}
	values.Add("wd", quiz)
	req, _ := http.NewRequest("GET", baidu_url+values.Encode(), nil)
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
