package lcdm_2000_api

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/tarm/serial"
)

const (
	RequestStart          byte = 0x04
	ResponseStart         byte = 0x01
	CommunicationIdentify byte = 0x50
	TextStart             byte = 0x02
	TextEnd               byte = 0x03
)

type Baud int

const (
	Baud9600  Baud = 9600
	Baud19200 Baud = 19200
)

type ResponseType byte

const (
	ErrorResponse ResponseType = 0x00
	AckResponse   ResponseType = 0x06
	NackResponse  ResponseType = 0x15
	EotResponse   ResponseType = 0x04
)

type StatusCode byte

const (
	Good                                             StatusCode = 0x30
	NormalStop                                       StatusCode = 0x31
	PickupError                                      StatusCode = 0x32
	UpperCheckSensorJam                              StatusCode = 0x33
	OverflowBill                                     StatusCode = 0x34
	JamExitOrEjectSensor                             StatusCode = 0x35
	JamDivertSensor                                  StatusCode = 0x36
	UndefinedCommand                                 StatusCode = 0x37
	UpperBillEnd                                     StatusCode = 0x38
	CheckSensorEjectSensorCountMismatched            StatusCode = 0x3A
	BillCountZeroOrOverflow                          StatusCode = 0x3B
	DivertTimeout                                    StatusCode = 0x3C
	BillCountError                                   StatusCode = 0x3D
	SensorError                                      StatusCode = 0x3E
	RejectTrayIsNotRecognised                        StatusCode = 0x3F
	LowerBillEnd                                     StatusCode = 0x40
	MotorStop                                        StatusCode = 0x41
	TimeoutCheckEjectSensor                          StatusCode = 0x42
	TimeoutDivertEjectSensor                         StatusCode = 0x43
	NoUpperCashBox                                   StatusCode = 0x45
	NoLowerCashBox                                   StatusCode = 0x46
	DispensingTimeout                                StatusCode = 0x47
	EjectSensorJam                                   StatusCode = 0x48
	DiverterNotOperatedNormallyOrSolenoidSensorError StatusCode = 0x49
	BillsNotDispensedDiverterAbnormal                StatusCode = 0x4A
	CountingBillsDivertCheckSensorMismatched         StatusCode = 0x4B
	LowerCheckSensorJam                              StatusCode = 0x4C
	CountingBillsEjectExitSensorMismatched           StatusCode = 0x4D
	ReverseJam                                       StatusCode = 0x4E
	BillDispensedFromWrongCashBox                    StatusCode = 0x4F
	TimeoutCheckDivertSensor                         StatusCode = 0x50
)

type CashboxStatusCode byte

const (
	Normal  CashboxStatusCode = 0x30
	NearEnd CashboxStatusCode = 0x31
)

type LCDMDispenser struct {
	config  *serial.Config
	port    *serial.Port
	logging bool
	open    bool
}

type SensorStatus struct {
	CheckSensor1   bool
	CheckSensor2   bool
	CheckSensor3   bool
	CheckSensor4   bool
	DivertSensor1  bool
	DivertSensor2  bool
	EjectSensor    bool
	ExitSensor     bool
	SolenoidSensor bool
	UpperNearEnd   bool
	LowerNearEnd   bool
	CashBoxUpper   bool
	CashBoxLower   bool
	RejectTray     bool
}

func NewConnection(path string, baud Baud, logging bool) (LCDMDispenser, error) {
	c := &serial.Config{Name: path, Baud: int(baud), ReadTimeout: 5 * time.Second, Parity: serial.ParityNone, StopBits: serial.Stop1,
		Size: 8}

	o, err := serial.OpenPort(c)

	res := LCDMDispenser{}

	if err != nil {
		return res, err
	}

	res.config = c
	res.port = o
	res.logging = logging
	res.open = true

	return res, nil
}

func (s *LCDMDispenser) Open() error {
	if s.port == nil || !s.open {
		return errors.New("port already opened")
	}

	p, err := serial.OpenPort(s.config)

	if err != nil {
		return err
	}

	s.port = p
	s.open = true

	return nil
}

func (s *LCDMDispenser) Close() error {
	if s.port == nil || !s.open {
		return errors.New("port not opened")
	}

	err := s.port.Close()
	s.open = false

	return err
}

