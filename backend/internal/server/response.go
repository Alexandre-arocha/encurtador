package server

import "github.com/gin-gonic/gin"

// errorBody é o envelope padrão de erro da API.
type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// respondError responde um erro no envelope padrão.
func (s *Server) respondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, errorBody{Error: errorDetail{Code: code, Message: message}})
}
