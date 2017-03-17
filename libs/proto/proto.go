package proto

import (
	"encoding/json"
	"errors"
	"fmt"
	"goim/libs/bufio"
	"goim/libs/bytes"
	"goim/libs/define"
	"goim/libs/encoding/binary"
	"goim/libs/net/websocket"

	"strconv"

	log "github.com/thinkboy/log4go"
)

// for tcp
const (
	MaxBodySize = int32(1 << 10)
)

const (
	// size
	PackSize      = 4
	HeaderSize    = 2
	VerSize       = 2
	OperationSize = 4
	SeqIdSize     = 4
	RawHeaderSize = PackSize + HeaderSize + VerSize + OperationSize + SeqIdSize
	MaxPackSize   = MaxBodySize + int32(RawHeaderSize)
	// offset
	PackOffset      = 0
	HeaderOffset    = PackOffset + PackSize
	VerOffset       = HeaderOffset + HeaderSize
	OperationOffset = VerOffset + VerSize
	SeqIdOffset     = OperationOffset + OperationSize
)

var (
	emptyProto    = Proto{}
	emptyJSONBody = []byte("{}")

	ErrProtoPackLen   = errors.New("default server codec pack length error")
	ErrProtoHeaderLen = errors.New("default server codec header length error")
)

var (
	ProtoReady  = &Proto{Operation: define.OP_PROTO_READY}
	ProtoFinish = &Proto{Operation: define.OP_PROTO_FINISH}
)

// Proto is a request&response written before every goim connect.  It is used internally
// but documented here as an aid to debugging, such as when analyzing
// network traffic.
// tcp:
// binary codec
// websocket & http:
// raw codec, with http header stored ver, operation, seqid
type Proto struct {
	Ver       int16           `json:"ver"`  // protocol version
	Operation int32           `json:"op"`   // operation for request
	SeqId     int32           `json:"seq"`  // sequence number chosen by client
	Body      json.RawMessage `json:"body"` // binary body bytes(json.RawMessage is []byte)
}

func (p *Proto) Reset() {
	*p = emptyProto
}

func (p *Proto) String() string {
	return fmt.Sprintf("\n-------- proto --------\nver: %d\nop: %d\nseq: %d\nbody: %v\n-----------------------", p.Ver, p.Operation, p.SeqId, p.Body)
}

func (p *Proto) WriteTo(b *bytes.Writer) {
	var (
		packLen = RawHeaderSize + int32(len(p.Body))
		buf     = b.Peek(RawHeaderSize)
	)
	binary.BigEndian.PutInt32(buf[PackOffset:], packLen)
	binary.BigEndian.PutInt16(buf[HeaderOffset:], int16(RawHeaderSize))
	binary.BigEndian.PutInt16(buf[VerOffset:], p.Ver)
	binary.BigEndian.PutInt32(buf[OperationOffset:], p.Operation)
	binary.BigEndian.PutInt32(buf[SeqIdOffset:], p.SeqId)
	if p.Body != nil {
		b.Write(p.Body)
	}
}

func (p *Proto) ReadTCP(rr *bufio.Reader) (err error) {
	var (
		bodyLen   int
		headerLen int16
		packLen   int32
		buf       []byte
	)
	if buf, err = rr.Pop(RawHeaderSize); err != nil {
		return
	}
	packLen = binary.BigEndian.Int32(buf[PackOffset:HeaderOffset])
	headerLen = binary.BigEndian.Int16(buf[HeaderOffset:VerOffset])
	p.Ver = binary.BigEndian.Int16(buf[VerOffset:OperationOffset])
	p.Operation = binary.BigEndian.Int32(buf[OperationOffset:SeqIdOffset])
	p.SeqId = binary.BigEndian.Int32(buf[SeqIdOffset:])
	if packLen > MaxPackSize {
		return ErrProtoPackLen
	}
	if headerLen != RawHeaderSize {
		return ErrProtoHeaderLen
	}
	if bodyLen = int(packLen - int32(headerLen)); bodyLen > 0 {
		p.Body, err = rr.Pop(bodyLen)
	} else {
		p.Body = nil
	}
	return
}

