package handler

import (
	httpReq "my-bbs/internal/handler/httprequest"
	httpResp "my-bbs/internal/handler/httpresponse"
	"my-bbs/internal/middleware"
	"my-bbs/internal/service"
	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/response"

	"github.com/gin-gonic/gin"
)

type MessageHandler struct {
	messageService *service.MessageService
}

func NewMessageHandler(messageService *service.MessageService) *MessageHandler {
	return &MessageHandler{messageService: messageService}
}

func (h *MessageHandler) CreateMessage(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.ReportError(c, bizerr.ErrUnauthorized)
		return
	}
	var req httpReq.CreateMessageRequest
	if err := httpReq.BindJSON(c, &req); err != nil {
		response.ReportError(c, err)
		return
	}
	if err := h.messageService.CreateMessage(c.Request.Context(), userID, req.Content); err != nil {
		response.ReportError(c, err)
		return
	}
	response.OKMsg(c, "留言提交成功")
}

func (h *MessageHandler) ListMyMessages(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.ReportError(c, bizerr.ErrUnauthorized)
		return
	}
	q, err := httpReq.BindPageQuery(c)
	if err != nil {
		response.ReportError(c, err)
		return
	}
	result, err := h.messageService.ListMyMessages(c.Request.Context(), userID, q)
	if err != nil {
		response.ReportError(c, err)
		return
	}
	response.OK(c, httpResp.NewMessagePageResponse(result))
}

func (h *MessageHandler) ListAllMessages(c *gin.Context) {
	actorID, ok := middleware.GetUserID(c)
	if !ok {
		response.ReportError(c, bizerr.ErrUnauthorized)
		return
	}
	q, err := httpReq.BindPageQuery(c)
	if err != nil {
		response.ReportError(c, err)
		return
	}
	result, err := h.messageService.ListAllMessages(c.Request.Context(), actorID, q)
	if err != nil {
		response.ReportError(c, err)
		return
	}
	response.OK(c, httpResp.NewMessagePageResponse(result))
}
