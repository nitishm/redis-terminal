/*
MIT License

Copyright (c) [2018] [Nitish Malhotra]

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/
package redisapi

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
)

type Redis struct {
	pool *redis.Pool
}

func NewRedis(server string) (r *Redis, err error) {
	p := &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
	r = &Redis{p}
	return
}

func (r *Redis) GetKeys(pattern string) (keys []string, err error) {
	conn := r.pool.Get()

	iter := 0
	for {
		arr, err := redis.Values(conn.Do("SCAN", iter, "MATCH", pattern))
		if err != nil {
			return keys, fmt.Errorf("error retrieving '%s' keys", pattern)
		}

		iter, _ = redis.Int(arr[0], nil)
		k, _ := redis.Strings(arr[1], nil)
		keys = append(keys, k...)

		if iter == 0 {
			break
		}
	}

	return keys, nil
}

func (r *Redis) GetValue(key string) (v interface{}, err error) {
	conn := r.pool.Get()
	t, err := r.GetType(key)
	if err != nil {
		return
	}

	switch t {
	case "hash":
		v, err = redis.StringMap(conn.Do("HGETALL", key))
		if err != nil {
			return "", err
		}
	case "list":
		v, err = redis.Strings(conn.Do("LRANGE", key, 0, -1))
		if err != nil {
			return "", err
		}
	case "string":
		v, err = redis.String(conn.Do("GET", key))
		if err != nil {
			return "", err
		}
	default:
		fmt.Printf("Case not supported")
		return
	}
	return
}

func (r *Redis) GetType(key string) (t string, err error) {
	conn := r.pool.Get()
	return redis.String(conn.Do("TYPE", key))
}

func PrintKey(r *Redis, key string) (s string, err error) {
	v, err := r.GetValue(key)
	if err != nil {
		return
	}

	b, err := json.MarshalIndent(v, "[green]", "\t")
	if err != nil {
		return
	}
	s = "[green]" + string(b)

	return
}
