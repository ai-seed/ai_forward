package clients

import (
	"context"
	"net/url"

	"ai-api-gateway/internal/infrastructure/logger"
)

// GenericForwardingService 通用转发服务
type GenericForwardingService struct {
	proxyClient GenericProxyClient
	logger      logger.Logger
}

// NewGenericForwardingService 创建通用转发服务
func NewGenericForwardingService(proxyClient GenericProxyClient, logger logger.Logger) *GenericForwardingService {
	return &GenericForwardingService{
		proxyClient: proxyClient,
		logger:      logger,
	}
}

// ForwardRequest 转发普通请求
func (s *GenericForwardingService) ForwardRequest(ctx context.Context, method, path string, headers map[string]string, body []byte, query url.Values) (*GenericProxyResponse, error) {
	return s.proxyClient.ForwardRequest(ctx, method, path, headers, body, query)
}

// ForwardStreamRequest 转发流式请求
func (s *GenericForwardingService) ForwardStreamRequest(ctx context.Context, method, path string, headers map[string]string, body []byte, query url.Values) (*StreamResponse, error) {
	return s.proxyClient.ForwardStreamRequest(ctx, method, path, headers, body, query)
}
