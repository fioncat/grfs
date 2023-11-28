package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/fioncat/grfs/types"
	bolt "go.etcd.io/bbolt"
)

var ErrMountPointNotFound = errors.New("could not find the mountpoint")

const boltMountPointBucketName = "mountpoint"

type boltMountPointMetadata struct {
	db *bolt.DB

	bucket []byte
}

func OpenBolt(cfg *types.Config) (types.MountPointMetadata, error) {
	path := filepath.Join(cfg.BaseDir, "metadata.db")
	db, err := bolt.Open(path, 0644, &bolt.Options{
		Timeout: cfg.OpenBoltTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("open boltdb: %w", err)
	}

	bucket := []byte(boltMountPointBucketName)
	err = ensureBoltBucket(db, bucket)
	if err != nil {
		return nil, err
	}

	return &boltMountPointMetadata{
		db:     db,
		bucket: bucket,
	}, nil
}

func ensureBoltBucket(db *bolt.DB, bucket []byte) error {
	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucket)
		return err
	})
	if err != nil {
		return fmt.Errorf("ensure bolt bucket %q: %v", string(bucket), err)
	}
	return nil
}

func (b *boltMountPointMetadata) Put(mp *types.MountPoint) error {
	key := []byte(mp.Repo.String())
	data, err := json.Marshal(mp)
	if err != nil {
		return fmt.Errorf("encode mountpoint to json: %w", err)
	}

	err = b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(b.bucket)
		return bucket.Put(key, data)
	})
	if err != nil {
		return fmt.Errorf("boltdb put: %w", err)
	}

	return nil
}

func (b *boltMountPointMetadata) Get(repo *types.Repository) (*types.MountPoint, error) {
	key := []byte(repo.String())

	var data []byte
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(b.bucket)
		data = bucket.Get(key)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("boltdb get: %w", err)
	}

	if len(data) == 0 {
		return nil, ErrMountPointNotFound
	}

	return b.decodeData(data)
}

func (b *boltMountPointMetadata) List() ([]*types.MountPoint, error) {
	var mps []*types.MountPoint
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(b.bucket)
		cursor := bucket.Cursor()
		for key, data := cursor.First(); key != nil; key, data = cursor.Next() {
			mp, err := b.decodeData(data)
			if err != nil {
				return fmt.Errorf("decode mountpoint %q: %w", string(key), err)
			}
			mps = append(mps, mp)
		}
		return nil
	})
	return mps, err
}

func (b *boltMountPointMetadata) Remove(mp *types.MountPoint) error {
	key := []byte(mp.Repo.String())
	err := b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(b.bucket)
		return bucket.Delete(key)
	})
	if err != nil {
		return fmt.Errorf("delete boltdb: %w", err)
	}

	return nil
}

func (b *boltMountPointMetadata) Close() error {
	return b.db.Close()
}

func (b *boltMountPointMetadata) decodeData(data []byte) (*types.MountPoint, error) {
	var mp types.MountPoint
	err := json.Unmarshal(data, &mp)
	if err != nil {
		return nil, fmt.Errorf("decode mountpoint json in metadata: %w", err)
	}

	err = mp.Validate()
	if err != nil {
		return nil, fmt.Errorf("validate mountpoint in metadata: %w", err)
	}

	return &mp, nil
}
