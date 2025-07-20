# init
go mod init github.com/username/project
go mod init <module-name>


# run
go run ./cmd/api (run all .go file under api)


# packages
## chi
(lightweight router)
go get -u github.com/go-chi/chi/v5

## CORs
(enable frontend to call api)
go get github.com/rs/cors

## air
(hot reloading)
go install github.com/air-verse/air@latest
air init
modify .air.toml
air

## godotenv
go get github.com/lpernett/godotenv

## validator
go get github.com/go-playground/validator/v10

## MongoDB
go get go.mongodb.org/mongo-driver/mongo
go get go.mongodb.org/mongo-driver/mongo/options
remember to whitelist the IP address (later on virtual server)

## mailer
go get github.com/sendgrid/sendgrid-go

## jwt
go get -u github.com/golang-jwt/jwt/v5

## AWS SDK
go get github.com/aws/aws-sdk-go-v2
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/service/s3


# omitempty
used for fields that are optional or auto-generated

# bson
all struct put/get info from mongoDB need to add `json...bson...`, so it can decode from db

# go functions
## *(pointer), &(dereferenair
ce)
generally ok to pass in value or reference (pointer),
but pass in reference is more efficient by saving memory - no copy created

## if
(write err as a 'if' initialization, so 'err' blocked inside if 
and can be redeclared outside )
if <initialization>; <condition> {
// code to execute if condition is true
}
if err := godotenv.Load(); err != nil {
log.Println("")
}

## make
make -> initialize and allocate memory for slices, maps, and channels.
make(map[key_type]value_type)
make([]int) -> append dynamically, least efficient
make([]int,5) -> len, cap = 5, all values initialed to 0, assign via indices, append will be make len+1, cap*2
make([]int, 0, 5) -> len=0, cap=5, pre-allocate memory to avoid reallocations, append, efficient

## fmt.ErrorF
(return error with format/context)
%v	value in a default format
    when printing structs, (%+v) adds field names
%w for error vales (wrap error)
%s for string
%d for decimal
## log.Println
(for non-critical logging, such is informational messages, warning, debugging...)
## log,Fatal
log the critical error and terminate the program with exit code

## context
a package to control timeouts, cancellations, or pass request-scoped values
commonly used for managing long-running operations that need to be canceled or timed out, 
such as database queries, HTTP requests...

context.TODO() -> placeholder with no specific context
context.WithTimeout(context.Background(), 10*time.Second) -> creates a new context with a deadline

## ...
ellipsis (...) in a function parameter list indicates a variadic parameter, 
meaning the function can accept zero or more values of the specified type.

## bson
"set" -> updates or creates a field with a specified value, replace if already exists 
        (use it for array will erase the whole array)
"addToSet" -> adds a value to an array only if it doesn’t already exist (ensures uniqueness)


# Authentication vs Authorization
Authentication -> verifies a user's identity/exists (who they are)
Authorization -> role and access/permits after being authenticated (what they can do)

## uuid (to generate plain token)
go get github.com/google/uuid

## SHA-256 (hash token) vs JWT token
SHA-256 (Secure Hash Algorithm) – hashing for storage
fixed length
same input to same output, irreversible hash
for token/API key storage in db

JWT (JSON Web Token) – authentication & authorization
JSON based, structured data, stateless token
store data (user roles, exp time) inside JWT, short-lived, stateless authentication (no db look up) 
for user/API authentication

## flat RBAC(Role-Based Access Control) vs Role precedence
Not using role precedence in this project since no role inheritance nor multiple roles per user



# TODO
autocompleairte mentions
mentions notification

Test AI image generator from frontend, unlock api (google account)
User activate front end url, need to set 'isSandbox' in mailer.go to false