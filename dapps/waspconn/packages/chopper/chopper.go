// package to help to split messages into smaller pieces and reassemble them
package chopper

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	waspconn2 "github.com/iotaledger/goshimmer/dapps/waspconn/packages/waspconn"
	"github.com/iotaledger/goshimmer/packages/binary/messagelayer/payload"
)

const (
	// for the final data packet to be not bigger than payload.MaxMessageSize
	// 4 - chunk id, 1 seq nr, 1 num chunks, 2 - data len
	chunkHeaderSize = 4 + 1 + 1 + 2
	maxTTL          = 5 * time.Minute
)

// special wrapper message for chunks of larger than buffer messages
type msgChunk struct {
	msgId       uint32
	chunkSeqNum byte
	numChunks   byte
	data        []byte
}

type dataInProgress struct {
	buffer      [][]byte
	ttl         time.Time
	numReceived int
}

var (
	nextId       uint32
	chopperMutex sync.Mutex
	chunks       = make(map[uint32]*dataInProgress)
)

// garbage collector
func init() {
	go func() {
		for {
			time.Sleep(10 * time.Second)
			toDelete := make([]uint32, 0)
			nowis := time.Now()
			chopperMutex.Lock()
			for id, dip := range chunks {
				if nowis.After(dip.ttl) {
					toDelete = append(toDelete, id)
				}
			}
			for _, id := range toDelete {
				delete(chunks, id)
			}
			chopperMutex.Unlock()
		}
	}()
}

func getNextMsgId() uint32 {
	chopperMutex.Lock()
	defer chopperMutex.Unlock()
	nextId++
	return nextId
}

// ChopData chops data into pieces and adds header to each piece for IncomingChunk function to reassemble it
// the size of each pieces is payload.MaxMessageSize - 3, for the header of the above protocol
func ChopData(data []byte, maxChunkSize uint16) ([][]byte, bool) {
	maxSizeWithoutHeader := maxChunkSize - chunkHeaderSize
	if len(data) <= payload.MaxMessageSize {
		return nil, false // no need to split
	}
	if len(data) > int(maxChunkSize)*255 {
		panic("ChopData: too long data to chop")
	}
	numChunks := byte(len(data) / int(maxSizeWithoutHeader))
	if len(data)%int(maxSizeWithoutHeader) > 0 {
		numChunks++
	}
	if numChunks < 2 {
		panic("ChopData: internal inconsistency 1")
	}
	id := getNextMsgId()
	ret := make([][]byte, 0, numChunks)
	var d []byte
	for i := byte(0); i < numChunks; i++ {
		if len(data) > int(maxSizeWithoutHeader) {
			d = data[:maxSizeWithoutHeader]
			data = data[maxSizeWithoutHeader:]
		} else {
			d = data
		}
		chunk := &msgChunk{
			msgId:       id,
			chunkSeqNum: i,
			numChunks:   numChunks,
			data:        d,
		}
		dtmp := chunk.encode()
		if len(dtmp) > int(maxChunkSize) {
			panic("ChopData: internal inconsistency 2")
		}
		ret = append(ret, dtmp)
	}
	return ret, true
}

func IncomingChunk(data []byte, maxChunkSize uint16) ([]byte, error) {
	maxSizeWithoutHeader := maxChunkSize - chunkHeaderSize
	msg := msgChunk{}
	if err := msg.decode(data, maxSizeWithoutHeader); err != nil {
		return nil, err
	}
	switch {
	case len(msg.data) > int(maxChunkSize):
		return nil, fmt.Errorf("too long data chunk")

	case msg.chunkSeqNum >= msg.numChunks:
		return nil, fmt.Errorf("wrong incoming data chunk seq number")
	}

	chopperMutex.Lock()
	defer chopperMutex.Unlock()

	dip, ok := chunks[msg.msgId]
	if !ok {
		dip = &dataInProgress{
			buffer: make([][]byte, int(msg.numChunks)),
			ttl:    time.Now().Add(maxTTL),
		}
		chunks[msg.msgId] = dip
	} else {
		if dip.buffer[msg.chunkSeqNum] != nil {
			return nil, fmt.Errorf("repeating seq number")
		}
	}
	dip.buffer[msg.chunkSeqNum] = msg.data
	dip.numReceived++

	if dip.numReceived != len(dip.buffer) {
		return nil, nil
	}
	// finished assembly of data chunks
	var buf bytes.Buffer
	for _, d := range dip.buffer {
		buf.Write(d)
	}
	delete(chunks, msg.msgId)
	return buf.Bytes(), nil
}

func (c *msgChunk) encode() []byte {
	var buf bytes.Buffer

	_ = waspconn2.WriteUint32(&buf, c.msgId)
	_ = waspconn2.WriteByte(&buf, c.numChunks)
	_ = waspconn2.WriteByte(&buf, c.chunkSeqNum)
	_ = waspconn2.WriteBytes16(&buf, c.data)
	return buf.Bytes()
}

func (c *msgChunk) decode(data []byte, maxChunkSizeWithoutHeader uint16) error {
	rdr := bytes.NewReader(data)
	if err := waspconn2.ReadUint32(rdr, &c.msgId); err != nil {
		return err
	}
	if err := waspconn2.ReadByte(rdr, &c.numChunks); err != nil {
		return err
	}
	if err := waspconn2.ReadByte(rdr, &c.chunkSeqNum); err != nil {
		return err
	}
	if data, err := waspconn2.ReadBytes16(rdr); err != nil {
		return err
	} else {
		c.data = data
	}
	if c.chunkSeqNum >= c.numChunks {
		return fmt.Errorf("wrong data chunk format")
	}
	if len(c.data) != int(maxChunkSizeWithoutHeader) && c.chunkSeqNum != c.numChunks-1 {
		return fmt.Errorf("wrong data chunk length")
	}
	return nil
}
