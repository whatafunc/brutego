### brutego - stage1

Task attached [Anti Bruteforce](./01-anti-bruteforce.md)

### Tests

go test -race -count=100 ./internal/bucket 
ok      github.com/whatafunc/brutego/internal/bucket    2.087s

### APIs Tests 
# For CheckAuth (expects login + password + ip):
curl -X POST http://localhost:8080/v1/auth/check \
  -H "Content-Type: application/json" \
  -d '{
    "login": "testuser",
    "password": "testpass123",
    "ip": "192.168.1.100"
  }'
{"ok":true}
...
{"ok":false}

# For ResetBucket (expects login + ip only):

curl -X POST http://localhost:8080/v1/bucket/reset \
  -H "Content-Type: application/json" \
  -d '{}'
{"code":3, "message":"login or ip must be provided", "details":[]}%

---

curl -X POST http://localhost:8080/v1/bucket/reset \
  -H "Content-Type: application/json" \
  -d '{
    "login": "testuser",
    "password": "testpass123",
    "ip": "192.168.1.100"
  }'
{}

---

grpcurl -plaintext -d '{
  "login": "testuser",
  "password": "testpass123",
  "ip": "192.168.1.100"
}' localhost:8081 antibruteforce.v1.AntiBruteforceService/CheckAuth

{
  "ok": true
}

---

grpcurl -plaintext -d '{
  "login": "testuser",
  "password": "testpass123",
  "ip": "192.168.1.100"
}' localhost:50051 antibruteforce.v1.AntiBruteforceService/ResetBucket
Error invoking method "antibruteforce.v1.AntiBruteforceService/ResetBucket": error getting request data: message type antibruteforce.v1.ResetBucketRequest has no known field named password

---

grpcurl -plaintext -d '{
  "login": "testuser",
  "ip": "192.168.1.100"
}' localhost:50051 antibruteforce.v1.AntiBruteforceService/ResetBucket
{}

# AddToBlacklist (expects subnet field):

grpcurl -plaintext -d '{
  "subnet": "192.168.1.0/24"
}' localhost:50051 antibruteforce.v1.AntiBruteforceService/AddToBlacklist
{}

---

grpcurl -plaintext -d '{
  "login": "testuser",      
  "password": "testpass123",                                             
  "ip": "192.168.1.100"
}' localhost:50051 antibruteforce.v1.AntiBruteforceService/CheckAuth
{}

grpcurl -plaintext -d '{
  "login": "testuser",
  "ip": "192.168.1.100"
}' localhost:50051 antibruteforce.v1.AntiBruteforceService/ResetBucket
{}

grpcurl -plaintext -d '{
  "login": "testuser",
  "password": "testpass123",
  "ip": "192.168.1.100"                                               
}' localhost:50051 antibruteforce.v1.AntiBruteforceService/CheckAuth
{}

grpcurl -plaintext -d '{
  "login": "testuser",
  "password": "testpass123",
  "ip": "192.168.1.100"
}' localhost:50051 antibruteforce.v1.AntiBruteforceService/CheckAuth
{}

grpcurl -plaintext -d '{
  "login": "testuser",
  "password": "testpass123",
  "ip": "192.168.1.100"
}' localhost:50051 antibruteforce.v1.AntiBruteforceService/CheckAuth
{}

grpcurl -plaintext -d '{
  "login": "testuser",
  "password": "testpass123",
  "ip": "192.168.5.100"
}' localhost:50051 antibruteforce.v1.AntiBruteforceService/CheckAuth
{
  "ok": true
}

# RemoveFromBlacklist (expects subnet field):

grpcurl -plaintext -d '{
  "subnet": "192.168.1.0/24"
}' localhost:50051 antibruteforce.v1.AntiBruteforceService/RemoveFromBlacklist
{}

grpcurl -plaintext -d '{
  "subnet": "192.168.5.0/24"
}' localhost:50051 antibruteforce.v1.AntiBruteforceService/RemoveFromBlacklist
ERROR:
  Code: NotFound
  Message: subnet not found: 192.168.5.0/24

grpcurl -plaintext -d '{
  "subnet": "192.168.1.0/24"
}' localhost:50051 antibruteforce.v1.AntiBruteforceService/RemoveFromBlacklist
{}

# AddToWhitelist (expects subnet field):

grpcurl -plaintext -d '{
  "subnet": "10.0.0.0/8"
}' localhost:50051 antibruteforce.v1.AntiBruteforceService/AddToWhitelist

grpcurl -plaintext -d '{
  "subnet": "192.168.1.0/24"
}' localhost:50051 antibruteforce.v1.AntiBruteforceService/AddToWhitelist

# RemoveFromWhitelist (expects subnet field):

grpcurl -plaintext -d '{
  "subnet": "10.0.0.0/8"
}' localhost:50051 antibruteforce.v1.AntiBruteforceService/RemoveFromWhitelist

grpcurl -plaintext -d '{
  "subnet": "192.168.1.0/24"
}' localhost:50051 antibruteforce.v1.AntiBruteforceService/RemoveFromWhitelist




# List of Expected Responses
ResetBucket: Returns {} (google.protobuf.Empty)

AddToBlacklist: Returns {} (google.protobuf.Empty)

RemoveFromBlacklist: Returns {} (google.protobuf.Empty)

AddToWhitelist: Returns {} (google.protobuf.Empty)

RemoveFromWhitelist: Returns {} (google.protobuf.Empty)

# List all methods
grpcurl -plaintext localhost:50051 list antibruteforce.v1.AntiBruteforceService
antibruteforce.v1.AntiBruteforceService.AddToBlacklist
antibruteforce.v1.AntiBruteforceService.AddToWhitelist
antibruteforce.v1.AntiBruteforceService.CheckAuth
antibruteforce.v1.AntiBruteforceService.RemoveFromBlacklist
antibruteforce.v1.AntiBruteforceService.RemoveFromWhitelist
antibruteforce.v1.AntiBruteforceService.ResetBucket


# Describe a specific method to see its request/response types
grpcurl -plaintext localhost:50051 describe antibruteforce.v1.AntiBruteforceService.ResetBucket
rpc ResetBucket ( .antibruteforce.v1.ResetBucketRequest ) returns ( .google.protobuf.Empty ) {
  option (.google.api.http) = { post: "/v1/bucket/reset", body: "*" };
}

...




