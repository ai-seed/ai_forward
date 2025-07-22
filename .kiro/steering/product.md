# AI API Gateway Product Overview

High-performance API gateway/proxy service for AI providers with comprehensive management capabilities.

## Core Features

- **Multi-Provider Support**: Routes requests to OpenAI, Anthropic, and other AI service providers
- **Quota Management**: API key-level usage quotas and rate limiting with async processing
- **Billing System**: Detailed usage logging and cost calculation with configurable precision
- **Load Balancing**: Round-robin, weighted, and least-connections strategies with failover
- **Model Management**: Maintains model information, pricing data, and provider compatibility
- **Function Calling**: Integrated search capabilities (Google, Bing, DuckDuckGo, etc.)
- **High Availability**: Health checks, automatic recovery, and distributed locking
- **Caching**: Redis-based multi-layer caching for optimal performance

## Target Use Cases

- API aggregation and management for AI services
- Cost tracking and billing for AI API usage
- Rate limiting and quota enforcement
- Load balancing across multiple AI providers
- Centralized authentication and authorization
- Function calling with web search integration

## Architecture Philosophy

- Clean Architecture with strict layer separation
- No foreign key constraints - application-layer relationship management
- High performance and scalability focus
- Comprehensive caching strategy with configurable TTLs
- Async processing for quota consumption
- JWT-based authentication with refresh tokens