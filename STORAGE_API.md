# Storage API Documentation

The Storage API provides comprehensive file management with support for public and private files, upload/download capabilities, and file metadata management.

## Features

- ✅ File upload with size validation
- ✅ Public and private file visibility
- ✅ File download
- ✅ File listing with pagination
- ✅ File metadata management
- ✅ Soft delete functionality
- ✅ Access control for private files
- ✅ Local storage (S3 ready for future implementation)

## Configuration

Storage settings are configured via environment variables in your `.env` file:

```bash
# Storage Configuration
STORAGE_TYPE=local              # Options: local, s3
STORAGE_BASE_PATH=./uploads     # Local storage directory
MAX_FILE_SIZE=10485760         # 10MB in bytes

# S3 Configuration (optional, for future use)
S3_BUCKET=
S3_REGION=us-east-1
S3_ACCESS_KEY=
S3_SECRET_KEY=
```

## API Endpoints

### Base URL
```
http://localhost:8080/api/v1/storage
```

### 1. Upload File

**Endpoint:** `POST /storage/upload`

**Authentication:** Required (Bearer Token)

**Content-Type:** `multipart/form-data`

**Form Fields:**
- `file` (required): The file to upload
- `visibility` (required): Either "public" or "private"
- `metadata` (optional): JSON string with custom metadata

**Example with cURL:**
```bash
curl -X POST http://localhost:8080/api/v1/storage/upload \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -F "file=@/path/to/document.pdf" \
  -F "visibility=private" \
  -F "metadata={\"category\":\"documents\",\"tags\":[\"important\"]}"
```

**Example Response:**
```json
{
  "success": true,
  "message": "File uploaded successfully",
  "data": {
    "file": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "user_id": "user-123",
      "file_name": "550e8400-e29b-41d4-a716-446655440000.pdf",
      "original_name": "document.pdf",
      "mime_type": "application/pdf",
      "size": 524288,
      "storage_type": "local",
      "visibility": "private",
      "metadata": {
        "category": "documents",
        "tags": ["important"]
      },
      "download_url": "http://localhost:8080/api/v1/storage/files/550e8400-e29b-41d4-a716-446655440000/download",
      "created_at": "2025-11-06T12:00:00Z",
      "updated_at": "2025-11-06T12:00:00Z"
    }
  }
}
```

---

### 2. List Files

**Endpoint:** `GET /storage/files`

