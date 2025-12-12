package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"newbeeHao.com/openapi/v2/common/response"
	"newbeeHao.com/openapi/v2/middleware/logger"
)

type BaseController struct {
	Logger logger.Logger
}

func (bc *BaseController) BindJSON(c *gin.Context, req interface{}) error {
	if err := c.ShouldBindJSON(req); err != nil {
		logger.Warnf(c.Request.Context(), "BindJSON error: %v", err)
		return err
	}
	return nil
}

func (bc *BaseController) BindQuery(c *gin.Context, req interface{}) error {
	if err := c.ShouldBindQuery(req); err != nil {
		logger.Warnf(c.Request.Context(), "ShouldBindQuery error: %v", err)
		return err
	}
	return nil
}

func (bc *BaseController) HandleError(c *gin.Context, status int, err error) {
	requestID := c.Request.Context().Value(constant.RequestIDKey).(string)
	errorResponse := &response.ErrorResponse{
		Code:    status,
		Message: err.Error(),
		Status:  http.StatusText(status),
	}
	response := response.Response{
		Error:     errorResponse,
		RequestId: requestID,
	}
	logger.Warnf(c.Request.Context(), "ErrorResponse: %#v", response)
	c.JSON(status, response)
}

func (bc *BaseController) SendResponse(c *gin.Context, data interface{}) {
	requestID := c.Request.Context().Value(constant.RequestIDKey).(string)
	response := response.Response{
		Result:    data,
		RequestId: requestID,
	}
	logger.Infof(c.Request.Context(), "Response: %#v", util.InterfaceToJson(response))
	c.JSON(http.StatusOK, response)
}
