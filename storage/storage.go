package storage

import (
	"clipboard/api"
	"encoding/json"
	"time"

	"go.etcd.io/bbolt"
)

type Impl struct {
	db *bbolt.DB
}

func (i *Impl) Get(timestamp string) (clip *api.ClippedImage, err error) {
	err = i.bucket(view, "clips", func(bucket *bbolt.Bucket) error {
		if value := bucket.Get([]byte(timestamp)); value != nil {
			clip = &api.ClippedImage{}
			return json.Unmarshal(value, clip)
		}
		return nil
	})
	return
}

func (i *Impl) Save(clip *api.ClippedImage) (err error) {
	var value []byte
	if value, err = json.Marshal(clip); err == nil {
		err = i.bucket(update, "clips", func(bucket *bbolt.Bucket) error {
			return bucket.Put([]byte(clip.TimeStamp), value)
		})
	}
	return
}

func (i *Impl) Delete(timestamp string) (err error) {
	return i.bucket(update, "clips", func(bucket *bbolt.Bucket) error {
		return bucket.Delete([]byte(timestamp))
	})
}

func (i *Impl) List() (timestamps []string, err error) {
	timestamps = []string{}
	err = i.bucket(view, "clips", func(bucket *bbolt.Bucket) error {
		cursor := bucket.Cursor()
		for k, _ := cursor.First(); k != nil; k, _ = cursor.Next() {
			timestamps = append(timestamps, string(k))
		}
		return nil
	})
	return
}

type operation int

const view operation = 0
const update operation = 1

func (i *Impl) bucket(op operation, name string, callback func(bucket *bbolt.Bucket) error) error {
	switch op {
	case update:
		return i.db.Update(func(tx *bbolt.Tx) error {
			return callback(tx.Bucket([]byte(name)))
		})
	case view:
		return i.db.View(func(tx *bbolt.Tx) error {
			return callback(tx.Bucket([]byte(name)))
		})
	default:
		panic("not possible")
	}
}

func (i *Impl) Open(path string, timeout time.Duration) (err error) {
	i.db, err = bbolt.Open(path, 0666, &bbolt.Options{Timeout: timeout})
	if err == nil {
		return i.db.Update(func(tx *bbolt.Tx) error {
			for _, bucket := range []string{"clips"} {
				_, err := tx.CreateBucketIfNotExists([]byte(bucket))
				if err != nil {
					return err
				}
			}
			return nil
		})
	}
	return
}

func (i *Impl) Close() (err error) {
	if i.db != nil {
		err = i.db.Close()
	}
	return
}

func New() api.StorageApi {
	return &Impl{}
}
