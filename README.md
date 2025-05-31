# Online Shop - Comprehensive E-commerce Platform

A modern, scalable e-commerce platform built with Go, implementing Domain-Driven Design (DDD), Command Query Responsibility Segregation (CQRS), and microservices architecture with advanced features including monitoring, queue systems, and automated testing.

## Architecture

### Technologies Used

- **Backend**: Go 1.21
- **Architecture Patterns**: DDD (Domain-Driven Design), CQRS (Command Query Responsibility Segregation)
- **Database**: PostgreSQL with GORM
- **Cache**: Redis
- **Search**: Elasticsearch
- **Communication**: gRPC, REST API
- **Authentication**: JWT (JSON Web Tokens)
- **Payment**: Midtrans Payment Gateway
- **Web Framework**: Gin
- **Containerization**: Docker & Docker Compose

### Project Structure

```
online-shop/
├── cmd/                    # Application entry points
│   ├── api/               # REST API server
│   ├── grpc/              # gRPC server
│   └── worker/            # Background workers
├── internal/              # Private application code
│   ├── domain/            # Domain models and business logic
│   │   ├── user/          # User domain
│   │   ├── product/       # Product domain
│   │   ├── order/         # Order domain
│   │   └── payment/       # Payment domain
│   ├── application/       # Application services (CQRS)
│   │   ├── commands/      # Command handlers
│   │   ├── queries/       # Query handlers
│   │   └── handlers/      # Application handlers
│   ├── infrastructure/    # External dependencies
│   │   ├── database/      # Database repositories
│   │   ├── redis/         # Redis cache
│   │   ├── elasticsearch/ # Search functionality
│   │   ├── grpc/          # gRPC services
│   │   └── payment/       # Payment providers
│   └── interfaces/        # Controllers and adapters
│       ├── http/          # HTTP handlers
│       ├── grpc/          # gRPC handlers
│       └── events/        # Event handlers
├── pkg/                   # Shared packages
│   ├── jwt/               # JWT utilities
│   ├── logger/            # Logging utilities
│   └── config/            # Configuration management
├── proto/                 # Protocol buffer definitions
├── migrations/            # Database migrations
├── docker-compose.yml     # Docker services
├── Dockerfile            # Application container
└── config.yaml           # Configuration file
```

## Features

### Core Features

1. **User Management**
   - User registration and authentication
   - JWT-based authorization
   - Role-based access control (Customer, Admin, Merchant)
   - Profile management

2. **Product Management**
   - Product catalog with categories
   - Product search with Elasticsearch
   - Stock management
   - Image support

3. **Order Management**
   - Shopping cart functionality
   - Order creation and tracking
   - Order status management
   - Order cancellation

4. **Payment Integration**
   - Midtrans payment gateway integration
   - Multiple payment methods support
   - Payment webhook handling
   - Refund processing

5. **Search & Discovery**
   - Full-text search with Elasticsearch
   - Category-based filtering
   - Price range filtering
   - Merchant filtering

### Technical Features

1. **CQRS Implementation**
   - Separate command and query models
   - Command handlers for write operations
   - Query handlers for read operations

2. **Domain-Driven Design**
   - Rich domain models
   - Domain services
   - Repository pattern
   - Aggregate roots

3. **Caching Strategy**
   - Redis for session management
   - Product and user caching
   - Cache invalidation strategies

4. **API Design**
   - RESTful API endpoints
   - gRPC services for internal communication
   - Comprehensive error handling
   - Request validation

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose
- PostgreSQL (if running locally)
- Redis (if running locally)
- Elasticsearch (if running locally)

### Installation

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd online-shop
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Configure the application**
   - Copy `config.yaml.example` to `config.yaml`
   - Update configuration values as needed
   - Set environment variables for sensitive data

4. **Run with Docker Compose (Recommended)**
   ```bash
   docker-compose up -d
   ```

5. **Or run locally**
   ```bash
   # Start dependencies
   docker-compose up -d postgres redis elasticsearch
   
   # Run the application
   go run cmd/api/main.go
   ```

### Configuration

The application can be configured through:

