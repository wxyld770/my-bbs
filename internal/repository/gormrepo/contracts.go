package gormrepo

import "my-bbs/internal/repository"

// 编译期断言：GORM Adapter 必须完整实现应用层定义的 Repository Port。
var (
	_ repository.UserRepository    = (*UserRepository)(nil)
	_ repository.UserReader        = (*UserRepository)(nil)
	_ repository.PostRepository    = (*PostRepository)(nil)
	_ repository.PostReader        = (*PostRepository)(nil)
	_ repository.HotPostReader     = (*HotPostReader)(nil)
	_ repository.CommentRepository = (*CommentRepository)(nil)
	_ repository.CommentCounter    = (*CommentRepository)(nil)
	_ repository.LikeRepository    = (*LikeRepository)(nil)
	_ repository.LikeReader        = (*LikeRepository)(nil)
	_ repository.SearchReader      = (*SearchReader)(nil)
)
