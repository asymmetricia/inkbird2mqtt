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
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
	btApi "github.com/muka/go-bluetooth/api"
	btAdapter "github.com/muka/go-bluetooth/bluez/profile/adapter"
	"github.com/muka/go-bluetooth/bluez/profile/device"
	"github.com/sirupsen/logrus"
)

var sensors []string
var sensorMu sync.Mutex

func BluetoothDiscover(ctx context.Context, log logrus.FieldLogger, adapter *btAdapter.Adapter1) {
	if err := adapter.FlushDevices(); err != nil {
		log.WithError(err).Warning("could not flush devices")
	} else {
		log.Debug("devices flushed")
	}

	discovery, cancel, err := btApi.Discover(adapter, nil)
	if err != nil {
		log.WithError(err).Fatal("could not start discovery")
	}
	defer cancel()
	log.Debug("discovery started")

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-discovery:
			if !ok {
				return
			}

			if event.Type == btAdapter.DeviceRemoved {
				continue
			}

			log = log.WithField("dev.address", event.Path)

			dev, err := device.NewDevice1(event.Path)
			if err != nil {
				log.Print(err)
				continue
			}
			log = log.WithField("dev.name", dev.Properties.Name)

			if dev.Properties.Name == "tps" {
				sensorMu.Lock()
				sensors = append(sensors, dev.Properties.Address)
				sensorMu.Unlock()
			}
		}
	}
}

func BluetoothPoll(ctx context.Context, log logrus.FieldLogger, adapter *btAdapter.Adapter1, readings chan<- Reading) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sensorMu.Lock()
			for _, address := range sensors {
				log = log.WithField("dev.address", address)

				// fetch device data from bluez, which will (perhaps later) be populated with
				// latest advertisement
				dev, err := adapter.GetDeviceByAddress(address)
				if err != nil {
					log.WithError(err).Warn("could not get device")
					continue
				}

				if dev == nil || dev.Properties == nil {
					continue
				}

				// bluez "stores" manufacturer data, which in this case is sub-optimal, because
				// this non-compliant inkbird stores its temperature in the manufacturer ID field
				mfd := dev.Properties.ManufacturerData

				// no manufacturer data yet, skip
				if len(mfd) == 0 {
					continue
				}

				// one entry- latest temp
				if len(mfd) == 1 {
					for temp := range mfd {
						var bat int
						variant, ok := mfd[temp].(dbus.Variant)
						if !ok {
							log.Warningf("mfd is %T, not dbus.Variant", mfd[temp])
						} else {
							bytes, ok := variant.Value().([]byte)
							if !ok {
								log.Warningf("mfd variant value is %T, not []byte", variant.Value())
							} else {
								bat = int(bytes[5])
							}
						}
						log.Printf("%f°C %d bat", float64(temp)/100, bat)
						readings <- Reading{
							Address: address,
							Value:   float64(temp) / 100,
							Battery: bat,
						}
					}
				}

				// one or more- RemoveDevice clears the advertisements, so next time we see one, it'll be the most recent.
				_ = adapter.RemoveDevice(dev.Path())
			}
			sensorMu.Unlock()
		}
	}
}
