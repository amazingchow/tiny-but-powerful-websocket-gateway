package extredis

import (
	"time"
)

const (
	DefaultTTL = time.Minute
)

type Item struct {
	key   string
	value interface{}
	ttl   time.Duration
}

func (item *Item) Key() string {
	return item.key
}

func (item *Item) SetKey(key string) {
	item.key = key
}

func (item *Item) Value() interface{} {
	return item.value
}

func (item *Item) SetValue(value interface{}) {
	item.value = value
}

func (item *Item) TTL() time.Duration {
	if item.ttl <= 0 {
		return 0
	}
	if item.ttl < time.Second {
		return DefaultTTL
	}
	return item.ttl
}

func (item *Item) SetTTL(ttl time.Duration) {
	item.ttl = ttl
}
