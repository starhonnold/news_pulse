package main

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	logger.Info("Testing News Pulse System")

	// Тестируем доступность API Gateway
	logger.Info("Testing API Gateway...")
	resp, err := http.Get("http://localhost:8080/health")
	if err != nil {
		logger.WithError(err).Error("API Gateway not accessible")
	} else {
		logger.WithField("status", resp.StatusCode).Info("API Gateway is accessible")
		resp.Body.Close()
	}

	// Тестируем доступность News Management Service
	logger.Info("Testing News Management Service...")
	resp, err = http.Get("http://localhost:8082/health")
	if err != nil {
		logger.WithError(err).Error("News Management Service not accessible")
	} else {
		logger.WithField("status", resp.StatusCode).Info("News Management Service is accessible")
		resp.Body.Close()
	}

	// Тестируем доступность Pulse Service
	logger.Info("Testing Pulse Service...")
	resp, err = http.Get("http://localhost:8083/health")
	if err != nil {
		logger.WithError(err).Error("Pulse Service not accessible")
	} else {
		logger.WithField("status", resp.StatusCode).Info("Pulse Service is accessible")
		resp.Body.Close()
	}

	// Тестируем доступность News Parsing Service
	logger.Info("Testing News Parsing Service...")
	resp, err = http.Get("http://localhost:8081/health")
	if err != nil {
		logger.WithError(err).Error("News Parsing Service not accessible")
	} else {
		logger.WithField("status", resp.StatusCode).Info("News Parsing Service is accessible")
		resp.Body.Close()
	}

	// Тестируем получение новостей через API Gateway
	logger.Info("Testing news retrieval...")
	resp, err = http.Get("http://localhost:8080/api/v1/news?limit=5")
	if err != nil {
		logger.WithError(err).Error("Failed to retrieve news")
	} else {
		logger.WithField("status", resp.StatusCode).Info("News retrieval successful")
		resp.Body.Close()
	}

	logger.Info("System test completed")
}
