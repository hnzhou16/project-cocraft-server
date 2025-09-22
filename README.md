# CoCraft Server - Backend API for Social Home Design Platform

CoCraft Server is the backend API that powers the CoCraft social platform, providing secure authentication, data management, AI-powered image generation, and real-time features for contractors, manufacturers, designers, and homeowners to collaborate on home design projects.

## Features

### Authentication & Security

- **JWT Authentication**: Secure token-based authentication with HTTP-only cookies
- **User Registration**: Email-based account creation with activation links via SendGrid
- **Role-Based Access Control**: Five distinct user roles (Admin, Contractor, Manufacturer, Designer, Homeowner) with granular permissions
- **Password Security**: Bcrypt password hashing and validation
- **Token Validation**: Middleware for protecting routes and validating user sessions

### User Management

- **User Profiles**: Complete user profile management with contact information and ratings
- **User Roles**: Support for different user types with role-specific permissions
- **Account Activation**: Email-based account activation system with TTL tokens
- **User Discovery**: Admin endpoints for user management and discovery
- **Rating System**: User rating and review system for building trust and credibility

### Social Features

- **Follow System**: Users can follow/unfollow other users with follower/following counts
- **User Relationships**: Track social connections and build professional networks
- **Profile Views**: Public profile viewing with role-based information display

### Content Management

- **Post Creation**: Create posts with text content, tags, and user mentions
- **Post Management**: Full CRUD operations for posts with ownership validation
- **Post Interactions**: Like/unlike functionality with real-time like counts
- **Post Discovery**: Get posts by user, andd search functionality

### Feed & Discovery

- **Personalized Feed**: Algorithm-driven feed based on user follows and interactions
- **Public Feed**: Limited public access for non-authenticated users
- **Search Functionality**: Full-text search across posts with filtering capabilities
- **Pagination**: Cursor-based pagination for efficient data loading

### Comments System

- **Nested Comments**: Support for comments and replies with parent-child relationships
- **Comment Management**: Create and retrieve comments with user information
- **Comment Counting**: Automatic comment count updates on posts

### Review System

- **User Reviews**: Users can leave reviews and ratings for other users
- **Review Management**: Create and delete reviews with rating calculations
- **Trust Building**: Aggregate ratings to build user credibility

### AI Integration

- **Image Generation**: OpenAI DALL-E integration for AI-powered image creation
- **History Retrieval**: Support for retrieving image history

### Cloud Storage

- **AWS S3 Integration**: Secure image upload and storage using AWS S3
- **Presigned URLs**: Secure, time-limited upload URLs for client-side uploads
- **Image Management**: Upload and delete operations with proper access control

### Email Services

- **SendGrid Integration**: Professional email delivery for account activation
- **Template System**: Structured email templates
- **Activation Emails**: Automated account activation email workflow

### API Architecture

- **RESTful Design**: Clean, RESTful API endpoints with proper HTTP methods
- **Middleware Stack**: Comprehensive middleware for logging, CORS, timeouts, and security
- **Error Handling**: Structured error responses with proper HTTP status codes
- **Request Validation**: Input validation and sanitization for all endpoints
- **CORS Support**: Configurable CORS for cross-origin requests

### Database & Performance

- **MongoDB Integration**: NoSQL database with connection pooling and optimization
- **Transaction Support**: ACID transactions for data consistency
- **Indexing**: Optimized database indexes for query performance
- **Connection Management**: Efficient connection pooling and timeout handling

### Infrastructure

- **Graceful Shutdown**: Proper server shutdown handling with cleanup
- **Health Checks**: Health check endpoints for monitoring and load balancing
- **Environment Configuration**: Flexible configuration via environment variables
- **Logging**: Structured logging with Zap for production monitoring

## Tech Stack

- **Language**: Go
- **Web Framework**: Chi Router
- **Database**: MongoDB with official Go driver
- **Authentication**: JWT with golang-jwt/jwt/v5
- **Cloud Storage**: AWS SDK v2 for S3 integration
- **Email**: SendGrid for transactional emails
- **AI**: OpenAI API for image generation
- **Logging**: Uber Zap for structured logging
- **Security**: Bcrypt for password hashing, CORS middleware
- **Validation**: Go Playground Validator

## Project Structure

