package nutsdb

import (
	"bytes"
	"errors"
	"strings"
	"time"

	"github.com/xujiajun/nutsdb/ds/list"
	"github.com/xujiajun/utils/strconv2"
)

// SeparatorForListKey represents separator for listKey
const SeparatorForListKey = "|"

// RPop removes and returns the last element of the list stored in the bucket at given bucket and key.
func (tx *Tx) RPop(bucket string, key []byte) (item []byte, err error) {
	item, err = tx.RPeek(bucket, key)
	if err != nil {
		return
	}

	return item, tx.push(bucket, key, DataRPopFlag, item)
}

// RPeek returns the last element of the list stored in the bucket at given bucket and key.
func (tx *Tx) RPeek(bucket string, key []byte) (item []byte, err error) {
	if err := tx.checkTxIsClosed(); err != nil {
		return nil, err
	}

	if _, ok := tx.db.ListIdx[bucket]; !ok {
		return nil, ErrBucket
	}

	item, _, err = tx.db.ListIdx[bucket].RPeek(string(key))

	return
}

// push sets values for list stored in the bucket at given bucket, key, flag and values.
func (tx *Tx) push(bucket string, key []byte, flag uint16, values ...[]byte) error {
	for _, value := range values {
		err := tx.put(bucket, key, value, Persistent, flag, uint64(time.Now().Unix()), DataStructureList)
		if err != nil {
			return err
		}
	}

	return nil
}

// RPush inserts the values at the tail of the list stored in the bucket at given bucket,key and values.
func (tx *Tx) RPush(bucket string, key []byte, values ...[]byte) error {
	if err := tx.checkTxIsClosed(); err != nil {
		return err
	}

	if strings.Contains(string(key), SeparatorForListKey) {
		return ErrSeparatorForListKey()
	}

	return tx.push(bucket, key, DataRPushFlag, values...)
}

// LPush inserts the values at the head of the list stored in the bucket at given bucket,key and values.
func (tx *Tx) LPush(bucket string, key []byte, values ...[]byte) error {
	if err := tx.checkTxIsClosed(); err != nil {
		return err
	}

	if strings.Contains(string(key), SeparatorForListKey) {
		return ErrSeparatorForListKey()
	}

	return tx.push(bucket, key, DataLPushFlag, values...)
}

// LPop removes and returns the first element of the list stored in the bucket at given bucket and key.
func (tx *Tx) LPop(bucket string, key []byte) (item []byte, err error) {
	item, err = tx.LPeek(bucket, key)
	if err != nil {
		return
	}

	return item, tx.push(bucket, key, DataLPopFlag, item)
}

// LPeek returns the first element of the list stored in the bucket at given bucket and key.
func (tx *Tx) LPeek(bucket string, key []byte) (item []byte, err error) {
	if err := tx.checkTxIsClosed(); err != nil {
		return nil, err
	}

	if _, ok := tx.db.ListIdx[bucket]; !ok {
		return nil, ErrBucket
	}

	item, err = tx.db.ListIdx[bucket].LPeek(string(key))

	return
}

// LSize returns the size of key in the bucket in the bucket at given bucket and key.
func (tx *Tx) LSize(bucket string, key []byte) (int, error) {
	if err := tx.checkTxIsClosed(); err != nil {
		return 0, err
	}

	if _, ok := tx.db.ListIdx[bucket]; !ok {
		return 0, ErrBucket
	}

	return tx.db.ListIdx[bucket].Size(string(key))
}

// LRange returns the specified elements of the list stored in the bucket at given bucket,key, start and end.
// The offsets start and stop are zero-based indexes 0 being the first element of the list (the head of the list),
// 1 being the next element and so on.
// Start and end can also be negative numbers indicating offsets from the end of the list,
// where -1 is the last element of the list, -2 the penultimate element and so on.
func (tx *Tx) LRange(bucket string, key []byte, start, end int) (list [][]byte, err error) {
	if err := tx.checkTxIsClosed(); err != nil {
		return nil, err
	}

	if _, ok := tx.db.ListIdx[bucket]; !ok {
		return nil, ErrBucket
	}

	return tx.db.ListIdx[bucket].LRange(string(key), start, end)
}

// LRem removes the first count occurrences of elements equal to value from the list stored in the bucket at given bucket,key,count.
// The count argument influences the operation in the following ways:
// count > 0: Remove elements equal to value moving from head to tail.
// count < 0: Remove elements equal to value moving from tail to head.
// count = 0: Remove all elements equal to value.
func (tx *Tx) LRem(bucket string, key []byte, count int) error {
	var err error

	size, err := tx.LSize(bucket, key)
	if err != nil {
		return err
	}

	if count >= size {
		return list.ErrCount
	}

	if count < 0 {
		index := size + count
		if index < 0 || index > size {
			return list.ErrCount
		}
	}

	return tx.push(bucket, key, DataLRemFlag, []byte(strconv2.IntToStr(count)))
}

// LSet sets the list element at index to value.
func (tx *Tx) LSet(bucket string, key []byte, index int, value []byte) error {
	var (
		err    error
		buffer bytes.Buffer
	)

	if err = tx.checkTxIsClosed(); err != nil {
		return err
	}

	if _, ok := tx.db.ListIdx[bucket]; !ok {
		return ErrBucket
	}

	if _, ok := tx.db.ListIdx[bucket].Items[string(key)]; !ok {
		return ErrKeyNotFound
	}

	size, _ := tx.LSize(bucket, key)
	if index < 0 || index >= size {
		return list.ErrIndexOutOfRange
	}

	buffer.Write(key)
	buffer.Write([]byte(SeparatorForListKey))
	indexBytes := []byte(strconv2.IntToStr(index))
	buffer.Write(indexBytes)
	newKey := buffer.Bytes()

	return tx.push(bucket, newKey, DataLSetFlag, value)
}

// LTrim trims an existing list so that it will contain only the specified range of elements specified.
// the offsets start and stop are zero-based indexes 0 being the first element of the list (the head of the list),
// 1 being the next element and so on.
// start and end can also be negative numbers indicating offsets from the end of the list,
// where -1 is the last element of the list, -2 the penultimate element and so on.
func (tx *Tx) LTrim(bucket string, key []byte, start, end int) error {
	var (
		err    error
		buffer bytes.Buffer
	)

	if err = tx.checkTxIsClosed(); err != nil {
		return err
	}

	if _, ok := tx.db.ListIdx[bucket]; !ok {
		return ErrBucket
	}

	if _, ok := tx.db.ListIdx[bucket].Items[string(key)]; !ok {
		return ErrKeyNotFound
	}

	if _, err := tx.LRange(bucket, key, start, end); err != nil {
		return err
	}

	buffer.Write(key)
	buffer.Write([]byte(SeparatorForListKey))
	buffer.Write([]byte(strconv2.IntToStr(start)))
	newKey := buffer.Bytes()

	return tx.push(bucket, newKey, DataLTrimFlag, []byte(strconv2.IntToStr(end)))
}

// ErrSeparatorForListKey returns when list key contains the SeparatorForListKey.
func ErrSeparatorForListKey() error {
	return errors.New("contain separator (" + SeparatorForListKey + ") for List key")
}
