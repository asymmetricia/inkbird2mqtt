# inkbird2mqtt

A tool that listens for bluetooth Low Energy broadcasts via bluez, looking for
Inkbird IBS-P01B broadcasts, and reporting them in to MQTT.

## Why?

Well, there are a few extant options ... but I like operating Golang tools
better than dealing with things like python, or homeassistant plugins. So, here
we are.

## Options

```
inkbird2mqtt [options]

  -log-level string
    	logrus log level- TRACE, DEBUG, INFO, WARN, ERROR, FATAL (env: LOG_LEVEL) (default "INFO")
  -mqtt-host string
    	hostname of the MQTT server to publish to (default "localhost")
  -mqtt-port int
    	port used to connect to MQTT (default 1883)
  -mqtt-prefix string
    	a prefix for the mqtt space to publish, messages are published to {mqtt-prefix}/{mac}, with mac in format 00-11-22-... (default "unifi")
  -mqtt-proto tcp
    	protocol used to connect to MQTT; options are tcp, `ssl`, `ws` (default "tcp")
```
