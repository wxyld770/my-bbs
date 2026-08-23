package search

import (
	"time"

	"my-bbs/internal/handler"
	"my-bbs/internal/repository/gormrepo"
	"my-bbs/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Module wires the read-only global-search use case.
type Module struct {
	Handler *handler.SearchHandler
}

func Initialize(db *gorm.DB, timeout time.Duration) *Module {
	searchRepo := gormrepo.NewSearchReader(db)
	userRepo := gormrepo.NewUserRepository(db)
	commentRepo := gormrepo.NewCommentRepository(db)
	likeRepo := gormrepo.NewLikeRepository(db)
	svc := service.NewSearchService(searchRepo, userRepo, commentRepo, likeRepo)
	hdl := handler.NewSearchHandler(svc, timeout)
	return &Module{Handler: hdl}
}

// Register implements router.RouteRegister.
func (m *Module) Register(r *gin.RouterGroup) {
	r.GET("/search", m.Handler.Search)
}
