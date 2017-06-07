package block

import (
	"errors"
	"time"

	"github.com/coreos/pkg/capnslog"
	"github.com/alternative-storage/torus"
	"github.com/alternative-storage/torus/models"
)

var clog = capnslog.NewPackageLogger("github.com/alternative-storage/torus", "block")

type Snapshot struct {
	Name     string
	When     time.Time
	INodeRef []byte
}

type blockMetadata interface {
	torus.MetadataService

	Lock(lease int64) error
	Unlock() error

	GetINode() (torus.INodeRef, error)
	SyncINode(torus.INodeRef) error

	CreateBlockVolume(vol *models.Volume) error
	DeleteVolume() error

	SaveSnapshot(name string) error
	GetSnapshots() ([]Snapshot, error)
	DeleteSnapshot(name string) error
}

func createBlockMetadata(mds torus.MetadataService, name string, vid torus.VolumeID) (blockMetadata, error) {
	switch mds.Kind() {
	case torus.EtcdMetadata:
		return createBlockEtcdMetadata(mds, name, vid)
	case torus.TempMetadata:
		return createBlockTempMetadata(mds, name, vid)
	default:
		return nil, errors.New("unimplemented for this kind of metadata")
	}
}
