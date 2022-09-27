package main

/*
   Copyright © 2022 Tricia Bogen

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	btApi "github.com/muka/go-bluetooth/api"
	btAdapter "github.com/muka/go-bluetooth/bluez/profile/adapter"
	"github.com/sirupsen/logrus"
)

const envLogLevel = "LOG_LEVEL"

func main() {
	defaultLevel := os.Getenv(envLogLevel)
	if defaultLevel == "" {
		defaultLevel = "INFO"
	}

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\ninkbird2mqtt Copyright © 2022 Tricia Bogen\n"+
			"This program comes with ABSOLUTELY NO WARRANTY.\n"+
			"This is free software, and you are welcome to redistribute it under certain conditions.\n")
	}

	logLevel := flag.String("log-level", defaultLevel, "logrus log level- "+
		"TRACE, DEBUG, INFO, WARN, ERROR, FATAL (env: "+envLogLevel+")")

	flag.Parse()

	lvl, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.SetLevel(lvl)

	logrus.Info("starting!")

	defer btApi.Exit()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())

	adapter, err := btAdapter.GetDefaultAdapter()
	if err != nil {
		log.Fatal(err)
	}

	log := logrus.WithField("adapter", adapter.Path())
	log.Debug("using default adapter")

	readings := make(chan Reading, 10)

	go BluetoothDiscover(ctx, log, adapter)
	go BluetoothPoll(ctx, log, adapter, readings)
	go MqttReport(ctx, log, readings)

	sig := <-sigs
	log.Printf("exiting on signal %v", sig)
	cancel()
}