func (p *Proto) WriteTCP(wr *bufio.Writer) (err error) {
	var (
		buf     []byte
		packLen int32
	)
	if p.Operation == define.OP_RAW {
		// write without buffer, job concact proto into raw buffer
		_, err = wr.WriteRaw(p.Body)
		return
	}
	packLen = RawHeaderSize + int32(len(p.Body))
	if buf, err = wr.Peek(RawHeaderSize); err != nil {
		return
	}
	binary.BigEndian.PutInt32(buf[PackOffset:], packLen)
	binary.BigEndian.PutInt16(buf[HeaderOffset:], int16(RawHeaderSize))
	binary.BigEndian.PutInt16(buf[VerOffset:], p.Ver)
	binary.BigEndian.PutInt32(buf[OperationOffset:], p.Operation)
	binary.BigEndian.PutInt32(buf[SeqIdOffset:], p.SeqId)
	if p.Body != nil {
		_, err = wr.Write(p.Body)
	}
	return
}

//修改后的明文读取
func (p *Proto) ReadWebsocket(ws *websocket.Conn) (err error) {
	var (
		// bodyLen   int
		// headerLen int16
		// packLen   int32
		buf []byte
	)
	if _, buf, err = ws.ReadMessage(); err != nil {
		return
	}
	log.Debug("ReadWebsocket...")
	log.Debug("buf is " + string(buf))
	log.Debug("Proto is :" + p.String())

	// if len(buf) < RawHeaderSize {
	// 	return ErrProtoPackLen
	// }
	// packLen = binary.BigEndian.Int32(buf[PackOffset:HeaderOffset])
	// headerLen = binary.BigEndian.Int16(buf[HeaderOffset:VerOffset])
	// p.Ver = binary.BigEndian.Int16(buf[VerOffset:OperationOffset])
	// p.Operation = binary.BigEndian.Int32(buf[OperationOffset:SeqIdOffset])
	// p.SeqId = binary.BigEndian.Int32(buf[SeqIdOffset:])
	// if packLen > MaxPackSize {
	// 	return ErrProtoPackLen
	// }
	// if headerLen != RawHeaderSize {
	// 	return ErrProtoHeaderLen
	// }
	// if bodyLen = int(packLen - int32(headerLen)); bodyLen > 0 {
	// 	p.Body = buf[headerLen:packLen]
	// } else {
	// 	p.Body = nil
	// }
	p.Body = buf // string(buf)
	return
}

//框架原来ReadWebsocket。二进制读取
func (p *Proto) ReadWebsocketBinary(ws *websocket.Conn) (err error) {
	var (
		bodyLen   int
		headerLen int16
		packLen   int32
		buf       []byte
	)
	if _, buf, err = ws.ReadMessage(); err != nil {
		return
	}

	if len(buf) < RawHeaderSize {
		return ErrProtoPackLen
	}
	packLen = binary.BigEndian.Int32(buf[PackOffset:HeaderOffset])
	headerLen = binary.BigEndian.Int16(buf[HeaderOffset:VerOffset])
	p.Ver = binary.BigEndian.Int16(buf[VerOffset:OperationOffset])
	p.Operation = binary.BigEndian.Int32(buf[OperationOffset:SeqIdOffset])
	p.SeqId = binary.BigEndian.Int32(buf[SeqIdOffset:])
	if packLen > MaxPackSize {
		return ErrProtoPackLen
	}
	if headerLen != RawHeaderSize {
		return ErrProtoHeaderLen
	}
	if bodyLen = int(packLen - int32(headerLen)); bodyLen > 0 {
		p.Body = buf[headerLen:packLen]
	} else {
		p.Body = nil
	}
	return
}

func (p *Proto) packBodyasArray() (js []json.RawMessage) {
	var (
		j json.RawMessage
	)
	offset := int32(PackOffset)
	buf := p.Body[:]
	for {
		if (len(buf[offset:])) < RawHeaderSize {
			// should not be here
			break
		}
		packLen := binary.BigEndian.Int32(buf[offset : offset+HeaderOffset])
		packBuf := buf[offset : offset+packLen]
		j = json.RawMessage(packBuf[RawHeaderSize:])
		js = append(js, j)
		offset += packLen
	}
	return
}

