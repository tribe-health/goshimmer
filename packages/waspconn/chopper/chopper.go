// package to help to split messages into smaller pieces and reassemble them
package chopper

import (
	"bytes"
	"fmt"
	"github.com/iotaledger/goshimmer/packages/waspconn"
	"github.com/iotaledger/hive.go/netutil/buffconn"
	"sync"
	"time"
)

const (
	// for the final data packet to be not bigger than buffconn.MaxMessageSize
	// 4 - chin id, 1 seq nr, 1 num chunks, 1 msg code
	maxChunkSize = buffconn.MaxMessageSize - 4 - 1 - 1 - 1
	maxTTL       = 5 * time.Minute
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

func ChopData(data []byte) [][]byte {
	if len(data) <= buffconn.MaxMessageSize {
		return [][]byte{data} // no need to split
	}
	if len(data) > maxChunkSize*255 {
		panic("too long data to chop")
	}
	numChunks := byte(len(data) / maxChunkSize)
	if len(data)%maxChunkSize > 0 {
		numChunks++
	}
	if numChunks < 2 {
		return [][]byte{data} // no need to split
	}
	id := getNextMsgId()
	ret := make([][]byte, 0, numChunks)
	var d []byte
	for i := byte(0); i < numChunks; i++ {
		if len(data) > maxChunkSize {
			d = data[:maxChunkSize]
			data = data[maxChunkSize:]
		} else {
			d = data
		}
		chunk := &msgChunk{
			msgId:       id,
			chunkSeqNum: i,
			numChunks:   numChunks,
			data:        d,
		}
		ret = append(ret, chunk.encode())
	}
	return ret
}

func IncomingChunk(data []byte) ([]byte, error) {
	msg := msgChunk{}
	if err := msg.decode(data); err != nil {
		return nil, err
	}
	switch {
	case len(msg.data) > maxChunkSize:
		return nil, fmt.Errorf("too long data chunk")

	case msg.chunkSeqNum >= msg.numChunks:
		return nil, fmt.Errorf("wrong incoming data chunk seq number")

	case msg.chunkSeqNum < msg.numChunks-1 && len(msg.data) != maxChunkSize:
		return nil, fmt.Errorf("wrong incoming data chunk size")
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

	_ = waspconn.WriteUint32(&buf, c.msgId)
	_ = waspconn.WriteByte(&buf, c.numChunks)
	_ = waspconn.WriteByte(&buf, c.chunkSeqNum)
	_ = waspconn.WriteBytes16(&buf, c.data)
	return buf.Bytes()
}

func (c *msgChunk) decode(data []byte) error {
	rdr := bytes.NewReader(data)
	if err := waspconn.ReadUint32(rdr, &c.msgId); err != nil {
		return err
	}
	if err := waspconn.ReadByte(rdr, &c.numChunks); err != nil {
		return err
	}
	if err := waspconn.ReadByte(rdr, &c.chunkSeqNum); err != nil {
		return err
	}
	if data, err := waspconn.ReadBytes16(rdr); err != nil {
		return err
	} else {
		c.data = data
	}
	if c.chunkSeqNum >= c.numChunks {
		return fmt.Errorf("wrong data chunk format")
	}
	if len(c.data) != maxChunkSize && c.chunkSeqNum != c.numChunks-1 {
		return fmt.Errorf("wrong data chunk length")
	}
	return nil
}
