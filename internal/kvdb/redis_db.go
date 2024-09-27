package kvdb

import (
	"context"

	"github.com/redis/go-redis/v9"
)

var (
	redisAddr = "127.0.0.1:6379"
)

type RedisDB struct {
	path   string
	client *redis.Client
}

func (db *RedisDB) WithDataPath(path string) *RedisDB {
	db.path = path
	return db
}

func (db *RedisDB) GetDbPath() string {
	return db.path
}

func (db *RedisDB) Open() error {
	db.client = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})
	return nil
}

func (db *RedisDB) Set(k, v []byte) error {
	ctx := context.Background()
	err := db.client.Set(ctx, string(k), string(v), 0).Err() //只是设置string类型
	return err
}

func (db *RedisDB) BatchSet(k, v [][]byte) error {
	ctx := context.Background()
	pipe := db.client.Pipeline() //通过 go-redis Pipeline 一次执行多个命令并读取返回值:
	for i, key := range k {
		pipe.Set(ctx, string(key), string(v[i]), 0)
	}

	_, err := pipe.Exec(ctx) //前面一个返回参数是执行语句
	return err
}

func (db *RedisDB) Get(k []byte) ([]byte, error) {
	ctx := context.Background()
	val, err := db.client.Get(ctx, string(k)).Result()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return []byte(val), nil
}

func (db *RedisDB) BatchGet(ks [][]byte) ([][]byte, error) {
	ctx := context.Background()
	values := make([][]byte, len(ks))
	for i, k := range ks {
		val, err := db.client.Get(ctx, string(k)).Result()
		if err == redis.Nil {
			values[i] = nil
		} else if err != nil {
			return nil, err
		} else {
			values[i] = []byte(val)
		}
	}
	return values, nil
}

func (db *RedisDB) Delete(k []byte) error {
	ctx := context.Background()
	return db.client.Del(ctx, string(k)).Err()
}

func (db *RedisDB) BatchDelete(ks [][]byte) error {
	ctx := context.Background()
	pipe := db.client.Pipeline()
	for _, k := range ks {
		pipe.Del(ctx, string(k))
	}

	_, err := pipe.Exec(ctx)
	return err
}

func (db *RedisDB) Has(k []byte) bool {
	ctx := context.Background()
	exists := db.client.Exists(ctx, string(k)).Val() > 0 //????
	return exists
}

func (db *RedisDB) IterDB(fn func(k, v []byte) error) int64 {
	ctx := context.Background()
	count := int64(0)
	cursor := uint64(0) //db.client.Scan函数要求的uint64
	for {
		var keys []string
		keys, cursor, _ = db.client.Scan(ctx, cursor, "*", 100).Result() //match 是一个可选的模式字符串，用于匹配特定的键。例如，使用 "*pattern*" 可以匹配包含 "pattern" 的所有键。
		for _, key := range keys {
			val, _ := db.client.Get(ctx, key).Result()
			if err := fn([]byte(key), []byte(val)); err != nil {
				break
			}
			count++
		}
		if cursor == 0 {
			break
		}
	}
	return count
}

func (db *RedisDB) IterKey(fn func(k []byte) error) int64 {
	ctx := context.Background()
	count := int64(0)
	cursor := uint64(0) //cursor 是一个 uint64 类型的值，表示扫描的游标。首次调用时，通常使用 0 作为起始游标。
	for {
		var keys []string
		keys, cursor, _ = db.client.Scan(ctx, cursor, "*", 100).Result() //最后一个参数（100） 是一个整数，表示希望每次迭代返回的大致数量。Redis 会尽力返回接近这个数量的元素，但实际上返回的数量可能会有所不同。
		for _, key := range keys {
			if err := fn([]byte(key)); err != nil {
				break
			}
			count++
		}
		if cursor == 0 {
			break
		}
	}
	return count
}

func (db *RedisDB) Close() error {
	return db.client.Close()
}