//修改后的明文写入
func (p *Proto) WriteWebsocket(ws *websocket.Conn) (err error) {
	var (
		buf     []byte
		packLen int
		// _body   []byte
	)
	log.Debug("WriteWebsocket...")
	log.Debug("Proto is :" + p.String())
	if p.Operation == define.OP_RAW {

		b := p.packBodyasArray()

		// var cm []byte
		for _, m := range b {
			// if codec {
			// 	// TODO 改进我
			// 	if cm, err = icodec.EncodeGzipBytes(m); err != nil {
			// 		goto failed
			// 	}
			// 	m = json.RawMessage(cm)
			// }
			// var _m []byte
			// if len(m) > 16 {

			if err = ws.WriteMessage(websocket.TextMessage, m); err != nil {
				goto failed
			}
		}

		log.Debug("OP_RAW--p.Body is :" + string(p.Body))
		log.Debug("RawHeaderSize is :" + strconv.Itoa(RawHeaderSize))
		//err = ws.WriteMessage(websocket.TextMessage, p.Body)
		return
	}
	packLen = RawHeaderSize + len(p.Body)
	if err = ws.WriteHeader(websocket.TextMessage, packLen); err != nil {
		log.Debug("WriteHeader--p.Body is :" + string(p.Body))
		log.Debug("RawHeaderSize is :" + strconv.Itoa(RawHeaderSize))
		return
	}
	if buf, err = ws.Peek(RawHeaderSize); err != nil {
		log.Debug("Peek--p.Body is :" + string(p.Body))
		log.Debug("RawHeaderSize is :" + strconv.Itoa(RawHeaderSize))
		return
	}

	log.Debug("buf is " + string(buf))

	// binary.BigEndian.PutInt32(buf[PackOffset:], int32(packLen))
	// binary.BigEndian.PutInt16(buf[HeaderOffset:], int16(RawHeaderSize))
	// binary.BigEndian.PutInt16(buf[VerOffset:], p.Ver)
	// binary.BigEndian.PutInt32(buf[OperationOffset:], p.Operation)
	// binary.BigEndian.PutInt32(buf[SeqIdOffset:], p.SeqId)
	if p.Body != nil {
		err = ws.WriteBody(p.Body)
	}
failed:
	if err != nil {
		log.Error("proto: %v, WriteWebsocket error(%v)", p, err)
	}

	return
}

// func (p *Proto) WriteWebsocket(ws *websocket.Conn) (err error) {
// 	var (
// 		buf     []byte
// 		packLen int
// 		// _body   []byte
// 	)
// 	log.Debug("WriteWebsocket...")
// 	log.Debug("Proto is :" + p.String())
// 	if p.Operation == define.OP_RAW {
// 		log.Debug("OP_RAW--p.Body is :" + string(p.Body))
// 		log.Debug("RawHeaderSize is :" + strconv.Itoa(RawHeaderSize))
// 		err = ws.WriteMessage(websocket.TextMessage, p.Body)
// 		return
// 	}
// 	packLen = RawHeaderSize + len(p.Body)
// 	if err = ws.WriteHeader(websocket.TextMessage, packLen); err != nil {
// 		log.Debug("WriteHeader--p.Body is :" + string(p.Body))
// 		log.Debug("RawHeaderSize is :" + strconv.Itoa(RawHeaderSize))
// 		return
// 	}
// 	if buf, err = ws.Peek(RawHeaderSize); err != nil {
// 		log.Debug("Peek--p.Body is :" + string(p.Body))
// 		log.Debug("RawHeaderSize is :" + strconv.Itoa(RawHeaderSize))
// 		return
// 	}

// 	log.Debug("buf is " + string(buf))

// 	// binary.BigEndian.PutInt32(buf[PackOffset:], int32(packLen))
// 	// binary.BigEndian.PutInt16(buf[HeaderOffset:], int16(RawHeaderSize))
// 	// binary.BigEndian.PutInt16(buf[VerOffset:], p.Ver)
// 	// binary.BigEndian.PutInt32(buf[OperationOffset:], p.Operation)
// 	// binary.BigEndian.PutInt32(buf[SeqIdOffset:], p.SeqId)
// 	if p.Body != nil {
// 		err = ws.WriteBody(p.Body)
// 	}
// 	return
// }

