package extredis

import (
	redis "github.com/redis/go-redis/v9"
)

var safeDecrLuaScript = redis.NewScript(`
local ret = '0'
local v = redis.call("GET", KEYS[1])
if v == false then
    redis.call("SET", KEYS[1], 0)
    ret = '0'
elseif v == '0' then
    ret = '0'
else
    ret = redis.call("DECR", KEYS[1])
end
return {
    ret
}
`)
