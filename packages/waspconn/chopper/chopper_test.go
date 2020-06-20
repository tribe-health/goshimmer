package chopper

import (
	"bytes"
	"crypto/rand"
	"github.com/iotaledger/hive.go/netutil/buffconn"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBasic(t *testing.T) {
	dataShort := make([]byte, 2000)
	_, _ = rand.Read(dataShort)

	dataLong := make([]byte, 80001)
	_, _ = rand.Read(dataLong)

	dataLong2 := make([]byte, 1000000)
	_, _ = rand.Read(dataLong2)

	dataExact := make([]byte, buffconn.MaxMessageSize)
	_, _ = rand.Read(dataExact)

	dataExact2 := make([]byte, 3*maxChunkSize)
	_, _ = rand.Read(dataExact2)

	choppedShort := ChopData(dataShort)
	assert.Equal(t, 1, len(choppedShort))
	assert.True(t, bytes.Equal(dataShort, choppedShort[0]))

	choppedExact := ChopData(dataExact)
	assert.Equal(t, 1, len(choppedExact))
	assert.True(t, bytes.Equal(dataExact, choppedExact[0]))

	choppedExact2 := ChopData(dataExact2)
	assert.Equal(t, 3, len(choppedExact2))

	choppedLong := ChopData(dataLong)
	assert.True(t, len(choppedLong) > 1)

	choppedLong2 := ChopData(dataLong2)
	assert.True(t, len(choppedLong2) > 1)

	for _, piece := range choppedExact2 {
		ret, err := IncomingChunk(piece)
		assert.NoError(t, err)
		if ret != nil {
			assert.True(t, bytes.Equal(dataExact2, ret))
		}
	}

	for _, piece := range choppedLong {
		ret, err := IncomingChunk(piece)
		assert.NoError(t, err)
		if ret != nil {
			assert.True(t, bytes.Equal(dataLong, ret))
		}
	}

	for i := len(choppedLong) - 1; i >= 0; i-- {
		ret, err := IncomingChunk(choppedLong2[i])
		assert.NoError(t, err)
		if ret != nil {
			assert.True(t, bytes.Equal(dataLong2, ret))
		}
	}
}
