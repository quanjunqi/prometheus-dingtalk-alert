package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Alert 结构体用于解析 Alertmanager 发来的告警
type Alert struct {
	Labels struct {
		Alertname  string `json:"alertname"`
		Endpoint   string `json:"endpoint"`
		Instance   string `json:"instance"`
		Job        string `json:"job"`
		Namespace  string `json:"namespace"`
		Owner      string `json:"owner"`
		Pod        string `json:"pod"`
		Prometheus string `json:"prometheus"`
		Region     string `json:"region"`
		Service    string `json:"service"`
		Severity   string `json:"severity"`
		Team       string `json:"team"`
		Container  string `json:"container"`
	}
	Annotations struct {
		AdditionalInfo string `json:"additionalInfo"`
		Description    string `json:"description"`
		Summary        string `json:"summary"`
		RunbookURL     string `json:"runbook_url"`
	}
	StartsAt     time.Time `json:"startsAt"`
	EndsAt       time.Time `json:"endsAt"`
	Status       string    `json:"status"`
	GeneratorURL string    `json:"generatorURL"`
}

// DingTalkMessage 用于构造发送到钉钉的消息
type DingTalkMessage struct {
	Msgtype  string `json:"msgtype"`
	Markdown struct {
		Title string `json:"title"`
		Text  string `json:"text"`
	} `json:"markdown"`
	At struct {
		AtMobiles []string `json:"atMobiles"`
		IsAtAll   bool     `json:"isAtAll"`
	} `json:"at"`
}

// Payload 代表整个 Alertmanager webhook 的数据结构
type WebhookData struct {
	Receiver    string  `json:"receiver"`
	Status      string  `json:"status"`
	Alerts      []Alert `json:"alerts"`
	GroupLabels struct {
		Alertname string `json:"alertname"`
		Job       string `json:"job"`
		Namespace string `json:"namespace"`
		Service   string `json:"service"`
	}
	CommonLabels struct {
		Alertname  string `json:"alertname"`
		Endpoint   string `json:"endpoint"`
		Instance   string `json:"instance"`
		Job        string `json:"job"`
		Namespace  string `json:"namespace"`
		Owner      string `json:"owner"`
		Pod        string `json:"pod"`
		Prometheus string `json:"prometheus"`
		Region     string `json:"region"`
		Service    string `json:"service"`
		Severity   string `json:"severity"`
		Team       string `json:"team"`
	}
	CommonAnnotations struct {
		AdditionalInfo string `json:"additionalInfo"`
		Description    string `json:"description"`
		Summary        string `json:"summary"`
		RunbookURL     string `json:"runbook_url"`
	}
	ExternalURL string `json:"externalURL"`
	Version     string `json:"version"`
	GroupKey    string `json:"groupKey"`
}

// 钉钉机器人的 webhook URL 和秘钥
const (
	dingTalkWebhook = "https://oapi.dingtalk.com/robot/send?access_token=6d2e340ee92d4cba36a125c2101cd0586f44516a6f11770c9eb5742a126d7fa6"
	dingTalkSecret  = "SEC9888a6963e1bfb3de8e121605a0cefb61c4cb310981a5fb8bf7eeec1b4c3cd5c"
)

func main() {
	r := gin.Default()
	r.POST("/webhook", alertHandler)
	r.Run(":8080") // 监听并在 0.0.0.0:8080 上启动服务
}

func alertHandler(c *gin.Context) {
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Println("body", err)
		return
	}
	fmt.Println(string(body))
	var webhookdata WebhookData
	if err = json.Unmarshal(body, &webhookdata); err != nil {
		fmt.Printf("Unmarshal err, %v\n", err)
		return
	}
	for _, alert := range webhookdata.Alerts {
		sendMessageToDingTalk(webhookdata, alert) //发送告警
	}

}

