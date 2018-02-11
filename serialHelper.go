package main

import (
	"errors"
	"go.bug.st/serial.v1"
	"go.bug.st/serial.v1/enumerator"
	"log"
	"strconv"
	"strings"
	"time"
)

var serialPort serial.Port
var SelectedPortId int
var SelectedDeviceId int

var SerialDevice1 string

func getDeviceNames() []string {
	var dn []string
	for _,d := range Cfg.Device {
		dn = append(dn, d.Name)
	}
	return dn
}

func getSerialPorts() (usbports []string, perr error) {
	//Devices.load()
	ports2, err := enumerator.GetDetailedPortsList()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	ports, err2 := serial.GetPortsList()
	if err2 != nil {
		log.Fatal(err)
		return nil, err2
	}
	if err != nil || err2 != nil {
		log.Fatal(err)
		return nil, err
	}
	if len(ports) == 0 {
		log.Println("No serial ports found!")
		return nil, nil
	}

	for _, port := range ports2 {
		if port.IsUSB {
			usbports = append(usbports, port.Name)
			for di, d := range Cfg.Device {
				if strings.ToLower(port.VID) == strings.ToLower(d.Vendor) && strings.ToLower(port.PID) == strings.ToLower(d.Product) {
					log.Printf("detected Device: %s\nportName: %s\n",Cfg.Device[di].Name, port.Name)
					SelectedPortId = len(usbports) - 1
					SelectedDeviceId = di
					SerialDevice1 = port.Name
				}
			}
		}
	}
	log.Printf("SelectedPortId: %d - SerialDevice1: %s - SelectedDeviceId: %d\n",SelectedPortId,SerialDevice1,SelectedDeviceId)
	return usbports, nil
}

func connectSerial(selSerialPort string) (err error) {
	c1 := make(chan int, 1)
	go func() {

		if selSerialPort == "" {
			err = errors.New("no device given")
			return
		}

		mode := &serial.Mode{
			BaudRate: Cfg.Device[SelectedDeviceId].Config.Serial.Baud,
			Parity:   serial.NoParity,
			DataBits: 8,
			StopBits: serial.OneStopBit,
		}
		serialPort, err = serial.Open(selSerialPort, mode)
		time.Sleep(time.Millisecond * time.Duration(Cfg.Device[SelectedDeviceId].Config.Serial.ConeectionTimeout))
		if err != nil {
			log.Println("error serial connect ", err)
		} else {
			c1 <- 1
		}
	}()
	select {
	case res := <-c1:
		log.Printf("serialPort %v  connected - res: %d\n", serialPort, res)
	case <-time.After(time.Second * time.Duration(Cfg.Device[SelectedDeviceId].Config.Serial.ConeectionTimeout)):
		err = errors.New("serial connection timeout")
	}

	if err != nil {
		return err
	}

	return
}

func sendSerialCmd(cmd string) {
	SerialResponse.Cmd = cmd
	SerialResponse.Code = -1
	SerialResponse.String = ""
	SerialResponse.Payload = ""

	log.Printf("send cmd: %s\n",cmd)
	temp := sendSerial(cmd)
	prepareResponse(temp)
	log.Printf("response:\n\tCode: %d\n\tString: %s \n\tPayloadLen: %d\n",SerialResponse.Code,SerialResponse.String,len(SerialResponse.Payload))
}

func sendSerial(cmdStr string) string {
	var resp string
	c1 := make(chan string)
	go func() {
		_, err := serialPort.Write([]byte(cmdStr + "\r\n"))
		if err != nil {
			log.Println("errro send serial: ", err, cmdStr)
		}
		time.Sleep(time.Millisecond * time.Duration(Cfg.Device[SelectedDeviceId].Config.Serial.WaitForReceive))
		resp = receiveSerial()
		c1 <- resp
	}()
	select {
	case resp := <-c1:
		return resp
	case <-time.After(time.Second * time.Duration(Cfg.Device[SelectedDeviceId].Config.Serial.ConeectionTimeout)):
		log.Println("sendSrial Timeout")
	}
	return resp
}

func receiveSerial() (resp string) {
	buff := make([]byte, 512)
	var err error
	var n = 0
	var c = 0
	for c < 1 {
		n, err = serialPort.Read(buff)
		if err != nil {
			log.Printf("error temp: %s - n %d - error (%s)\n", resp, n, err)
		}
		c = c + n
	}
	return string(buff[:c])
}

func deviceInfo(longInfo string) (shortInfo string) {
	shortInfo = "undefined"
	toks := strings.Split(longInfo, " ")
	if len(toks) >= 3 {
		shortInfo = toks[0] + " " + toks[1]
	} else {
		shortInfo = longInfo
	}
	return
}

func prepareResponse(res string) {
	var result []string
	res = strings.Replace(res, "\n", "#", -1)
	res = strings.Replace(res, "\r", "#", -1)
	res = strings.Replace(res, "##", "#", -1)

	if !strings.Contains(res, ":") {
		log.Println("no response given")
		serialPort.ResetInputBuffer()
		return
	}
	temp2 = strings.Split(res, ":")
	if len(temp2[1]) >= 2 {
		result = append(result, temp2[0])
		SerialResponse.Code, _ = strconv.Atoi(temp2[0])
		temp := strings.Split(temp2[1], "#")
		if len(temp) > 0 {
			for i, s := range temp {
				switch i {
				case 0:
					SerialResponse.String = s
				case 1:
					SerialResponse.Payload = s
				}
				if s != "" {
					result = append(result, s)
				}
			}
		}
	}
}
