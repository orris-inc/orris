package repository

import (
	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// NodeGroupRepositoryImpl implements the node.NodeGroupRepository interface
type NodeGroupRepositoryImpl struct {
	db                    *gorm.DB
	mapper                mappers.NodeGroupMapper
	nodeMapper            mappers.NodeMapper
	trojanConfigRepo      *TrojanConfigRepository
	shadowsocksConfigRepo *ShadowsocksConfigRepository
	logger                logger.Interface
}

// NewNodeGroupRepository creates a new node group repository instance
func NewNodeGroupRepository(db *gorm.DB, logger logger.Interface) node.NodeGroupRepository {
	return &NodeGroupRepositoryImpl{
		db:                    db,
		mapper:                mappers.NewNodeGroupMapper(),
		nodeMapper:            mappers.NewNodeMapper(),
		trojanConfigRepo:      NewTrojanConfigRepository(db, logger),
		shadowsocksConfigRepo: NewShadowsocksConfigRepository(db, logger),
		logger:                logger,
	}
}