1. **Configuration file** (`config.yaml`)
2. **Environment variables** (takes precedence over config file)

Key configuration sections:

- `server`: HTTP server settings
- `database`: PostgreSQL connection settings
- `redis`: Redis connection settings
- `elasticsearch`: Elasticsearch connection settings
- `jwt`: JWT token settings
- `midtrans`: Payment gateway settings
- `grpc`: gRPC server settings

### Environment Variables

```bash
# Database
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres
DATABASE_DBNAME=online_shop
DATABASE_SSLMODE=disable

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Elasticsearch
ELASTICSEARCH_URL=http://localhost:9200

# JWT
JWT_SECRET_KEY=your-super-secret-jwt-key-here
JWT_EXPIRY_HOURS=24

# Midtrans
MIDTRANS_SERVER_KEY=your-midtrans-server-key
MIDTRANS_CLIENT_KEY=your-midtrans-client-key
MIDTRANS_ENVIRONMENT=sandbox

# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=12000

# gRPC
GRPC_HOST=0.0.0.0
GRPC_PORT=12001
```

## API Documentation

### Authentication Endpoints

- `POST /api/v1/users/register` - User registration
- `POST /api/v1/users/login` - User login
- `GET /api/v1/users/profile` - Get user profile (authenticated)
- `PUT /api/v1/users/profile` - Update user profile (authenticated)
- `PUT /api/v1/users/password` - Change password (authenticated)

### Product Endpoints

- `GET /api/v1/products/search` - Search products
- `GET /api/v1/products/:id` - Get product details
- `GET /api/v1/products/categories` - List categories

### Order Endpoints

- `POST /api/v1/orders` - Create order (authenticated)
- `GET /api/v1/orders` - Get user orders (authenticated)
- `GET /api/v1/orders/:id` - Get order details (authenticated)
- `PUT /api/v1/orders/:id/cancel` - Cancel order (authenticated)

### Payment Endpoints

- `POST /api/v1/payments/webhook` - Payment webhook (Midtrans)

### Example Requests

#### User Registration
```bash
curl -X POST http://localhost:12000/api/v1/users/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123",
    "first_name": "John",
    "last_name": "Doe",
    "phone": "+1234567890"
  }'
```

#### User Login
```bash
curl -X POST http://localhost:12000/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

#### Search Products
```bash
curl "http://localhost:12000/api/v1/products/search?q=laptop&limit=10&offset=0"
```

#### Create Order
```bash
curl -X POST http://localhost:12000/api/v1/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "items": [
      {
        "product_id": "product-uuid",
        "quantity": 2
      }
    ],
    "shipping_address": {
      "street": "123 Main St",
      "city": "Jakarta",
      "state": "DKI Jakarta",
      "postal_code": "12345",
      "country": "Indonesia"
    }
  }'
```

## Development

### Running Tests

```bash
go test ./...
```

### Building the Application

```bash
go build -o bin/api cmd/api/main.go
```

### Database Migrations

The application automatically runs migrations on startup. Manual migration can be done by:

```bash
go run cmd/migrate/main.go
```

### Adding New Features

1. **Domain Layer**: Add new domain models in `internal/domain/`
2. **Application Layer**: Add commands/queries in `internal/application/`
3. **Infrastructure Layer**: Add repositories and external services in `internal/infrastructure/`
4. **Interface Layer**: Add HTTP handlers in `internal/interfaces/http/`

## Deployment

### Docker Deployment

1. **Build the image**
   ```bash
   docker build -t online-shop:latest .
   ```

2. **Run with Docker Compose**
   ```bash
   docker-compose up -d
   ```

### Production Considerations

1. **Security**
   - Use strong JWT secret keys
   - Enable HTTPS/TLS
   - Implement rate limiting
   - Use secure database connections

2. **Performance**
   - Configure connection pooling
   - Implement proper caching strategies
   - Use CDN for static assets
   - Monitor application metrics

3. **Scalability**
   - Use load balancers
   - Implement horizontal scaling
   - Use database read replicas
   - Consider microservice decomposition

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support and questions, please open an issue in the repository or contact the development team.