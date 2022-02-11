package action

import (
	"github.com/hectorgimenez/koolo/internal/hid"
	"math/rand"
	"time"
)

func Run(sequence ...HIDOperation) {
	for _, op := range sequence {
		op.execute()
		time.Sleep(op.delay())
	}
}

type HIDOperation interface {
	execute()
	delay() time.Duration
}

type delayedOperation struct {
	delayAfterOperation time.Duration
}

// Add random 1-30% extra delay to all operations
func (d delayedOperation) delay() time.Duration {
	rand.Seed(time.Now().UnixNano())
	randomPct := ((float32(rand.Intn(30-1+1)) + 1) / 100) + 1
	delayedMs := int64(float32(d.delayAfterOperation.Milliseconds()) * randomPct)

	return time.Duration(delayedMs) * time.Millisecond
}

type MouseDisplacement struct {
	delayedOperation
	x int
	y int
}

func (m MouseDisplacement) execute() {
	hid.MovePointer(m.x, m.y)
}

func NewMouseDisplacement(x, y int, delay time.Duration) MouseDisplacement {
	return MouseDisplacement{
		delayedOperation: delayedOperation{delayAfterOperation: delay},
		x:                x,
		y:                y,
	}
}