// func (p *Proto) packBodyasArray() (js []json.RawMessage) {
// 	var (
// 		j json.RawMessage
// 	)
// 	offset := int32(PackOffset)
// 	buf := p.Body[:]
// 	for {
// 		if (len(buf[offset:])) < RawHeaderSize {
// 			// should not be here
// 			break
// 		}
// 		packLen := binary.BigEndian.Int32(buf[offset : offset+HeaderOffset])
// 		packBuf := buf[offset : offset+packLen]
// 		j = json.RawMessage(packBuf[RawHeaderSize:])
// 		js = append(js, j)
// 		offset += packLen
// 	}
// 	return
// }

// func (c *Conn) WriteJSON(v interface{}) error {
// 	w, err := c.NextWriter(TextMessage)
// 	if err != nil {
// 		return err
// 	}
// 	err1 := json.NewEncoder(w).Encode(v)
// 	err2 := w.Close()
// 	if err1 != nil {
// 		return err1
// 	}
// 	return err2
// }

// func (p *Proto) WriteWebsocket(wr *websocket.Conn) (err error) {
// 	if p.Body == nil {
// 		p.Body = emptyJSONBody
// 	}
// 	switch p.Operation {
// 	case define.OP_RAW:
// 		b := p.packBodyasArray()
// 		var cm []byte
// 		for _, m := range b {
// 			// if codec {
// 			// 	// TODO 改进我
// 			// 	// if cm, err = icodec.EncodeGzipBytes(m); err != nil {
// 			// 	// 	goto failed
// 			// 	// }
// 			// 	m = json.RawMessage(cm)
// 			// }
// 			if err = wr.WriteMessage(websocket.TextMessage, m); err != nil {
// 				goto failed
// 			}
// 		}
// 	case define.OP_HEARTBEAT_REPLY:
// 		// if codec {
// 		// 	var (
// 		// 		m   []byte
// 		// 		tmp []byte
// 		// 	)

// 		// 	if tmp, err = json.Marshal(p); err != nil {
// 		// 		return
// 		// 	}
// 		// 	if m, err = icodec.EncodeGzipBytes(tmp); err != nil {
// 		// 		return
// 		// 	}
// 		// 	err = wr.WriteMessage(websocket.TextMessage, m)
// 		// 	return
// 		// }
// 		err = wr.WriteJSON(p)
// 	default:
// 		// if codec {
// 		// 	if p.Body, err = icodec.EncodeGzipBytes(p.Body); err != nil {
// 		// 		goto failed
// 		// 	}
// 		// }
// 		err = wr.WriteMessage(websocket.TextMessage, p.Body)
// 	}
// failed:
// 	if err != nil {
// 		log.Error("proto: %v, WriteWebsocket error(%v)", p, err)
// 	}
// 	return
// }

//框架原来WriteWebsocket。二进制写入
func (p *Proto) WriteWebsocketBinary(ws *websocket.Conn) (err error) {
	var (
		buf     []byte
		packLen int
	)
	if p.Operation == define.OP_RAW {
		err = ws.WriteMessage(websocket.BinaryMessage, p.Body)
		return
	}
	packLen = RawHeaderSize + len(p.Body)
	if err = ws.WriteHeader(websocket.BinaryMessage, packLen); err != nil {
		return
	}
	if buf, err = ws.Peek(RawHeaderSize); err != nil {
		return
	}
	binary.BigEndian.PutInt32(buf[PackOffset:], int32(packLen))
	binary.BigEndian.PutInt16(buf[HeaderOffset:], int16(RawHeaderSize))
	binary.BigEndian.PutInt16(buf[VerOffset:], p.Ver)
	binary.BigEndian.PutInt32(buf[OperationOffset:], p.Operation)
	binary.BigEndian.PutInt32(buf[SeqIdOffset:], p.SeqId)
	if p.Body != nil {
		err = ws.WriteBody(p.Body)
	}
	return
}
