package pow

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"errors"
	"sync"
	"time"

	"github.com/iotaledger/goshimmer/packages/pow"
	"github.com/iotaledger/goshimmer/plugins/config"
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/hive.go/logger"
	_ "golang.org/x/crypto/blake2b" // required by crypto.BLAKE2b_512
)

var (
	// ErrMessageTooSmall is returned when the message is smaller than the 8-byte nonce.
	ErrMessageTooSmall = errors.New("message too small")
)

// parameters
var (
	hash = crypto.BLAKE2b_512

	// configured via parameters
	difficultyMutex sync.RWMutex
	difficulty      int
	numWorkers      int
	timeout         time.Duration

	powEvents  *PowEvents
	eventsOnce sync.Once
)

var (
	log *logger.Logger

	workerOnce sync.Once
	worker     *pow.Worker
)

// Worker returns the PoW worker instance of the PoW plugin.
func Worker() *pow.Worker {
	workerOnce.Do(func() {
		difficultyMutex.Lock()
		defer difficultyMutex.Unlock()
		log = logger.NewLogger(PluginName)
		// load the parameters
		difficulty = config.Node().Int(CfgPOWDifficulty)
		numWorkers = config.Node().Int(CfgPOWNumThreads)
		timeout = config.Node().Duration(CfgPOWTimeout)
		// create the worker
		worker = pow.New(hash, numWorkers)
		// ensure events are initialized
		Events()
	})
	return worker
}

// Events returns the pow events.
func Events() *PowEvents {
	eventsOnce.Do(func() {
		// init the events
		powEvents = &PowEvents{events.NewEvent(powDoneEventCaller)}
	})
	return powEvents
}

// DoPOW performs the PoW on the provided msg and returns the nonce.
func DoPOW(msg []byte) (uint64, error) {
	content, err := powData(msg)
	if err != nil {
		return 0, err
	}

	difficultyMutex.RLock()
	defer difficultyMutex.RUnlock()

	// get the PoW worker
	worker := Worker()

	log.Debugw("start PoW", "difficulty", difficulty, "numWorkers", numWorkers)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	start := time.Now()
	nonce, err := worker.Mine(ctx, content[:len(content)-pow.NonceBytes], difficulty)
	duration := time.Since(start)

	ev := &PowDoneEvent{
		Difficulty: difficulty,
		Duration:   duration,
	}

	Events().PowDone.Trigger(ev)

	log.Debugw("PoW stopped", "nonce", nonce, "err", err)

	return nonce, err
}

// powData returns the bytes over which PoW should be computed.
func powData(msgBytes []byte) ([]byte, error) {
	contentLength := len(msgBytes) - ed25519.SignatureSize
	if contentLength < pow.NonceBytes {
		return nil, ErrMessageTooSmall
	}
	return msgBytes[:contentLength], nil
}

// Tune changes pow difficulty at runtime.
func Tune(d int) {
	difficultyMutex.Lock()
	defer difficultyMutex.Unlock()

	difficulty = d
}
