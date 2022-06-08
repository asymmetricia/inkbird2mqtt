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
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
)

var mqttHost = flag.String("mqtt-host", "localhost",
	"hostname of the MQTT server to publish to")
var mqttPrefix = flag.String("mqtt-prefix", "inkbird",
	"a prefix for the mqtt space to publish, messages are published to "+
		"{mqtt-prefix}/{mac}, with mac in format 00-11-22-...")
var mqttPort = flag.Int("mqtt-port", 1883, "port used to connect to MQTT")
var mqttProto = flag.String("mqtt-proto", "tcp", "protocol used to connect "+
	"to MQTT; options are `tcp`, `ssl`, `ws`")

type Reading struct {
	Address string
	Value   float64
	Battery int
}

func MqttReport(ctx context.Context, log logrus.FieldLogger, readings <-chan Reading) {
	log = log.WithFields(logrus.Fields{
		"mqtt-host":  *mqttHost,
		"mqtt-port":  *mqttPort,
		"mqtt-proto": *mqttProto,
	})

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", *mqttHost, *mqttPort))
	if err != nil {
		log.WithError(err).Fatal("could not validate mqtt config")
	}
	conn.Close()

	opts := mqtt.NewClientOptions().SetClientID("inkbird2mqtt")
	opts.AddBroker(fmt.Sprintf("%s://%s:%d", *mqttProto, *mqttHost, *mqttPort))
	client := mqtt.NewClient(opts)

	log.Info("connecting to MQTT server...")
	token := client.Connect()
	token.WaitTimeout(time.Minute)
	if err := token.Error(); err != nil {
		log.WithError(err).Fatal("mqtt connect failed")
	}
	if !client.IsConnected() {
		log.Fatalf("timeout waiting for mqtt to connect")
	}
	log.Info("connected to MQTT")
	defer client.Disconnect(1000)

	for {
		select {
		case <-ctx.Done():
			return
		case reading, ok := <-readings:
			if !ok {
				return
			}
			mqttToken := client.Publish(
				filepath.Join(
					*mqttPrefix,
					strings.Replace(reading.Address, ":", "-", -1)),
				1,
				true,
				strconv.FormatFloat(reading.Value, 'f', 2, 64))

			if mqttToken.Error() == nil {
				mqttToken = client.Publish(
					filepath.Join(
						*mqttPrefix,
						strings.Replace(reading.Address, ":", "-", -1),
						"battery"),
					1,
					true,
					strconv.FormatInt(int64(reading.Battery), 10))
			}

			if err := mqttToken.Error(); err != nil {
				log.WithError(err).Error("error while publishing")
			}
		}
	}
}