```
project-cocraft-server/
├── main/                     # Main application and HTTP handlers
│   ├── main.go              # Application entry point and configuration
│   ├── zz_api.go            # Route definitions and server setup
│   ├── auth.go              # Authentication handlers
│   ├── user.go              # User management handlers
│   ├── post.go              # Post management handlers
│   ├── comment.go           # Comment system handlers
│   ├── review.go            # Review system handlers
│   ├── feed.go              # Feed and discovery handlers
│   ├── image.go             # Image upload handlers
│   ├── ai.go                # AI image generation handlers
│   ├── health.go            # Health check handlers
│   ├── middleware.go        # Custom middleware functions
│   ├── validator.go         # Request validation logic
│   ├── json.go              # JSON response utilities
│   └── error.go             # Error handling utilities
├── internal/                # Internal packages
│   ├── ai/                  # AI integration
│   │   └── image.go         # OpenAI image generation client
│   ├── auth/                # Authentication system
│   │   ├── auth.go          # Authentication interface
│   │   └── jwt.go           # JWT implementation
│   ├── aws/                 # AWS integration
│   │   └── presigner.go     # S3 presigned URL generation
│   ├── db/                  # Database connection
│   │   └── db.go            # MongoDB connection management
│   ├── mailer/              # Email services
│   │   ├── mailer.go        # Email interface
│   │   ├── sendgrid.go      # SendGrid implementation
│   │   └── templates/       # Email templates
│   ├── security/            # Security utilities
│   │   ├── password.go      # Password hashing
│   │   └── role.go          # Role-based permissions
│   └── storage/             # Data access layer
│       ├── collection.go    # Storage interface and MongoDB setup
│       ├── user.go          # User data operations
│       ├── post.go          # Post data operations
│       ├── comment.go       # Comment data operations
│       ├── review.go        # Review data operations
│       ├── follow.go        # Follow relationship operations
│       ├── invite.go        # User invitation operations
│       └── pagination.go    # Cursor-based pagination
├── bin/                     # Compiled binaries
├── docs/                    # API documentation
├── scripts/                 # Build and deployment scripts
├── go.mod                   # Go module definition
├── go.sum                   # Go module checksums
├── .air.toml               # Hot reload configuration
└── .gitignore              # Git ignore rules
```

## API Endpoints

### Authentication
- `POST /authentication/user` - Register new user
- `POST /authentication/token` - Login and get JWT token
- `GET /authentication/validate` - Validate current token

### User Management
- `PUT /user/activate/{token}` - Activate user account
- `GET /user/me` - Get current user profile
- `GET /user/{userID}/profile` - Get user profile by ID
- `GET /user/{userID}/reviews` - Get user reviews
- `GET /user/admin` - Get all users (admin only)

### Social Features
- `GET /user/{userID}/following` - Get users being followed
- `GET /user/{userID}/follow-status` - Check follow status
- `POST /user/{userID}/follow` - Follow user
- `DELETE /user/{userID}/follow` - Unfollow user

### Posts
- `POST /post` - Create new post
- `GET /post/{postID}` - Get post by ID
- `GET /post/user/{userID}` - Get posts by user
- `PATCH /post/{postID}` - Update post (owner only)
- `DELETE /post/{postID}` - Delete post (owner only)
- `PATCH /post/{postID}/like` - Toggle like on post

### Comments
- `GET /post/{postID}/comment` - Get post comments
- `POST /post/{postID}/comment` - Create comment

### Reviews
- `POST /review/create-review` - Create user review
- `DELETE /review/{reviewID}/delete-review` - Delete review

### Feed & Discovery
- `GET /feed/public` - Get public feed
- `GET /feed/user` - Get personalized user feed
- `GET /feed/trending` - Get trending posts
- `GET /feed/search` - Search posts

### AI Features
- `POST /ai/generate-image` - Generate AI image
- `POST /ai/refine-image` - Refine existing AI image

### File Management
- `POST /user/upload-image` - Get presigned upload URL
- `DELETE /user/delete-image` - Delete uploaded image

### System
- `GET /health` - Health check endpoint


## Security Features

- **JWT Authentication**: Secure token-based authentication
- **Role-Based Access Control**: Granular permissions system
- **Password Hashing**: Bcrypt for secure password storage
- **CORS Protection**: Configurable cross-origin resource sharing
- **Request Validation**: Input sanitization and validation
- **Secure Headers**: Security middleware for HTTP headers
- **Rate Limiting**: Protection against abuse (configurable)

## Performance Optimizations

- **Connection Pooling**: Efficient database connection management
- **Cursor Pagination**: Memory-efficient data pagination
- **Indexing**: Optimized database queries with proper indexes
- **Graceful Shutdown**: Proper resource cleanup on shutdown