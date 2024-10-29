package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Config struct {
	Server struct {
		Address string
		Port    int
	}
	API struct {
		GPTGodURL string
	}
	ModelMapping map[string]string
}

var config = Config{
	Server: struct {
		Address string
		Port    int
	}{
		Address: "127.0.0.1",
		Port:    9998,
	},
	API: struct {
		GPTGodURL string
	}{
		GPTGodURL: "https://api.gptgod.online/v1/chat/completions",
	},
	ModelMapping: map[string]string{
		"ggl-4":                       "gpt-4-turbo",
		"gclaude-3-5-sonnet":          "claude-3-5-sonnet-20240620",
		"gclaude-3-5-sonnet-20241022": "claude-3-5-sonnet-20241022",
		"gpt-4o":                      "gpt-4o",
	},
}

func main() {
	// 设置路由
	r := gin.Default()
	r.POST("/chat/completions", handleChat)

	addr := fmt.Sprintf("%s:%d", config.Server.Address, config.Server.Port)
	log.Printf("Server listening on %s", addr)
	r.Run(addr)
}

func handleChat(c *gin.Context) {
	headers := make(map[string]string)

	// 收集所有需要的请求头
	headers["Content-Type"] = c.GetHeader("Content-Type")
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
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	log.Printf("Forwarding request to: %s", config.API.GPTGodURL)

	log.Printf("Request parameters: %s", string(jsonBody))

	req, err := http.NewRequest("POST", config.API.GPTGodURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}

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

	log.Printf("Response status code: %d", resp.StatusCode)

	return responseBody, nil
}