func (s *LCDMDispenser) Status() (StatusCode, SensorStatus, error) {
	status := SensorStatus{}

	err := sendRequest(s, 0x46, []byte{})

	if err != nil {
		return 0, status, err
	}

	response, err := readResponse(s)

	if err != nil {
		return 0, status, err
	}

	status.CheckSensor1 = (response[2] & (1 << 0)) != 0
	status.CheckSensor2 = (response[2] & (1 << 1)) != 0
	status.CheckSensor3 = (response[3] & (1 << 3)) != 0
	status.CheckSensor4 = (response[3] & (1 << 4)) != 0
	status.DivertSensor1 = (response[2] & (1 << 2)) != 0
	status.DivertSensor2 = (response[2] & (1 << 3)) != 0
	status.EjectSensor = (response[2] & (1 << 4)) != 0
	status.ExitSensor = (response[2] & (1 << 5)) != 0
	status.SolenoidSensor = (response[3] & (1 << 0)) != 0
	status.UpperNearEnd = (response[2] & (1 << 6)) != 0
	status.LowerNearEnd = (response[3] & (1 << 5)) != 0
	status.CashBoxUpper = (response[3] & (1 << 1)) != 0
	status.CashBoxLower = (response[3] & (1 << 2)) != 0
	status.RejectTray = (response[3] & (1 << 6)) != 0

	return StatusCode(response[1]), status, err
}

func (s *LCDMDispenser) Reset() error {
	err := sendRequest(s, 0x44, []byte{})

	if err != nil {
		return err
	}

	_, err = readResponse(s)

	if err != nil {
		return err
	}

	return nil
}

func (s *LCDMDispenser) UpperDispense(count byte) (StatusCode, CashboxStatusCode, byte, byte, error) {
	err := sendRequest(s, 0x45, []byte(fmt.Sprintf("%02d", count)))

	if err != nil {
		return 0, 0, 0, 0, err
	}

	response, err := readResponse(s)

	if err != nil {
		return 0, 0, 0, 0, err
	}

	val, _ := strconv.ParseUint(string(response[0:2]), 10, 8)
	checkSensor := byte(val)
	val, _ = strconv.ParseUint(string(response[2:34]), 10, 8)
	exitSensor := byte(val)

	return StatusCode(response[4]), CashboxStatusCode(response[5]), checkSensor, exitSensor, nil
}

func (s *LCDMDispenser) LowerDispense(count byte) (StatusCode, CashboxStatusCode, byte, byte, error) {
	err := sendRequest(s, 0x55, []byte(fmt.Sprintf("%02d", count)))

	if err != nil {
		return 0, 0, 0, 0, err
	}

	response, err := readResponse(s)

	if err != nil {
		return 0, 0, 0, 0, err
	}

	val, _ := strconv.ParseUint(string(response[0:2]), 10, 8)
	checkSensor := byte(val)
	val, _ = strconv.ParseUint(string(response[2:34]), 10, 8)
	exitSensor := byte(val)

	return StatusCode(response[4]), CashboxStatusCode(response[5]), checkSensor, exitSensor, nil
}

func (s *LCDMDispenser) Dispense(upperCount byte, lowerCount byte) (StatusCode, CashboxStatusCode, byte, byte, byte, byte, error) {
	err := sendRequest(s, 0x55, []byte(fmt.Sprintf("%02d%02d", upperCount, lowerCount)))

	if err != nil {
		return 0, 0, 0, 0, 0, 0, err
	}

	response, err := readResponse(s)

	if err != nil {
		return 0, 0, 0, 0, 0, 0, err
	}

	val, _ := strconv.ParseUint(string(response[0:2]), 10, 8)
	upperCheckSensor := byte(val)
	val, _ = strconv.ParseUint(string(response[2:34]), 10, 8)
	upperExitSensor := byte(val)

	val, _ = strconv.ParseUint(string(response[0:2]), 10, 8)
	lowerCheckSensor := byte(val)
	val, _ = strconv.ParseUint(string(response[2:34]), 10, 8)
	lowerExitSensor := byte(val)

	return StatusCode(response[8]), CashboxStatusCode(response[9]), upperCheckSensor, upperExitSensor, lowerCheckSensor, lowerExitSensor, nil
}

func (s *LCDMDispenser) RomVersion() (string, string, error) {
	err := sendRequest(s, 0x47, []byte{})

	if err != nil {
		return "", "", err
	}

	response, err := readResponse(s)

	if err != nil {
		return "", "", err
	}

	return string(response[2:4]), string(response[4:8]), nil
}

