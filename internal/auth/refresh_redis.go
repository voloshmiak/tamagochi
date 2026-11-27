package auth

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/voloshmiak/tamagochi/internal/platform/clock"
)

type RedisRefreshStore struct {
	Client *redis.Client
	Clock  clock.Clock
}

const (
	keyToken   = "refresh:tok:"
	keyFamily  = "refresh:fam:"
	keyRevoked = "refresh:rev:"
)

var saveScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[3]) == 1 then return 0 end
redis.call('HSET', KEYS[1], 'family', ARGV[2], 'user', ARGV[3], 'used', '0', 'exp', ARGV[4])
redis.call('PEXPIREAT', KEYS[1], ARGV[4])
redis.call('SADD', KEYS[2], ARGV[1])
redis.call('PEXPIREAT', KEYS[2], ARGV[4])
return 1
`)

var consumeScript = redis.NewScript(`
local v = redis.call('HMGET', KEYS[1], 'family', 'user', 'used', 'exp')
if not v[1] then return {0} end
if v[3] == '1' then return {2, v[1], v[2], v[4]} end
redis.call('HSET', KEYS[1], 'used', '1')
return {1, v[1], v[2], v[4]}
`)

var revokeScript = redis.NewScript(`
local family = redis.call('HGET', KEYS[1], 'family')
if not family then return 0 end
local famKey = ARGV[2] .. family
for _, id in ipairs(redis.call('SMEMBERS', famKey)) do
  redis.call('DEL', ARGV[1] .. id)
end
local ttl = redis.call('PTTL', famKey)
if ttl < 1 then ttl = 1 end
redis.call('DEL', famKey)
redis.call('SET', ARGV[3] .. family, '1', 'PX', ttl)
return 1
`)

func (s RedisRefreshStore) Save(ctx context.Context, rec RefreshRecord) error {
	keys := []string{keyToken + rec.ID, keyFamily + rec.FamilyID, keyRevoked + rec.FamilyID}
	ok, err := saveScript.Run(ctx, s.Client, keys, rec.ID, rec.FamilyID, rec.UserID, rec.ExpiresAt.UnixMilli()).Int()
	if err != nil {
		return fmt.Errorf("save refresh token: %w", err)
	}
	if ok == 0 {
		return ErrRefreshRevoked
	}
	return nil
}

func (s RedisRefreshStore) Consume(ctx context.Context, id string) (RefreshRecord, error) {
	res, err := consumeScript.Run(ctx, s.Client, []string{keyToken + id}).Slice()
	if err != nil {
		return RefreshRecord{}, fmt.Errorf("consume refresh token: %w", err)
	}
	status, _ := res[0].(int64)
	if status == 0 {
		return RefreshRecord{}, ErrRefreshUnknown
	}
	family, _ := res[1].(string)
	user, _ := res[2].(string)
	expStr, _ := res[3].(string)
	expMs, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		return RefreshRecord{}, fmt.Errorf("refresh token %s: bad expiry %q", id, expStr)
	}
	rec := RefreshRecord{ID: id, FamilyID: family, UserID: user, ExpiresAt: time.UnixMilli(expMs).UTC()}
	if !rec.ExpiresAt.After(s.Clock.Now()) {
		return RefreshRecord{}, ErrRefreshUnknown
	}
	if status == 2 {
		return rec, ErrRefreshUsed
	}
	return rec, nil
}

func (s RedisRefreshStore) RevokeSession(ctx context.Context, id string) error {
	err := revokeScript.Run(ctx, s.Client, []string{keyToken + id}, keyToken, keyFamily, keyRevoked).Err()
	if err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("revoke refresh session: %w", err)
	}
	return nil
}
