package main

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

func main() {
	// Initialize logger
	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Log startup message
	log.Info("Image Processing API starting...")
	fmt.Println("Image Processing API starting...")
}