func (s *LCDMDispenser) Ack() {
	_, _ = s.port.Write([]byte{0x06})
}

func (s *LCDMDispenser) Nack() {
	_, _ = s.port.Write([]byte{0x15})
}

func readResponse(v *LCDMDispenser) ([]byte, error) {
	resp, err := readRespCode(v)

	if err != nil {
		return nil, err
	}

	if resp != AckResponse {
		return nil, errors.New("Response not ACK")
	}

	data, err := readRespData(v)

	if err != nil {
		return nil, err
	}

	v.Ack()

	time.Sleep(time.Millisecond * 200)

	return data, nil
}

func readRespCode(v *LCDMDispenser) (ResponseType, error) {
	var buf []byte
	innerBuf := make([]byte, 256)

	totalRead := 0
	readTriesCount := 0
	maxReadCount := 1050

	for ; ; {
		readTriesCount += 1

		if readTriesCount >= maxReadCount {
			return ErrorResponse, fmt.Errorf("Reads tries exceeded")
		}

		n, err := v.port.Read(innerBuf)

		if err != nil {
			return ErrorResponse, err
		}

		totalRead += n
		buf = append(buf, innerBuf[:n]...)

		if totalRead < 1 {
			continue
		}
		break
	}

	if buf[0] == 0x06 {
		if v.logging {
			fmt.Printf("<- ACK\n")
		}
		return AckResponse, nil // TODO Ack
	}

	if buf[0] == 0x15 {
		if v.logging {
			fmt.Printf("<- NAK\n")
		}
		return NackResponse, nil
	}

	if buf[0] == 0x04 {
		if v.logging {
			fmt.Printf("<- EOT\n")
		}
		return EotResponse, nil
	}

	return ErrorResponse, nil
}

func readRespData(v *LCDMDispenser) ([]byte, error) {
	var buf []byte
	innerBuf := make([]byte, 256)

	totalRead := 0
	readTriesCount := 0
	maxReadCount := 1050

	lastRead := false

	for ; ; {
		readTriesCount += 1

		if readTriesCount >= maxReadCount {
			return nil, fmt.Errorf("Reads tries exceeded")
		}

		n, err := v.port.Read(innerBuf)

		if err != nil {
			return nil, err
		}

		totalRead += n
		buf = append(buf, innerBuf[:n]...)

		if len(buf) > 2 && buf[len(buf)-2] == TextEnd {
			lastRead = true
		}

		if lastRead == false {
			continue
		}

		break
	}

	if buf[0] != ResponseStart || buf[1] != CommunicationIdentify {
		fmt.Printf("<- %X\n", buf)
		return nil, fmt.Errorf("Response format invalid")
	}

	crc := buf[len(buf)-1]

	buf = buf[:len(buf)-1]

	crc2 := getChecksum(buf)

	if crc != crc2 {
		return nil, fmt.Errorf("Response verification failed")
	}

	if buf[2] != TextStart || buf[len(buf)-1] != TextEnd {
		return nil, fmt.Errorf("Response format invalid")
	}

	buf = buf[4 : len(buf)-1]

	if v.logging {
		fmt.Printf("<- %X\n", buf)
	}

	return buf, nil
}

func sendRequest(v *LCDMDispenser, commandCode byte, bytesData ...[]byte) error {
	if !v.open {
		return errors.New("serial port is closed")
	}

	buf := new(bytes.Buffer)

	length := 6

	for _, b := range bytesData {
		length += len(b)
	}

	_ = binary.Write(buf, binary.LittleEndian, RequestStart)
	_ = binary.Write(buf, binary.LittleEndian, CommunicationIdentify)
	_ = binary.Write(buf, binary.LittleEndian, TextStart)
	_ = binary.Write(buf, binary.LittleEndian, commandCode)

	for _, data := range bytesData {
		_ = binary.Write(buf, binary.LittleEndian, data)
	}

	_ = binary.Write(buf, binary.LittleEndian, TextEnd)

	crc := getChecksum(buf.Bytes())

	_ = binary.Write(buf, binary.LittleEndian, crc)

	if v.logging {
		fmt.Printf("-> %X\n", buf.Bytes())
	}

	_, err := v.port.Write(buf.Bytes())

	return err
}

func getChecksum(data []byte) byte {
	chksum := byte(0)

	for _, b := range data {
		chksum = chksum ^ b
	}

	return chksum
}