func sendMessageToDingTalk(webhookdata WebhookData, alert Alert) {

	message := DingTalkMessage{
		Msgtype: "markdown",
		Markdown: struct {
			Title string "json:\"title\""
			Text  string "json:\"text\""
		}{
			Title: "Prometheus Alert",
			Text:  nodeMessage(webhookdata, alert),
		},
		// At: struct {
		// 	AtMobiles []string `json:"atMobiles"`
		// 	IsAtAll   bool     `json:"isAtAll"`
		// }{
		// 	AtMobiles: []string{alert.Labels.Owner},
		// 	IsAtAll:   false,
		// },
	}
	requestBody, _ := json.Marshal(message)
	timestamp := time.Now().UnixNano() / 1000000
	sign := generateSign(timestamp)

	req, _ := http.NewRequest("POST", dingTalkWebhook+"&timestamp="+strconv.FormatInt(timestamp, 10)+"&sign="+sign, bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending message to DingTalk:", err)
		return
	}
	defer resp.Body.Close()
}

func nodeMessage(webhookdata WebhookData, alert Alert) string {
	// 获取 Instance 值
	var message string
	switch webhookdata.Receiver {
	case "webhook_alert":
		switch alert.Annotations.AdditionalInfo {
		case "node":
			if alert.Status == "firing" {
				message = "### 告警信息\n" +
					"- **主题**: " + alert.Labels.Alertname + "\n" +
					"- **实例**: " + alert.Labels.Instance + "\n" +
					"- **告警级别**: " + webhookdata.CommonLabels.Severity + "\n" +
					"- **主机地域**: " + alert.Labels.Region + "\n" +
					"- **告警内容**: " + alert.Annotations.Description + "\n" +
					"- **告警详情**: " + alert.Annotations.Summary + "\n" +
					"- **开始时间**: " + alert.StartsAt.Format("2006-01-02 15:04:05") + "\n\n"
			} else if webhookdata.Status == "resolved" {
				message = "### 恢复信息\n" +
					"- **主题**: " + alert.Labels.Alertname + "\n" +
					"- **实例**: " + alert.Labels.Instance + "\n" +
					"- **告警级别**: " + webhookdata.CommonLabels.Severity + "\n" +
					"- **告警内容**: " + alert.Annotations.Description + "\n" +
					"- **告警详情**: " + alert.Annotations.Summary + "\n" +
					"- **开始时间**: " + alert.StartsAt.Format("2006-01-02 15:04:05") + "\n" +
					"- **结束时间**: " + alert.EndsAt.Format("2006-01-02 15:04:05") + "\n\n"
			}
		case "k8s":
			if alert.Status == "firing" {
				message = "### 告警信息\n" +
					"- **主题**: " + alert.Labels.Alertname + "\n" +
					"- **实例**: " + alert.Labels.Instance + "\n" +
					"- **命名空间**: " + alert.Labels.Namespace + "\n" +
					"- **容器**: " + alert.Labels.Container + "\n" +
					"- **告警级别**: " + webhookdata.CommonLabels.Severity + "\n" +
					"- **告警内容**: " + alert.Annotations.Description + "\n" +
					"- **告警详情**: " + alert.Annotations.Summary + "\n"
				// 仅在 RunbookURL 不为空时添加
				if runbookURL := alert.Annotations.RunbookURL; runbookURL != "" {
					message += "- **告警日志**: [查看详情](" + runbookURL + ")\n"
				}
				message += "- **开始时间**: " + alert.StartsAt.Format("2006-01-02 15:04:05") + "\n\n"
			} else if webhookdata.Status == "resolved" {
				message = "### 恢复信息\n" +
					"- **主题**: " + alert.Labels.Alertname + "\n" +
					"- **实例**: " + alert.Labels.Instance + "\n" +
					"- **命名空间**: " + alert.Labels.Namespace + "\n" +
					"- **容器**: " + alert.Labels.Container + "\n" +
					"- **告警级别**: " + webhookdata.CommonLabels.Severity + "\n" +
					"- **告警内容**: " + alert.Annotations.Description + "\n" +
					"- **告警详情**: " + alert.Annotations.Summary + "\n" +
					"- **开始时间**: " + alert.StartsAt.Format("2006-01-02 15:04:05") + "\n" +
					"- **结束时间**: " + alert.EndsAt.Format("2006-01-02 15:04:05") + "\n\n"
			}
		}

	}
	return message
}

// 格式化告警信息
// 可根据需要自定义消息格式

func generateSign(timestamp int64) string {
	// 生成钉钉签名
	stringToSign := strconv.FormatInt(timestamp, 10) + "\n" + dingTalkSecret
	hmac256 := hmac.New(sha256.New, []byte(dingTalkSecret))
	hmac256.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(hmac256.Sum(nil))
}