**Authentication:** Optional (shows public files + user's private files if authenticated)

**Query Parameters:**
- `visibility` (optional): Filter by "public" or "private"
- `page` (optional): Page number (default: 1)
- `limit` (optional): Items per page (default: 20, max: 100)

**Example with cURL:**
```bash
# List all accessible files (public + your private files)
curl http://localhost:8080/api/v1/storage/files?page=1&limit=10

# List only public files
curl http://localhost:8080/api/v1/storage/files?visibility=public

# List only your private files (requires authentication)
curl http://localhost:8080/api/v1/storage/files?visibility=private \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

**Example Response:**
```json
{
  "success": true,
  "message": "Files retrieved successfully",
  "data": {
    "files": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "file_name": "550e8400-e29b-41d4-a716-446655440000.pdf",
        "original_name": "document.pdf",
        "mime_type": "application/pdf",
        "size": 524288,
        "storage_type": "local",
        "visibility": "private",
        "download_url": "http://localhost:8080/api/v1/storage/files/550e8400-e29b-41d4-a716-446655440000/download",
        "created_at": "2025-11-06T12:00:00Z",
        "updated_at": "2025-11-06T12:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "limit": 10,
    "total_pages": 1
  }
}
```

---

### 3. Get File Metadata

**Endpoint:** `GET /storage/files/{id}`

**Authentication:** Optional (required for private files)

**Example with cURL:**
```bash
# Get public file metadata
curl http://localhost:8080/api/v1/storage/files/550e8400-e29b-41d4-a716-446655440000

# Get private file metadata (requires authentication)
curl http://localhost:8080/api/v1/storage/files/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

---

### 4. Download File

**Endpoint:** `GET /storage/files/{id}/download`

**Authentication:** Optional (required for private files)

**Example with cURL:**
```bash
# Download public file
curl -O http://localhost:8080/api/v1/storage/files/550e8400-e29b-41d4-a716-446655440000/download

# Download private file (requires authentication)
curl -O http://localhost:8080/api/v1/storage/files/550e8400-e29b-41d4-a716-446655440000/download \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"

# Download with custom filename
curl http://localhost:8080/api/v1/storage/files/550e8400-e29b-41d4-a716-446655440000/download \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -o my-document.pdf
```

---

### 5. Update File

**Endpoint:** `PUT /storage/files/{id}`

**Authentication:** Required (must be file owner)

**Content-Type:** `application/json`

**Request Body:**
```json
{
  "visibility": "public",
  "metadata": "{\"updated\":true}"
}
```

**Example with cURL:**
```bash
curl -X PUT http://localhost:8080/api/v1/storage/files/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "visibility": "public",
    "metadata": "{\"category\":\"public-docs\"}"
  }'
```

**Example Response:**
```json
{
  "success": true,
  "message": "File updated successfully",
  "data": {
    "file": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "visibility": "public",
      "metadata": {
        "category": "public-docs"
      },
      "updated_at": "2025-11-06T13:00:00Z"
    }
  }
}
```

---

### 6. Delete File

**Endpoint:** `DELETE /storage/files/{id}`

**Authentication:** Required (must be file owner)

**Example with cURL:**
```bash
curl -X DELETE http://localhost:8080/api/v1/storage/files/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

**Example Response:**
```json
{
  "success": true,
  "message": "File deleted successfully",
  "data": null
}
```

---

## Access Control

### Public Files
- Can be uploaded by authenticated users
- Can be viewed and downloaded by anyone (no authentication required)
- Can only be updated/deleted by the owner

### Private Files
- Can be uploaded by authenticated users
- Can only be viewed and downloaded by the owner
- Can only be updated/deleted by the owner

---

## File Size Limits

The default maximum file size is **10MB** (10,485,760 bytes). You can configure this in your `.env` file:

```bash
MAX_FILE_SIZE=20971520  # 20MB
```

---

## Error Responses

### 400 Bad Request
```json
{
  "success": false,
  "message": "No file provided"
}
```

### 401 Unauthorized
```json
{
  "success": false,
  "message": "Authentication required"
}
```

### 403 Forbidden
```json
{
  "success": false,
  "message": "Access denied"
}
```

### 404 Not Found
```json
{
  "success": false,
  "message": "File not found"
}
```

### 413 Request Entity Too Large
```json
{
  "success": false,
  "message": "File too large",
  "data": "file size exceeds maximum allowed size of 10485760 bytes"
}
```

### 422 Validation Error
```json
{
  "success": false,
  "message": "Validation failed",
  "errors": [
    {
      "code": "VALIDATION_ERROR",
      "message": "Key: 'UploadRequest.Visibility' Error:Field validation for 'Visibility' failed on the 'oneof' tag",
      "field": ""
    }
  ]
}
```

---

## Testing with Postman

### Step 1: Get Access Token
1. POST to `/api/v1/users/login` or `/api/v1/oauth2/token`
2. Copy the `access_token` from the response

### Step 2: Upload File
1. Create a new POST request to `/api/v1/storage/upload`
2. Set Authorization header: `Bearer YOUR_ACCESS_TOKEN`
3. Go to Body → form-data
4. Add fields:
   - `file` (File type) → Select your file
   - `visibility` (Text) → "private" or "public"
   - `metadata` (Text, optional) → JSON string

### Step 3: List Files
1. Create a new GET request to `/api/v1/storage/files`
2. Add Authorization header if viewing private files
3. Add query parameters: `page=1&limit=10`

### Step 4: Download File
1. Create a new GET request to `/api/v1/storage/files/{file-id}/download`
2. Add Authorization header if downloading private files
3. Click Send and Download

---

## File Structure

```
/Applications/MAMP/htdocs/gogin/
├── internal/
│   └── modules/
│       └── storage/
│           ├── dto.go        # Request/Response structures
│           ├── service.go    # Business logic
│           ├── handlers.go   # HTTP handlers
│           └── module.go     # Module registration
├── uploads/                  # Local file storage (gitignored)
└── STORAGE_API.md           # This documentation
```

---

## Database Schema

The `files` table structure:

```sql
CREATE TABLE files (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36),
    file_name VARCHAR(255) NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(100),
    size BIGINT NOT NULL,
    path TEXT NOT NULL,
    storage_type VARCHAR(20) DEFAULT 'local',
    visibility VARCHAR(20) DEFAULT 'private',
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);
```

---

## Future Enhancements

- ✅ Local storage (implemented)
- ⏳ S3 storage support
- ⏳ File versioning
- ⏳ Folder/directory support
- ⏳ File sharing with expiring links
- ⏳ Thumbnail generation for images
- ⏳ Virus scanning integration

---

## Support

For issues or questions, please check:
- Swagger documentation: `http://localhost:8080/swagger/index.html`
- API logs in your terminal
- Database files table for stored file records
