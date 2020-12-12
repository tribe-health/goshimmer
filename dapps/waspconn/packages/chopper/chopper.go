// package to help to split messages into smaller pieces and reassemble them
package chopper

// Copyright 2020 IOTA Stiftung
// SPDX-License-Identifier: Apache-2.0

import (
	"bytes"
	"fmt"
	"sync"
	"time"
)

const (
	// for the final data packet to be not bigger than tangle.MaxMessageSize
	// 4 - chunk id, 1 seq nr, 1 num chunks, 2 - data len
	chunkHeaderSize = 4 + 1 + 1 + 2
	maxTTL          = 5 * time.Minute
)

type Chopper struct {
	nextId  uint32
	mutex   *sync.Mutex
	chunks  map[uint32]*dataInProgress
	closeCh chan bool
}

type dataInProgress struct {
	buffer      [][]byte
	ttl         time.Time
	numReceived int
}

func NewChopper() *Chopper {
	c := Chopper{
		nextId:  0,
		mutex:   &sync.Mutex{},
		chunks:  make(map[uint32]*dataInProgress),
		closeCh: make(chan bool),
	}
	go c.cleanupLoop()
	return &c
}

func (c *Chopper) Close() {
	close(c.closeCh)
}

func (c *Chopper) cleanupLoop() {
	for {
		select {
		case <-c.closeCh:
			return
		case <-time.After(10 * time.Second):
			toDelete := make([]uint32, 0)
			nowis := time.Now()
			c.mutex.Lock()
			for id, dip := range c.chunks {
				if nowis.After(dip.ttl) {
					toDelete = append(toDelete, id)
				}
			}
			for _, id := range toDelete {
				delete(c.chunks, id)
			}
			c.mutex.Unlock()
		}
	}
}

func (c *Chopper) getNextMsgId() uint32 {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.nextId++
	return c.nextId
}

// ChopData chops data into pieces and adds header to each piece for IncomingChunk function to reassemble it
// the size of each pieces is maxChunkSize - 3, for the header of the above protocol
func (c *Chopper) ChopData(data []byte, maxChunkSize uint16) ([][]byte, bool) {
	maxSizeWithoutHeader := maxChunkSize - chunkHeaderSize
	if len(data) <= int(maxChunkSize) { // [KP] Was compared with tangle.MaxMessageSize
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
	id := c.getNextMsgId()
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

func (c *Chopper) IncomingChunk(data []byte, maxChunkSize uint16) ([]byte, error) {
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

	c.mutex.Lock()
	defer c.mutex.Unlock()

	dip, ok := c.chunks[msg.msgId]
	if !ok {
		dip = &dataInProgress{
			buffer: make([][]byte, int(msg.numChunks)),
			ttl:    time.Now().Add(maxTTL),
		}
		c.chunks[msg.msgId] = dip
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
	delete(c.chunks, msg.msgId)
	return buf.Bytes(), nil
}
