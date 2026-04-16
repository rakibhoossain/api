package buffers

import "sync"

// GenericBuffer emulates a common memory buffer that stores items until size limit or cron timer invokes flush
type GenericBuffer struct {
	name  string
	items []interface{}
	mu    sync.Mutex
}

func NewGenericBuffer(name string) *GenericBuffer {
	return &GenericBuffer{
		name:  name,
		items: make([]interface{}, 0),
	}
}

func (b *GenericBuffer) Add(item interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.items = append(b.items, item)
}

func (b *GenericBuffer) flushLogic(batch []interface{}) error {
	// Abstract generic flush if required...
	return nil
}
