-- Sliding Window Rate Limiter Lua Script
-- Implements a sliding window algorithm for distributed rate limiting
--
-- KEYS[1] = rate limit key
-- ARGV[1] = current timestamp in microseconds
-- ARGV[2] = window size in microseconds
-- ARGV[3] = maximum requests allowed
-- ARGV[4] = request weight (usually 1)
--
-- Returns: {allowed (0/1), current_count, ttl_ms, reset_at_ms}

local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local max_requests = tonumber(ARGV[3])
local weight = tonumber(ARGV[4]) or 1

-- Calculate window boundaries
local window_start = now - window

-- Remove expired entries (outside the current window)
redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

-- Count current requests in window
local current_count = redis.call('ZCARD', key)

-- Check if adding this request would exceed the limit
if current_count + weight > max_requests then
    -- Rate limited - return info about when to retry
    local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
    local reset_at = now + window
    if oldest and #oldest >= 2 then
        reset_at = tonumber(oldest[2]) + window
    end
    local ttl_ms = math.ceil((reset_at - now) / 1000)
    if ttl_ms < 0 then ttl_ms = 0 end
    
    return {0, current_count, ttl_ms, math.ceil(reset_at / 1000)}
end

-- Add the new request with current timestamp as score
-- Use timestamp + random suffix to avoid collisions
local member = now .. ':' .. redis.call('INCR', key .. ':seq')
redis.call('ZADD', key, now, member)

-- Set TTL on the key to clean up after window expires
redis.call('PEXPIRE', key, math.ceil(window / 1000))

-- Also set TTL on sequence counter
redis.call('PEXPIRE', key .. ':seq', math.ceil(window / 1000))

-- Calculate remaining and reset time
local new_count = current_count + weight
local reset_at = now + window
local ttl_ms = math.ceil(window / 1000)

return {1, new_count, ttl_ms, math.ceil(reset_at / 1000)}
