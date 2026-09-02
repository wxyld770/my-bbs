package httprequest

type RegisterRequest struct {
	Username   string `json:"username" binding:"required,min=3,max=64"`
	Password   string `json:"password" binding:"required,min=6,max=64"`
	Nickname   string `json:"nickname" binding:"max=64"`
	InviteCode string `json:"invite_code" binding:"required"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=64"`
}

type UpdateProfileRequest struct {
	Nickname     string `json:"nickname" binding:"max=64"`
	Introduction string `json:"introduction" binding:"max=1024"`
}

type UpdateAvatarRequest struct {
	AvatarURL *string `json:"avatar_url" binding:"required,max=2048"`
}
