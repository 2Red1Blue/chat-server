package main

import (
	"encoding/json" // 添加这行
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Server struct {
		Address string `yaml:"address"`
		Port    int    `yaml:"port"`
	} `yaml:"server"`
	API struct {
		GPTGodURL string `yaml:"gptgod_url"`
	} `yaml:"api"`
	ModelMapping map[string]string `yaml:"model_mapping"`
}

var config Config

func main() {
	loadConfig()
	//设置路由
	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*") // 允许所有域名访问，或替换为特定域名
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept")

		// 处理预检请求
		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			return
		}

		c.Next()
	})

	r.POST("/chat/completions", handleChat)

	addr := fmt.Sprintf("%s:%d", config.Server.Address, config.Server.Port)
	log.Printf("Server listening on %s", addr)
	r.Run(addr)
}

func loadConfig() {
	data, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("Error parsing config file: %v", err)
	}
}

func handleChat(c *gin.Context) {
	headers := make(map[string]string)

	// 收集所有需要的请求头
	headers["Content-Type"] = c.GetHeader("Content-Type")
	// 只有当 Accept 头存在时才添加
	if accept := c.GetHeader("Accept"); accept != "" {
		headers["Accept"] = accept
	}
	headers["Authorization"] = c.GetHeader("Authorization")

	if headers["Authorization"] == "" || !strings.HasPrefix(headers["Authorization"], "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid Authorization header"})
		return
	}

	var requestBody map[string]interface{}
	if err := c.BindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	log.Printf("Received request: %+v", requestBody)

	model, ok := requestBody["model"].(string)
	if ok {
		if mappedModel, exists := config.ModelMapping[model]; exists {
			requestBody["model"] = mappedModel
		}
	}

	response, err := forwardRequest(requestBody, headers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Received response: %s", string(response))

	c.Data(http.StatusOK, "application/json", response)
}

func forwardRequest(requestBody map[string]interface{}, headers map[string]string) ([]byte, error) {
	jsonBody, err := json.Marshal(requestBody) // 这里使用了 json
	if err != nil {
		return nil, err
	}

	// 打印实际发送的请求地址
	log.Printf("Forwarding request to: %s", config.API.GPTGodURL)

	// 打印实际发送的请求参数
	log.Printf("Request parameters: %s", string(jsonBody))

	req, err := http.NewRequest("POST", config.API.GPTGodURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}

	// 设置所有请求头
	for key, value := range headers {
		req.Header.Set(key, value)
		log.Printf("Setting header: %s: %s", key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 打印响应状态码
	log.Printf("Response status code: %d", resp.StatusCode)

	return responseBody, nil
}
