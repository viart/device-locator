package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/viper"
	"github.com/viart/device-locator/pkg/fmip"
	"github.com/viart/device-locator/pkg/mqtt"
)

type Config struct {
	Mqtt     mqtt.Cfg
	Accounts []fmip.Credentials
}

func main() {
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/device-locator/")
	viper.SetConfigName("config")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalln("No configuration file found")
	}

	var cfg Config
	err := viper.Unmarshal(&cfg)
	if err != nil {
		log.Fatalf("Error parsing config file, %v", err)
	}

	mqtt, err := mqtt.New(cfg.Mqtt)
	if err != nil {
		log.Fatalf("Can't connet to MQTT server, %v", err)
	}

	session := fmip.NewISession()

	done := make(chan bool, 1)
	errs := make(chan error)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigs:
			fmt.Println(" Got shutdown event, exiting gracefully ...")
			done <- true
		case err := <-errs:
			log.Fatalln(err)
			done <- true
		}
	}()

	for _, account := range cfg.Accounts {
		go func(account fmip.Credentials) {
			res, err := session.Init(account.Username, account.Password)
			if err != nil {
				errs <- fmt.Errorf("Unable to init the iClient: %s", err)
			}

			prsID := res.ServerContext.PrsID
			authToken := res.ServerContext.AuthToken

			mqtt.Track(account.Username, res)

			for {
				// 5s+
				time.Sleep(time.Duration(5+rand.Intn(55)) * time.Second)
				res, err := session.Refresh(account.Username, prsID, authToken)
				if err != nil {
					errs <- fmt.Errorf("Unable to refresh the iClient: %s", err)
				}

				mqtt.Track(account.Username, res)
			}
		}(account)
	}

	<-done
}
